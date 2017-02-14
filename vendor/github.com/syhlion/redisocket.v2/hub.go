package redisocket

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/websocket"
)

type User interface {
	Trigger(event string, data []byte) (err error)
	Close()
}

type WebsocketOptional struct {
	WriteWait      time.Duration
	PongWait       time.Duration
	PingPeriod     time.Duration
	MaxMessageSize int64
	Upgrader       websocket.Upgrader
}

var (
	DefaultWebsocketOptional = WebsocketOptional{
		WriteWait:      10 * time.Second,
		PongWait:       60 * time.Second,
		PingPeriod:     (60 * time.Second * 9) / 10,
		MaxMessageSize: 512,
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}
)

var APPCLOSE = errors.New("APP_CLOSE")

type EventHandler func(event string, b []byte) ([]byte, error)

type ReceiveMsgHandler func([]byte) error

func NewSender(m *redis.Pool) (e *Sender) {

	return &Sender{
		redisManager: m,
	}
}

type Sender struct {
	redisManager *redis.Pool
}

func (s *Sender) GetChannels(channelPrefix string, pattern string) (channels []string, err error) {
	conn := s.redisManager.Get()
	defer conn.Close()
	channels, err = redis.Strings(conn.Do("keys", channelPrefix+":::"+pattern))
	return
}

func (s *Sender) Push(channelPrefix, event string, data []byte) (val int, err error) {
	conn := s.redisManager.Get()
	defer conn.Close()
	val, err = redis.Int(conn.Do("PUBLISH", channelPrefix+event, data))
	err = s.redisManager.Get().Flush()
	return
}

//NewApp It's create a Hub
func NewHub(m *redis.Pool, debug bool) (e *Hub) {

	l := log.New(os.Stdout, "[redisocket.v2]", log.Lshortfile|log.Ldate|log.Lmicroseconds)
	return &Hub{

		Config:       DefaultWebsocketOptional,
		redisManager: m,
		psc:          &redis.PubSubConn{m.Get()},
		RWMutex:      new(sync.RWMutex),
		subjects:     make(map[string]map[User]bool),
		subscribers:  make(map[User]map[string]bool),
		closeSign:    make(chan int),
		closeflag:    false,
		debug:        debug,
		log:          l,
	}

}
func (e *Hub) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (c *Client, err error) {
	ws, err := e.Config.Upgrader.Upgrade(w, r, responseHeader)
	c = &Client{
		ws:      ws,
		send:    make(chan []byte, 4096),
		RWMutex: new(sync.RWMutex),
		hub:     e,
		events:  make(map[string]EventHandler),
	}
	return
}

type Hub struct {
	ChannelPrefix string
	Config        WebsocketOptional
	psc           *redis.PubSubConn
	redisManager  *redis.Pool
	subjects      map[string]map[User]bool
	subscribers   map[User]map[string]bool
	closeSign     chan int
	closeflag     bool
	debug         bool
	*sync.RWMutex
	log *log.Logger
}

func (a *Hub) Ping() (err error) {
	_, err = a.redisManager.Get().Do("PING")
	if err != nil {
		return
	}
	return
}
func (a *Hub) logger(format string, v ...interface{}) {
	if a.debug {
		a.log.Printf(format, v...)
	}
}

func (a *Hub) Register(event string, c User) (err error) {
	a.Lock()

	defer a.Unlock()
	//observer map
	if m, ok := a.subscribers[c]; !ok {
		events := make(map[string]bool)
		events[event] = true
		a.subscribers[c] = events
	} else {
		m[event] = true
	}

	//event map
	if clients, ok := a.subjects[event]; !ok {
		clients := make(map[User]bool)
		clients[c] = true
		a.subjects[event] = clients
	} else {
		clients[c] = true
	}
	return
}

func (a *Hub) Unregister(event string, c User) (err error) {

	a.Lock()
	defer a.Unlock()

	//observer map
	if m, ok := a.subscribers[c]; ok {
		delete(m, event)
		if len(m) == 0 {
			delete(a.subscribers, c)
		}
	}
	//event map
	if m, ok := a.subjects[event]; ok {
		delete(m, c)
		if len(m) == 0 {
			delete(a.subjects, event)
		}
	}

	return
}

func (a *Hub) UnregisterAll(c *Client) {
	a.Lock()
	m, ok := a.subscribers[c]
	a.Unlock()
	if ok {
		for e, _ := range m {
			a.Unregister(e, c)
		}
	}
	a.Lock()
	delete(a.subscribers, c)
	a.Unlock()
	return
}
func (a *Hub) recordSubjcet() {
	go func() {
		t := time.NewTicker(time.Minute * 10)
		defer func() {
			t.Stop()
		}()
		for {
			select {
			case <-t.C:
				conn := a.redisManager.Get()
				conn.Send("MULTI")
				for key, _ := range a.subjects {
					conn.Send("SET", a.ChannelPrefix+":::"+key, time.Now().Unix())
					conn.Send("EXPIRE", time.Minute*11)
				}
				conn.Do("EXEC")
				conn.Close()
			}

		}
	}()
}
func (a *Hub) listenRedis() <-chan error {

	errChan := make(chan error, 1)
	go func() {
		for {
			switch v := a.psc.Receive().(type) {
			case redis.PMessage:

				//過濾掉前綴
				channel := strings.Replace(v.Channel, a.ChannelPrefix, "", 1)

				//過濾掉星號
				channel = strings.Replace(channel, "*", "", 1)
				a.RLock()
				clients, ok := a.subjects[channel]
				a.RUnlock()
				if !ok {
					continue
				}

				a.logger("channel:%s\taction:push start\tmsg:%s\tconnect clients:%v", channel, v.Data, len(clients))
				for c, _ := range clients {
					c.Trigger(channel, v.Data)
				}
				a.logger("channel:%s\taction:push over\tmsg:%s\tconnect clients:%v", channel, v.Data, len(clients))

			case error:
				errChan <- v

				break
			}
		}
	}()
	return errChan
}

func (a *Hub) close() {
	a.closeflag = true
	for c, _ := range a.subscribers {
		c.Close()
	}
}
func (a *Hub) Listen(channelPrefix string) error {
	a.ChannelPrefix = channelPrefix
	a.psc.PSubscribe(channelPrefix + "*")
	a.recordSubjcet()
	redisErr := a.listenRedis()
	select {
	case e := <-redisErr:
		a.close()
		return e
	case <-a.closeSign:
		a.close()
		return APPCLOSE

	}
}
func (a *Hub) Close() {
	if !a.closeflag {
		a.closeSign <- 1
		close(a.closeSign)
	}
	return

}

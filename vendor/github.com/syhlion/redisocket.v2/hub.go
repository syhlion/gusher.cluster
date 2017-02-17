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
	Trigger(event string, p *Payload) (err error)
	Close()
}

type Payload struct {
	Data           []byte
	PrepareMessage *websocket.PreparedMessage
	IsPrepare      bool
}

type ReceiveMsg struct {
	Channels    map[string]EventHandler
	Sub         bool
	ResponseMsg []byte
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

type EventHandler func(event string, payload *Payload) error

type ReceiveMsgHandler func([]byte) (*ReceiveMsg, error)

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
	channels, err = redis.Strings(conn.Do("smembers", channelPrefix+"channels"))
	return
}

func (s *Sender) Push(channelPrefix, event string, data []byte) (val int, err error) {
	conn := s.redisManager.Get()
	defer conn.Close()
	val, err = redis.Int(conn.Do("PUBLISH", channelPrefix+event, data))
	return
}

//NewApp It's create a Hub
func NewHub(m *redis.Pool, debug bool) (e *Hub) {

	l := log.New(os.Stdout, "[redisocket.v2]", log.Lshortfile|log.Ldate|log.Lmicroseconds)
	pool := &Pool{

		subjects:    make(map[string]map[User]bool),
		subscribers: make(map[User]map[string]bool),
		trigger:     make(chan *eventPayload),
		reg:         make(chan *registerPayload),
		unreg:       make(chan *unregisterPayload),
		unregAll:    make(chan *unregisterAllPayload),
		close:       make(chan int),
	}
	go pool.Run()
	return &Hub{

		Config:       DefaultWebsocketOptional,
		redisManager: m,
		psc:          &redis.PubSubConn{m.Get()},
		Pool:         pool,
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
		send:    make(chan *Payload, 4096),
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
	*Pool
	closeSign chan int
	closeflag bool
	debug     bool
	log       *log.Logger
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
func (a *Hub) CountOnlineUsers() (i int) {
	return len(a.Pool.subscribers)
}
func (a *Hub) CountChannels() (i int) {
	return len(a.Pool.subjects)
}

/*
func (a *Hub) recordSubjcet() {
	go func() {
		t := time.NewTicker(time.Minute * 1)
		defer func() {
			t.Stop()
		}()
		for {
			select {
			case <-t.C:
				conn := a.redisManager.Get()
				conn.Send("MULTI")
				for key, _ := range a.subjects {
					conn.Send("SADD", a.ChannelPrefix+"channels", key)
					conn.Send("EXPIRE", a.ChannelPrefix+"channels", 2*60)
				}
				conn.Do("EXEC")
				conn.Close()
			}

		}
	}()
}
*/
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
				pMsg, err := websocket.NewPreparedMessage(websocket.TextMessage, v.Data)
				if err != nil {
					continue
				}
				p := &Payload{
					PrepareMessage: pMsg,
					IsPrepare:      true,
				}
				a.Trigger(channel, p)

			case error:
				errChan <- v

				break
			}
		}
	}()
	return errChan
}

func (a *Hub) Listen(channelPrefix string) error {
	a.ChannelPrefix = channelPrefix
	a.psc.PSubscribe(channelPrefix + "*")
	//a.recordSubjcet()
	redisErr := a.listenRedis()
	select {
	case e := <-redisErr:
		return e
	}
}
func (a *Hub) Close() {
	a.Stop()
	return

}

package redisocket

import (
	"errors"
	"fmt"
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
	Event          string
}

type ReceiveMsg struct {
	Event        string
	EventHandler EventHandler
	Sub          bool
	ResponseMsg  []byte
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
			CheckOrigin: func(r *http.Request) bool { return true },
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

type BatchData struct {
	Event string
	Data  []byte
}

func (s *Sender) GetChannels(channelPrefix string, appKey string, pattern string) (channels []string, err error) {
	keyPrefix := fmt.Sprintf("%s%s@channels:", channelPrefix, appKey)
	conn := s.redisManager.Get()
	defer conn.Close()
	tmp, err := redis.Strings(conn.Do("keys", keyPrefix+pattern))
	channels = make([]string, len(tmp))
	for _, v := range tmp {
		channel := strings.Replace(v, keyPrefix, "", -1)
		channels = append(channels, channel)
	}

	return
}
func (s *Sender) GetOnlineByChannel(channelPrefix string, appKey string, channel string) (online []string, err error) {
	memberKey := fmt.Sprintf("%s%s@channels:%s", channelPrefix, appKey, channel)
	conn := s.redisManager.Get()
	defer conn.Close()
	online, err = redis.Strings(conn.Do("smembers", memberKey))
	return
}
func (s *Sender) GetOnline(channelPrefix string, appKey string) (online []string, err error) {
	memberKey := fmt.Sprintf("%s%s@online", channelPrefix, appKey)
	conn := s.redisManager.Get()
	defer conn.Close()
	online, err = redis.Strings(conn.Do("smembers", memberKey))
	return
}

func (s *Sender) PushBatch(channelPrefix, appKey string, data []BatchData) {
	conn := s.redisManager.Get()
	defer conn.Close()
	for _, d := range data {
		conn.Do("PUBLISH", channelPrefix+appKey+"@"+d.Event, d.Data)
	}
	return
}

func (s *Sender) Push(channelPrefix, appKey string, event string, data []byte) (val int, err error) {
	conn := s.redisManager.Get()
	defer conn.Close()
	val, err = redis.Int(conn.Do("PUBLISH", channelPrefix+appKey+"@"+event, data))
	return
}

//NewApp It's create a Hub
func NewHub(m *redis.Pool, debug bool) (e *Hub) {

	l := log.New(os.Stdout, "[redisocket.v2]", log.Lshortfile|log.Ldate|log.Lmicroseconds)
	pool := &Pool{

		users:     make(map[*Client]bool),
		broadcast: make(chan *eventPayload),
		join:      make(chan *Client),
		leave:     make(chan *Client),
		rpool:     m,
	}
	go pool.Run()
	return &Hub{

		Config:       DefaultWebsocketOptional,
		redisManager: m,
		psc:          &redis.PubSubConn{m.Get()},
		Pool:         pool,
		debug:        debug,
		log:          l,
	}

}
func (e *Hub) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header, uid string, prefix string) (c *Client, err error) {
	ws, err := e.Config.Upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		return
	}
	c = &Client{
		prefix:  prefix,
		uid:     uid,
		ws:      ws,
		send:    make(chan *Payload, 32),
		RWMutex: new(sync.RWMutex),
		hub:     e,
		events:  make(map[string]EventHandler),
	}
	e.Join(c)
	return
}

type Hub struct {
	ChannelPrefix string
	Config        WebsocketOptional
	psc           *redis.PubSubConn
	redisManager  *redis.Pool
	*Pool
	debug bool
	log   *log.Logger
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
	return len(a.Pool.users)
}
func (a *Hub) listenRedis() <-chan error {

	errChan := make(chan error, 1)
	go func() {
		for {
			switch v := a.psc.Receive().(type) {
			case redis.PMessage:

				//過濾掉前綴
				channel := strings.Replace(v.Channel, a.ChannelPrefix, "", -1)
				//過濾掉@ 之前的字
				sch := strings.SplitN(channel, "@", 2)
				if len(sch) != 2 {
					continue
				}

				//過濾掉星號
				channel = strings.Replace(sch[1], "*", "", -1)
				pMsg, err := websocket.NewPreparedMessage(websocket.TextMessage, v.Data)
				if err != nil {
					continue
				}
				p := &Payload{
					PrepareMessage: pMsg,
					IsPrepare:      true,
				}
				a.Broadcast(channel, p)

			case error:
				errChan <- v

				break
			}
		}
	}()
	return errChan
}

func (a *Hub) Listen(channelPrefix string) error {
	a.Pool.channelPrefix = channelPrefix
	a.ChannelPrefix = channelPrefix
	a.psc.PSubscribe(channelPrefix + "*")
	redisErr := a.listenRedis()
	select {
	case e := <-redisErr:
		return e
	}
}
func (a *Hub) Close() {
	return

}

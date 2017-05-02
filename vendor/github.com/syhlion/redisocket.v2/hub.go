package redisocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
)

//User client interface
type User interface {
	Trigger(event string, p *Payload) (err error)
	Close()
}

//Payload reciev from redis
type Payload struct {
	Data           []byte
	PrepareMessage *websocket.PreparedMessage
	IsPrepare      bool
	Event          string
}

//WebsocketOptional  init websocket hub config
type WebsocketOptional struct {
	ScanInterval   time.Duration
	WriteWait      time.Duration
	PongWait       time.Duration
	PingPeriod     time.Duration
	MaxMessageSize int64
	Upgrader       websocket.Upgrader
}
type socketPayload struct {
	Sid  string      `json:"sid"`
	Data interface{} `json:"data"`
}
type userPayload struct {
	Uid  string      `json:"uid"`
	Data interface{} `json:"data"`
}

var (
	//DefaultWebsocketOptional default config
	DefaultWebsocketOptional = WebsocketOptional{
		ScanInterval:   30 * time.Second,
		WriteWait:      10 * time.Second,
		PongWait:       60 * time.Second,
		PingPeriod:     (60 * time.Second * 9) / 10,
		MaxMessageSize: 512,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
)

//EventHandler event handler
type EventHandler func(event string, payload *Payload) error

//ReceiveMsgHandler client receive msg
type ReceiveMsgHandler func([]byte) ([]byte, error)

//NewSender return sender  send to hub
func NewSender(m *redis.Pool) (e *Sender) {

	return &Sender{
		redisManager: m,
	}
}

//Sender struct
type Sender struct {
	redisManager *redis.Pool
}

//BatchData push batch data struct
type BatchData struct {
	Event string
	Data  []byte
}

//GetChannels get all sub channels
func (s *Sender) GetChannels(channelPrefix string, appKey string, pattern string) (channels []string, err error) {
	keyPrefix := fmt.Sprintf("%s%s@channels:", channelPrefix, appKey)
	conn := s.redisManager.Get()
	defer conn.Close()
	tmp, err := redis.Strings(conn.Do("keys", keyPrefix+pattern))
	channels = make([]string, 0)
	for _, v := range tmp {
		channel := strings.Replace(v, keyPrefix, "", -1)
		if channel == "" {
			continue
		}
		channels = append(channels, channel)
	}

	return
}

//GetOnlineByChannel get all online user by  channel
func (s *Sender) GetOnlineByChannel(channelPrefix string, appKey string, channel string) (online []string, err error) {
	memberKey := fmt.Sprintf("%s%s@channels:%s", channelPrefix, appKey, channel)
	conn := s.redisManager.Get()
	defer conn.Close()
	nt := time.Now().Unix()
	dt := nt - 120
	online, err = redis.Strings(conn.Do("ZRANGEBYSCORE", memberKey, dt, nt))
	return
}

//GetOnline get all online user
func (s *Sender) GetOnline(channelPrefix string, appKey string) (online []string, err error) {
	memberKey := fmt.Sprintf("%s%s@online", channelPrefix, appKey)
	conn := s.redisManager.Get()
	defer conn.Close()
	nt := time.Now().Unix()
	dt := nt - 120
	online, err = redis.Strings(conn.Do("ZRANGEBYSCORE", memberKey, dt, nt))
	return
}

//PushBatch push batch data
func (s *Sender) PushBatch(channelPrefix, appKey string, data []BatchData) {
	conn := s.redisManager.Get()
	defer conn.Close()
	for _, d := range data {
		conn.Do("PUBLISH", channelPrefix+appKey+"@"+d.Event, d.Data)
	}
	return
}

//PushToSid  push to user socket id
func (s *Sender) PushToSid(channelPrefix, appKey string, uid string, data interface{}) (val int, err error) {
	conn := s.redisManager.Get()
	defer conn.Close()
	u := userPayload{
		Uid:  uid,
		Data: data,
	}
	d, err := json.Marshal(u)
	if err != nil {
		return
	}
	val, err = redis.Int(conn.Do("PUBLISH", channelPrefix+appKey+"@"+"#GUSHERFUNC-TOUID#", d))
	return
}

//PushTo  push to user socket
func (s *Sender) PushToUid(channelPrefix, appKey string, uid string, data interface{}) (val int, err error) {
	conn := s.redisManager.Get()
	defer conn.Close()
	u := userPayload{
		Uid:  uid,
		Data: data,
	}
	d, err := json.Marshal(u)
	if err != nil {
		return
	}
	val, err = redis.Int(conn.Do("PUBLISH", channelPrefix+appKey+"@"+"#GUSHERFUNC-TOSID#", d))
	return
}

//Push push single data
func (s *Sender) Push(channelPrefix, appKey string, event string, data []byte) (val int, err error) {
	conn := s.redisManager.Get()
	defer conn.Close()
	val, err = redis.Int(conn.Do("PUBLISH", channelPrefix+appKey+"@"+event, data))
	return
}

//NewHub It's create a Hub
func NewHub(m *redis.Pool, debug bool) (e *Hub) {

	l := log.New(os.Stdout, "[redisocket.v2]", log.Lshortfile|log.Ldate|log.Lmicroseconds)
	pool := &pool{
		users:         make(map[*Client]bool),
		broadcastChan: make(chan *eventPayload, 4096),
		joinChan:      make(chan *Client),
		leaveChan:     make(chan *Client),
		kickSidChan:   make(chan string),
		kickUidChan:   make(chan string),
		uPayloadChan:  make(chan *uPayload, 100),
		sPayloadChan:  make(chan *sPayload, 100),
		shutdownChan:  make(chan int, 1),
		rpool:         m,
	}
	mq := &messageQuene{
		freeBufferChan: make(chan *buffer, 100),
		serveChan:      make(chan *buffer),
		pool:           pool,
	}
	mq.run()
	return &Hub{

		messageQuene: mq,
		Config:       DefaultWebsocketOptional,
		redisManager: m,
		psc:          &redis.PubSubConn{Conn: m.Get()},
		pool:         pool,
		debug:        debug,
		closeSign:    make(chan int, 1),
		log:          l,
	}

}

//Upgrade gorilla websocket wrap upgrade method
func (e *Hub) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header, uid string, prefix string) (c *Client, err error) {
	ws, err := e.Config.Upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		return
	}
	sid := uuid.NewV1()
	c = &Client{
		prefix:  prefix,
		uid:     uid,
		sid:     sid.String(),
		ws:      ws,
		send:    make(chan *Payload, 64),
		RWMutex: new(sync.RWMutex),
		hub:     e,
		events:  make(map[string]EventHandler),
	}
	e.join(c)
	return
}

//Hub client hub
type Hub struct {
	ChannelPrefix string
	messageQuene  *messageQuene
	Config        WebsocketOptional
	psc           *redis.PubSubConn
	redisManager  *redis.Pool
	*pool
	debug     bool
	log       *log.Logger
	closeSign chan int
}

//Ping ping redis server
func (e *Hub) Ping() (err error) {
	_, err = e.redisManager.Get().Do("PING")
	if err != nil {
		return
	}
	return
}
func (e *Hub) logger(format string, v ...interface{}) {
	if e.debug {
		e.log.Printf(format, v...)
	}
}

//CountOnlineUsers return online user total
func (e *Hub) CountOnlineUsers() (i int) {
	return len(e.pool.users)
}
func (e *Hub) listenRedis() <-chan error {

	errChan := make(chan error, 1)
	go func() {
		for {
			switch v := e.psc.Receive().(type) {
			case redis.PMessage:

				//過濾掉前綴
				channel := strings.Replace(v.Channel, e.ChannelPrefix, "", -1)
				//過濾掉@ 之前的字
				sch := strings.SplitN(channel, "@", 2)
				if len(sch) != 2 {
					continue
				}

				//過濾掉星號
				channel = strings.Replace(sch[1], "*", "", -1)
				if channel == "#GUSHERFUNC-TOUID#" {
					up := &userPayload{}
					err := json.Unmarshal(v.Data, up)
					if err != nil {
						continue
					}
					b, err := json.Marshal(up.Data)
					if err != nil {
						continue
					}
					e.toUid(up.Uid, b)
					continue
				}
				if channel == "#GUSHERFUNC-TOSID#" {
					up := &socketPayload{}
					err := json.Unmarshal(v.Data, up)
					if err != nil {
						continue
					}
					b, err := json.Marshal(up.Data)
					if err != nil {
						continue
					}
					e.toSid(up.Sid, b)
					continue
				}
				pMsg, err := websocket.NewPreparedMessage(websocket.TextMessage, v.Data)
				if err != nil {
					continue
				}
				p := &Payload{
					PrepareMessage: pMsg,
					IsPrepare:      true,
				}
				e.broadcast(channel, p)

			case error:
				errChan <- v

				break
			}
		}
	}()
	return errChan
}

//Listen hub start
//it's block method
func (e *Hub) Listen(channelPrefix string) error {
	e.pool.channelPrefix = channelPrefix
	e.ChannelPrefix = channelPrefix
	e.psc.PSubscribe(channelPrefix + "*")
	redisErr := e.listenRedis()
	e.pool.scanInterval = e.Config.ScanInterval
	poolErr := e.pool.run()
	select {
	case er := <-redisErr:
		e.pool.shutdown()
		return er
	case er := <-poolErr:
		return er
	case <-e.closeSign:
		e.pool.shutdown()
		return nil
	}
}

//Close close hub & close every client
func (e *Hub) Close() {
	e.closeSign <- 1
	return

}

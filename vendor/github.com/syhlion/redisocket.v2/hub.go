package redisocket

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// User client interface
type User interface {
	Trigger(event string, p *Payload) (err error)
	Close()
}

// Payload reciev from redis
type Payload struct {
	Len            int
	Data           []byte
	PrepareMessage *websocket.PreparedMessage
	IsPrepare      bool
	Event          string
}

// WebsocketOptional  init websocket hub config
type WebsocketOptional struct {
	ScanInterval   time.Duration
	WriteWait      time.Duration
	PongWait       time.Duration
	PingPeriod     time.Duration
	MaxMessageSize int64
	MessageWorkers int // inbound 訊息處理的 worker 數(<=0 用 defaultMessageWorkers)
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
type reloadChannelPayload struct {
	Uid      string   `json:"uid"`
	Channels []string `json:"data"`
}
type addChannelPayload struct {
	Uid     string `json:"uid"`
	Channel string `json:"data"`
}

var (
	//DefaultWebsocketOptional default config
	DefaultWebsocketOptional = WebsocketOptional{
		ScanInterval:   30 * time.Second,
		WriteWait:      10 * time.Second,
		PongWait:       60 * time.Second,
		PingPeriod:     (60 * time.Second * 9) / 10,
		MaxMessageSize: 512,
		MessageWorkers: defaultMessageWorkers,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
)

// EventHandler event handler
type EventHandler func(event string, payload *Payload) error

// ReceiveMsgHandler client receive msg
type ReceiveMsgHandler func([]byte) ([]byte, error)

// NewSender return sender  send to hub
func NewSender(m *redis.Pool) (e *Sender) {

	return &Sender{
		broker:   newRedisBroker(m),
		presence: newRedisPresence(m),
	}
}

// NewSenderWithBrokerAndPresence 注入 broker 與 presence(NATS-native 的 publish 端;
// presence 通常為 memoryPresence,本機無成員、查詢時 request/reply 聚合各節點)。
func NewSenderWithBrokerAndPresence(broker Broker, presence Presence) *Sender {
	return &Sender{broker: broker, presence: presence}
}

// Sender struct(publish 走 broker、presence 查詢走 presence;兩者皆可換後端)
type Sender struct {
	broker   Broker
	presence Presence
}

// BatchData push batch data struct
type BatchData struct {
	Event string
	Data  []byte
}

// GetChannels get all sub channels
func (s *Sender) GetChannels(channelPrefix string, appKey string, pattern string) (channels []string, err error) {
	return s.presence.Channels(channelPrefix, appKey, pattern)
}

// GetOnlineByChannel get all online user by  channel
func (s *Sender) GetOnlineByChannel(channelPrefix string, appKey string, channel string) (online []string, err error) {
	return s.presence.OnlineByChannel(channelPrefix, appKey, channel)
}

// GetOnline get all online user
func (s *Sender) GetOnline(channelPrefix string, appKey string) (online []string, err error) {
	return s.presence.Online(channelPrefix, appKey)
}

// PushBatch push batch data
func (s *Sender) PushBatch(channelPrefix, appKey string, data []BatchData) {
	for _, d := range data {
		s.broker.Publish(channelPrefix, appKey, d.Event, d.Data)
	}
	return
}

// PushToSid  push to user socket id
func (s *Sender) PushToSid(channelPrefix, appKey string, uid string, data interface{}) (val int, err error) {
	u := socketPayload{
		Sid:  uid,
		Data: data,
	}
	d, err := json.Marshal(u)
	if err != nil {
		return
	}
	val, err = s.broker.Publish(channelPrefix, appKey, "#GUSHERFUNC-TOSID#", d)
	return
}

// PushTo  push to user socket
func (s *Sender) PushToUid(channelPrefix, appKey string, uid string, data interface{}) (val int, err error) {
	u := userPayload{
		Uid:  uid,
		Data: data,
	}
	d, err := json.Marshal(u)
	if err != nil {
		return
	}
	val, err = s.broker.Publish(channelPrefix, appKey, "#GUSHERFUNC-TOUID#", d)
	return
}

// ReloadChannel  reload user channel list
func (s *Sender) ReloadChannel(channelPrefix, appKey string, uid string, channels []string) (val int, err error) {
	u := reloadChannelPayload{
		Uid:      uid,
		Channels: channels,
	}
	d, err := json.Marshal(u)
	if err != nil {
		return
	}
	val, err = s.broker.Publish(channelPrefix, appKey, "#GUSHERFUNC-RELOADCHANEL#", d)
	return
}

// AddChannel  append channel to user channel list
func (s *Sender) AddChannel(channelPrefix, appKey string, uid string, channel string) (val int, err error) {
	u := addChannelPayload{
		Uid:     uid,
		Channel: channel,
	}
	d, err := json.Marshal(u)
	if err != nil {
		return
	}
	val, err = s.broker.Publish(channelPrefix, appKey, "#GUSHERFUNC-ADDCHANEL#", d)
	return
}

// Push push single data
func (s *Sender) Push(channelPrefix, appKey string, event string, data []byte) (val int, err error) {
	return s.broker.Publish(channelPrefix, appKey, event, data)
}

// NewHub It's create a Hub (Redis 後端:bus 與 presence 都用同一個 redis pool)
func NewHub(m *redis.Pool, log *slog.Logger, debug bool) (e *Hub) {
	return NewHubWithBroker(newRedisBroker(m), m, log, debug)
}

// NewHubWithBroker 注入 bus 後端(broker),presence 用 presencePool(redis)。
// 用於「NATS broker + redis presence」過渡組合。
func NewHubWithBroker(broker Broker, presencePool *redis.Pool, log *slog.Logger, debug bool) (e *Hub) {
	return newHub(broker, newRedisPresence(presencePool), presencePool, log, debug)
}

// NewHubWithBrokerAndPresence 同時注入 broker 與 presence,完全不需要 redis。
// NATS-native 路線用(natsBroker + memoryPresence)。
func NewHubWithBrokerAndPresence(broker Broker, presence Presence, log *slog.Logger, debug bool) (e *Hub) {
	return newHub(broker, presence, nil, log, debug)
}

func newHub(broker Broker, presence Presence, redisManager *redis.Pool, log *slog.Logger, debug bool) (e *Hub) {

	quit := make(chan struct{})
	stat := &Statistic{
		inMemChannel:  make(chan int, 8192),
		outMemChannel: make(chan int, 8192),
		inMsgChannel:  make(chan int, 8192),
		outMsgChannel: make(chan int, 8192),
		l:             log,
		quit:          quit,
	}
	go stat.Run()
	pool := &pool{
		stat:               stat,
		users:              make(map[*Client]bool),
		broadcastChan:      make(chan *eventPayload, 4096),
		joinChan:           make(chan *Client),
		leaveChan:          make(chan *Client),
		kickSidChan:        make(chan string),
		kickUidChan:        make(chan string),
		uPayloadChan:       make(chan *uPayload, 4096),
		uReloadChannelChan: make(chan *uReloadChannelPayload, 4096),
		uAddChannelChan:    make(chan *uAddChannelPayload, 4096),
		sPayloadChan:       make(chan *sPayload, 4096),
		quit:               quit,
		presence:           presence,
	}
	mq := &messageQuene{
		freeBufferChan: make(chan *buffer, 8192),
		serveChan:      make(chan *buffer, 8192),
		pool:           pool,
		quit:           quit,
	}
	// 註:workers 在 Listen() 啟動,讓使用者可於 NewHub 後、Listen 前調整
	// Config.MessageWorkers。

	return &Hub{

		messageQuene: mq,
		Config:       DefaultWebsocketOptional,
		redisManager: redisManager,
		broker:       broker,
		pool:         pool,
		debug:        debug,
		quit:         quit,
		log:          log,
	}

}

// Upgrade gorilla websocket wrap upgrade method
func (e *Hub) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header, uid string, prefix string, auth *Auth) (c *Client, err error) {
	ws, err := e.Config.Upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		return
	}
	sid := uuid.New() // V4 隨機 UUID(取代 satori NewV1 的可預測時間/MAC 版)
	c = &Client{
		prefix:  prefix,
		uid:     uid,
		sid:     sid.String(),
		ws:      ws,
		send:    make(chan *Payload, 256),
		RWMutex: new(sync.RWMutex),
		hub:     e,
		events:  make(map[string]EventHandler),
		auth:    auth,
	}
	e.join(c)
	return
}

// Hub client hub
type Hub struct {
	ChannelPrefix string
	messageQuene  *messageQuene
	Config        WebsocketOptional
	broker        Broker
	redisManager  *redis.Pool
	*pool
	debug     bool
	log       *slog.Logger
	quit      chan struct{}
	closeOnce sync.Once
}

// Ping ping redis server(NATS-native 路線無 redis,redisManager 為 nil 時為 no-op)
func (e *Hub) Ping() (err error) {
	if e.redisManager == nil {
		return nil
	}
	_, err = e.redisManager.Get().Do("PING")
	if err != nil {
		return
	}
	return
}
func (e *Hub) logger(format string, v ...interface{}) {
	if e.debug {
		e.log.Debug(fmt.Sprintf(format, v...))
	}
}

// CountOnlineUsers return online user total
func (e *Hub) CountOnlineUsers() (i int) {
	return e.pool.onlineCount()
}

// dispatchLoop 消費 broker 收到的事件,做控制事件分派或頻道廣播。
func (e *Hub) dispatchLoop(msgs <-chan BrokerEvent) {
	for ev := range msgs {
		e.handleEvent(ev.Event, ev.Data)
	}
}

// handleEvent 處理單一 bus 事件:#GUSHERFUNC-*# 為控制事件,其餘為一般頻道廣播。
func (e *Hub) handleEvent(channel string, data []byte) {
	switch channel {
	case "#GUSHERFUNC-TOUID#":
		up := &userPayload{}
		if err := json.Unmarshal(data, up); err != nil {
			return
		}
		b, err := json.Marshal(up.Data)
		if err != nil {
			return
		}
		e.toUid(up.Uid, b)
	case "#GUSHERFUNC-TOSID#":
		up := &socketPayload{}
		if err := json.Unmarshal(data, up); err != nil {
			return
		}
		b, err := json.Marshal(up.Data)
		if err != nil {
			return
		}
		e.toSid(up.Sid, b)
	case "#GUSHERFUNC-RELOADCHANEL#":
		up := &reloadChannelPayload{}
		if err := json.Unmarshal(data, up); err != nil {
			return
		}
		e.reloadUidChannels(up.Uid, up.Channels)
	case "#GUSHERFUNC-ADDCHANEL#":
		up := &addChannelPayload{}
		if err := json.Unmarshal(data, up); err != nil {
			return
		}
		e.addUidChannels(up.Uid, up.Channel)
	default:
		pMsg, err := websocket.NewPreparedMessage(websocket.TextMessage, data)
		if err != nil {
			return
		}
		p := &Payload{
			Len:            len(data),
			PrepareMessage: pMsg,
			IsPrepare:      true,
		}
		e.broadcast(channel, p)
	}
}

// Listen hub start
// it's block method
func (e *Hub) Listen(channelPrefix string) error {
	e.pool.channelPrefix = channelPrefix
	e.ChannelPrefix = channelPrefix
	msgs, busErr := e.broker.Subscribe(channelPrefix)
	go e.dispatchLoop(msgs)
	e.messageQuene.run(e.Config.MessageWorkers)
	e.pool.scanInterval = e.Config.ScanInterval
	poolErr := e.pool.run()
	select {
	case er := <-busErr:
		e.Close()
		return er
	case er := <-poolErr:
		e.Close()
		return er
	case <-e.quit:
		return nil
	}
}

// Close 優雅關閉 Hub:停止 bus 訂閱、pool、message workers、statistic 與所有 client。
// 可安全重複呼叫(sync.Once)。
func (e *Hub) Close() {
	e.closeOnce.Do(func() {
		close(e.quit)           // 收掉 pool.run / message workers / statistic / 各 input 方法
		e.broker.Close()        // 收掉訂閱 goroutine → msgs 關閉 → dispatchLoop 結束
		e.pool.presence.Close() // 收掉 presence(memoryPresence 退訂;redis 為 no-op)
	})
}

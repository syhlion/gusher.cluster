package redisocket

import (
	"errors"
	"time"

	"github.com/gomodule/redigo/redis"
)

type eventPayload struct {
	payload *Payload
	event   string
}

// pool 用
type uPayload struct {
	uid  string `json:"uid"`
	data []byte `json:"data"`
}
type uReloadChannelPayload struct {
	uid      string   `json:"uid"`
	channels []string `json:"channels"`
}
type uAddChannelPayload struct {
	uid     string `json:"uid"`
	channel string `json:"channel"`
}
type sPayload struct {
	sid  string `json:"uid"`
	data []byte `json:"data"`
}

type pool struct {
	users              map[*Client]bool
	broadcastChan      chan *eventPayload
	joinChan           chan *Client
	leaveChan          chan *Client
	shutdownChan       chan int
	kickUidChan        chan string
	kickSidChan        chan string
	uPayloadChan       chan *uPayload
	uReloadChannelChan chan *uReloadChannelPayload
	uAddChannelChan    chan *uAddChannelPayload
	sPayloadChan       chan *sPayload
	rpool              *redis.Pool
	channelPrefix      string
	scanInterval       time.Duration
	msgTotal           int64
	msgByteSum         int64
}

func (h *pool) run() <-chan error {
	errChan := make(chan error, 1)
	go func() {
		t := time.NewTicker(h.scanInterval)
		defer func() {
			t.Stop()
			err := errors.New("pool close")
			errChan <- err
		}()
		for {
			select {
			case p := <-h.broadcastChan:
				for u := range h.users {
					u.Trigger(p.event, p.payload)
				}
			case <-h.shutdownChan:
				for u := range h.users {
					u.Close()
				}
			case n := <-h.kickSidChan:
				for u := range h.users {
					if u.sid == n {
						u.Close()
					}
				}
			case n := <-h.uReloadChannelChan:
				for u := range h.users {
					if u.uid == n.uid {

						u.SetChannels(n.channels)

					}
				}
			case n := <-h.uAddChannelChan:
				for u := range h.users {
					if u.uid == n.uid {

						u.AddChannel(n.channel)

					}
				}
			case s := <-h.kickUidChan:
				for u := range h.users {
					if u.uid == s {
						u.Close()
					}
				}
			case n := <-h.uPayloadChan:
				for u := range h.users {
					if u.uid == n.uid {

						u.Send(n.data)
					}
				}
			case n := <-h.sPayloadChan:
				for u := range h.users {
					if u.sid == n.sid {
						u.Send(n.data)
					}
				}
			case u := <-h.joinChan:
				statistic.AddMem()
				h.users[u] = true
			case u := <-h.leaveChan:
				if _, ok := h.users[u]; ok {
					statistic.SubMem()
					close(u.send)
					delete(h.users, u)
				}
			case <-t.C:
				h.syncOnline()

			}

		}
	}()
	return errChan
}
func (h *pool) toUid(uid string, d []byte) {
	u := &uPayload{uid: uid, data: d}
	h.uPayloadChan <- u
}
func (h *pool) reloadUidChannels(uid string, channels []string) {
	u := &uReloadChannelPayload{uid: uid, channels: channels}
	h.uReloadChannelChan <- u
}
func (h *pool) addUidChannels(uid string, channel string) {
	u := &uAddChannelPayload{uid: uid, channel: channel}
	h.uAddChannelChan <- u
}
func (h *pool) toSid(sid string, d []byte) {
	u := &sPayload{sid: sid, data: d}
	h.sPayloadChan <- u
}
func (h *pool) shutdown() {
	h.shutdownChan <- 1
}
func (h *pool) kickUid(uid string) {
	h.kickUidChan <- uid
}
func (h *pool) kickSid(sid string) {
	h.kickSidChan <- sid
}
func (h *pool) syncOnline() (err error) {
	conn := h.rpool.Get()
	defer conn.Close()
	t := time.Now()
	nt := t.Unix()
	dt := t.Unix() - 86400
	conn.Send("MULTI")
	for u := range h.users {
		if u.uid != "" {
			conn.Send("ZADD", h.channelPrefix+u.prefix+"@"+"online", "CH", nt, u.uid)
		}
		u.RLock()
		for e := range u.events {
			conn.Send("ZADD", h.channelPrefix+u.prefix+"@"+"channels:"+e, "CH", nt, u.uid)
			conn.Send("EXPIRE", h.channelPrefix+u.prefix+"@"+"channels:"+e, 300)
		}
		u.RUnlock()
		conn.Send("EXPIRE", h.channelPrefix+u.prefix+"@"+"online", 300)
	}
	conn.Do("EXEC")
	tmp, err := redis.Strings(conn.Do("keys", h.channelPrefix+"*"))
	if err != nil {
		return
	}
	//刪除過時的key
	conn.Send("MULTI")
	for _, k := range tmp {
		conn.Send("ZREMRANGEBYSCORE", k, dt, nt-60)
	}
	conn.Do("EXEC")
	return
}
func (h *pool) broadcast(event string, p *Payload) {
	h.broadcastChan <- &eventPayload{p, event}
}
func (h *pool) join(c *Client) {
	h.joinChan <- c
}
func (h *pool) leave(c *Client) {
	h.leaveChan <- c
}

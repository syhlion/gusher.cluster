package redisocket

import (
	"errors"
	"time"

	"github.com/garyburd/redigo/redis"
)

type eventPayload struct {
	payload *Payload
	event   string
}

type pool struct {
	users          map[*Client]bool
	broadcastChan  chan *eventPayload
	joinChan       chan *Client
	leaveChan      chan *Client
	shutdownChan   chan int
	kickChan       chan string
	freeBufferChan chan *buffer
	serveChan      chan *buffer
	rpool          *redis.Pool
	channelPrefix  string
	scanInterval   time.Duration
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
			case n := <-h.kickChan:
				for u := range h.users {
					if u.uid == n {
						u.Close()
					}
				}
			case b := <-h.serveChan:
				h.serve(b)
			case u := <-h.joinChan:
				h.users[u] = true
			case u := <-h.leaveChan:
				if _, ok := h.users[u]; ok {
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
func (h *pool) shutdown() {
	h.shutdownChan <- 1
}
func (h *pool) kick(uid string) {
	h.kickChan <- uid
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
		for e := range u.events {
			conn.Send("ZADD", h.channelPrefix+u.prefix+"@"+"channels:"+e, "CH", nt, u.uid)
			conn.Send("EXPIRE", h.channelPrefix+u.prefix+"@"+"channels:"+e, 300)
		}
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

func (h *pool) serve(buffer *buffer) {
	receiveMsg, err := buffer.client.re(buffer.buffer.Bytes())
	if err == nil {
		buffer.client.Send(receiveMsg)
	} else {
		buffer.client.Close()
	}
	buffer.reset(nil)
	select {
	case h.freeBufferChan <- buffer:
	default:
	}
	return
}

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

type Pool struct {
	users         map[*Client]bool
	broadcast     chan *eventPayload
	join          chan *Client
	leave         chan *Client
	shutdown      chan int
	kick          chan string
	rpool         *redis.Pool
	channelPrefix string
}

func (h *Pool) Run() <-chan error {
	errChan := make(chan error, 1)
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer func() {
			t.Stop()
			err := errors.New("pool close")
			errChan <- err
		}()
		for {
			select {
			case p := <-h.broadcast:
				for u, _ := range h.users {
					u.Trigger(p.event, p.payload)
				}
			case <-h.shutdown:
				for u, _ := range h.users {
					u.Close()
				}
			case n := <-h.kick:
				for u, _ := range h.users {
					if u.uid == n {
						u.Close()
					}
				}
			case u := <-h.join:
				h.users[u] = true
			case u := <-h.leave:
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
func (a *Pool) Shutdown() {
	a.shutdown <- 1
}
func (a *Pool) Kick(uid string) {
	a.kick <- uid
}
func (a *Pool) syncOnline() (err error) {
	conn := a.rpool.Get()
	defer conn.Close()
	t := time.Now()
	nt := t.Unix()
	dt := t.Unix() - 86400
	for u, _ := range a.users {
		if u.uid != "" {
			conn.Do("ZADD", a.channelPrefix+u.prefix+"@"+"online", "CH", nt, u.uid)
		}
		for e, _ := range u.events {
			conn.Do("ZADD", a.channelPrefix+u.prefix+"@"+"channels:"+e, "CH", nt, u.uid)
			conn.Do("EXPIRE", a.channelPrefix+u.prefix+"@"+"channels:"+e, 300)
		}
		conn.Do("EXPIRE", a.channelPrefix+u.prefix+"@"+"online", 300)
	}
	tmp, err := redis.Strings(conn.Do("keys", a.channelPrefix+"*"))
	if err != nil {
		return
	}
	for _, k := range tmp {
		conn.Do("ZREMRANGEBYSCORE", k, dt, nt-60)
	}
	return
}
func (a *Pool) Broadcast(event string, p *Payload) {
	a.broadcast <- &eventPayload{p, event}
}
func (a *Pool) Join(c *Client) {
	a.join <- c
}
func (a *Pool) Leave(c *Client) {
	a.leave <- c
}

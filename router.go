package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/garyburd/redigo/redis"
	"github.com/syhlion/redisocket"
)

type User struct {
	eventLauncher redisocket.App
	conn          redisocket.Subscriber
	tag           string
	channel       string
}

type WsManager struct {
	users map[*User]bool
	*sync.RWMutex
	pool *redis.Pool
}

func (u *User) AfterReadStream(b []byte) (err error) {
	return
}
func (u *User) BeforeWriteStream(b []byte) (data []byte, err error) {
	return b, nil
}
func (u *User) Listen() (err error) {
	return u.conn.Listen(u)
}
func (u *User) Close() {
	u.conn.Close()
}

func (wm *WsManager) Connect(w http.ResponseWriter, r *http.Request) {
	var (
		tag, channel string
	)
	s, err := rsocket.NewClient(w, r)
	if err != nil {
		log.Println(err)
		return
	}
	/**/
	//訂閱頻道
	rsocket.Subscribe(tag, s)
	rsocket.Subscribe(channel, s)
	c := wm.pool.Get()
	c.Send("MULTI")
	c.Send("INCR", tag)
	c.Send("INCR", channel)
	reply, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		s.Close()
		log.Println(err)
		return
	}
	var (
		tag_reply     int64
		channel_reply int64
	)
	if _, err := redis.Scan(reply, &tag_reply, &channel_reply); err != nil {
		s.Close()
		log.Println(err)
		return
	}
	/**/

	u := &User{rsocket, s, tag, channel}
	wm.Lock()
	wm.users[u] = true
	wm.Unlock()
	u.Listen()
	wm.Dieconnect(u)

}

func (wm *WsManager) Dieconnect(u *User) {
	wm.Lock()
	delete(wm.users, u)
	wm.Unlock()

	c := wm.pool.Get()
	c.Send("MULTI")
	c.Send("DECR", u.tag)
	c.Send("DECR", u.channel)
	reply, err := redis.Values(c.Do("EXEC"))
	if err != nil {
		log.Println(err)
		return
	}
	var (
		tag_reply     int64
		channel_reply int64
	)
	if _, err := redis.Scan(reply, &tag_reply, &channel_reply); err != nil {
		log.Println(err)
		return
	}
	if tag_reply == 0 {
		c.Do("DEL", u.tag)
	}
	if channel_reply == 0 {
		c.Do("DEL", u.channel)
	}
}
func (wm *WsManager) Close() {
	for u, _ := range wm.users {
		u.Close()
	}
}

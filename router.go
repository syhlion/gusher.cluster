package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/garyburd/redigo/redis"
	"github.com/syhlion/redisocket.v2"
	"github.com/syhlion/requestwork.v2"
)

type User struct {
	id      string
	channel map[string]bool
	redisocket.Subscriber
}

type WsManager struct {
	users map[*User]bool
	*sync.RWMutex
	pool   *redis.Pool
	worker *requestwork.Worker
}

func (wm *WsManager) Connect(w http.ResponseWriter, r *http.Request) {

	s, err := rsocket.NewClient(w, r)
	if err != nil {
		log.Println(err)
		return
	}

	id, channel, err := func() (id string, channel map[string]bool, err error) {
		channel = make(map[string]bool)
		if s, ok := r.Context().Value("auth").(Auth); ok {
			for _, c := range s.Channel {
				channel[c] = false
			}
			id = s.UserId
		} else {
			err = errors.New("type error")
		}
		return
	}()
	if err != nil {
		return
	}

	u := &User{id, channel, s}
	wm.Lock()
	wm.users[u] = true
	wm.Unlock()
	u.Listen(func(data []byte) (err error) {
		h := func(data []byte) (d []byte, err error) {
			return data, nil
		}
		var packet Packet
		err = json.Unmarshal(data, &packet)
		if err != nil {
			return
		}

		//訂閱處理
		if packet.Action == Subscribe {
			for _, c := range packet.Content {
				if b, ok := u.channel[c]; ok && !b {
					u.Subscribe(c, h)
					u.channel[c] = true
				}
			}
		}

		//反訂閱處理
		if packet.Action == UnSubscribe {
			for _, c := range packet.Content {
				if b, ok := u.channel[c]; ok && b {
					u.Unsubscribe(c)
					u.channel[c] = false
				}
			}
		}

		return
	})
	wm.Disconnect(u)

}

func (wm *WsManager) Disconnect(u *User) {
	wm.Lock()
	delete(wm.users, u)
	wm.Unlock()

}
func (wm *WsManager) Close() {
	for u, _ := range wm.users {
		u.Close()
	}
}

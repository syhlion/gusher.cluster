package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/garyburd/redigo/redis"
	"github.com/syhlion/redisocket.v2"
	"github.com/syhlion/requestwork.v2"
)

type User struct {
	id      string
	channel map[string]bool
	*redisocket.Client
}

type WsManager struct {
	users map[*User]bool
	*sync.RWMutex
	pool   *redis.Pool
	worker *requestwork.Worker
}

func (wm *WsManager) Count() int {
	return len(wm.users)
}

func (wm *WsManager) Connect(w http.ResponseWriter, r *http.Request) {

	id, channel, err := func() (id string, channel map[string]bool, err error) {
		channel = make(map[string]bool)
		auth := r.Context().Value("auth")
		if s, ok := auth.(Auth); ok {
			for _, c := range s.Channels {
				channel[c] = false
			}
			id = s.UserId
		} else {
			err = errors.New("auth type error")
		}
		return
	}()
	app_key := r.Context().Value("app_key").(string)
	if err != nil {
		logger.GetRequestEntry(r).Warnf("parse from context error %s", err)
		http.Error(w, "app_key error", 401)
		return
	}
	s, err := rsocket.NewClient(w, r)
	if err != nil {
		logger.GetRequestEntry(r).Warnf("upgrade ws connection %s", err)
		http.Error(w, err.Error(), 401)
		return
	}

	u := &User{id, channel, s}
	wm.Lock()
	wm.users[u] = true
	wm.Unlock()
	logger.GetRequestEntry(r).Debug("user Listen Start")
	err = u.Listen(func(data []byte) (err error) {
		h := func(channel string, data []byte) (d []byte, err error) {
			return data, nil
		}
		var command = ChannelCommand{}
		err = json.Unmarshal(data, &command)
		if err != nil {
			return
		}

		var reply []byte
		logger.GetRequestEntry(r).Debugf("client receive command %s", data)
		//訂閱處理
		if command.Event == SubscribeEvent {
			if _, ok := u.channel[command.Data.Channel]; ok {
				logger.GetRequestEntry(r).Debugf("sub %s@%s channel", app_key, command.Data.Channel)
				u.Subscribe(app_key+"@"+command.Data.Channel, h)
				u.channel[command.Data.Channel] = true
				command.Event = SubscribeReplySucceeded
				reply, err = json.Marshal(command)
				if err != nil {
					logger.GetRequestEntry(r).Debugf("sub sucess reply %s", err)
				}
			} else {
				command.Event = SubscribeReplyError
				reply, err = json.Marshal(command)
				if err != nil {
					logger.GetRequestEntry(r).Debugf("sub error reply %s", err)
				}

			}

			u.Send(reply)
			return
		}

		//反訂閱處理
		if command.Event == UnSubscribeEvent {
			if _, ok := u.channel[command.Data.Channel]; ok {
				logger.GetRequestEntry(r).Debugf("unsub %s@%s channel", app_key, command.Data.Channel)
				u.Unsubscribe(app_key + "@" + command.Data.Channel)
				u.channel[command.Data.Channel] = false
				command.Event = UnSubscribeReplySucceeded
				reply, err = json.Marshal(command)
				if err != nil {
					logger.GetRequestEntry(r).Debugf("unsub sucess reply %s", err)
				}
			} else {
				command.Event = UnSubscribeReplyError
				reply, err = json.Marshal(command)
				if err != nil {
					logger.GetRequestEntry(r).Debugf("unsub error reply %s", err)
				}
			}
			u.Send(reply)
			return
		}

		return
	})
	if err != nil {
		logger.GetRequestEntry(r).Infof("disconnect %s", err)
	}
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

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/buger/jsonparser"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/syhlion/redisocket.v2"
	"github.com/syhlion/requestwork.v2"
)

type UserHandler func(u *User) (err error)

var DefaultSubHandler = func(channel string, data []byte) (d []byte, err error) {
	return data, nil
}

type User struct {
	id       string
	channels map[string]bool
	appKey   string
	request  *http.Request
	*redisocket.Client
	isLogin bool
}

type WsManager struct {
	users map[*User]bool
	*sync.RWMutex
	pool *redis.Pool
	*redisocket.Hub
	worker *requestwork.Worker
}

func (wm *WsManager) Count() int {
	return len(wm.users)
}

func (wm *WsManager) Connect(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	appKey := params["app_key"]
	if appKey == "" {
		logger.GetRequestEntry(r).Warn("app_key is nil")
		http.Error(w, "app_key is nil", 401)
		return
	}
	s, err := wm.Upgrade(w, r, nil)
	if err != nil {
		logger.GetRequestEntry(r).Warnf("upgrade ws connection %s", err)
		return
	}

	u := &User{
		appKey:  appKey,
		request: r,
		isLogin: false,
		Client:  s,
	}
	wm.Lock()
	wm.users[u] = true
	wm.Unlock()
	logger.GetRequestEntry(r).Debug("user listen start")
	time.AfterFunc(15*time.Second, func() {
		if !u.isLogin {
			logger.GetRequestEntry(u.request).Debug("login timeout")
			u.Close()
		}
	})
	err = u.Listen(func(data []byte) (err error) {
		logger.GetRequestEntry(r).Debugf("client receive command %s", data)
		//訂閱處理
		if !u.isLogin {
			val, err := jsonparser.GetString(data, "event")
			if err != nil {
				return err
			}
			if val != LoginEvent {
				s := fmt.Sprintf("event error %s", val)
				return errors.New(s)
			}
			d, _, _, err := jsonparser.Get(data, "data", "jwt")
			if err != nil {
				logger.Debug(err)
				return err
			}
			v := url.Values{}

			v.Add("data", string(d))
			req, err := http.NewRequest("POST", decode_service, bytes.NewBufferString(v.Encode()))

			if err != nil {
				logger.GetRequestEntry(r).Warn(err)
				return err
			}
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Add("Content-Length", strconv.Itoa(len(v.Encode())))

			logger.GetRequestEntry(u.request).Debugf("login message: %s", d)
			ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
			a := &JwtPack{}
			err = wm.worker.Execute(ctx, req, func(resp *http.Response, e error) (err error) {
				if e != nil {
					logger.Debug(e)
					return e
				}
				defer resp.Body.Close()
				err = json.NewDecoder(resp.Body).Decode(a)
				if err != nil {
					logger.Debug(err)
					return
				}
				return
			})
			if err != nil {
				return err
			}
			if a.Gusher.AppKey != u.appKey {
				err = errors.New("app_key error")
				return err
			}
			logger.GetRequestEntry(u.request).Debugf("login parse sucess: %v", a)
			if len(a.Gusher.Channels) == 0 {
				err = errors.New("no channels")
				return err
			}
			channels := make(map[string]bool)
			for _, c := range a.Gusher.Channels {
				channels[c] = false
			}
			u.id = a.Gusher.UserId
			u.channels = channels
			u.isLogin = true
			return nil
		}

		h, err := CommanRouter(data)
		if err != nil {
			return
		}
		err = h(u)
		if err != nil {
			return
		}

		return
	})
	if err != nil {
		logger.GetRequestEntry(r).Debugf("disconnect %s", err)
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
func SubscribeCommand(data []byte) (h UserHandler, err error) {

	channel, err := jsonparser.GetString(data, "channel")
	if err != nil {
		return
	}
	h = func(u *User) (err error) {
		command := &ChannelCommand{}
		var reply []byte
		if _, ok := u.channels[channel]; ok {
			logger.GetRequestEntry(u.request).Debugf("sub %s@%s channel", u.appKey, channel)
			u.On(u.appKey+"@"+channel, DefaultSubHandler)
			u.channels[channel] = true
			command.Event = SubscribeReplySucceeded
			command.Data.Channel = channel
			reply, err = json.Marshal(command)
			if err != nil {
				logger.GetRequestEntry(u.request).Debugf("sub sucess reply %s", err)
			}
		} else {
			command.Event = SubscribeReplyError
			reply, err = json.Marshal(command)
			if err != nil {
				logger.GetRequestEntry(u.request).Debugf("sub error reply %s", err)
			}

		}

		u.Send(reply)
		return
	}
	return
}
func UnSubscribeCommand(data []byte) (h UserHandler, err error) {
	channel, err := jsonparser.GetString(data, "channel")
	if err != nil {
		return
	}
	h = func(u *User) (err error) {
		command := &ChannelCommand{}
		var reply []byte
		//反訂閱處理
		if _, ok := u.channels[channel]; ok {
			logger.GetRequestEntry(u.request).Debugf("unsub %s@%s channel", u.appKey, channel)
			u.Off(u.appKey + "@" + channel)
			u.channels[command.Data.Channel] = false
			command.Event = UnSubscribeReplySucceeded
			command.Data.Channel = channel
			reply, err = json.Marshal(command)
			if err != nil {
				logger.GetRequestEntry(u.request).Debugf("unsub sucess reply %s", err)
			}
		} else {
			command.Event = UnSubscribeReplyError
			reply, err = json.Marshal(command)
			if err != nil {
				logger.GetRequestEntry(u.request).Debugf("unsub error reply %s", err)
			}
		}
		u.Send(reply)
		return
	}
	return
}

func CommanRouter(data []byte) (h UserHandler, err error) {

	val, err := jsonparser.GetString(data, "event")
	if err != nil {
		return
	}
	d, _, _, err := jsonparser.Get(data, "data")
	if err != nil {
		return
	}
	switch val {
	case SubscribeEvent:
		h, err = SubscribeCommand(d)
		break
	case UnSubscribeEvent:
		h, err = UnSubscribeCommand(d)
		break
	default:
		err = errors.New("event errors")
		break
	}
	return
}

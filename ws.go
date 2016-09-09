package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/buger/jsonparser"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/syhlion/redisocket.v2"
	"github.com/syhlion/requestwork.v2"
)

var DefaultSubHandler = func(channel string, data []byte) (d []byte, err error) {
	return data, nil
}

type User struct {
	id       string
	channels map[string]bool
	app_key  string
	request  *http.Request
	*redisocket.Client
	isLogin bool
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
	params := mux.Vars(r)
	app_key := params["app_key"]
	if app_key == "" {
		logger.GetRequestEntry(r).Warn("app_key is nil")
		http.Error(w, "app_key is nil", 401)
		return
	}
	s, err := rsocket.NewClient(w, r)
	if err != nil {
		logger.GetRequestEntry(r).Warnf("upgrade ws connection %s", err)
		http.Error(w, err.Error(), 401)
		return
	}

	u := &User{
		app_key: app_key,
		request: r,
		isLogin: false,
		Client:  s,
	}
	wm.Lock()
	wm.users[u] = true
	wm.Unlock()
	logger.GetRequestEntry(r).Debug("user Listen Start")
	time.AfterFunc(15*time.Second, func() {
		if !u.isLogin {
			logger.GetRequestEntry(u.request).Debug("login timeout")
			u.Close()
		}
	})
	err = u.Listen(func(data []byte) (err error) {
		logger.GetRequestEntry(r).Debugf("client receive command %s", data)
		//訂閱處理
		err = CommanRouter(data, u)
		if err != nil {
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
func LoginCommand(data []byte, u *User) (err error) {
	if u.isLogin {
		return
	}
	d, _, _, err := jsonparser.Get(data, "jwt")
	if err != nil {
		logger.GetRequestEntry(u.request).Debug(err)
		return
	}

	a := &Auth{}
	logger.GetRequestEntry(u.request).Debugf("login message: %s", d)
	err = client.Call("JWT_RSA_Decoder.Decode", d, a)
	if err != nil {
		return
	}
	logger.GetRequestEntry(u.request).Debugf("login parse scuess: %v", a)
	if len(a.Channels) == 0 {
		err = errors.New("no channels")
		return
	}
	u.id = a.UserId
	channels := make(map[string]bool)
	for _, c := range a.Channels {
		channels[c] = false
	}
	u.id = a.UserId
	u.channels = channels
	u.isLogin = true
	return

}
func SubscribeCommand(data []byte, u *User) (err error) {

	channel, err := jsonparser.GetString(data, "channel")
	if err != nil {
		return
	}
	command := &ChannelCommand{}
	var reply []byte
	if _, ok := u.channels[channel]; ok {
		logger.GetRequestEntry(u.request).Debugf("sub %s@%s channel", u.app_key, channel)
		u.Subscribe(u.app_key+"@"+channel, DefaultSubHandler)
		u.channels[channel] = true
		command.Event = SubscribeReplySucceeded
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
func UnSubscribeCommand(data []byte, u *User) (err error) {
	channel, err := jsonparser.GetString(data, "channel")
	if err != nil {
		return
	}
	command := &ChannelCommand{}
	var reply []byte
	//反訂閱處理
	if _, ok := u.channels[command.Data.Channel]; ok {
		logger.GetRequestEntry(u.request).Debugf("unsub %s@%s channel", u.app_key, channel)
		u.Unsubscribe(u.app_key + "@" + channel)
		u.channels[command.Data.Channel] = false
		command.Event = UnSubscribeReplySucceeded
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

func CommanRouter(data []byte, u *User) (err error) {

	val, err := jsonparser.GetString(data, "event")
	if err != nil {
		return
	}
	d, _, _, err := jsonparser.Get(data, "data")
	if err != nil {
		return
	}
	switch val {
	case LoginEvent:
		err = LoginCommand(d, u)
		break
	case SubscribeEvent:
		err = SubscribeCommand(d, u)
		break
	case UnSubscribeEvent:
		err = UnSubscribeCommand(d, u)
		break
	}
	return
}

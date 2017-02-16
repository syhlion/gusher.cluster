package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/buger/jsonparser"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/syhlion/greq"
	"github.com/syhlion/redisocket.v2"
)

type UserHandler func(u *User) (err error)

var DefaultSubHandler = func(channel string, p *redisocket.Payload) (err error) {
	return nil
}

type User struct {
	id       string
	channels map[string]bool
	appKey   string
	request  *http.Request
	*redisocket.Client
}

type WsManager struct {
	users map[*User]bool
	*sync.RWMutex
	pool *redis.Pool
	*redisocket.Hub
	client *greq.Client
}

func (wm *WsManager) Auth(sc SlaveConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		jwt := r.FormValue("jwt")

		v := url.Values{}
		v.Add("data", jwt)
		req, err := http.NewRequest("POST", sc.DecodeServiceAddr, bytes.NewBufferString(v.Encode()))

		if err != nil {
			logger.GetRequestEntry(r).Warn(err)
			http.Error(w, "jwt decode fail", http.StatusUnauthorized)
			return
		}
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Content-Length", strconv.Itoa(len(v.Encode())))

		b, _, err := wm.client.Post(sc.DecodeServiceAddr, v)
		if err != nil {
			logger.GetRequestEntry(r).Warn(err)
			http.Error(w, "jwt decode fail", http.StatusUnauthorized)
			return
		}
		a := &JwtPack{}
		err = json.Unmarshal(b, a)
		if err != nil {
			logger.GetRequestEntry(r).Warn(err)
			http.Error(w, "jwt decode fail", http.StatusUnauthorized)
			return
		}
		logger.GetRequestEntry(r).Debugf("request jwt: %s", jwt)
		conn := wm.pool.Get()
		defer conn.Close()
		b, err = json.Marshal(a.Gusher)
		if err != nil {
			logger.GetRequestEntry(r).Debug(err)
			http.Error(w, "jwt decode fail", http.StatusUnauthorized)
			return
		}
		uid := uuid.NewV1()
		conn.Send("SET", uid.String(), string(b))
		conn.Send("EXPIRE", uid.String(), 60)
		conn.Flush()
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(struct {
			Token string `json:"token"`
		}{
			Token: uid.String(),
		})
		return
	}

}

func (wm *WsManager) Count() int {
	return len(wm.users)
}

func (wm *WsManager) Connect(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	appKey := params["app_key"]
	token := r.FormValue("token")
	if appKey == "" || token == "" {
		logger.GetRequestEntry(r).Warn("app_key or token is nil")
		http.Error(w, "app_key is nil", http.StatusUnauthorized)
		return
	}
	conn := wm.pool.Get()
	reply, err := redis.Bytes(conn.Do("GET", token))
	if err != nil {
		conn.Close()
		logger.GetRequestEntry(r).Warn(err)
		http.Error(w, "token error", http.StatusUnauthorized)
		return
	}
	conn.Close()
	auth := Auth{}
	err = json.Unmarshal(reply, &auth)
	if err != nil {
		logger.GetRequestEntry(r).Warn(err)
		http.Error(w, "token error", http.StatusUnauthorized)
		return
	}
	if appKey != auth.AppKey {
		http.Error(w, "appkey error", http.StatusUnauthorized)
		return
	}
	channels := make(map[string]bool)
	for _, v := range auth.Channels {
		channels[v] = false
	}

	s, err := wm.Upgrade(w, r, nil)
	if err != nil {
		logger.GetRequestEntry(r).Warnf("upgrade ws connection %s", err)
		return
	}

	u := &User{
		appKey:   appKey,
		request:  r,
		Client:   s,
		channels: channels,
	}
	wm.Lock()
	wm.users[u] = true
	wm.Unlock()
	logger.GetRequestEntry(r).Debug("user listen start")
	err = u.Listen(func(data []byte) (err error) {
		logger.GetRequestEntry(r).Debugf("client receive command %s", data)
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
	u.Close()
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

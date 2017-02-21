package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/buger/jsonparser"
	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/syhlion/greq"
	"github.com/syhlion/redisocket.v2"
)

var DefaultSubHandler = func(channel string, p *redisocket.Payload) (err error) {
	return nil
}

func WsAuth(sc SlaveConfig, pool *redis.Pool, reqClient *greq.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		jwt := r.FormValue("jwt")

		v := url.Values{}
		v.Add("data", jwt)

		b, _, err := reqClient.Post(sc.DecodeServiceAddr, v)
		if err != nil {
			logger.WithError(err).Warn("post decode service error")
			http.Error(w, "jwt decode fail", http.StatusUnauthorized)
			return
		}
		a := &JwtPack{}
		err = json.Unmarshal(b, a)
		if err != nil {
			logger.WithError(err).Warn("json marshal error")
			http.Error(w, "jwt decode fail", http.StatusUnauthorized)
			return
		}
		b, err = json.Marshal(a.Gusher)
		if err != nil {
			logger.WithError(err).Warn("json marshl error")
			http.Error(w, "jwt decode fail", http.StatusUnauthorized)
			return
		}
		uid := uuid.NewV1()
		conn := pool.Get()
		defer conn.Close()
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

func WsConnect(sc SlaveConfig, pool *redis.Pool, rHub *redisocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		appKey := params["app_key"]
		token := r.FormValue("token")
		if appKey == "" || token == "" {
			logger.Warn("app_key or token is nil")
			http.Error(w, "app_key is nil", http.StatusUnauthorized)
			return
		}
		conn := pool.Get()
		reply, err := redis.Bytes(conn.Do("GET", token))
		if err != nil {
			conn.Close()
			logger.WithError(err).Warn("redis get error")
			http.Error(w, "token error", http.StatusUnauthorized)
			return
		}
		conn.Close()
		auth := Auth{}
		err = json.Unmarshal(reply, &auth)
		if err != nil {
			logger.WithError(err).Warn("json unmarshal error")
			http.Error(w, "token error", http.StatusUnauthorized)
			return
		}
		if appKey != auth.AppKey {
			http.Error(w, "appkey error", http.StatusUnauthorized)
			return
		}

		s, err := rHub.Upgrade(w, r, nil)
		if err != nil {
			logger.WithError(err).Warnf("upgrade ws connection error")
			return
		}
		defer s.Close()

		s.Listen(func(data []byte) (msg *redisocket.ReceiveMsg, err error) {
			h, err := CommanRouter(data)
			if err != nil {
				return
			}

			d, _, _, err := jsonparser.Get(data, "data")
			if err != nil {
				return
			}
			return h(appKey, auth, d)
		})
		return
	}

}

func SubscribeCommand(appkey string, auth Auth, data []byte) (msg *redisocket.ReceiveMsg, err error) {

	channel, err := jsonparser.GetString(data, "channel")
	if err != nil {
		return
	}
	msg = &redisocket.ReceiveMsg{
		EventHandler: DefaultSubHandler,
		Sub:          true,
	}
	command := &ChannelCommand{}
	exist := false
	for _, ch := range auth.Channels {
		if ch == channel {
			exist = true
			break
		}
	}
	var reply []byte
	if exist {
		msg.Event = appkey + "@" + channel
		command.Event = SubscribeReplySucceeded
		command.Data.Channel = channel
		reply, err = json.Marshal(command)
		if err != nil {
			return
		}
		msg.ResponseMsg = reply
	} else {
		command.Event = SubscribeReplyError
		reply, err = json.Marshal(command)
		if err != nil {
			return
		}
		msg.ResponseMsg = reply

	}

	return
}
func UnSubscribeCommand(appkey string, auth Auth, data []byte) (msg *redisocket.ReceiveMsg, err error) {
	channel, err := jsonparser.GetString(data, "channel")
	if err != nil {
		return
	}
	exist := false
	for _, ch := range auth.Channels {
		if ch == channel {
			exist = true
			break
		}
	}
	msg = &redisocket.ReceiveMsg{
		Sub: false,
	}
	command := &ChannelCommand{}
	var reply []byte
	//反訂閱處理
	if exist {
		msg.Event = appkey + "@" + channel
		command.Event = UnSubscribeReplySucceeded
		command.Data.Channel = channel
		reply, err = json.Marshal(command)
		if err != nil {
			return
		}
		msg.ResponseMsg = reply
	} else {
		command.Event = UnSubscribeReplyError
		reply, err = json.Marshal(command)
		if err != nil {
			return
		}
		msg.ResponseMsg = reply
	}
	return
}

func CommanRouter(data []byte) (fn func(appkey string, auth Auth, data []byte) (msg *redisocket.ReceiveMsg, err error), err error) {

	val, err := jsonparser.GetString(data, "event")
	if err != nil {
		return
	}
	switch val {
	case SubscribeEvent:
		return SubscribeCommand, nil
	case UnSubscribeEvent:
		return UnSubscribeCommand, nil
	default:
		err = errors.New("event errors")
		break
	}
	return
}

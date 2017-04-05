package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/Sirupsen/logrus"
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

type commandResponse struct {
	sub     bool
	handler func(string, *redisocket.Payload) (err error)
	msg     []byte
	event   string
}

func Ping() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	}
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
func WtfConnect(sc SlaveConfig, pool *redis.Pool, rHub *redisocket.Hub) http.HandlerFunc {
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

		s, err := rHub.Upgrade(w, r, nil, auth.UserId, appKey)
		if err != nil {
			logger.WithError(err).Warnf("upgrade ws connection error")
			return
		}
		defer s.Close()

		s.Listen(func(data []byte) (b []byte, err error) {
			logger.WithFields(logrus.Fields{
				"data": string(data),
			}).Info("receive start")
			h, err := CommanRouter(data)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"data": string(data),
				}).WithError(err).Warn("command router error")
				return
			}

			d, _, _, err := jsonparser.Get(data, "data")
			if err != nil {
				logger.WithFields(logrus.Fields{
					"data": string(data),
				}).WithError(err).Warn("jsonparser data error")
				return
			}
			logger.WithFields(logrus.Fields{
				"data":  string(data),
				"pdata": string(d),
			}).Info("receive to sub")
			res, err := h(appKey, auth, d)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"data":  string(data),
					"pdata": string(d),
				}).WithError(err).Warn("sub error")
				return
			}
			if res.sub {
				s.On(res.event, res.handler)
			} else {
				s.Off(res.event)
			}
			return res.msg, nil

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

		s, err := rHub.Upgrade(w, r, nil, auth.UserId, appKey)
		if err != nil {
			logger.WithError(err).Warnf("upgrade ws connection error")
			return
		}
		defer s.Close()

		s.Listen(func(data []byte) (b []byte, err error) {
			h, err := CommanRouter(data)
			if err != nil {
				return
			}

			d, _, _, err := jsonparser.Get(data, "data")
			if err != nil {
				return
			}
			res, err := h(appKey, auth, d)
			if err != nil {
				return
			}
			if res.sub {
				s.On(res.event, res.handler)
			} else {
				s.Off(res.event)
			}
			return res.msg, nil
		})
		return
	}

}

func SubscribeCommand(appkey string, auth Auth, data []byte) (msg *commandResponse, err error) {

	channel, err := jsonparser.GetString(data, "channel")
	if err != nil {
		return
	}
	msg = &commandResponse{
		handler: DefaultSubHandler,
		sub:     true,
	}
	command := &ChannelCommand{}
	exist := false
	for _, ch := range auth.Channels {
		//新增萬用字元  如果找到這個 任何頻道皆可訂閱
		if ch == "*" {
			exist = true
			break
		}
		ech := regexp.QuoteMeta(ch)
		rch := strings.Replace(ech, `\*`, "*", -1)
		r, err := regexp.Compile(rch)
		if err != nil {
			break
		}
		if r.MatchString(channel) {
			exist = true
			break
		}
	}
	var reply []byte
	if exist {
		msg.event = channel
		command.Event = SubscribeReplySucceeded
		command.Data.Channel = channel
		reply, err = json.Marshal(command)
		if err != nil {
			return
		}
		msg.msg = reply
	} else {
		command.Event = SubscribeReplyError
		reply, err = json.Marshal(command)
		if err != nil {
			return
		}
		msg.msg = reply

	}

	return
}
func UnSubscribeCommand(appkey string, auth Auth, data []byte) (msg *commandResponse, err error) {
	channel, err := jsonparser.GetString(data, "channel")
	if err != nil {
		return
	}
	exist := false
	for _, ch := range auth.Channels {
		//新增萬用字元  如果找到這個 任何頻道皆可訂閱
		if ch == "*" {
			exist = true
			break
		}
		ech := regexp.QuoteMeta(ch)
		rch := strings.Replace(ech, `\*`, "*", -1)
		r, err := regexp.Compile(rch)
		if err != nil {
			break
		}
		if r.MatchString(channel) {
			exist = true
			break
		}
	}
	msg = &commandResponse{
		sub: false,
	}
	command := &ChannelCommand{}
	var reply []byte
	//反訂閱處理
	if exist {
		msg.event = channel
		command.Event = UnSubscribeReplySucceeded
		command.Data.Channel = channel
		reply, err = json.Marshal(command)
		if err != nil {
			return
		}
		msg.msg = reply
	} else {
		command.Event = UnSubscribeReplyError
		reply, err = json.Marshal(command)
		if err != nil {
			return
		}
		msg.msg = reply
	}
	return
}

func CommanRouter(data []byte) (fn func(appkey string, auth Auth, data []byte) (msg *commandResponse, err error), err error) {

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

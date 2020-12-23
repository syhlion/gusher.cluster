package main

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/syhlion/greq"
	"github.com/syhlion/redisocket.v2"
)

var DefaultSubHandler = func(channel string, p *redisocket.Payload) (err error) {
	return nil
}

type commandResponse struct {
	cmdType   string
	handler   func(string, *redisocket.Payload) (err error)
	msg       []byte
	data      string
	multiData []string //multi sub use
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
func WtfConnect(sc SlaveConfig, pool *redis.Pool, jobPool *redis.Pool, rHub *redisocket.Hub, reqClient *greq.Client) http.HandlerFunc {
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
			logger.WithError(err).Warn("token get nil")
			http.Error(w, "token error", http.StatusUnauthorized)
			return
		}
		conn.Close()
		auth := &redisocket.Auth{}
		err = json.Unmarshal(reply, auth)
		if err != nil {
			logger.WithError(err).Warn("json unmarshal error")
			http.Error(w, "token error", http.StatusUnauthorized)
			return
		}
		if appKey != auth.AppKey {
			http.Error(w, "appkey error", http.StatusUnauthorized)
			return
		}

		s, err := rHub.Upgrade(w, r, nil, auth.UserId, appKey, auth)
		if err != nil {
			logger.WithError(err).Warnf("upgrade ws connection error")
			return
		}
		defer s.Close()

		s.Listen(func(data []byte) (b []byte, err error) {
			logger.WithFields(logrus.Fields{
				"data": string(data),
			}).Info("receive start")
			h, err := CommanRouter(data, jobPool)
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
			res, err := h(appKey, s.GetAuth(), d, s.SocketId(), true)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"data":  string(data),
					"pdata": string(d),
					"res":   res,
				}).WithError(err).Warn("sub error")
				return
			}
			switch res.cmdType {
			case "SUB":
				s.On(res.data, res.handler)
			case "UNSUB":
				s.Off(res.data)

			}
			logger.WithFields(logrus.Fields{
				"data":  string(data),
				"pdata": string(d),
				"res":   res,
			}).Info("receive to sub")
			return res.msg, nil

		})
		return
	}

}

func WsConnect(sc SlaveConfig, pool *redis.Pool, jobPool *redis.Pool, rHub *redisocket.Hub, reqClient *greq.Client) http.HandlerFunc {
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
			logger.WithError(err).Warn("token get nil")
			http.Error(w, "token error", http.StatusUnauthorized)
			return
		}
		conn.Close()
		auth := &redisocket.Auth{}
		err = json.Unmarshal(reply, auth)
		if err != nil {
			logger.WithError(err).Warn("json unmarshal error")
			http.Error(w, "token error", http.StatusUnauthorized)
			return
		}
		if appKey != auth.AppKey {
			http.Error(w, "appkey error", http.StatusUnauthorized)
			return
		}

		s, err := rHub.Upgrade(w, r, nil, auth.UserId, appKey, auth)
		if err != nil {
			logger.WithError(err).Warnf("upgrade ws connection error")
			return
		}
		defer s.Close()

		s.Listen(func(data []byte) (b []byte, err error) {
			h, err := CommanRouter(data, jobPool)
			if err != nil {
				logger.WithError(err).Info("router error")
				return
			}
			d, _, _, err := jsonparser.Get(data, "data")
			if err != nil {
				logger.WithError(err).Info("get data error")
				return
			}
			debug, err := jsonparser.GetBoolean(data, "debug")
			if err != nil {
				debug = false
			}
			res, err := h(appKey, s.GetAuth(), d, s.SocketId(), debug)
			if err != nil {
				logger.WithError(err).Info("handler error")
				return
			}
			switch res.cmdType {
			case "SUB":
				s.On(res.data, res.handler)
			case "MULTISUB":
				for _, v := range res.multiData {
					s.On(v, res.handler)
				}
			case "UNSUB":
				s.Off(res.data)

			}
			return res.msg, nil
		})
		return
	}

}
func MultiSubscribeCommand(appkey string, auth redisocket.Auth, data []byte, socketId string, debug bool) (msg *commandResponse, err error) {

	multiChannel := make([]string, 0)
	_, err = jsonparser.ArrayEach(data, func(v []byte, dataType jsonparser.ValueType, offset int, err error) {
		if err != nil {
			return
		}
		multiChannel = append(multiChannel, string(v))

	}, "multi_channel")
	msg = &commandResponse{
		handler: DefaultSubHandler,
		cmdType: "MULTISUB",
	}
	command := &ChannelCommand{}
	var exist bool
	for _, ch := range auth.Channels {
		//新增萬用字元  如果找到這個 任何頻道皆可訂閱
		if ch == "*" {
			exist = true
			break
		}
	}
	subChannels := make([]string, 0)
	if exist {
		subChannels = multiChannel
	} else {
		isMatch := true
		for _, ch := range multiChannel {
			if !InArray(ch, auth.Channels) {
				isMatch = false
				break
			}
		}
		if isMatch {
			subChannels = multiChannel
		}
	}
	var reply []byte
	if len(subChannels) > 0 {
		msg.multiData = subChannels
		command.Event = MultiSubscribeReplySucceeded
		command.SocketId = socketId
		command.Data.Channel = subChannels
		reply, err = json.Marshal(command)
		if err != nil {
			return
		}
		msg.msg = reply
	} else {

		//TODO 需重構 不讓他進入訂閱模式
		msg.cmdType = ""
		command.Event = MultiSubscribeReplyError
		command.SocketId = socketId
		command.Data.Channel = multiChannel
		reply, err = json.Marshal(command)
		if err != nil {
			return
		}
		msg.msg = reply

	}

	return
}

func SubscribeCommand(appkey string, auth redisocket.Auth, data []byte, socketId string, debug bool) (msg *commandResponse, err error) {

	channel, err := jsonparser.GetString(data, "channel")
	if err != nil {
		return
	}
	msg = &commandResponse{
		handler: DefaultSubHandler,
		cmdType: "SUB",
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
		rch := strings.Replace(ech, `\*`, ".+", -1)
		r := regexp.MustCompile("^" + rch + "$")

		if r.MatchString(channel) {
			exist = true
			break
		}
	}
	var reply []byte
	if exist {
		msg.data = channel
		command.SocketId = socketId
		command.Event = SubscribeReplySucceeded
		command.Data.Channel = channel
		reply, err = json.Marshal(command)
		if err != nil {
			return
		}
		msg.msg = reply
	} else {

		//TODO 需重構 不讓他進入訂閱模式
		msg.cmdType = ""
		command.SocketId = socketId
		command.Event = SubscribeReplyError
		command.Data.Channel = channel
		reply, err = json.Marshal(command)
		if err != nil {
			return
		}
		msg.msg = reply

	}

	return
}
func QueryChannelCommand(appkey string, auth redisocket.Auth, data []byte, socketId string, debug bool) (msg *commandResponse, err error) {
	msg = &commandResponse{
		handler: DefaultSubHandler,
		cmdType: "QUERYCHANNEL",
	}

	command := &QueryChannelResponse{}
	command.Event = QueryChannelReplySucceeded
	command.SocketId = socketId
	command.Data = struct {
		Channels []string `json:"channels"`
	}{
		Channels: auth.Channels,
	}

	reply, err := json.Marshal(command)
	if err != nil {
		return
	}
	msg.msg = reply
	return
}

func PingPongCommand(appkey string, auth redisocket.Auth, data []byte, socketId string, debug bool) (msg *commandResponse, err error) {
	msg = &commandResponse{
		handler: DefaultSubHandler,
		cmdType: "PING",
	}

	command := &PongResponse{}
	command.Event = QueryChannelReplySucceeded
	command.SocketId = socketId
	command.Data = data
	command.Time = time.Now().Unix()

	reply, err := json.Marshal(command)
	if err != nil {
		return
	}
	msg.msg = reply
	return
}
func Remote(pool *redis.Pool) func(string, redisocket.Auth, []byte, string, bool) (msg *commandResponse, err error) {
	return func(appkey string, auth redisocket.Auth, data []byte, socketId string, debug bool) (msg *commandResponse, err error) {

		remote, err := jsonparser.GetString(data, "remote")
		if err != nil {
			return
		}
		uid, err := jsonparser.GetString(data, "uid")
		if err != nil {
			return
		}
		payload, _, _, err := jsonparser.Get(data, "payload")
		if err != nil {
			return
		}
		p := JsonCheck(string(payload))
		msg = &commandResponse{
			cmdType: "REMOTE",
		}
		var reply []byte
		command := &RemoteCommand{}
		command.Data.Remote = remote
		b, ok := auth.Remotes[remote]

		//沒有這個remote 返回錯誤訊息不斷線
		if !ok || !b {
			command.Event = RemoteReplyError
			command.SocketId = socketId
			reply, err = json.Marshal(command)
			if err != nil {
				return
			}
			msg.msg = reply
			return
		}
		wp := WorkerPayload{
			UserId:   auth.UserId,
			Data:     p,
			Uid:      uid,
			SocketId: socketId,
			AppKey:   auth.AppKey,
		}
		d, err := json.Marshal(wp)
		conn := pool.Get()
		defer conn.Close()
		_, err = conn.Do("RPUSH", auth.AppKey+"@"+remote, d)
		if err != nil {
			return
		}
		command.Event = RemoteReplySucceeded
		command.SocketId = socketId
		reply, err = json.Marshal(command)
		if err != nil {
			return
		}
		if debug {
			msg.msg = reply
		}

		return
	}

}
func UnSubscribeCommand(appkey string, auth redisocket.Auth, data []byte, socketId string, debug bool) (msg *commandResponse, err error) {
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
		rch := strings.Replace(ech, `\*`, ".+", -1)
		r := regexp.MustCompile("^" + rch + "$")

		if r.MatchString(channel) {
			exist = true
			break
		}
	}
	msg = &commandResponse{
		cmdType: "UNSUB",
	}
	command := &ChannelCommand{}
	var reply []byte
	//反訂閱處理
	if exist {
		msg.data = channel
		command.Event = UnSubscribeReplySucceeded
		command.SocketId = socketId
		command.Data.Channel = channel
		reply, err = json.Marshal(command)
		if err != nil {
			return
		}
		msg.msg = reply
	} else {
		msg.data = channel

		//TODO 需重構 先不讓他進入訂閱模式
		msg.cmdType = ""
		command.Event = UnSubscribeReplyError
		command.SocketId = socketId
		command.Data.Channel = channel
		reply, err = json.Marshal(command)
		if err != nil {
			return
		}
		msg.msg = reply
	}
	return
}

func CommanRouter(data []byte, pool *redis.Pool) (fn func(appkey string, auth redisocket.Auth, data []byte, socketId string, debug bool) (msg *commandResponse, err error), err error) {

	val, err := jsonparser.GetString(data, "event")
	if err != nil {
		return
	}
	switch val {
	case RemoteEvent:
		return Remote(pool), nil
	case QueryChannelEvent:
		return QueryChannelCommand, nil
	case SubscribeEvent:
		return SubscribeCommand, nil
	case MultiSubscribeEvent:
		return MultiSubscribeCommand, nil
	case UnSubscribeEvent:
		return UnSubscribeCommand, nil
	case PingEvent:
		return PingPongCommand, nil
	default:
		err = errors.New("event errors")
		break
	}
	return
}

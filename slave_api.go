package main

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/buger/jsonparser"
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

// Healthz is a liveness probe: 200 as long as the process is serving.
func Healthz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}
}

// Version reports the build version. GET /version
func Version(v string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(v))
	}
}

// POST /v1/auth  body: {jwt} -> {token}
func WsAuth(sc SlaveConfig, pubKey *rsa.PublicKey) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		var req JwtRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "body decode fail", http.StatusBadRequest)
			return
		}
		jwtStr := req.Jwt
		// 本機驗 JWT;通過即把「JWT 本身」當 token 回給 client(無狀態、無 redis)。
		// client 後續 /ws?token=<JWT>,該端點再本機驗一次 → 全程零 redis、保留原兩步流程。
		if _, err := Decode(pubKey, jwtStr); err != nil {
			logger.WithError(err).Warn("jwt local decode error")
			http.Error(w, "jwt decode fail", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(struct {
			Token string `json:"token"`
		}{Token: jwtStr})
	}
}

// WsConnect upgrades GET /v1/apps/{app}/ws?token=<JWT> to a WebSocket.
// token 參數即 JWT,本機 RSA 公鑰驗證 → auth,無 redis。
func WsConnect(sc SlaveConfig, pubKey *rsa.PublicKey, rHub *redisocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appKey := r.PathValue("app")
		token := r.URL.Query().Get("token") // token 即 JWT
		if appKey == "" || token == "" {
			logger.Warn("app_key or token is nil")
			http.Error(w, "app_key is nil", http.StatusUnauthorized)
			return
		}
		jp, err := Decode(pubKey, token)
		if err != nil {
			logger.WithError(err).Warn("jwt decode error")
			http.Error(w, "token error", http.StatusUnauthorized)
			return
		}
		auth := jp.Gusher
		if appKey != auth.AppKey {
			http.Error(w, "appkey error", http.StatusUnauthorized)
			return
		}

		s, err := rHub.Upgrade(w, r, nil, auth.UserId, appKey, &auth)
		if err != nil {
			logger.WithError(err).Warnf("upgrade ws connection error")
			return
		}
		defer s.Close()

		t1 := time.Now()
		logger.WithFields(Fields{
			"conn_at":   t1,
			"socket_id": s.SocketId(),
			"user_id":   auth.UserId,
		}).Info("connect")
		s.Listen(func(data []byte) (b []byte, err error) {
			h, err := CommanRouter(data)
			if err != nil {
				logger.WithField("socket_id", s.SocketId()).WithError(err).Info("router error")
				return
			}
			d, _, _, err := jsonparser.Get(data, "data")
			if err != nil {
				logger.WithField("socket_id", s.SocketId()).WithError(err).Info("get data error")
				return
			}
			debug, err := jsonparser.GetBoolean(data, "debug")
			if err != nil {
				debug = false
			}
			res, err := h(appKey, s.GetAuth(), d, s.SocketId(), debug)
			if err != nil {
				logger.WithField("socket_id", s.SocketId()).WithError(err).Info("handler error")
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
		t2 := time.Now()
		logger.WithFields(
			Fields{
				"conn_at":            t1,
				"conn_end":           t2,
				"socket_id":          s.SocketId(),
				"user_id":            auth.UserId,
				"conn_duration":      fmt.Sprintf("%v", t2.Sub(t1)),
				"conn_duration_nano": t2.Sub(t1),
			}).Info("disconnect")
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

		// cmdType 留空 → WsConnect 不會呼叫 s.On,故不進訂閱模式;只回錯誤訊息。
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

		// cmdType 留空 → WsConnect 不會呼叫 s.On,故不進訂閱模式;只回錯誤訊息。
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

		// cmdType 留空 → WsConnect 不會呼叫 s.On,故不進訂閱模式;只回錯誤訊息。
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

func CommanRouter(data []byte) (fn func(appkey string, auth redisocket.Auth, data []byte, socketId string, debug bool) (msg *commandResponse, err error), err error) {

	val, err := jsonparser.GetString(data, "event")
	if err != nil {
		return
	}
	switch val {
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

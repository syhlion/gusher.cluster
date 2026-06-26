package main

import (
	"crypto/rsa"
	"net/http"
	"regexp"

	redisocket "github.com/syhlion/redisocket.v2"
)

// writeJSON marshals v and writes it as a 200 JSON response.
func writeJSON(w http.ResponseWriter, r *http.Request, v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		logger.GetRequestEntry(r).WithError(err).Warn("json marshal error")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("json marshal error"))
		return
	}
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

// decodeBody reads a JSON request body into v. Returns false (and writes a 400)
// on malformed input.
func decodeBody(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		logger.GetRequestEntry(r).WithError(err).Warn("body decode error")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("body decode error"))
		return false
	}
	return true
}

// GET /v1/apps/{app}/channels/count
func GetAllChannelCount(rsender *redisocket.Sender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		channels, err := rsender.GetChannels(listenChannelPrefix, r.PathValue("app"), "*")
		if err != nil {
			logger.GetRequestEntry(r).WithError(err).Warn("get channels error")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("get channels error"))
			return
		}
		writeJSON(w, r, struct {
			Count int `json:"count"`
		}{Count: len(channels)})
	}
}

// GET /v1/apps/{app}/channels
func GetAllChannel(rsender *redisocket.Sender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		channels, err := rsender.GetChannels(listenChannelPrefix, r.PathValue("app"), "*")
		if err != nil {
			logger.GetRequestEntry(r).WithError(err).Warn("get channels error")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("get channels error"))
			return
		}
		writeJSON(w, r, channels)
	}
}

// GET /v1/apps/{app}/channels/{channel}/users/count
func GetOnlineCountByChannel(rsender *redisocket.Sender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		online, err := rsender.GetOnlineByChannel(listenChannelPrefix, r.PathValue("app"), r.PathValue("channel"))
		if err != nil {
			logger.GetRequestEntry(r).WithError(err).Warn("get online error")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("get online error"))
			return
		}
		writeJSON(w, r, struct {
			Count int `json:"count"`
		}{Count: len(online)})
	}
}

// GET /v1/apps/{app}/channels/{channel}/users
func GetOnlineByChannel(rsender *redisocket.Sender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		online, err := rsender.GetOnlineByChannel(listenChannelPrefix, r.PathValue("app"), r.PathValue("channel"))
		if err != nil {
			logger.GetRequestEntry(r).WithError(err).Warn("get online error")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("get online error"))
			return
		}
		writeJSON(w, r, online)
	}
}

// GET /v1/apps/{app}/users/count
func GetOnlineCount(rsender *redisocket.Sender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		online, err := rsender.GetOnline(listenChannelPrefix, r.PathValue("app"))
		if err != nil {
			logger.GetRequestEntry(r).WithError(err).Warn("get online error")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("get online error"))
			return
		}
		writeJSON(w, r, struct {
			Count int `json:"count"`
		}{Count: len(online)})
	}
}

// GET /v1/apps/{app}/users
func GetOnline(rsender *redisocket.Sender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		online, err := rsender.GetOnline(listenChannelPrefix, r.PathValue("app"))
		if err != nil {
			logger.GetRequestEntry(r).WithError(err).Warn("get online error")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("get online error"))
			return
		}
		writeJSON(w, r, online)
	}
}

// POST /v1/apps/{app}/sockets/{socket}/messages  body: {data}
func PushToSocket(rsender *redisocket.Sender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		app := r.PathValue("app")
		socketID := r.PathValue("socket")
		var req DataRequest
		if !decodeBody(w, r, &req) {
			return
		}
		rsender.PushToSid(listenChannelPrefix, app, socketID, req.Data)
		writeJSON(w, r, struct {
			SocketId string      `json:"socket_id"`
			Data     interface{} `json:"data"`
		}{SocketId: socketID, Data: req.Data})
	}
}

// POST /v1/apps/{app}/users/{user}/channels  body: {channel}
func AddUserChannels(rsender *redisocket.Sender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		app := r.PathValue("app")
		userID := r.PathValue("user")
		var req AddChannelRequest
		if !decodeBody(w, r, &req) {
			return
		}
		if req.Channel == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("channel empty error"))
			return
		}
		rsender.AddChannel(listenChannelPrefix, app, userID, req.Channel)

		a := ChannelInfoData{}
		a.Data = struct {
			Channel string `json:"channel"`
		}{Channel: req.Channel}
		a.Event = AddChannelEvent
		rsender.PushToUid(listenChannelPrefix, app, userID, a)

		writeJSON(w, r, struct {
			UserId string      `json:"user_id"`
			Data   interface{} `json:"data"`
		}{UserId: userID, Data: req.Channel})
	}
}

// PUT /v1/apps/{app}/users/{user}/channels  body: {channels}
func ReloadUserChannels(rsender *redisocket.Sender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		app := r.PathValue("app")
		userID := r.PathValue("user")
		var req ReloadChannelsRequest
		if !decodeBody(w, r, &req) {
			return
		}
		rsender.ReloadChannel(listenChannelPrefix, app, userID, req.Channels)

		a := ChannelInfoData{}
		a.Data = struct {
			Channels []string `json:"channels"`
		}{Channels: req.Channels}
		a.Event = ReloadChannelEvent
		rsender.PushToUid(listenChannelPrefix, app, userID, a)

		writeJSON(w, r, struct {
			UserId string      `json:"user_id"`
			Data   interface{} `json:"data"`
		}{UserId: userID, Data: req.Channels})
	}
}

// POST /v1/apps/{app}/users/{user}/messages  body: {data}
func PushToUser(rsender *redisocket.Sender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		app := r.PathValue("app")
		userID := r.PathValue("user")
		var req DataRequest
		if !decodeBody(w, r, &req) {
			return
		}
		rsender.PushToUid(listenChannelPrefix, app, userID, req.Data)
		writeJSON(w, r, struct {
			UserId string      `json:"user_id"`
			Data   interface{} `json:"data"`
		}{UserId: userID, Data: req.Data})
	}
}

// POST /v1/apps/{app}/messages/batch  body: [{channel,event,data}, ...]
func PushBatchMessage(rsender *redisocket.Sender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		app := r.PathValue("app")
		batchData := make([]BatchData, 0)
		if !decodeBody(w, r, &batchData) {
			return
		}
		bd := make([]redisocket.BatchData, 0, len(batchData))
		for _, data := range batchData {
			d, err := json.Marshal(struct {
				Channel string      `json:"channel"`
				Event   string      `json:"event"`
				Data    interface{} `json:"data"`
			}{Channel: data.Channel, Event: data.Event, Data: data.Data})
			if err != nil {
				logger.GetRequestEntry(r).Warn(err)
				continue
			}
			bd = append(bd, redisocket.BatchData{Data: d, Event: data.Channel})
		}
		rsender.PushBatch(listenChannelPrefix, app, bd)
		writeJSON(w, r, struct {
			Total int `json:"total"`
		}{Total: len(batchData)})
	}
}

// POST /v1/apps/{app}/messages  body: {channel_pattern, event, data}
func PushMessageByPattern(rsender *redisocket.Sender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		app := r.PathValue("app")
		var req PatternMessageRequest
		if !decodeBody(w, r, &req) {
			return
		}
		if req.ChannelPattern == "" || req.Event == "" {
			logger.GetRequestEntry(r).Warn("empty param")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("empty param"))
			return
		}
		re, err := regexp.Compile(req.ChannelPattern)
		if err != nil {
			logger.GetRequestEntry(r).Warn("pattern error")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("channel_pattern cant regex"))
			return
		}
		chs, err := rsender.GetChannels(listenChannelPrefix, app, "*")
		if err != nil {
			logger.GetRequestEntry(r).Warn("get channel error")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("get channel error"))
			return
		}
		var match int
		for _, v := range chs {
			if !re.MatchString(v) {
				continue
			}
			match++
			d, err := json.Marshal(struct {
				Channel string      `json:"channel"`
				Event   string      `json:"event"`
				Data    interface{} `json:"data"`
			}{Channel: v, Event: req.Event, Data: req.Data})
			if err != nil {
				logger.GetRequestEntry(r).Warn(err)
				continue
			}
			if _, err = rsender.Push(listenChannelPrefix, app, v, d); err != nil {
				logger.GetRequestEntry(r).Warn(err)
				continue
			}
		}
		writeJSON(w, r, struct {
			Total   int    `json:"total"`
			Pattern string `json:"pattern"`
		}{Total: match, Pattern: req.ChannelPattern})
	}
}

// POST /v1/apps/{app}/channels/{channel}/messages  body: {event, data}
func PushMessage(rsender *redisocket.Sender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		app := r.PathValue("app")
		channel := r.PathValue("channel")
		var req MessageRequest
		if !decodeBody(w, r, &req) {
			return
		}
		if req.Event == "" {
			logger.GetRequestEntry(r).Warn("empty event")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("empty event"))
			return
		}
		d, err := json.Marshal(struct {
			Channel string      `json:"channel"`
			Event   string      `json:"event"`
			Data    interface{} `json:"data"`
		}{Channel: channel, Event: req.Event, Data: req.Data})
		if err != nil {
			logger.GetRequestEntry(r).Warn(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("data error"))
			return
		}
		if _, err = rsender.Push(listenChannelPrefix, app, channel, d); err != nil {
			logger.GetRequestEntry(r).Warn(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("push error"))
			return
		}
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(d)
	}
}

// POST /v1/auth/decode  body: {jwt}
func DecodeJWT(key *rsa.PublicKey) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req JwtRequest
		if !decodeBody(w, r, &req) {
			return
		}
		auth, err := Decode(key, req.Jwt)
		if err != nil {
			logger.GetRequestEntry(r).Warnf("error:%s", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("data error"))
			return
		}
		writeJSON(w, r, auth)
	}
}

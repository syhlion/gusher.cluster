package main

import (
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"
	redisocket "github.com/syhlion/redisocket.v2"
)

func GetAllChannel(rsender *redisocket.Sender) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		channels, err := rsender.GetChannels(listenChannelPrefix, params["app_key"], "*")
		if err != nil {
			logger.GetRequestEntry(r).WithError(err).Warn("get redis error")
			w.WriteHeader(400)
			w.Write([]byte("get redis error"))
		}
		b, err := json.Marshal(channels)
		if err != nil {
			logger.GetRequestEntry(r).WithError(err).Warn("json marshal error")
			w.WriteHeader(400)
			w.Write([]byte("json marshal error"))
		}
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(b)
		return
	}
}
func GetOnlineByChannel(rsender *redisocket.Sender) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		online, err := rsender.GetOnlineByChannel(listenChannelPrefix, params["app_key"], params["channel"])
		if err != nil {
			logger.GetRequestEntry(r).WithError(err).Warn("get redis error")
			w.WriteHeader(400)
			w.Write([]byte("get redis error"))
		}
		b, err := json.Marshal(online)
		if err != nil {
			logger.GetRequestEntry(r).WithError(err).Warn("json marshal error")
			w.WriteHeader(400)
			w.Write([]byte("json marshal error"))
		}
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(b)
		return
	}
}
func GetOnline(rsender *redisocket.Sender) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		online, err := rsender.GetOnline(listenChannelPrefix, params["app_key"])
		if err != nil {
			logger.GetRequestEntry(r).WithError(err).Warn("get redis error")
			w.WriteHeader(400)
			w.Write([]byte("get redis error"))
		}
		b, err := json.Marshal(online)
		if err != nil {
			logger.GetRequestEntry(r).WithError(err).Warn("json marshal error")
			w.WriteHeader(400)
			w.Write([]byte("json marshal error"))
		}
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(b)
		return
	}
}
func PushToSocket(rsender *redisocket.Sender) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		app_key := params["app_key"]
		if app_key == "" {
			logger.GetRequestEntry(r).Warn("empty param")
			w.WriteHeader(400)
			w.Write([]byte("empty param"))
			return
		}
		socket_id := params["socket_id"]
		data := r.FormValue("data")
		j := JsonCheck(data)
		rsender.PushToSid(listenChannelPrefix, app_key, socket_id, j)
		push := struct {
			SocketId string      `json:"socket_id"`
			Data     interface{} `json:"data"`
		}{
			SocketId: socket_id,
			Data:     data,
		}
		d, err := json.Marshal(push)
		if err != nil {
			logger.GetRequestEntry(r).Warn(err)
			w.WriteHeader(400)
			w.Write([]byte("data error"))
			return
		}
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(d)
		return
	}
}
func PushToUser(rsender *redisocket.Sender) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		app_key := params["app_key"]
		if app_key == "" {
			logger.GetRequestEntry(r).Warn("empty param")
			w.WriteHeader(400)
			w.Write([]byte("empty param"))
			return
		}
		user_id := params["user_id"]
		data := r.FormValue("data")
		j := JsonCheck(data)
		rsender.PushToUid(listenChannelPrefix, app_key, user_id, j)
		push := struct {
			UserId string      `json:"user_id"`
			Data   interface{} `json:"data"`
		}{
			UserId: user_id,
			Data:   data,
		}
		d, err := json.Marshal(push)
		if err != nil {
			logger.GetRequestEntry(r).Warn(err)
			w.WriteHeader(400)
			w.Write([]byte("data error"))
			return
		}
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(d)
		return
	}
}

func PushBatchMessage(rsender *redisocket.Sender) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		app_key := params["app_key"]
		if app_key == "" {
			logger.GetRequestEntry(r).Warn("empty param")
			w.WriteHeader(400)
			w.Write([]byte("empty param"))
			return
		}
		data := r.FormValue("batch_data")
		if data == "" {
			logger.GetRequestEntry(r).Warn("empty batch data")
			w.WriteHeader(400)
			w.Write([]byte("empty batch data"))
			return
		}
		batchData := make([]BatchData, 0)
		byteData := []byte(data)
		err := json.Unmarshal(byteData, &batchData)
		if err != nil {
			logger.GetRequestEntry(r).Warn(err)
			w.WriteHeader(400)
			w.Write([]byte("data error"))
			return
		}
		l := len(batchData)
		bd := make([]redisocket.BatchData, l)
		for _, data := range batchData {
			push := struct {
				Channel string      `json:"channel"`
				Event   string      `json:"event"`
				Data    interface{} `json:"data"`
			}{
				Channel: data.Channel,
				Event:   data.Event,
				Data:    data.Data,
			}
			d, err := json.Marshal(push)
			if err != nil {
				logger.GetRequestEntry(r).Warn(err)
				continue
			}
			b := redisocket.BatchData{
				Data:  d,
				Event: data.Channel,
			}
			bd = append(bd, b)

		}
		rsender.PushBatch(listenChannelPrefix, app_key, bd)
		response := struct {
			Total int `json:"total"`
			Cap   int `json:"cap"`
		}{
			Total: len(batchData),
			Cap:   len(byteData),
		}
		d, err := json.Marshal(response)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte("response marshal error"))
			return
		}
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(d)
		return
	}
}

func PushMessageByPattern(rsender *redisocket.Sender) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		app_key := params["app_key"]
		channelPattern := r.FormValue("channel_pattern")
		event := r.FormValue("event")
		if app_key == "" || channelPattern == "" || event == "" {
			logger.GetRequestEntry(r).Warn("empty param")
			w.WriteHeader(400)
			w.Write([]byte("empty param"))
			return
		}
		re, err := regexp.Compile(channelPattern)
		if err != nil {
			logger.GetRequestEntry(r).Warn("pattern error")
			w.WriteHeader(400)
			w.Write([]byte("channel_pattern cant regex"))
			return

		}
		chs, err := rsender.GetChannels(listenChannelPrefix, app_key, "*")
		if err != nil {
			logger.GetRequestEntry(r).Warn("get channel error")
			w.WriteHeader(400)
			w.Write([]byte("get channel error"))
			return

		}

		data := r.FormValue("data")
		if data == "" {
			logger.GetRequestEntry(r).Warn("empty data")
			w.WriteHeader(400)
			w.Write([]byte("empty data"))
			return
		}
		jsonData := JsonCheck(data)

		var match int
		for _, v := range chs {
			if re.MatchString(v) {
				match++
				push := struct {
					Channel string      `json:"channel"`
					Event   string      `json:"event"`
					Data    interface{} `json:"data"`
				}{
					Channel: v,
					Event:   event,
					Data:    jsonData,
				}
				d, err := json.Marshal(push)
				if err != nil {
					logger.GetRequestEntry(r).Warn(err)
					continue
				}
				_, err = rsender.Push(listenChannelPrefix, app_key, v, d)
				if err != nil {
					logger.GetRequestEntry(r).Warn(err)
					continue
				}
			}
		}
		response := struct {
			Total   int    `json:"total"`
			Pattern string `json:"pattern"`
		}{
			Total:   match,
			Pattern: channelPattern,
		}
		d, err := json.Marshal(response)
		if err != nil {
			w.WriteHeader(400)
			w.Write([]byte("response marshal error"))
			return
		}
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(d)
		return
	}
}
func PushMessage(rsender *redisocket.Sender) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		app_key := params["app_key"]
		channel := params["channel"]
		event := params["event"]
		if app_key == "" || channel == "" || event == "" {
			logger.GetRequestEntry(r).Warn("empty param")
			w.WriteHeader(400)
			w.Write([]byte("empty param"))
			return
		}

		data := r.FormValue("data")
		if data == "" {
			logger.GetRequestEntry(r).Warn("empty data")
			w.WriteHeader(400)
			w.Write([]byte("empty data"))
			return
		}
		jsonData := JsonCheck(data)

		push := struct {
			Channel string      `json:"channel"`
			Event   string      `json:"event"`
			Data    interface{} `json:"data"`
		}{
			Channel: channel,
			Event:   event,
			Data:    jsonData,
		}
		d, err := json.Marshal(push)
		if err != nil {
			logger.GetRequestEntry(r).Warn(err)
			w.WriteHeader(400)
			w.Write([]byte("data error"))
			return
		}
		_, err = rsender.Push(listenChannelPrefix, app_key, channel, d)
		if err != nil {
			logger.GetRequestEntry(r).Warn(err)
			w.WriteHeader(400)
			w.Write([]byte("data error"))
			return
		}
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		w.Write(d)
		return
	}
}
func DecodeJWT(key *rsa.PublicKey) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		data := r.FormValue("data")
		auth, err := Decode(key, data)
		if err != nil {
			logger.GetRequestEntry(r).Warnf("error:%s, post data:%s", err, data)
			w.WriteHeader(400)
			w.Write([]byte("data error"))
			return
		}
		if err = json.NewEncoder(w).Encode(auth); err != nil {
			logger.GetRequestEntry(r).Warnf("error:%s", err)
			w.WriteHeader(400)
			w.Write([]byte("parse error"))
		}
		return
	}
}

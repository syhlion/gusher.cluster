package main

import (
	"crypto/rsa"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	redisocket "github.com/syhlion/redisocket.v2"
)

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
		_, err = rsender.Push(listenChannelPrefix, app_key+"@"+channel, d)
		if err != nil {
			logger.GetRequestEntry(r).Warn(err)
			w.WriteHeader(400)
			w.Write([]byte("data error"))
			return
		}
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

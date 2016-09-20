package main

import (
	"encoding/json"
	"net/http"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
)

func PushMessage(rpool *redis.Pool) func(w http.ResponseWriter, r *http.Request) {
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

		push := struct {
			Channel string      `json:"channel"`
			Event   string      `json:"event"`
			Data    interface{} `json:"data"`
		}{
			Channel: channel,
			Event:   event,
			Data:    data,
		}
		d, err := json.Marshal(push)
		if err != nil {
			logger.GetRequestEntry(r).Warn(err)
			w.WriteHeader(400)
			w.Write([]byte("data error"))
			return
		}
		conn := rpool.Get()
		defer conn.Close()
		_, err = conn.Do("PUBLISH", app_key+"@"+channel, d)
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
func SystemInfo(s *SlaveInfos) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(s.Info())
		return
	}
}

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
)

func checkKey(app_key string) (err error) {
	conn := rpool.Get()
	defer conn.Close()
	reply, err := redis.Int(conn.Do("HEXISTS", app_key, "url"))
	if err != nil || reply == 0 {
		err = errors.New("empty app_key")
	}
	return
}

func CheckAppKey(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	app_key := params["app_key"]
	if app_key == "" {
		logger.GetRequestEntry(r).Warn("empty app_key")
		w.WriteHeader(400)
		w.Write([]byte("empty app_key"))
		return
	}
	err := checkKey(app_key)
	if err != nil {
		logger.GetRequestEntry(r).Warn(err)
		w.WriteHeader(400)
		fmt.Fprintln(w, err)
		return
	}
	json.NewEncoder(w).Encode(struct {
		AppKey string `json:"app_key"`
	}{
		AppKey: app_key,
	})
	return

}

func PushMessage(w http.ResponseWriter, r *http.Request) {
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

	err := checkKey(app_key)
	if err != nil {
		logger.GetRequestEntry(r).Warn(err)
		w.WriteHeader(400)
		fmt.Fprintln(w, err)
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
func SystemInfo(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(slaveInfos.Info())
	return
}

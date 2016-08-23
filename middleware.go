package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
)

func HttpUse(h http.HandlerFunc, middleware ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for _, m := range middleware {
		h = m(h)
	}

	return h
}

func AuthMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		app_key := params["app_key"]
		if app_key != "" {
			log.Println("app_key nil")
			return
		}

		auth := r.FormValue("auth")
		if auth != "" {
			http.Error(w, "auth nil", 401)
			return
		}

		c := rpool.Get()

		//redis 格式 app_key url http://test.com

		reply, err := redis.Int(c.Do("HEXISTS", app_key, "url"))
		if err != nil || reply == 0 {
			http.Error(w, "auth process error", 401)
			return
		}

		url, err := redis.String(c.Do("HGET", app_key, "url"))
		if err != nil {
			http.Error(w, "auth process error", 401)
			return
		}
		req, err := http.NewRequest("POST", url, nil)
		if err != nil {
			http.Error(w, "auth process error", 401)
			return
		}
		ctx, _ := context.WithTimeout(r.Context(), 30*time.Second)
		err = worker.Execute(ctx, req, func(resp *http.Response, err error) (e error) {
			if err != nil {
				return
			}
			defer resp.Body.Close()

			a := &Auth{}
			//TODO read json and parse
			err = json.NewDecoder(resp.Body).Decode(a)
			if err != nil {
				return
			}
			ctx := r.Context()
			ctx = context.WithValue(ctx, "auth", a)
			r = r.WithContext(ctx)
			return
		})
		h.ServeHTTP(w, r)

		return
	}
}

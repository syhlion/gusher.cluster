package main

import (
	"context"
	"encoding/json"
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
		if app_key == "" {
			logger.RequestWarn(r, "app_key nil")
			http.Error(w, "auth nil", 401)
			return
		}

		auth := r.FormValue("auth")
		if auth == "" {
			logger.RequestWarn(r, "auth nil")
			http.Error(w, "auth nil", 401)
			return
		}

		c := rpool.Get()

		//redis 格式 app_key url http://test.com

		reply, err := redis.Int(c.Do("HEXISTS", app_key, "url"))
		if err != nil || reply == 0 {
			logger.RequestWarn(r, err)
			http.Error(w, "auth process error", 401)
			return
		}

		url, err := redis.String(c.Do("HGET", app_key, "url"))
		if err != nil {
			logger.RequestWarn(r, err)
			http.Error(w, "auth process error", 401)
			return
		}
		req, err := http.NewRequest("POST", url, nil)
		if err != nil {
			logger.RequestWarn(r, err)
			http.Error(w, "auth process error", 401)
			return
		}
		ctx, _ := context.WithTimeout(r.Context(), 30*time.Second)
		err = worker.Execute(ctx, req, func(resp *http.Response, err error) (e error) {
			if err != nil {
				logger.RequestWarn(r, err)
				return
			}
			defer resp.Body.Close()

			a := Auth{}
			err = json.NewDecoder(resp.Body).Decode(&a)
			if err != nil {
				logger.RequestWarn(r, err)
				return
			}
			ctx := r.Context()
			ctx = context.WithValue(ctx, "auth", a)
			ctx = context.WithValue(ctx, "app_key", app_key)
			r = r.WithContext(ctx)
			return
		})
		if err != nil {
			logger.RequestWarn(r, err)
			http.Error(w, "auth error", 401)
			return
		}
		h.ServeHTTP(w, r)

	}
}

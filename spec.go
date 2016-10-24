package main

import (
	jwt "github.com/dgrijalva/jwt-go"
)

const (
	LoginEvent                = "gusher.login"
	SubscribeEvent            = "gusher.subscribe"
	UnSubscribeEvent          = "gusher.unsubscribe"
	SubscribeReplySucceeded   = "subscribe_succeeded"
	SubscribeReplyError       = "subscribe_error"
	UnSubscribeReplySucceeded = "unsubscribe_succeeded"
	UnSubscribeReplyError     = "unsubscribe_error"
)

type InternalCommand struct {
	Event string `json:"event"`
}

type ChannelCommand struct {
	InternalCommand
	Data ChannelData `json:"data"`
}
type ChannelData struct {
	Channel string `json:"channel"`
}

type JwtPack struct {
	Gusher Auth `json:"gusher"`
	jwt.StandardClaims
}
type Auth struct {
	Channels []string `json:"channels"`
	UserId   string   `json:"user_id"`
	AppKey   string   `json:"app_key"`
}

/*rpc use*/
type ServerInfo struct {
	Ip             string `json:"ip"`
	LocalListen    string `json:"local_listen"`
	Version        string `json:"version"`
	RunTimeVersion string `json:"runtime_version"`
	NumCpu         int    `json:"cpu"`
	MemAllcoated   uint64 `json:"usage-memory"`
	Goroutines     int    `json:"goroutines"`
	Connections    int    `json:"connections"`
	SendInterval   string `json:"send_interval"`
	UpdateTime     int64  `json:"update_time"`
}

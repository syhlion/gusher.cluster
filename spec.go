package main

import (
	jwt "github.com/dgrijalva/jwt-go"
)

const (
	RemoteEvent               = "gusher.remote"
	LoginEvent                = "gusher.login"
	SubscribeEvent            = "gusher.subscribe"
	UnSubscribeEvent          = "gusher.unsubscribe"
	RemoteReplySucceeded      = "gusher.remote_succeeded"
	RemoteReplyError          = "gusher.remote_error"
	SubscribeReplySucceeded   = "gusher.subscribe_succeeded"
	SubscribeReplyError       = "gusher.subscribe_error"
	UnSubscribeReplySucceeded = "gusher.unsubscribe_succeeded"
	UnSubscribeReplyError     = "gusher.unsubscribe_error"
)

type BatchData struct {
	Channel string      `json:"channel"`
	Event   string      `json:"event"`
	Data    interface{} `json:"data"`
}

type InternalCommand struct {
	Event string `json:"event"`
}
type RemoteCommand struct {
	InternalCommand
	Data RemoteData `json:"data"`
}
type RemoteData struct {
	Remote string      `json:"remote"`
	Msg    interface{} `json:"msg"`
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
	Channels []string        `json:"channels"`
	UserId   string          `json:"user_id"`
	AppKey   string          `json:"app_key"`
	Remotes  map[string]bool `json:"remotes"`
}
type WorkerPayload struct {
	UserId   string      `json:"user_id"`
	SocketId string      `json:"socket_id"`
	Uid      string      `json:"uid"`
	AppKey   string      `json:"app_key"`
	Data     interface{} `json:"data"`
}

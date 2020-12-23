package main

import (
	jwt "github.com/dgrijalva/jwt-go"
	redisocket "github.com/syhlion/redisocket.v2"
)

const (
	PingEvent                    = "gusher.ping"
	QueryChannelEvent            = "gusher.querychannel"
	QueryChannelReplySucceeded   = "gusher.querychannel_succeeded"
	QueryChannelReplyError       = "gusher.querychannel_error"
	AddChannelEvent              = "gusher.addchannel"
	ReloadChannelEvent           = "gusher.reloadchannel"
	PongReplySucceeded           = "gusher.pong_succeeded"
	RemoteEvent                  = "gusher.remote"
	LoginEvent                   = "gusher.login"
	SubscribeEvent               = "gusher.subscribe"
	MultiSubscribeEvent          = "gusher.multi_subscribe"
	UnSubscribeEvent             = "gusher.unsubscribe"
	RemoteReplySucceeded         = "gusher.remote_succeeded"
	RemoteReplyError             = "gusher.remote_error"
	SubscribeReplySucceeded      = "gusher.subscribe_succeeded"
	SubscribeReplyError          = "gusher.subscribe_error"
	MultiSubscribeReplySucceeded = "gusher.multi_subscribe_succeeded"
	MultiSubscribeReplyError     = "gusher.multi_subscribe_error"
	UnSubscribeReplySucceeded    = "gusher.unsubscribe_succeeded"
	UnSubscribeReplyError        = "gusher.unsubscribe_error"
)

type BatchData struct {
	Channel string      `json:"channel"`
	Event   string      `json:"event"`
	Data    interface{} `json:"data"`
}

type InternalCommand struct {
	Event    string `json:"event"`
	SocketId string `json:"socket_id"`
}
type RemoteCommand struct {
	InternalCommand
	Data RemoteData `json:"data"`
}

type ChannelInfoData struct {
	InternalCommand
	Data interface{} `json:"data"`
}

type RemoteData struct {
	Remote string      `json:"remote"`
	Msg    interface{} `json:"msg"`
}
type PingCommand struct {
	InternalCommand
	Data interface{} `json:"data"`
}
type PongResponse struct {
	InternalCommand
	Data interface{} `json:"data"`
	Time int64       `json:"time"`
}
type QueryChannelResponse struct {
	InternalCommand
	Data interface{} `json:"data"`
}

type ChannelCommand struct {
	InternalCommand
	Data ChannelData `json:"data"`
}
type ChannelData struct {
	Channel interface{} `json:"channel"`
}

type JwtPack struct {
	Gusher redisocket.Auth `json:"gusher"`
	jwt.StandardClaims
}

/*
type Auth struct {
	Channels []string        `json:"channels"`
	UserId   string          `json:"user_id"`
	AppKey   string          `json:"app_key"`
	Remotes  map[string]bool `json:"remotes"`
}
*/
type WorkerPayload struct {
	UserId   string      `json:"user_id"`
	SocketId string      `json:"socket_id"`
	Uid      string      `json:"uid"`
	AppKey   string      `json:"app_key"`
	Data     interface{} `json:"data"`
}

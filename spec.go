package main

import (
	jwt "github.com/golang-jwt/jwt"
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
	LoginEvent                   = "gusher.login"
	SubscribeEvent               = "gusher.subscribe"
	MultiSubscribeEvent          = "gusher.multi_subscribe"
	UnSubscribeEvent             = "gusher.unsubscribe"
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
type ChannelInfoData struct {
	InternalCommand
	Data interface{} `json:"data"`
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

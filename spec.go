package main

import (
	jwt "github.com/golang-jwt/jwt/v5"
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

// ---- REST request bodies (JSON) ----

// MessageRequest is the body for publishing to a channel: {event, data}.
type MessageRequest struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// DataRequest is the body for a direct push to a user or socket: {data}.
type DataRequest struct {
	Data interface{} `json:"data"`
}

// PatternMessageRequest is the body for a pattern publish across an app.
type PatternMessageRequest struct {
	ChannelPattern string      `json:"channel_pattern"`
	Event          string      `json:"event"`
	Data           interface{} `json:"data"`
}

// AddChannelRequest adds a single channel to a user: {channel}.
type AddChannelRequest struct {
	Channel string `json:"channel"`
}

// ReloadChannelsRequest replaces a user's channel set: {channels}.
type ReloadChannelsRequest struct {
	Channels []string `json:"channels"`
}

// JwtRequest carries a JWT for /v1/auth and /v1/auth/decode: {jwt}.
type JwtRequest struct {
	Jwt string `json:"jwt"`
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
	jwt.RegisteredClaims
}

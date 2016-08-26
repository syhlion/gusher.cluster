package main

const (
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
	Id      string `json:"id"`
	Channel string `json:"channel"`
}

type CommonMessage struct {
	Channel string      `json:"channel"`
	Event   string      `json:"event"`
	Data    interface{} `json:"data"`
}

type Auth struct {
	Channels []string `json:"channels"`
	UserId   string   `json:"user_id"`
}

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

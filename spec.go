package main

const Subscribe = "Sub"
const UnSubscribe = "UnSub"

type Packet struct {
	Action  string   `json:action`
	Content []string `json:content`
}

type Auth struct {
	Channel []string `json:channel`
	UserId  string   `json:user_id`
}

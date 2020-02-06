package redisocket

type Auth struct {
	Channels []string        `json:"channels"`
	UserId   string          `json:"user_id"`
	AppKey   string          `json:"app_key"`
	Remotes  map[string]bool `json:"remotes"`
}

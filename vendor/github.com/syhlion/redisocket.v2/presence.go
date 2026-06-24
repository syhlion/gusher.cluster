package redisocket

import (
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
)

// PresenceMember 描述一個在線成員(批次更新用)。
type PresenceMember struct {
	AppKey   string
	Uid      string
	Channels []string
}

// Presence 抽象「在線/頻道成員」追蹤:誰在線、某頻道有誰、列頻道。
// 現役為 Redis sorted-set 實作;NATS 後端將以 per-node 記憶體 + request/reply 取代,
// 屆時連線層不需改動(只換注入的實作)。
type Presence interface {
	// Touch 更新單一成員的在線時戳(online + 指定 channels);client 訂閱時呼叫。
	Touch(prefix, appKey, uid string, channels []string) error
	// Sync 批次更新所有成員 + 清理過期成員(週期性呼叫)。
	Sync(prefix string, members []PresenceMember) error
	// OnlineByChannel 回傳某頻道近期在線的 uid。
	OnlineByChannel(prefix, appKey, channel string) ([]string, error)
	// Online 回傳某 app 近期在線的 uid。
	Online(prefix, appKey string) ([]string, error)
	// Channels 回傳某 app 符合 pattern 的頻道名。
	Channels(prefix, appKey, pattern string) ([]string, error)
	// Close 釋放資源(memoryPresence 退訂查詢主題;redisPresence 為 no-op)。
	Close() error
}

// redisPresence 以 Redis sorted set 實作 Presence(現役後端,行為與原 syncOnline/Get* 一致)。
// 註:沿用既有 KEYS / 120s 時窗 / 週期 ZREMRANGEBYSCORE 清理;不在此優化(NATS 版會整個取代)。
type redisPresence struct {
	pool *redis.Pool
}

func newRedisPresence(pool *redis.Pool) *redisPresence {
	return &redisPresence{pool: pool}
}

// Close 為 no-op(redis presence 不持有需釋放的訂閱)。
func (p *redisPresence) Close() error { return nil }

func (p *redisPresence) Touch(prefix, appKey, uid string, channels []string) error {
	if uid == "" && len(channels) == 0 {
		return nil
	}
	conn := p.pool.Get()
	defer conn.Close()
	nt := time.Now().Unix()
	conn.Send("MULTI")
	if uid != "" {
		conn.Send("ZADD", prefix+appKey+"@online", "CH", nt, uid)
	}
	for _, ch := range channels {
		conn.Send("ZADD", prefix+appKey+"@channels:"+ch, "CH", nt, uid)
	}
	_, err := conn.Do("EXEC")
	return err
}

func (p *redisPresence) Sync(prefix string, members []PresenceMember) error {
	conn := p.pool.Get()
	defer conn.Close()
	t := time.Now()
	nt := t.Unix()
	dt := t.Unix() - 86400
	conn.Send("MULTI")
	for _, m := range members {
		if m.Uid != "" {
			conn.Send("ZADD", prefix+m.AppKey+"@online", "CH", nt, m.Uid)
		}
		for _, e := range m.Channels {
			conn.Send("ZADD", prefix+m.AppKey+"@channels:"+e, "CH", nt, m.Uid)
			conn.Send("EXPIRE", prefix+m.AppKey+"@channels:"+e, 300)
		}
		conn.Send("EXPIRE", prefix+m.AppKey+"@online", 300)
	}
	conn.Do("EXEC")
	// 清理過期成員(沿用 KEYS,NATS 版不需要)
	tmp, err := redis.Strings(conn.Do("keys", prefix+"*"))
	if err != nil {
		return err
	}
	conn.Send("MULTI")
	for _, k := range tmp {
		conn.Send("ZREMRANGEBYSCORE", k, dt, nt-60)
	}
	conn.Do("EXEC")
	return nil
}

func (p *redisPresence) OnlineByChannel(prefix, appKey, channel string) ([]string, error) {
	conn := p.pool.Get()
	defer conn.Close()
	nt := time.Now().Unix()
	dt := nt - 120
	return redis.Strings(conn.Do("ZRANGEBYSCORE", prefix+appKey+"@channels:"+channel, dt, nt))
}

func (p *redisPresence) Online(prefix, appKey string) ([]string, error) {
	conn := p.pool.Get()
	defer conn.Close()
	nt := time.Now().Unix()
	dt := nt - 120
	return redis.Strings(conn.Do("ZRANGEBYSCORE", prefix+appKey+"@online", dt, nt))
}

func (p *redisPresence) Channels(prefix, appKey, pattern string) ([]string, error) {
	keyPrefix := prefix + appKey + "@channels:"
	conn := p.pool.Get()
	defer conn.Close()
	tmp, err := redis.Strings(conn.Do("keys", keyPrefix+pattern))
	if err != nil {
		return nil, err
	}
	channels := make([]string, 0)
	for _, v := range tmp {
		channel := strings.Replace(v, keyPrefix, "", -1)
		if channel == "" {
			continue
		}
		channels = append(channels, channel)
	}
	return channels, nil
}

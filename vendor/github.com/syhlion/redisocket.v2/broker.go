package redisocket

import (
	"strings"

	"github.com/gomodule/redigo/redis"
)

// BrokerEvent 是從跨節點匯流排收到的一則訊息。
// Event 為已去前綴、去星號的頻道名(可能是 #GUSHERFUNC-*# 控制事件或一般頻道)。
type BrokerEvent struct {
	AppKey string
	Event  string
	Data   []byte
}

// Broker 抽象「跨節點訊息匯流排」的 publish / subscribe。
// 連線層(ws hub)只依賴此介面,不綁定具體後端 —— 現役為 Redis pub/sub,
// 後續可替換為 NATS,連線/分派邏輯不需改動。
type Broker interface {
	// Publish 發佈一則訊息到 prefix+appKey 下的 event 主題,回傳收到的訂閱者數。
	Publish(prefix, appKey, event string, data []byte) (int, error)
	// Subscribe 在 prefix 下訂閱所有訊息,回傳 (event,data) 串流與錯誤串流。
	// 會啟動背景 goroutine;呼叫 Close 結束,屆時 msgs 會被關閉。
	Subscribe(prefix string) (msgs <-chan BrokerEvent, errs <-chan error)
	// Close 釋放底層連線/訂閱。
	Close() error
}

// redisBroker 以 Redis pub/sub 實作 Broker(現役後端)。
type redisBroker struct {
	pool *redis.Pool
	psc  *redis.PubSubConn
	done chan struct{}
}

func newRedisBroker(pool *redis.Pool) *redisBroker {
	return &redisBroker{pool: pool}
}

func (b *redisBroker) Publish(prefix, appKey, event string, data []byte) (int, error) {
	conn := b.pool.Get()
	defer conn.Close()
	return redis.Int(conn.Do("PUBLISH", prefix+appKey+"@"+event, data))
}

func (b *redisBroker) Subscribe(prefix string) (<-chan BrokerEvent, <-chan error) {
	msgs := make(chan BrokerEvent, 4096)
	errs := make(chan error, 1)
	b.done = make(chan struct{})
	// 用 pool.Dial() 取「專用 raw 連線」(非 pooled activeConn)。如此 Close 時關閉它
	// 中斷阻塞中的 Receive,是 race-free 的(pooled activeConn 的簿記才會與 Receive race)。
	conn, err := b.pool.Dial()
	if err != nil {
		errs <- err
		close(msgs)
		return msgs, errs
	}
	b.psc = &redis.PubSubConn{Conn: conn}
	b.psc.PSubscribe(prefix + "*")
	go func() {
		defer close(msgs)
		for {
			switch v := b.psc.Receive().(type) {
			case redis.Message:
				// 去前綴 → appKey@event
				channel := strings.Replace(v.Channel, prefix, "", -1)
				sch := strings.SplitN(channel, "@", 2)
				if len(sch) != 2 {
					continue
				}
				// 去星號(pattern 殘留)
				event := strings.Replace(sch[1], "*", "", -1)
				msgs <- BrokerEvent{AppKey: sch[0], Event: event, Data: v.Data}
			case error:
				// Close() 關閉連線會讓 Receive 回錯;若是正常關閉(done 已關)就乾淨退出,
				// 否則才回報為真錯誤(觸發上層重啟/收斂)。
				select {
				case <-b.done:
				default:
					errs <- v
				}
				return
			}
		}
	}()
	return msgs, errs
}

// Close 關閉 done 與底層連線:連線關閉會中斷阻塞中的 Receive,goroutine 見 done
// 後乾淨退出(不回報為錯誤)。raw 連線的 Close 與 Receive 並發是安全的。
func (b *redisBroker) Close() error {
	if b.done != nil {
		close(b.done)
	}
	if b.psc != nil {
		return b.psc.Close()
	}
	return nil
}

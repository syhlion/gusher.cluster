package redisocket

import (
	"strings"

	"github.com/nats-io/nats.go"
)

// eventHeader 是 NATS 訊息上攜帶事件名(頻道)的 header key。
// 用 header 帶 event、body 帶原始 bytes —— 免編碼,event/頻道名含任何字元
// (含 ".")都安全,避免把頻道名塞進 subject token 造成的限制。
const eventHeader = "e"

// natsBroker 以 NATS core pub/sub 實作 Broker。
//
// Subject 映射:bus 訊息收在 "<prefix>ch." 命名空間下,publish 到
// "<prefix>ch.<appKey>"、訂閱 "<prefix>ch.>"。如此 presence 的
// "<prefix>presence.*"、remote 的 "<prefix>rpc.*" 才不會被 bus 的萬用吃掉。
// (與 redisBroker 行為對齊:收全部、由 Hub 本地依 event 分派。)
const natsBusNS = "ch."

type natsBroker struct {
	nc   *nats.Conn
	sub  *nats.Subscription
	done chan struct{}
}

func NewNATSBroker(nc *nats.Conn) Broker {
	return &natsBroker{nc: nc}
}

func (b *natsBroker) Publish(prefix, appKey, event string, data []byte) (int, error) {
	msg := nats.NewMsg(prefix + natsBusNS + appKey)
	msg.Header.Set(eventHeader, event)
	msg.Data = data
	// NATS core 無「收到的訂閱者數」概念,回 0。
	return 0, b.nc.PublishMsg(msg)
}

func (b *natsBroker) Subscribe(prefix string) (<-chan BrokerEvent, <-chan error) {
	msgs := make(chan BrokerEvent, 4096)
	errs := make(chan error, 1)
	// 用 ChanSubscribe + 自有 goroutine,Close 時可關閉 msgs(讓 dispatchLoop 結束)。
	natsMsgs := make(chan *nats.Msg, 4096)
	busPrefix := prefix + natsBusNS
	sub, err := b.nc.ChanSubscribe(busPrefix+">", natsMsgs)
	if err != nil {
		errs <- err
		close(msgs)
		return msgs, errs
	}
	b.sub = sub
	b.done = make(chan struct{})
	go func() {
		defer close(msgs)
		for {
			select {
			case m := <-natsMsgs:
				msgs <- BrokerEvent{
					AppKey: strings.TrimPrefix(m.Subject, busPrefix),
					Event:  m.Header.Get(eventHeader),
					Data:   m.Data,
				}
			case <-b.done:
				return
			}
		}
	}()
	return msgs, errs
}

func (b *natsBroker) Close() error {
	if b.sub != nil {
		b.sub.Unsubscribe()
	}
	if b.done != nil {
		close(b.done)
	}
	return nil
}

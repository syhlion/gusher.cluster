package redisocket

// defaultMessageWorkers 是處理 inbound 訊息的 worker goroutine 數。
// TODO(Phase E/F):改為可設定 + 可停止(graceful shutdown)。
const defaultMessageWorkers = 1024

type messageQuene struct {
	serveChan      chan *buffer
	freeBufferChan chan *buffer
	pool           *pool
	quit           chan struct{}
}

func (m *messageQuene) worker() {
	for {
		select {
		case b := <-m.serveChan:
			m.serve(b)
		case <-m.quit:
			return
		}
	}
}
func (m *messageQuene) run(workers int) {
	if workers <= 0 {
		workers = defaultMessageWorkers
	}
	for i := 0; i < workers; i++ {
		go m.worker()
	}
}

func (m *messageQuene) serve(buffer *buffer) {
	receiveMsg, err := buffer.client.re(buffer.buffer.Bytes())
	if err == nil {
		byteCount := len(receiveMsg)
		if byteCount > 0 {
			m.pool.toSid(buffer.client.sid, receiveMsg)
		}
	} else {
		m.pool.kickSid(buffer.client.sid)
	}
	buffer.reset(nil)
	select {
	case m.freeBufferChan <- buffer:
	default:
	}
	return
}

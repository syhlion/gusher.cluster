package redisocket

import (
	"log"
)

type messageQuene struct {
	serveChan      chan *buffer
	freeBufferChan chan *buffer
	pool           *pool
}

func (m *messageQuene) worker() {
	for {
		select {
		case b := <-m.serveChan:
			m.serve(b)
		}
	}
	log.Println("[redisocket.v2] message quene crash")
}
func (m *messageQuene) run() {
	for i := 0; i < 1024; i++ {
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

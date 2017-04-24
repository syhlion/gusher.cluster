package redisocket

import (
	"log"
)

type messageQuene struct {
	serveChan      chan *buffer
	freeBufferChan chan *buffer
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
	for i := 0; i <= 5; i++ {
		go m.worker()
	}
}

func (m *messageQuene) serve(buffer *buffer) {
	receiveMsg, err := buffer.client.re(buffer.buffer.Bytes())
	if err == nil {
		if len(receiveMsg) > 0 {
			buffer.client.Send(receiveMsg)
		}
	} else {
		buffer.client.Close()
	}
	buffer.reset(nil)
	select {
	case m.freeBufferChan <- buffer:
	default:
	}
	return
}

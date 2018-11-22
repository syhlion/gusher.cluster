package redisocket

import (
	"log"
	"time"
)

type Statistic struct {
	inMem         int
	outMem        int
	inMsg         int
	inByte        int
	outMsg        int
	outByte       int
	inMemChannel  chan int
	outMemChannel chan int
	inMsgChannel  chan int
	outMsgChannel chan int
	lastFlushTime time.Time
	l             *log.Logger
}

func (s *Statistic) AddMem() {
	select {
	case s.inMemChannel <- 1:
	default:
	}
}
func (s *Statistic) SubMem() {
	select {
	case s.outMemChannel <- 1:
	default:
	}
}
func (s *Statistic) AddInMsg(byteLength int) {
	select {
	case s.inMsgChannel <- byteLength:
	default:
	}
}
func (s *Statistic) AddOutMsg(byteLength int) {
	select {
	case s.outMsgChannel <- byteLength:
	default:
	}
}

func (s *Statistic) Run() {

	// 10 second flush statistic
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-s.inMemChannel:
			s.inMem = s.inMem + 1
		case <-s.outMemChannel:
			s.outMem = s.outMem + 1
		case i := <-s.inMsgChannel:
			s.inByte = s.inByte + i
			s.inMsg = s.inMsg + 1
		case o := <-s.outMsgChannel:
			s.outByte = s.outByte + o
			s.outMsg = s.outMsg + 1
		case <-t.C:
			//clear statistic
			s.lastFlushTime = time.Now()
			s.Flush(s.lastFlushTime)
			s.inByte = 0
			s.inMsg = 0
			s.outMsg = 0
			s.outByte = 0
			s.outMem = 0
			s.inMem = 0
		}
	}
}
func (s *Statistic) Flush(t time.Time) {

	s.l.Printf(" [message] in-count: %d, in-byte: %d, out-count: %d, out-byte: %d, [member] in: %d, out: %d", s.inMsg, s.inByte, s.outMsg, s.outByte, s.inMem, s.outMem)
}

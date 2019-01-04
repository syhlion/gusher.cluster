package redisocket

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

//Client gorilla websocket wrap struct
type Client struct {
	prefix string
	sid    string
	uid    string
	ws     *websocket.Conn
	events map[string]EventHandler
	send   chan *Payload
	*sync.RWMutex
	re  ReceiveMsgHandler
	hub *Hub
}

func (c *Client) SocketId() string {
	return c.sid
}

//On event.  client on event
func (c *Client) On(event string, h EventHandler) {
	c.Lock()
	c.events[event] = h
	c.Unlock()
	conn := c.hub.rpool.Get()
	defer func() {
		conn.Close()
	}()
	nt := time.Now().Unix()
	conn.Send("MULTI")
	if c.uid != "" {
		conn.Send("ZADD", c.hub.ChannelPrefix+c.prefix+"@"+"online", "CH", nt, c.uid)
	}
	conn.Send("ZADD", c.hub.ChannelPrefix+c.prefix+"@"+"channels:"+event, "CH", nt, c.uid)
	conn.Do("EXEC")

	return
}

//Off event. client off event
func (c *Client) Off(event string) {
	c.Lock()
	delete(c.events, event)
	c.Unlock()
	return
}

//Trigger event. trigger client reigster event
func (c *Client) Trigger(event string, p *Payload) (err error) {
	c.RLock()
	_, ok := c.events[event]
	c.RUnlock()
	if !ok {
		return errors.New("No Event")
	}

	select {
	case c.send <- p:
	default:
		c.hub.logger("user %s disconnect  err: trigger buffer full", c.uid)
		c.Close()
	}
	return
}

//Send message. write msg to client
func (c *Client) Send(data []byte) {
	p := &Payload{
		Len:       len(data),
		Data:      data,
		IsPrepare: false,
	}
	select {
	case c.send <- p:
	default:
		c.hub.logger("user %s disconnect  err: send buffer full", c.uid)
		c.Close()
	}
	return
}

func (c *Client) write(msgType int, data []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(c.hub.Config.WriteWait))
	return c.ws.WriteMessage(msgType, data)
}
func (c *Client) writePreparedMessage(data *websocket.PreparedMessage) error {
	c.ws.SetWriteDeadline(time.Now().Add(c.hub.Config.WriteWait))
	return c.ws.WritePreparedMessage(data)
}

func (c *Client) readPump() {

	defer func() {
		c.hub.leave(c)
		c.Close()

	}()
	c.ws.SetReadLimit(c.hub.Config.MaxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(c.hub.Config.PongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(c.hub.Config.PongWait)); return nil })
	for {
		msgType, reader, err := c.ws.NextReader()
		if err != nil {
			c.hub.logger("user %s disconnect  err: websocket read out of max message size", c.uid)
			return
		}
		if msgType != websocket.TextMessage {
			c.hub.logger("user %s disconnect  err: send message type not text message", c.uid)
			continue
		}

		var buf *buffer
		select {
		case buf = <-c.hub.messageQuene.freeBufferChan:
			buf.reset(c)
		default:
			// None free, so allocate a new one.
			buf = &buffer{buffer: bytes.NewBuffer(make([]byte, 0, c.hub.Config.MaxMessageSize)), client: c}
		}
		_, err = io.Copy(buf.buffer, reader)
		if err != nil {
			buf.reset(nil)
			c.hub.logger("user %s disconnect  err: copy buffer error", c.uid)
			return
		}
		statistic.AddInMsg(buf.buffer.Len())
		select {
		case c.hub.messageQuene.serveChan <- buf:
		default:
			c.hub.logger("user %s disconnect  err: server receive busy", c.uid)
			return

		}

	}

}

//Close client. disconnect client
func (c *Client) Close() {
	c.ws.Close()
	return
}

//Listen client
//client start listen
//it's block method
func (c *Client) Listen(re ReceiveMsgHandler) {
	c.re = re
	go c.writePump()
	c.readPump()
}

func (c *Client) writePump() {
	t := time.NewTicker(c.hub.Config.PingPeriod)
	defer func() {
		t.Stop()
		c.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				c.hub.logger("user %s disconnect  err: channel receive error", c.uid)
				return
			}

			c.RLock()
			h, ok := c.events[msg.Event]
			c.RUnlock()
			if ok {
				if h != nil {
					err := h(msg.Event, msg)
					if err != nil {
						c.hub.logger("user %s disconnect  err: event callback execute error", c.uid)
						return
					}
				} else {
					c.hub.logger("user %s disconnect  err: no event callback", c.uid)
					return
				}
			}
			statistic.AddOutMsg(msg.Len)
			if msg.IsPrepare {

				if err := c.writePreparedMessage(msg.PrepareMessage); err != nil {
					c.hub.logger("user %s disconnect  err: write prepared message  %s", c.uid, err)
					return
				}
			} else {
				if err := c.write(websocket.TextMessage, msg.Data); err != nil {
					c.hub.logger("user %s disconnect  err: write normal message  %s", c.uid, err)
					return
				}

			}

		case <-t.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				c.hub.logger("user %s disconnect  err: ping message  %s", c.uid, err)
				return
			}
			//超過時間 都沒有事件訂閱 就斷線處理
			if len(c.events) == 0 {
				c.hub.logger("user %s disconnect  err: timeout to subscribe", c.uid)
				return
			}

		}
	}

}

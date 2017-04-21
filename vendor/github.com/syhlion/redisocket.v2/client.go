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

	c.send <- p
	return
}

//Send message. write msg to client
func (c *Client) Send(data []byte) {
	p := &Payload{
		Data:      data,
		IsPrepare: false,
	}
	c.send <- p
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
			return
		}
		if msgType != websocket.TextMessage {
			continue
		}
		var buf *buffer
		select {
		case buf = <-c.hub.pool.freeBufferChan:
			buf.reset(c)
		default:
			// None free, so allocate a new one.
			buf = &buffer{buffer: bytes.NewBuffer(make([]byte, 0, c.hub.Config.MaxMessageSize)), client: c}
		}
		_, err = io.Copy(buf.buffer, reader)
		if err != nil {
			buf.reset(nil)
			return
		}
		c.hub.pool.serveChan <- buf

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
				return
			}

			c.RLock()
			h, ok := c.events[msg.Event]
			if ok {
				if h != nil {
					err := h(msg.Event, msg)
					if err != nil {
						return
					}
				} else {
					return
				}
			}
			c.RUnlock()
			if msg.IsPrepare {

				if err := c.writePreparedMessage(msg.PrepareMessage); err != nil {
					return
				}
			} else {
				if err := c.write(websocket.TextMessage, msg.Data); err != nil {
					return
				}

			}

		case <-t.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				return
			}
			//超過時間 都沒有事件訂閱 就斷線處理
			if len(c.events) == 0 {
				return
			}

		}
	}

}

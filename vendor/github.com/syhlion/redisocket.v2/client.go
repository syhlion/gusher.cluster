package redisocket

import (
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	ws     *websocket.Conn
	events map[string]EventHandler
	send   chan *Payload
	*sync.RWMutex
	re  ReceiveMsgHandler
	hub *Hub
}

func (c *Client) On(event string, h EventHandler) {
	c.Lock()
	c.events[event] = h
	c.Unlock()

	return
}
func (c *Client) Off(event string) {
	c.Lock()
	delete(c.events, event)
	c.Unlock()
	return
}

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
		c.hub.Leave(c)
		c.Close()
	}()
	c.ws.SetReadLimit(c.hub.Config.MaxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(c.hub.Config.PongWait))
	c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(c.hub.Config.PongWait)); return nil })
	data := make([]byte, 256)
	for {
		msgType, reader, err := c.ws.NextReader()
		if err != nil {
			return
		}
		if msgType != websocket.TextMessage {
			continue
		}
		_, err = reader.Read(data)
		if err != nil {
			return
		}

		receiveMsg, err := c.re(data)
		if err != nil {
			return
		}
		if receiveMsg.Event != "" || receiveMsg.EventHandler == nil {
			if receiveMsg.Sub {
				c.On(receiveMsg.Event, receiveMsg.EventHandler)
			} else {
				c.Off(receiveMsg.Event)
			}
		}

		c.Send(receiveMsg.ResponseMsg)
	}
	return

}
func (c *Client) Close() {
	c.ws.Close()
	return
}

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
		close(c.send)
	}()
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}

			h, ok := c.events[msg.Event]
			if ok {
				err := h(msg.Event, msg)
				if err != nil {
					return
				}
			}
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

		}
	}
	return

}

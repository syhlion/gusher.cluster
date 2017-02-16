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
	send   chan *websocket.PreparedMessage
	*sync.RWMutex
	re  ReceiveMsgHandler
	hub *Hub
}

func (c *Client) On(event string, h EventHandler) error {
	c.Lock()
	c.events[event] = h
	c.Unlock()
	return c.hub.Register(event, c)
}
func (c *Client) Off(event string) error {
	c.Lock()
	delete(c.events, event)
	c.Unlock()
	return c.hub.Unregister(event, c)
}

func (c *Client) Trigger(event string, pMsg *websocket.PreparedMessage, origin []byte) (err error) {
	c.RLock()
	h, ok := c.events[event]
	c.RUnlock()
	if !ok {
		return errors.New("No Event")
	}

	b, err := h(event, pMsg, origin)

	if err != nil {
		return
	}
	c.send <- b
	return
}

/*
func (c *Client) Send(data []byte) {
	c.send <- data
	return
}
*/

func (c *Client) write(msgType int, data []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(c.hub.Config.WriteWait))
	return c.ws.WriteMessage(msgType, data)
}

func (c *Client) readPump() <-chan error {

	errChan := make(chan error)
	go func() {
		defer func() {
			c.Close()
		}()
		c.ws.SetReadLimit(c.hub.Config.MaxMessageSize)
		c.ws.SetReadDeadline(time.Now().Add(c.hub.Config.PongWait))
		c.ws.SetPongHandler(func(string) error { c.ws.SetReadDeadline(time.Now().Add(c.hub.Config.PongWait)); return nil })
		for {
			msgType, data, err := c.ws.ReadMessage()
			if err != nil {
				errChan <- err
				close(errChan)
				return
			}
			if msgType != websocket.TextMessage {
				continue
			}

			err = c.re(data)
			if err != nil {
				errChan <- err
				return
			}
		}
	}()
	return errChan

}
func (c *Client) Send(data []byte) (err error) {
	return c.write(websocket.TextMessage, data)
}
func (c *Client) Close() {
	c.hub.UnregisterAll(c)
	c.ws.Close()
	return
}

func (c *Client) Listen(re ReceiveMsgHandler) (err error) {
	defer c.Close()
	c.re = re
	writeErr := c.writePump()
	readErr := c.readPump()
	select {
	case e := <-writeErr:
		return e
	case e := <-readErr:
		return e
	}
}

func (c *Client) writePump() <-chan error {
	errChan := make(chan error)
	go func() {
		t := time.NewTicker(c.hub.Config.PingPeriod)
		defer func() {
			c.Close()
			t.Stop()
		}()
		for {
			select {
			case msg, ok := <-c.send:
				if !ok {

					errChan <- c.write(websocket.CloseMessage, []byte{})
					close(errChan)
					return
				}

				if err := c.ws.WritePreparedMessage(msg); err != nil {
					errChan <- err
					close(errChan)
					return
				}

			case <-t.C:
				if err := c.write(websocket.PingMessage, []byte{}); err != nil {
					errChan <- err
					close(errChan)
					return
				}

			}
		}
	}()
	return errChan

}

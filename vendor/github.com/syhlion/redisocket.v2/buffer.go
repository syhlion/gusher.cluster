package redisocket

import "bytes"

type buffer struct {
	buffer *bytes.Buffer
	client *Client
}

func (b *buffer) reset(c *Client) {
	b.buffer.Reset()
	b.client = c
}

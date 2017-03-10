package redisocket

import "bytes"

type Buffer struct {
	buffer *bytes.Buffer
	client *Client
}

func (b *Buffer) Reset(c *Client) {
	b.buffer.Reset()
	b.client = c
}

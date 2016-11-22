package redisocket

import (
	"testing"

	"github.com/garyburd/redigo/redis"
)

var (
	testChannel = []string{"TEST_A", "TEST_B", "TEST_C"}
	rpool       = redis.NewPool(func() (conn redis.Conn, err error) {
		return
	}, 10)
	hub    = NewHub(rpool, true)
	client = &Client{}
)

func init() {
	for _, v := range testChannel {
		hub.Register(v, client)
	}
}

func TestRegister(t *testing.T) {
	for _, v := range testChannel {
		if _, ok := hub.subjects[v]; !ok {
			t.Errorf("no subjects %s", v)
		}

		if _, ok := hub.subscribers[client]; !ok {
			t.Errorf("no subscribers %s", client)
		}
		channels := hub.subscribers[client]
		if _, ok := channels[v]; !ok {
			t.Errorf("subscriber no this event  %s", v)
		}
	}
}

func TestUnregister(t *testing.T) {
	for _, v := range testChannel {
		hub.Unregister(v, client)
		if _, ok := hub.subjects[v]; ok {
			t.Errorf("nodelete subjects %s", v)
		}

		channels := hub.subscribers[client]
		if _, ok := channels[v]; ok {
			t.Errorf("subscriber no delete this event  %s", v)
		}
	}
	if _, ok := hub.subscribers[client]; ok {
		t.Errorf("no delete subscribers %s", client)
	}
}

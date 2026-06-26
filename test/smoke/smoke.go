// Command smoke is a correctness check against a running gusher.cluster stack
// (see docker-compose). It drives the full path — auth -> ws connect ->
// subscribe -> master push -> receive — across N connections and exits
// non-zero if any connection fails to receive the pushed message. Unlike
// test/conn-test (a load/latency tool that only logs), this returns a proper
// exit code so it can gate `make smoke` / CI.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	var (
		authURL = env("SMOKE_AUTH_URL", "http://127.0.0.1:8888/v1/auth")
		wsURL   = env("SMOKE_WS_URL", "ws://127.0.0.1:8888/v1/apps/TEST/ws")
		pushURL = env("SMOKE_PUSH_URL", "http://127.0.0.1:7777/v1/apps/TEST/channels/AA/messages")
		jwt     = os.Getenv("SMOKE_JWT")
		subMsg  = env("SMOKE_SUBSCRIBE", `{"event":"gusher.subscribe","data":{"channel":"AA"}}`)
		event   = env("SMOKE_EVENT", "EVENT")
		payload = env("SMOKE_PAYLOAD", "hello-smoke")
	)
	conns, _ := strconv.Atoi(env("SMOKE_CONNECTIONS", "50"))
	if conns < 1 {
		conns = 1
	}
	if jwt == "" {
		fatal("SMOKE_JWT is empty")
	}

	// 1) /v1/auth — local JWT verify
	resp, err := http.Post(authURL, "application/json", strings.NewReader(`{"jwt":"`+jwt+`"}`))
	if err != nil {
		fatal("auth request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fatal("auth status = %d, want 200", resp.StatusCode)
	}
	log.Printf("auth ok; opening %d connections", conns)

	// 2) open N ws connections and subscribe
	dialURL := wsURL + "?token=" + jwt
	clients := make([]*websocket.Conn, 0, conns)
	for i := 0; i < conns; i++ {
		c, _, err := websocket.DefaultDialer.Dial(dialURL, nil)
		if err != nil {
			fatal("ws dial #%d: %v", i, err)
		}
		c.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := c.WriteMessage(websocket.TextMessage, []byte(subMsg)); err != nil {
			fatal("subscribe write #%d: %v", i, err)
		}
		c.SetReadDeadline(time.Now().Add(10 * time.Second))
		if _, reply, err := c.ReadMessage(); err != nil {
			fatal("subscribe reply #%d: %v", i, err)
		} else if !strings.Contains(string(reply), "subscribe_succeeded") {
			fatal("subscribe #%d not succeeded: %s", i, reply)
		}
		clients = append(clients, c)
	}
	log.Printf("all %d subscribed; pushing", conns)

	// 3) every client waits for the pushed message
	var received int64
	var wg sync.WaitGroup
	for i, c := range clients {
		wg.Add(1)
		go func(i int, c *websocket.Conn) {
			defer wg.Done()
			defer c.Close()
			c.SetReadDeadline(time.Now().Add(15 * time.Second))
			_, msg, err := c.ReadMessage()
			if err != nil {
				log.Printf("client #%d read: %v", i, err)
				return
			}
			if strings.Contains(string(msg), payload) {
				atomic.AddInt64(&received, 1)
			} else {
				log.Printf("client #%d unexpected message: %s", i, msg)
			}
		}(i, c)
	}

	// 4) master push (once) — fan-out to all subscribers
	pResp, err := http.Post(pushURL, "application/json",
		strings.NewReader(`{"event":"`+event+`","data":"`+payload+`"}`))
	if err != nil {
		fatal("push request: %v", err)
	}
	pResp.Body.Close()
	if pResp.StatusCode != http.StatusOK {
		fatal("push status = %d, want 200", pResp.StatusCode)
	}

	wg.Wait()
	got := atomic.LoadInt64(&received)
	if int(got) != conns {
		fatal("only %d/%d clients received the message", got, conns)
	}
	log.Printf("SMOKE PASS: %d/%d clients received the pushed message", got, conns)
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "SMOKE FAIL: "+format+"\n", a...)
	os.Exit(1)
}

// Command loadtest opens N WebSocket connections to a gusher slave, subscribes
// each to a channel, then triggers one master push and measures the per-client
// fan-out latency (push → received) as p50/p99/max.
//
// Single-box numbers are bounded by your fd/CPU/ports; to reach ~100k spread
// the clients across several loadtest hosts and several gusher slave nodes.
// See docs/LOAD-TEST.md.
//
//	go run ./test/loadtest \
//	  -n 5000 -channel AA \
//	  -ws ws://127.0.0.1:8888/ws/TEST \
//	  -auth http://127.0.0.1:8888/auth \
//	  -push http://127.0.0.1:7777/push/TEST/AA/notify \
//	  -jwt "$JWT"
package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	n := flag.Int("n", 1000, "number of concurrent connections")
	wsAPI := flag.String("ws", "", "ws endpoint, e.g. ws://host:8888/ws/TEST")
	authAPI := flag.String("auth", "", "auth endpoint, e.g. http://host:8888/auth")
	pushAPI := flag.String("push", "", "master push endpoint, e.g. http://host:7777/push/TEST/AA/notify")
	jwt := flag.String("jwt", "", "JWT (gusher claim with the channel authorized)")
	channel := flag.String("channel", "AA", "channel to subscribe")
	dial := flag.Int("dial-concurrency", 500, "max concurrent dials")
	flag.Parse()
	if *wsAPI == "" || *authAPI == "" || *pushAPI == "" || *jwt == "" {
		flag.Usage()
		os.Exit(2)
	}

	// 1. auth once → token (the verified JWT)
	token := authToken(*authAPI, *jwt)

	// 2. open N connections, subscribe each
	var (
		connected int64
		recvTimes = make([]time.Duration, *n)
		received  = make([]int32, *n) // 0/1 per client
		pushAt    atomic.Int64        // unix-nano of the push
		subWG     sync.WaitGroup
		doneWG    sync.WaitGroup
		sem       = make(chan struct{}, *dial)
		conns     = make([]*websocket.Conn, *n)
	)
	subResp := "subscribe_succeeded"
	subMsg := fmt.Sprintf(`{"event":"gusher.subscribe","data":{"channel":%q}}`, *channel)

	fmt.Printf("dialing %d connections ...\n", *n)
	for i := 0; i < *n; i++ {
		subWG.Add(1)
		doneWG.Add(1)
		go func(i int) {
			defer doneWG.Done()
			sem <- struct{}{}
			c, _, err := websocket.DefaultDialer.Dial(*wsAPI+"?token="+url.QueryEscape(token), nil)
			<-sem
			if err != nil {
				subWG.Done()
				return
			}
			conns[i] = c
			atomic.AddInt64(&connected, 1)
			_ = c.WriteMessage(websocket.TextMessage, []byte(subMsg))
			subbed := false
			for {
				_, data, err := c.ReadMessage()
				if err != nil {
					if !subbed {
						subWG.Done()
					}
					return
				}
				if !subbed && strings.Contains(string(data), subResp) {
					subbed = true
					subWG.Done()
					continue
				}
				if subbed {
					recvTimes[i] = time.Duration(time.Now().UnixNano() - pushAt.Load())
					atomic.StoreInt32(&received[i], 1)
					return
				}
			}
		}(i)
	}
	subWG.Wait()
	fmt.Printf("connected=%d subscribed; pushing ...\n", atomic.LoadInt64(&connected))

	// 3. one push, measure fan-out
	pushAt.Store(time.Now().UnixNano())
	if _, err := http.PostForm(*pushAPI, url.Values{"data": {"loadtest"}}); err != nil {
		fmt.Println("push error:", err)
		os.Exit(1)
	}
	waitOrTimeout(&doneWG, 30*time.Second)
	for _, c := range conns {
		if c != nil {
			c.Close()
		}
	}

	// 4. report
	var lat []time.Duration
	var got int
	for i := 0; i < *n; i++ {
		if atomic.LoadInt32(&received[i]) == 1 {
			lat = append(lat, recvTimes[i])
			got++
		}
	}
	sort.Slice(lat, func(a, b int) bool { return lat[a] < lat[b] })
	fmt.Println("──────── result ────────")
	fmt.Printf("connections attempted : %d\n", *n)
	fmt.Printf("connections subscribed: %d\n", connected)
	fmt.Printf("messages received     : %d (%.1f%%)\n", got, pct(got, int(connected)))
	if len(lat) > 0 {
		fmt.Printf("fan-out latency p50   : %v\n", lat[len(lat)*50/100])
		fmt.Printf("fan-out latency p99   : %v\n", lat[min(len(lat)-1, len(lat)*99/100)])
		fmt.Printf("fan-out latency max   : %v\n", lat[len(lat)-1])
	}
}

func authToken(authAPI, jwt string) string {
	resp, err := http.PostForm(authAPI, url.Values{"jwt": {jwt}})
	if err != nil {
		fmt.Println("auth error:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	var buf [4096]byte
	nn, _ := resp.Body.Read(buf[:])
	body := string(buf[:nn])
	i := strings.Index(body, `"token":"`)
	if i < 0 {
		fmt.Println("auth: no token in response:", body)
		os.Exit(1)
	}
	body = body[i+len(`"token":"`):]
	return body[:strings.IndexByte(body, '"')]
}

func waitOrTimeout(wg *sync.WaitGroup, d time.Duration) {
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(d):
	}
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) * 100 / float64(b)
}

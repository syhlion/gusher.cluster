# redisocket.v2

[![Go Report Card](https://goreportcard.com/badge/github.com/syhlion/redisocket.v2)](https://goreportcard.com/report/github.com/syhlion/redisocket.v2)

A WebSocket **hub engine**: it holds ws connections and fans out messages across
nodes, with the cross-node **bus** and **presence** behind pluggable interfaces.
Pick a **Redis** or **NATS** backend — swapping it never touches the connection
layer. (This is the engine behind [gusher.cluster](https://github.com/syhlion/gusher.cluster).)

📐 **Architecture**: [English](docs/ARCHITECTURE.md) · [繁體中文](docs/ARCHITECTURE.zh-TW.md)

## Install

```sh
go get github.com/syhlion/redisocket.v2
```

## Usage

The engine takes a `*slog.Logger` (output is the caller's choice). Pick a backend
when you build the hub:

```go
// NATS backend (bus + per-node presence; no Redis)
nc, _ := nats.Connect("nats://127.0.0.1:4222")
broker := redisocket.NewNATSBroker(nc)
presence, _ := redisocket.NewMemoryPresence(nc, "app.")
hub := redisocket.NewHubWithBrokerAndPresence(broker, presence, slog.Default(), false)

// — or — Redis backend
// pool := &redis.Pool{ Dial: func() (redis.Conn, error) { return redis.Dial("tcp", ":6379") } }
// hub := redisocket.NewHub(pool, slog.Default(), false)

go hub.Listen("app.") // channelPrefix; blocks
defer hub.Close()     // graceful shutdown (goleak-clean)

http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
	auth := &redisocket.Auth{AppKey: "appKey", UserId: "Scott", Channels: []string{"*"}}
	c, err := hub.Upgrade(w, r, nil, auth.UserId, auth.AppKey, auth)
	if err != nil {
		return
	}
	c.Listen(func(data []byte) (resp []byte, err error) {
		// handle an inbound client message; return bytes to echo back
		return nil, nil
	})
})
```

Publish into a channel from anywhere (e.g. a separate publish API node):

```go
sender := redisocket.NewSenderWithBrokerAndPresence(broker, presence) // NATS
// sender := redisocket.NewSender(pool)                               // Redis
sender.Push("app.", "appKey", "AA", []byte(`{"hello":"world"}`))      // → subscribers of channel AA
```

## Configurable logger (optional)

```go
lg, closeLog, _ := redisocket.NewLogger(redisocket.LogConfig{
	Output: redisocket.LogBoth, File: "/var/log/app.log", // stdout / file / both
	Format: "json", MaxSizeMB: 100, MaxBackups: 7, Compress: true, // logrotate
})
defer closeLog()
```

## Docs

- [Architecture](docs/ARCHITECTURE.md) — engine internals, Broker / Presence, graceful shutdown, testing
- [API Reference](https://pkg.go.dev/github.com/syhlion/redisocket.v2)

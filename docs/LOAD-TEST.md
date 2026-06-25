# Load testing

> 🌐 **English** · [繁體中文](LOAD-TEST.zh-TW.md)

`test/loadtest` opens **N** WebSocket connections to a slave, subscribes each to
a channel, triggers **one** master push, and reports the per-client **fan-out
latency** (push → received) as p50 / p99 / max plus the delivery rate.

```sh
go run ./test/loadtest -n 5000 -channel AA \
  -ws   ws://127.0.0.1:8888/ws/TEST \
  -auth http://127.0.0.1:8888/auth \
  -push http://127.0.0.1:7777/push/TEST/AA/notify \
  -jwt  "$JWT"
```

(Generate `$JWT` with [`test/jwtgenerate`](../test/jwtgenerate); the channel must
be in the JWT's `channels`.)

## Baseline (single box)

One slave + one master + the load tool on the **same** dev machine, local NATS,
`ulimit -n 100000`:

| Connections | Delivered | fan-out p50 | fan-out p99 |
|---|---|---|---|
| 5,000 | 100% | 7.3 ms | 13 ms |
| 10,000 | 100% | 17 ms | 31 ms |

Latency scales ~linearly with fan-out size; a single push reaches 10k clients in
~30 ms p99 on one box. These are **lower bounds** — a real deployment spreads the
work (below), so per-node fan-out stays small.

## Reaching ~100k connections

100k on one box is fd/CPU/port-bound; the design scales **horizontally** instead:

- **Multiple slave nodes** — connections shard across slaves; each node only
  fans out its own subscribers, and only receives the NATS subjects it
  subscribed to. A push reaches all slaves via NATS at once.
- **Multiple load hosts** — run `loadtest` from several machines (a single box
  caps out on fds / outbound ports well before 100k).
- **Tuning** — raise `ulimit -n` (e.g. 1M) and `net.ipv4.ip_local_port_range`,
  `net.core.somaxconn`, `net.ipv4.tcp_tw_reuse` on both sides.

## What to watch

- **Delivery rate** must stay ~100%. A slow client whose send buffer fills is
  dropped by design (back-pressure) — watch for that under burst.
- **Fan-out p99** per node — keep the per-node subscriber count in a range that
  holds your latency target; add slave nodes to lower it.
- **NATS** is not stressed by connection count (it only carries inter-node
  messages); watch its message rate, not the 100k.

## See also

- [ARCHITECTURE](ARCHITECTURE.md) — why connection count doesn't hit NATS
- `test/loadtest/loadtest.go` — the tool

# 壓力測試

> 🌐 [English](LOAD-TEST.md) · **繁體中文**

`test/loadtest` 會對 slave 開 **N** 條 WebSocket 連線、各自訂閱一個頻道,觸發 **一次**
master 推播,然後回報每個 client 的 **fan-out 延遲**(推播→收到)p50 / p99 / max
與送達率。

```sh
go run ./test/loadtest -n 5000 -channel AA \
  -ws   ws://127.0.0.1:8888/ws/TEST \
  -auth http://127.0.0.1:8888/auth \
  -push http://127.0.0.1:7777/push/TEST/AA/notify \
  -jwt  "$JWT"
```

(用 [`test/jwtgenerate`](../test/jwtgenerate) 產 `$JWT`;頻道要在 JWT 的 `channels` 內。)

## 基線(單機)

一 slave + 一 master + 壓測工具跑在**同一台** dev 機、本機 NATS、`ulimit -n 100000`:

| 連線數 | 送達率 | fan-out p50 | fan-out p99 |
|---|---|---|---|
| 5,000 | 100% | 7.3 ms | 13 ms |
| 10,000 | 100% | 17 ms | 31 ms |

延遲隨 fan-out 大小約略線性成長;單機一次推播 ~30 ms p99 觸及 1 萬 client。這是
**下界**——實際部署會把工作分散(見下),每個節點的 fan-out 維持很小。

## 衝到 ~10 萬連線

10 萬塞在一台會被 fd / CPU / port 卡死;這個架構靠**水平擴展**:

- **多個 slave 節點** — 連線分散到各 slave;每個節點只 fan-out 自己的訂閱者、也只
  收自己訂閱的 NATS subject。一次推播經 NATS 同時到所有 slave。
- **多台壓測機** — 從多台跑 `loadtest`(單機在到 10 萬之前就會卡 fd / 對外 port)。
- **調校** — 兩端都拉高 `ulimit -n`(如 1M)與 `net.ipv4.ip_local_port_range`、
  `net.core.somaxconn`、`net.ipv4.tcp_tw_reuse`。

## 該盯什麼

- **送達率**要維持 ~100%。送出 buffer 滿的慢 client 會被刻意丟棄(back-pressure)
  ——尖峰時注意這個。
- **每節點 fan-out p99** — 把每節點訂閱者數控在能達到延遲目標的範圍;要降就加 slave。
- **NATS** 不會被連線數壓到(它只走節點間訊息);看它的訊息速率,不是那 10 萬。

## 延伸

- [架構](ARCHITECTURE.zh-TW.md) — 為什麼連線數不會打到 NATS
- `test/loadtest/loadtest.go` — 工具

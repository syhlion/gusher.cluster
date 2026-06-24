# gusher.cluster 架構（NATS）

> 🌐 [English](ARCHITECTURE.md) · **繁體中文**

gusher.cluster 是自架的即時推播服務（仿 Pusher）。瀏覽器以 **WebSocket** 連到
**slave** 節點並訂閱頻道;後端 **POST** 到 **master** 節點把訊息推進頻道;訊息在節點
之間經 **NATS** 傳遞。**完全沒有 Redis**——bus、presence、auth 全在 NATS / 本機。

> 圖的原始檔是 `diagrams/` 裡的 `.drawio`,用 draw.io 開啟可編輯。

## 系統總覽

![system](diagrams/system.drawio.png)

- **slave** — 持有 ws 連線、本機驗 JWT（RSA 公鑰）、訂閱其 client 頻道對應的 NATS
  subject、把進來的訊息 fan-out。內嵌
  [redisocket.v2](https://github.com/syhlion/redisocket.v2) 引擎。
- **master** — 無狀態的 publish / REST API（`/push/...`、`/{app}/online`、
  `/{app}/channels`）。它不持有連線;在線/頻道查詢靠 NATS request/reply 匯總各 slave。
- **NATS** — 單一後端:
  - bus:subject `gusher.ch.<appKey>`（節點只收自己訂閱的頻道）;
  - presence:`gusher.presence.query`（request/reply scatter-gather,無 store）。

兩種角色都是無狀態、可水平擴展。

## 授權（本機 JWT,無 Redis）

JWT 帶 `gusher` claim——`{app_key, user_id, channels}`,RS256 簽。流程（對 client
不變,但無狀態）:

1. `POST /auth {jwt}` → slave 用 RSA **公鑰**本機驗（`helper.Decode`),把 JWT 本身
   當 `token` 回傳。
2. `GET /ws/{app_key}?token=<JWT>` → slave 再把 token-as-JWT 本機驗一次、upgrade。

沒有 decode service、沒有 token store——舊的 `RPUSH`/`SET` redis 路徑全數移除。

## 即時訊息流

- **訂閱** — client 送 `{"event":"gusher.subscribe","data":{"channel":"AA"}}`;
  slave 依 JWT 的 `channels`（萬用字元 / regex）比對授權後註冊訂閱。
- **推播** — `POST /push/{app_key}/{channel}/{event}` → master publish 到
  `gusher.ch.<appKey>` → 每個訂閱該頻道的 slave 把訊息寫給對應的 ws client。
- **presence** — master 的 `GET /{app_key}/online` 等,經 NATS scatter-gather 各
  slave 後合併。

## 遷移（Redis → NATS）

![evolution](diagrams/evolution.drawio.png)

過去 bus（pub/sub）、presence（sorted set）、auth-token store 全在 Redis。現在分別
變成 NATS subject 路由、per-node 記憶體 + request/reply、本機 JWT 驗證。讓這變成「換
後端」而非「重寫」的引擎層抽象,見
[redisocket.v2 架構](https://github.com/syhlion/redisocket.v2/blob/master/docs/ARCHITECTURE.zh-TW.md)。

## 日誌

輸出 **stdout / file / both** + 輪替,由環境變數驅動（`GUSHER_LOG_OUTPUT`、
`GUSHER_LOG_FILE`、`GUSHER_LOG_FORMAT`、`GUSHER_LOG_*`）。app（logrus）與引擎
（slog）寫到同一目的地——見 `logsetup.go`。

## 執行

`docker compose -f docker-compose/docker-compose.yml up --build` 會起
`nats` + `gusher-master` + `gusher-slave`（無 Redis）。在 compose 檔旁放一個
`public.pem`。必要環境變數:`GUSHER_NATS_ADDR`、`GUSHER_PUBLIC_PEM_FILE`,以及
API listen / prefix。

## 延伸

- [redisocket.v2 架構](https://github.com/syhlion/redisocket.v2/blob/master/docs/ARCHITECTURE.zh-TW.md)
  — ws hub 引擎（Broker / Presence 抽象）
- `doc/protocal.md` — client WebSocket 協定（events）
- `docker-compose/` — NATS 部署

# Gusher.Cluster

[![Build Status](https://drone.syhlion.tw/api/badges/syhlion/gusher.cluster/status.svg)](https://drone.syhlion.tw/syhlion/gusher.cluster)
 [![Stars](https://img.shields.io/github/stars/syhlion/gusher.cluster.svg)](https://github.com/syhlion/gusher.cluster)
 [![Go](https://img.shields.io/github/go-mod/go-version/syhlion/gusher.cluster.svg)](go.mod)
 [![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
 [![Backed by NATS](https://img.shields.io/badge/backed%20by-NATS-27AAE1.svg)](https://nats.io)
 [![docs English](https://img.shields.io/badge/docs-English-lightgrey.svg)](README.md)
 [![docs 繁體中文](https://img.shields.io/badge/docs-%E7%B9%81%E9%AB%94%E4%B8%AD%E6%96%87-blue.svg)](README.zh-TW.md)

自架的即時推播服務（仿 Pusher 風格）。瀏覽器以 **WebSocket** 連線並訂閱頻道；
後端則 **POST** 把訊息推進頻道。可水平擴展，**以 NATS 為後端——不需要 Redis**。

📐 **架構文件**：[English](docs/ARCHITECTURE.md) · [繁體中文](docs/ARCHITECTURE.zh-TW.md)

## 運作原理

- **slave**——持有 WebSocket 連線，在本地驗證 JWT，並把訊息扇出給訂閱者。
- **master**——無狀態的 REST API，用來**推播**訊息與**查詢**在線狀態（presence）。
- **NATS**——在節點之間傳遞訊息（匯流排），並以 request/reply 回答 presence 查詢。
  沒有 Redis、沒有 token store、沒有 decode 服務。

```
client ──ws──▶ slave ──subscribe──▶  NATS  ◀──publish── master ◀──POST── backend
```

## 環境需求

- **NATS**（唯一的後端）——`nats-server` 2.10 以上。
- 一組 **RSA 金鑰對**——master/slave 以**公鑰**驗證 JWT；由你自己的認證服務以私鑰簽發 JWT。
  完整的「產生 → 簽 → 驗證 → 輪替」教學見 [docs/KEYS.zh-TW.md](docs/KEYS.zh-TW.md)。

## 執行方式

### docker-compose（最快）

把你的 `public.pem` 放到 compose 檔旁邊，然後：

```sh
docker compose -f docker-compose/docker-compose.yml up --build
```

會起 `nats` + `gusher-master`（`:7777`）+ `gusher-slave`（`:8888`），不含 Redis。

### 從原始碼編譯

```sh
go build -ldflags "-X main.name=gusher" -o gusher.cluster .

# slave
GUSHER_NATS_ADDR=nats://127.0.0.1:4222 GUSHER_PUBLIC_PEM_FILE=./public.pem \
GUSHER_API_LISTEN=:8888 GUSHER_API_URI_PREFIX=/ ./gusher.cluster slave

# master
GUSHER_NATS_ADDR=nats://127.0.0.1:4222 GUSHER_PUBLIC_PEM_FILE=./public.pem \
GUSHER_MASTER_API_LISTEN=:7777 GUSHER_MASTER_URI_PREFIX=/ ./gusher.cluster master
```

完整環境變數清單見 `slave.env.example` / `master.env.example`。

## 維運

- **健康檢查**：`GET /ping`（liveness）· `GET /ready`（readiness——只有在 NATS
  連線正常時才回 200）。
- **NATS 認證**：設定 `GUSHER_NATS_CREDS=/path/to/app.creds` 使用 user credentials；
  TLS 則用 `tls://` 位址（或在 NATS server 設定）。client 會自動重連。

## 客端流程

1. `POST /auth` 帶 `jwt=<JWT>` → `{"token":"<JWT>"}`（JWT 在本地驗證後直接當 token
   回傳——無狀態、不需 store）。
2. `GET /ws/{app_key}?token=<token>` → WebSocket。以
   `{"event":"gusher.subscribe","data":{"channel":"AA"}}` 訂閱。
3. 後端推播：`POST /push/{app_key}/{channel}/{event}` 帶 `data=...`。

JWT 帶有 `gusher` claim——`{"app_key","user_id","channels"}`——以 **RS256** 簽章。
完整 WebSocket 協定見 [doc/protocal.md](./doc/protocal.md)，REST API 見
[doc/api.md](./doc/api.md)。

## 日誌

輸出方式可選，並支援輪替（rotation），透過環境變數設定：

| 環境變數 | 可選值 |
|---|---|
| `GUSHER_LOG_OUTPUT` | `stdout`（預設）/ `file` / `both` |
| `GUSHER_LOG_FILE` | 路徑（用於 `file` / `both`） |
| `GUSHER_LOG_FORMAT` | `json`（預設）/ `text` |
| `GUSHER_LOG_MAX_SIZE_MB` / `_MAX_BACKUPS` / `_MAX_AGE_DAYS` / `_COMPRESS` | 輪替設定 |

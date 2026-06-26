# Gusher.Cluster

[![Stars](https://img.shields.io/github/stars/syhlion/gusher.cluster.svg)](https://github.com/syhlion/gusher.cluster)
[![Build Status](https://drone.syhlion.tw/api/badges/syhlion/gusher.cluster/status.svg)](https://drone.syhlion.tw/syhlion/gusher.cluster)
[![Go](https://img.shields.io/github/go-mod/go-version/syhlion/gusher.cluster.svg)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Backed by NATS](https://img.shields.io/badge/backed%20by-NATS-27AAE1.svg)](https://nats.io)
[![Docker](https://img.shields.io/docker/v/syhlion/gusher.cluster?sort=semver&logo=docker&logoColor=white&label=docker)](https://hub.docker.com/r/syhlion/gusher.cluster)
[![docs English](https://img.shields.io/badge/docs-English-lightgrey.svg)](README.md)
[![docs 繁體中文](https://img.shields.io/badge/docs-%E7%B9%81%E9%AB%94%E4%B8%AD%E6%96%87-blue.svg)](README.zh-TW.md)

自架的即時推播服務（仿 Pusher 風格）。瀏覽器以 **WebSocket** 連線並訂閱頻道；
後端則 **POST** 把訊息推進頻道。可水平擴展，**以 NATS 為後端——不需要 Redis**。

## 架構

![system](docs/diagrams/system.drawio.png)

- **slave**——持有 WebSocket 連線，在本地以 RSA 公鑰驗證 JWT，並把訊息扇出給訂閱者。
- **master**——無狀態的 REST API，用來**推播**訊息與**查詢**在線狀態（presence）。
- **NATS**——在節點之間傳遞訊息（匯流排），並以 request/reply 回答 presence 查詢。
  沒有 Redis、沒有 token store、沒有 decode 服務。

```
client ──ws──▶ slave ──subscribe──▶  NATS  ◀──publish── master ◀──POST── backend
```

兩種角色都是無狀態且可水平擴展。完整說明（匯流排、presence、從 Redis 的演進）見
[docs/ARCHITECTURE.zh-TW.md](docs/ARCHITECTURE.zh-TW.md)。

## 環境需求

- **NATS**（唯一的後端）——`nats-server` 2.10 以上。
- 一組 **RSA 金鑰對**——master/slave 以**公鑰**驗證 JWT；由你自己的認證服務以私鑰簽發 JWT。
  完整的「產生 → 簽 → 驗證 → 輪替」教學見 [docs/KEYS.zh-TW.md](docs/KEYS.zh-TW.md)。

## 快速開始（Docker Compose）

stack 已內附一把**示範 RSA 金鑰**（`docker-compose/public.pem`，對應
`test/key/private.pem`），開箱即跑、免設定：

```sh
docker compose -f docker-compose/docker-compose.yml up --build
```

會起 `nats` + `gusher-master`（`:7777`）+ `gusher-slave`（`:8888`），不含 Redis。

**驗證效果** — 一行指令把 stack 拉起、簽一張 JWT、用 WebSocket 訂閱、推一則訊息
並驗證有收到，最後自動拆掉：

```sh
make smoke
```

或對著跑起來的 stack 手動測——用示範金鑰簽 token，再 auth → 連線 → 推播：

```sh
# 1. 簽一張 JWT（claims：app_key TEST、channels AA/BB）
go run test/jwtgenerate/jwtgenerate.go gen --private-key test/key/private.pem
# 2. 換成 session token
curl -s localhost:8888/v1/auth -d '{"jwt":"<JWT>"}'
# 3. 開 socket：ws://localhost:8888/v1/apps/TEST/ws?token=<token>
#    然後訂閱：{"event":"gusher.subscribe","data":{"channel":"AA"}}
# 4. 從任一後端推播——訂閱中的 socket 就會收到
curl -s localhost:7777/v1/apps/TEST/channels/AA/messages -d '{"event":"EVENT","data":{"hi":"there"}}'
```

**換成自己的金鑰**（正式部署用）——產生一組 RSA 金鑰對，把公鑰放到 compose 檔旁邊
（或讓 `GUSHER_PUBLIC_PEM_FILE` 指過去）：

```sh
make rsakey        # 產生 private.pem + public.pem
```

完整金鑰生命週期（產生 → 簽 → 驗證 → 輪替）見 [docs/KEYS.zh-TW.md](docs/KEYS.zh-TW.md)。

## 執行方式（從原始碼編譯）

```sh
go build -ldflags "-X main.name=gusher" -o gusher.cluster .

# slave
GUSHER_NATS_ADDR=nats://127.0.0.1:4222 GUSHER_PUBLIC_PEM_FILE=./public.pem \
GUSHER_API_LISTEN=:8888 ./gusher.cluster slave

# master
GUSHER_NATS_ADDR=nats://127.0.0.1:4222 GUSHER_PUBLIC_PEM_FILE=./public.pem \
GUSHER_MASTER_API_LISTEN=:7777 ./gusher.cluster master
```

完整環境變數清單見 `slave.env.example` / `master.env.example`。

## 容器映像（Docker Hub）

repo 內的 `docker-compose.yml` 與 `example/` 都是**從原始碼建**（開發 / demo 用）。
若要**不自己 build、直接部署**,從 Docker Hub 拉發佈好的 image——
[**syhlion/gusher.cluster**](https://hub.docker.com/r/syhlion/gusher.cluster),
每個 release 都有 tag(`:3.0.0` / `:3` / `:latest`):

```sh
docker pull syhlion/gusher.cluster:latest
# image 以「角色」當啟動指令;給它一個可連的 NATS ＋ 你的公鑰:
docker run --rm -p 7777:7777 \
  -e GUSHER_NATS_ADDR=nats://your-nats:4222 \
  -e GUSHER_MASTER_API_LISTEN=:7777 \
  -e GUSHER_PUBLIC_PEM_FILE=/public.pem \
  -v "$PWD/public.pem:/public.pem:ro" \
  syhlion/gusher.cluster:latest master
```

想在自己的 compose 裡「拉 image 而非 build」,把 `build:` 區塊換成
`image: syhlion/gusher.cluster:<tag>` 即可。

## 客端流程

1. `POST /v1/auth` 帶 `{"jwt":"<JWT>"}` → `{"token":"<JWT>"}`（JWT 在本地驗證後直接
   當 token 回傳——無狀態、不需 store）。
2. `GET /v1/apps/{app}/ws?token=<token>` → WebSocket。以
   `{"event":"gusher.subscribe","data":{"channel":"AA"}}` 訂閱。
3. 後端推播：`POST /v1/apps/{app}/channels/{channel}/messages` 帶
   `{"event":"...","data":...}`。

JWT 帶有 `gusher` claim——`{"app_key","user_id","channels"}`——以 **RS256** 簽章。
完整 WebSocket 協定見 [doc/protocal.md](./doc/protocal.md)，REST API 見
[doc/api.md](./doc/api.md)。

## 維運

- **健康檢查**：`GET /healthz`（liveness）· `GET /readyz`（readiness——只有在 NATS
  連線正常時才回 200）。
- **Console / 統計**：master 在 `GET /ui` 提供單頁 console（全域連線數/人數 ＋ 逐 app
  頻道），背後走 `GET /v1/stats` 與 `GET /v1/apps`。
- **NATS 認證**：設定 `GUSHER_NATS_CREDS=/path/to/app.creds` 使用 user credentials；
  TLS 則用 `tls://` 位址（或在 NATS server 設定）。client 會自動重連。

## 日誌

輸出方式可選，並支援輪替（rotation），透過環境變數設定：

| 環境變數 | 可選值 |
|---|---|
| `GUSHER_LOG_OUTPUT` | `stdout`（預設）/ `file` / `both` |
| `GUSHER_LOG_FILE` | 路徑（用於 `file` / `both`） |
| `GUSHER_LOG_FORMAT` | `json`（預設）/ `text` |
| `GUSHER_LOG_MAX_SIZE_MB` / `_MAX_BACKUPS` / `_MAX_AGE_DAYS` / `_COMPRESS` | 輪替設定 |

## 文件

- [example/](example/) — **可直接跑的 demo**：在後端打字、前端即時看到（一行 `docker compose` 啟動）
- [docs/ARCHITECTURE.zh-TW.md](docs/ARCHITECTURE.zh-TW.md) — 架構、NATS 匯流排/presence、從 Redis 的演進（含圖）
- [doc/protocal.md](./doc/protocal.md) — WebSocket 協定
- [doc/api.md](./doc/api.md) — REST API
- [docs/KEYS.zh-TW.md](docs/KEYS.zh-TW.md) — RSA 金鑰：產生 → 簽 → 驗證 → 輪替
- [docs/LOAD-TEST.zh-TW.md](docs/LOAD-TEST.zh-TW.md) — 壓力測試

## 測試

```
go test ./...     # 單元 + e2e（e2e 會在行程內起一顆 NATS，不需外部依賴）
```

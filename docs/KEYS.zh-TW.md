# 金鑰與 JWT 簽發

> 🌐 [English](KEYS.md) · **繁體中文**

gusher 以 **RS256 JWT** 驗證客端。本頁涵蓋整個金鑰生命週期：產生金鑰對、簽出
token、把公鑰交給 gusher，以及輪替（rotation）。

## 誰簽、誰驗

gusher **從不簽** token——它只負責**驗證**。簽是**你自己的認證服務**做的事。

```
你的 Auth 服務 ──(用 private.pem 以 RS256 簽)──▶ JWT ──▶ 玩家瀏覽器
                                                          │
玩家帶 JWT 打 gusher slave /auth、/ws ◀───────────────────┘
gusher slave/master ──(用 public.pem 驗章)──▶ 簽章正確才放行
```

| 金鑰 | 存在哪 | 機密性 |
|---|---|---|
| `private.pem` | **只**在你的認證服務 | **機密——絕不外流、絕不交給 gusher** |
| `public.pem` | gusher master & slave（`GUSHER_PUBLIC_PEM_FILE`） | 公開——可自由散布 |

演算法固定為 **RS256**。每個 token 都必須帶 `gusher` claim：
`{"app_key", "user_id", "channels"}`。

## 1. 產生金鑰對（一次性）

```sh
make rsakey
# 等同於：
#   openssl genrsa -out private.pem 2048
#   openssl rsa -in private.pem -pubout -out public.pem
```

這會產生 `private.pem`（留在你的認證服務）與 `public.pem`（交給 gusher）。RSA
最少 2048 位元，用 4096 也可以。

## 2. 簽一個 JWT

**正式環境**——你的認證服務用 `private.pem` 以 RS256 簽，payload 放 `gusher`
claim。虛擬碼：

```
claims = { "gusher": { "app_key": "TEST", "user_id": "U1", "channels": ["AA","BB"] } }
token  = jwt.sign(claims, private_pem, algorithm = "RS256")
```

**開發 / 測試**——用內建工具：

```sh
go run test/jwtgenerate/jwtgenerate.go gen \
  --private-key test/key/private.pem \
  --payload '{"gusher":{"user_id":"U1","channels":["AA","BB"],"app_key":"TEST"}}'
```

它會印出一個已簽章的 token，可直接貼給 `POST /auth`。

## 3. 把公鑰交給 gusher

master 與 slave 都需要：

```sh
GUSHER_PUBLIC_PEM_FILE=./public.pem ./gusher.cluster slave   # master 同理
```

用 docker-compose 時，把 `public.pem` 放到 compose 檔旁邊即可——它會被掛載進兩個
container。

## 4. 端到端驗證

```sh
# 1. 用上面的開發工具簽出 token → TOKEN
# 2. 拿去 /auth 兌換
curl -s -X POST http://127.0.0.1:8888/auth -d "jwt=$TOKEN"
#    → {"token":"<JWT>"}
# 3. 開 WebSocket：GET /ws/{app_key}?token=<token>
```

`make smoke` 對著實際的 compose stack 跑的就是這整套流程。

## 輪替金鑰

公鑰是在啟動時讀入，所以輪替是「重啟」而非熱重載：

1. 產生新的金鑰對。
2. 把新的 `private.pem` 佈到認證服務，讓新 token 改用它簽。
3. 更新 `GUSHER_PUBLIC_PEM_FILE`（或替換掛載的 `public.pem`），重啟 master & slave。

gusher 同一時間只信任一把公鑰。為避免讓已發出、仍在流通的 token 失效，換鑰時請先
排空（drain），或接受切換過程中有一小段舊 token 驗證失敗的空窗。

## 延伸閱讀

- [doc/protocal.md](../doc/protocal.md)——完整 WebSocket 協定與 JWT claim 結構。
- [doc/api.md](../doc/api.md)——含 `/auth` 的 REST API。
- [ARCHITECTURE.zh-TW.md](ARCHITECTURE.zh-TW.md)——認證在整體設計中的位置。

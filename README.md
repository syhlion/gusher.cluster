# Gusher.Cluster

[![Build Status](https://drone.syhlion.tw/api/badges/syhlion/gusher.cluster/status.svg)](https://drone.syhlion.tw/syhlion/gusher.cluster)
 [![Stars](https://img.shields.io/github/stars/syhlion/gusher.cluster.svg)](https://github.com/syhlion/gusher.cluster)
 [![Go](https://img.shields.io/github/go-mod/go-version/syhlion/gusher.cluster.svg)](go.mod)
 [![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
 [![Backed by NATS](https://img.shields.io/badge/backed%20by-NATS-27AAE1.svg)](https://nats.io)
 [![Docs](https://img.shields.io/badge/docs-EN%20%7C%20%E7%B9%81%E4%B8%AD-success.svg)](docs/)

> 🌐 **English** · [繁體中文](README.zh-TW.md)

Self-hosted realtime push service (Pusher-style). Browsers hold a **WebSocket**
and subscribe to channels; backends **POST** to push a message into a channel.
Horizontally scalable, **backed by NATS — no Redis**.

📐 **Architecture**: [English](docs/ARCHITECTURE.md) · [繁體中文](docs/ARCHITECTURE.zh-TW.md)

## How it works

- **slave** — holds the WebSocket connections, verifies the JWT locally, and
  fans out messages to subscribers.
- **master** — a stateless REST API to **push** messages and **query** presence.
- **NATS** — carries messages between nodes (bus) and answers presence queries
  (request/reply). No Redis, no token store, no decode service.

```
client ──ws──▶ slave ──subscribe──▶  NATS  ◀──publish── master ◀──POST── backend
```

## Requirements

- **NATS** (the only backend) — `nats-server` 2.10+
- An **RSA key pair** — master/slave verify the JWT with the **public key**;
  your own auth service signs JWTs with the private key. See
  [docs/KEYS.md](docs/KEYS.md) for the full generate → sign → verify → rotate
  walkthrough.

## Run

### docker-compose (quickest)

Put your `public.pem` next to the compose file, then:

```sh
docker compose -f docker-compose/docker-compose.yml up --build
```

Brings up `nats` + `gusher-master` (`:7777`) + `gusher-slave` (`:8888`), no Redis.

### From source

```sh
go build -ldflags "-X main.name=gusher" -o gusher.cluster .

# slave
GUSHER_NATS_ADDR=nats://127.0.0.1:4222 GUSHER_PUBLIC_PEM_FILE=./public.pem \
GUSHER_API_LISTEN=:8888 GUSHER_API_URI_PREFIX=/ ./gusher.cluster slave

# master
GUSHER_NATS_ADDR=nats://127.0.0.1:4222 GUSHER_PUBLIC_PEM_FILE=./public.pem \
GUSHER_MASTER_API_LISTEN=:7777 GUSHER_MASTER_URI_PREFIX=/ ./gusher.cluster master
```

See `slave.env.example` / `master.env.example` for the full env list.

## Ops

- **Health**: `GET /ping` (liveness) · `GET /ready` (readiness — 200 only while
  NATS is connected).
- **NATS auth**: set `GUSHER_NATS_CREDS=/path/to/app.creds` for user credentials;
  use a `tls://` address (or NATS server config) for TLS. The client auto-reconnects.

## Client flow

1. `POST /auth` with `jwt=<JWT>` → `{"token":"<JWT>"}` (the JWT is verified
   locally and returned as the token — stateless, no store).
2. `GET /ws/{app_key}?token=<token>` → WebSocket. Subscribe with
   `{"event":"gusher.subscribe","data":{"channel":"AA"}}`.
3. Backend pushes: `POST /push/{app_key}/{channel}/{event}` with `data=...`.

The JWT carries the `gusher` claim — `{"app_key","user_id","channels"}` — signed
**RS256**. See [doc/protocal.md](./doc/protocal.md) for the full WebSocket
protocol and [doc/api.md](./doc/api.md) for the REST API.

## Logging

Output is selectable and rotated, via env:

| Env | Values |
|---|---|
| `GUSHER_LOG_OUTPUT` | `stdout` (default) / `file` / `both` |
| `GUSHER_LOG_FILE` | path (for `file` / `both`) |
| `GUSHER_LOG_FORMAT` | `json` (default) / `text` |
| `GUSHER_LOG_MAX_SIZE_MB` / `_MAX_BACKUPS` / `_MAX_AGE_DAYS` / `_COMPRESS` | rotation |

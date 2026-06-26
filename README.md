# Gusher.Cluster

[![Stars](https://img.shields.io/github/stars/syhlion/gusher.cluster.svg)](https://github.com/syhlion/gusher.cluster)
[![Build Status](https://drone.syhlion.tw/api/badges/syhlion/gusher.cluster/status.svg)](https://drone.syhlion.tw/syhlion/gusher.cluster)
[![Go](https://img.shields.io/github/go-mod/go-version/syhlion/gusher.cluster.svg)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Backed by NATS](https://img.shields.io/badge/backed%20by-NATS-27AAE1.svg)](https://nats.io)
[![Docker](https://img.shields.io/docker/v/syhlion/gusher.cluster?sort=semver&logo=docker&logoColor=white&label=docker)](https://hub.docker.com/r/syhlion/gusher.cluster)
[![docs English](https://img.shields.io/badge/docs-English-blue.svg)](README.md)
[![docs 繁體中文](https://img.shields.io/badge/docs-%E7%B9%81%E9%AB%94%E4%B8%AD%E6%96%87-lightgrey.svg)](README.zh-TW.md)

Self-hosted realtime push service (Pusher-style). Browsers hold a **WebSocket**
and subscribe to channels; backends **POST** to push a message into a channel.
Horizontally scalable, **backed by NATS — no Redis**.

## Architecture

![system](docs/diagrams/system.drawio.png)

- **slave** — holds the WebSocket connections, verifies the JWT locally (RSA
  public key), and fans out messages to subscribers.
- **master** — a stateless REST API to **push** messages and **query** presence.
- **NATS** — carries messages between nodes (bus) and answers presence queries
  (request/reply). No Redis, no token store, no decode service.

```
client ──ws──▶ slave ──subscribe──▶  NATS  ◀──publish── master ◀──POST── backend
```

Both roles are stateless and scale horizontally. Full write-up (bus, presence,
evolution from Redis) in [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Requirements

- **NATS** (the only backend) — `nats-server` 2.10+.
- An **RSA key pair** — master/slave verify the JWT with the **public key**;
  your own auth service signs JWTs with the private key. See
  [docs/KEYS.md](docs/KEYS.md) for the full generate → sign → verify → rotate
  walkthrough.

## Quick start (Docker Compose)

The stack ships a **demo RSA key** (`docker-compose/public.pem`, matched by
`test/key/private.pem`), so it runs out of the box — no setup:

```sh
docker compose -f docker-compose/docker-compose.yml up --build
```

Brings up `nats` + `gusher-master` (`:7777`) + `gusher-slave` (`:8888`), no Redis.

**See it work** — one command brings the stack up, signs a JWT, subscribes over
WebSocket, pushes a message and asserts it arrives, then tears down:

```sh
make smoke
```

Or test by hand against the running stack — sign a token with the demo key, then
auth → connect → push:

```sh
# 1. sign a JWT (claims: app_key TEST, channels AA/BB)
go run test/jwtgenerate/jwtgenerate.go gen --private-key test/key/private.pem
# 2. exchange it for a session token
curl -s localhost:8888/v1/auth -d '{"jwt":"<JWT>"}'
# 3. open the socket: ws://localhost:8888/v1/apps/TEST/ws?token=<token>
#    then subscribe: {"event":"gusher.subscribe","data":{"channel":"AA"}}
# 4. push from any backend — the subscribed socket receives it
curl -s localhost:7777/v1/apps/TEST/channels/AA/messages -d '{"event":"EVENT","data":{"hi":"there"}}'
```

**Use your own keys** (for real deployments) — generate an RSA pair and drop the
public half next to the compose file (or point `GUSHER_PUBLIC_PEM_FILE` at it):

```sh
make rsakey        # writes private.pem + public.pem
```

Full key lifecycle (generate → sign → verify → rotate) in [docs/KEYS.md](docs/KEYS.md).

## Run (from source)

```sh
go build -ldflags "-X main.name=gusher" -o gusher.cluster .

# slave
GUSHER_NATS_ADDR=nats://127.0.0.1:4222 GUSHER_PUBLIC_PEM_FILE=./public.pem \
GUSHER_API_LISTEN=:8888 ./gusher.cluster slave

# master
GUSHER_NATS_ADDR=nats://127.0.0.1:4222 GUSHER_PUBLIC_PEM_FILE=./public.pem \
GUSHER_MASTER_API_LISTEN=:7777 ./gusher.cluster master
```

See `slave.env.example` / `master.env.example` for the full env list.

## Container image

The repo's `docker-compose.yml` and `example/` **build from source** (for dev /
the demo). To deploy without building, pull the published image from Docker Hub —
[**syhlion/gusher.cluster**](https://hub.docker.com/r/syhlion/gusher.cluster),
tagged per release (`:3.0.0` / `:3` / `:latest`):

```sh
docker pull syhlion/gusher.cluster:latest
# the image takes the role as its command; give it a reachable NATS + your public key:
docker run --rm -p 7777:7777 \
  -e GUSHER_NATS_ADDR=nats://your-nats:4222 \
  -e GUSHER_MASTER_API_LISTEN=:7777 \
  -e GUSHER_PUBLIC_PEM_FILE=/public.pem \
  -v "$PWD/public.pem:/public.pem:ro" \
  syhlion/gusher.cluster:latest master
```

To pull instead of build in your own compose, swap the `build:` block for
`image: syhlion/gusher.cluster:<tag>`.

## Client flow

1. `POST /v1/auth` with `{"jwt":"<JWT>"}` → `{"token":"<JWT>"}` (the JWT is
   verified locally and returned as the token — stateless, no store).
2. `GET /v1/apps/{app}/ws?token=<token>` → WebSocket. Subscribe with
   `{"event":"gusher.subscribe","data":{"channel":"AA"}}`.
3. Backend pushes: `POST /v1/apps/{app}/channels/{channel}/messages` with
   `{"event":"...","data":...}`.

The JWT carries the `gusher` claim — `{"app_key","user_id","channels"}` — signed
**RS256**. See [doc/protocal.md](./doc/protocal.md) for the full WebSocket
protocol and [doc/api.md](./doc/api.md) for the REST API.

## Ops

- **Health**: `GET /healthz` (liveness) · `GET /readyz` (readiness — 200 only while
  NATS is connected).
- **Console / stats**: the master serves a single-page console at `GET /ui` (global
  connections/users + per-app channels) over `GET /v1/stats` and `GET /v1/apps`.
- **NATS auth**: set `GUSHER_NATS_CREDS=/path/to/app.creds` for user credentials;
  use a `tls://` address (or NATS server config) for TLS. The client auto-reconnects.

## Logging

Output is selectable and rotated, via env:

| Env | Values |
|---|---|
| `GUSHER_LOG_OUTPUT` | `stdout` (default) / `file` / `both` |
| `GUSHER_LOG_FILE` | path (for `file` / `both`) |
| `GUSHER_LOG_FORMAT` | `json` (default) / `text` |
| `GUSHER_LOG_MAX_SIZE_MB` / `_MAX_BACKUPS` / `_MAX_AGE_DAYS` / `_COMPRESS` | rotation |

## Docs

- [example/](example/) — **runnable demo**: type on a backend, watch it arrive live on a frontend (one `docker compose` command)
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — architecture, NATS bus/presence, evolution from Redis (with diagrams)
- [doc/protocal.md](./doc/protocal.md) — WebSocket protocol
- [doc/api.md](./doc/api.md) — REST API
- [docs/KEYS.md](docs/KEYS.md) — RSA keys: generate → sign → verify → rotate
- [docs/LOAD-TEST.md](docs/LOAD-TEST.md) — load testing

## Tests

```
go test ./...     # unit + e2e (e2e spins up an in-process NATS, no external deps)
```

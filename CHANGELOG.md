## [Unreleased]

### [Added]

- **Global observability endpoints** (master): `GET /v1/stats` (totals across all
  apps) and `GET /v1/apps` (per-app breakdown), each reporting **connections**
  (exact — summed sockets) and **users** (approximate — summed per-node distinct
  uids). Backed by the new `redisocket.v2` `Stats` scatter-gather (bumped to
  v1.1.0). Counts refresh on each presence sync (`GUSHER_SCAN_INTERVAL`, default 30s).

### [Changed / API]

- **HTTP API redesigned to a clean resource-oriented REST shape** and moved off
  `gorilla/mux` to the stdlib `net/http.ServeMux` (Go 1.22+ method routing; one
  fewer dependency). Resources are `apps`, `channels`, `users`, `sockets`,
  `messages`; the HTTP verb carries the action and request bodies are JSON
  (`{event,data}`) instead of form values. Examples:
  - publish: `POST /v1/apps/{app}/channels/{channel}/messages` (was `POST /push/{app}/{channel}/{event}`)
  - pattern / batch: `POST /v1/apps/{app}/messages` / `.../messages/batch`
  - to user / socket: `POST /v1/apps/{app}/users/{user}/messages` / `.../sockets/{socket}/messages`
  - a user's channels: `POST` (add) / `PUT` (replace) `/v1/apps/{app}/users/{user}/channels`
  - presence: `GET /v1/apps/{app}/channels[/count]`, `GET /v1/apps/{app}/users[/count]`, `GET /v1/apps/{app}/channels/{channel}/users[/count]`
  - auth / ws: `POST /v1/auth`, `GET /v1/apps/{app}/ws?token=`; decode: `POST /v1/auth/decode`
- **Health probes renamed** `/ping`→`/healthz`, `/ready`→`/readyz`; added `GET /version`.
- **Dropped the configurable URI prefix** (`GUSHER_MASTER_URI_PREFIX` /
  `GUSHER_API_URI_PREFIX`) and the legacy `/wtf` ws alias — the API path is fixed
  at `/v1`. **Breaking**, hard cutover (no aliases).

## [v2.0.0] - 2026-06-25

> **Breaking**: the backend is now NATS (no Redis); env and the `remote` feature
> changed. See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

### [Changed]

- **Migrate from Redis to NATS** — realtime bus (subject routing) and presence
  (per-node memory + request/reply) now run on NATS; **Redis is removed**.
- **Local JWT auth** — verify the JWT with the RSA public key in-process; no
  decode service, no token store. The `/auth` → `/ws?token=` flow is unchanged.
- Adopt the modernized `redisocket.v2` v1.0.0 engine (slog logger; output
  stdout/file/both + log rotation via `GUSHER_LOG_*`).
- NATS reconnect/creds support; `/ready` readiness probe; pprof bound to
  localhost; `/ws` and `/wtf` share one handler.

### [Removed]

- The `remote` feature (`gusher.remote`, fire-and-forget RPUSH) — unused. A
  future client→backend channel will use NATS request/reply.
- All Redis env (`GUSHER_REDIS_*`, `GUSHER_JOB_REDIS_*`, `GUSHER_DECODE_SERVICE`).

## [v1.13.2]

- fix jwt bug change package github.com/dgrijalva/jwt-go  to https://github.com/golang-jwt/jwt

## [v1.13.1]

- fix gorilla websocket CVE bug
- fix jsonparser CVE bug

## [v1.13.0]

- add env switch log to josn or text (default json)
- log change to json


## [v1.12.0]

- add websocket reply message add field socket_id


## [v1.11.0]

- add remote reponse switch

## [v1.10.1]

- fix remote dont close
- update docker base image



## [v1.9.0]

- add reload channel api

## [v1.8.6]

- fix docker-compose
- fix start job redis empty error msg


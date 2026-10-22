## [Unreleased]

### [Security]

- Go toolchain 1.25 → 1.27.3 (go.mod `go 1.27.0` + `toolchain go1.27.3`,
  Docker base image `golang:1.27.3-alpine`, CI pinned to 1.27.3). Fixes
  CVE-2026-111943 (crypto/x509 `Verify` did not reject the revoked eMudhra
  intermediate certificates). Go 1.25 is out of support.
- runtime image `alpine:3.20` → `alpine:3.24` (3.20 is EOL), for both
  `docker/Dockerfile` and `example/app/Dockerfile`.

## [v3.0.1] - 2026-09-04

### [Security]

- bump `github.com/buger/jsonparser` v1.1.1 → v1.1.2 (Dependabot high,
  GHSA-6g7g-w4f8-9c9x: denial of service on crafted input).

### [CI]

- **CI moved from Drone to GitHub Actions** (`.github/workflows/ci.yml`); the
  self-hosted Drone at drone.syhlion.tw is retired (its README badge had been
  404ing). `test` (build, vet, `go test -race`) runs on master pushes, PRs and
  tags; `docker` builds `docker/Dockerfile` and pushes `syhlion/gusher.cluster`
  on `v*` tags only, stamping the tag into `main.version`. Not carried over:
  the cross-platform tarballs + draft GitHub release and the Telegram
  notification (releases are cut by hand with `gh`). `.drone.yml` removed.
  The Docker Hub credentials are read from the repo's `dockerhub` environment.

## [v3.0.0] - 2026-06-26

> **Breaking**: HTTP API redesigned to a clean resource-oriented REST shape under
> `/v1` (see below). No external consumers yet — hard cutover, no aliases.

### [Added]

- **Engineering console** (master): `GET /ui` — a single self-contained HTML page
  (no build, no auth) showing master health, global connection/user totals, a
  per-app breakdown, and per-channel online counts for an entered app_key, with
  optional 3s auto-refresh. Mirrors gua's console; a probe + API-validation
  surface, not an end-user product.
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


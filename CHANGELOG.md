## [Unreleased]

### [Changed]

- **Migrate from Redis to NATS** — realtime bus (subject routing) and presence
  (per-node memory + request/reply) now run on NATS; **Redis is removed**.
- **Local JWT auth** — verify the JWT with the RSA public key in-process; no
  decode service, no token store. The `/auth` → `/ws?token=` flow is unchanged.
- Adopt the modernized `redisocket.v2` engine (slog logger; output
  stdout/file/both + log rotation via `GUSHER_LOG_*`).
- pprof bound to localhost; `/ws` and `/wtf` share one handler.

### [Removed]

- The `remote` feature (`gusher.remote`, fire-and-forget RPUSH) — unused. A
  future client→backend channel will use NATS request/reply.
- All Redis env (`GUSHER_REDIS_*`, `GUSHER_JOB_REDIS_*`, `GUSHER_DECODE_SERVICE`).

### [Fix]

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


# gusher.cluster HTTP API

All endpoints are versioned under `/v1`; operational probes live at the root.
Request and response bodies are JSON.

- **slave** serves the client-facing auth + WebSocket endpoints.
- **master** serves the backend publish + presence-query endpoints.

## Operational (both roles)

| Method | Path | Notes |
|---|---|---|
| GET | `/healthz` | liveness — `200 ok` while the process serves |
| GET | `/readyz` | readiness — `200 ready` when NATS is connected, `503` otherwise |
| GET | `/version` | build version string |
| GET | `/ui` | master only — single-page engineering console (global stats + per-app channels) |

## Master observability

Counts refresh on each presence sync (`GUSHER_SCAN_INTERVAL`, default 30s).
`connections` is exact; `users` is an approximate sum across nodes (a user
connected to multiple nodes is counted per node).

| Method | Path | Response |
|---|---|---|
| GET | `/v1/stats` | `{"apps":3,"connections":348,"users":312}` — totals across all apps |
| GET | `/v1/apps` | `[{"app":"TEST","connections":120,"users":110}, ...]` — per-app, sorted |

## Slave API

### Auth — `POST /v1/auth`

Body: `{"jwt":"<JWT>"}`. The JWT is verified locally with the RSA public key and
returned as the session token (stateless — no store).

```json
{ "token": "<JWT>" }
```

JWT [ref](https://jwt.io) · [example](https://github.com/syhlion/gusher.cluster/blob/master/jwt.example)

### Connect — `GET /v1/apps/{app}/ws?token=<token>`

Upgrades to a WebSocket. Subscribe over the socket with
`{"event":"gusher.subscribe","data":{"channel":"AA"}}`.

## Master API

### Presence queries

| Method | Path | Response |
|---|---|---|
| GET | `/v1/apps/{app}/channels` | `["channel1","channel2", ...]` |
| GET | `/v1/apps/{app}/channels/count` | `{"count":3}` |
| GET | `/v1/apps/{app}/channels/{channel}/users` | `["user_id", ...]` |
| GET | `/v1/apps/{app}/channels/{channel}/users/count` | `{"count":3}` |
| GET | `/v1/apps/{app}/users` | `["user_id", ...]` |
| GET | `/v1/apps/{app}/users/count` | `{"count":3}` |

### Publish to a channel — `POST /v1/apps/{app}/channels/{channel}/messages`

Body:

```json
{ "event": "notify", "data": { "key": "value" } }
```

`data` may be any JSON value (object, string, number). Response echoes the
delivered envelope: `{"channel","event","data"}`.

### Publish by channel pattern — `POST /v1/apps/{app}/messages`

Body:

```json
{ "channel_pattern": "^App", "event": "notify", "data": { "key": "value" } }
```

`channel_pattern` is a regular expression matched against the app's live
channels. Response: `{"total":1,"pattern":"^App"}`.

### Batch publish — `POST /v1/apps/{app}/messages/batch`

Body is a JSON array of messages:

```json
[
  { "channel": "public", "event": "notify", "data": "test" },
  { "channel": "public", "event": "notify", "data": { "username": "test" } }
]
```

Response: `{"total":2}`.

### Push to a user — `POST /v1/apps/{app}/users/{user}/messages`

Body: `{"data": {"key":"value"}}` → `{"user_id":"...","data":...}`.

### Push to a socket — `POST /v1/apps/{app}/sockets/{socket}/messages`

Body: `{"data": {"key":"value"}}` → `{"socket_id":"...","data":...}`.

### Add a channel to a user — `POST /v1/apps/{app}/users/{user}/channels`

Body: `{"channel":"aa"}` → `{"user_id":"...","data":"aa"}`.

### Replace a user's channels — `PUT /v1/apps/{app}/users/{user}/channels`

Body: `{"channels":["gg","ff"]}` → `{"user_id":"...","data":["gg","ff"]}`.

### Decode a JWT (debug) — `POST /v1/auth/decode`

Body: `{"jwt":"<JWT>"}`.

```json
{ "gusher": { "channels": [], "user_id": "", "app_key": "" } }
```

---

* note1: a `channels` slice containing `"*"` lets the user subscribe to all channels.
* note2: `*` glob is supported, e.g. `t*st` matches `test`, `app*` matches `apple`.

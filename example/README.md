# gusher.cluster example — realtime push demo

A one-command, self-contained demo: type a message on one side and watch it
arrive **live** on the other over a WebSocket. It shows the whole path —
backend publish → master → NATS → slave → browser — with no Redis and no setup.

## Run

From the repo root:

```sh
docker compose -f example/docker-compose.yml up --build
```

Then open **<http://localhost:8080>**.

- **Left pane (backend → publish)** — type a message and hit *send*. The demo
  backend proxies it to the master: `POST /v1/apps/TEST/channels/demo/messages`.
- **Right pane (frontend → subscribe)** — a browser WebSocket to the slave
  (`/v1/apps/TEST/ws`), subscribed to channel `demo`. Whatever you send on the
  left pops up here instantly. Open a second tab to see the fan-out.

## What's in the stack

| Service | Role |
|---|---|
| `nats` | the only backend (bus + presence) |
| `gusher-master` (`:7777`, internal) | publish/REST API — the demo proxies to it |
| `gusher-slave` (`:8888`) | holds the WebSocket the browser connects to |
| `demo` (`:8080`) | tiny stdlib Go backend: serves the page, signs a demo JWT, proxies publishes |

## How auth works here

gusher verifies a JWT (RS256) signed by **your** auth service. This demo reuses
the repo's demo keypair (`test/key/`): the `demo` backend signs a token with the
private key (so the browser never sees it) and master/slave verify with the
matching public key. In production you'd swap in your own keys — see
[../docs/KEYS.md](../docs/KEYS.md).

> Demo only — no auth on the demo backend, fixed app/channel (`TEST`/`demo`).
> For the full HTTP API see [../doc/api.md](../doc/api.md).

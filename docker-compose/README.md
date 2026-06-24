# gusher.cluster docker-compose (NATS)

Starts `nats` + `gusher-master` (`:7777`) + `gusher-slave` (`:8888`) — no Redis.

## Usage

Put your RSA `public.pem` in this directory (for a quick test:
`cp ../test/key/public.pem .`), then from the repo root:

```sh
docker compose -f docker-compose/docker-compose.yml up --build
```

master/slave verify the JWT with that public key. Env is set inline in
`docker-compose.yml` (`GUSHER_NATS_ADDR`, `GUSHER_LOG_*`, listen / prefix).

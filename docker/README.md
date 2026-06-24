# gusher.cluster Dockerfile

Multi-stage build from the local source + `vendor/` (Go 1.25). The image
entrypoint is the `gusher.cluster` binary — run it as `master` or `slave`.

## Build

```sh
docker build -f docker/Dockerfile -t gusher.cluster:nats .
```

Or just use [`docker-compose/`](../docker-compose), which builds this image and
brings up NATS + master + slave.

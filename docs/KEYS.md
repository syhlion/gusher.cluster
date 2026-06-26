# Keys & JWT signing

> 🌐 **English** · [繁體中文](KEYS.zh-TW.md)

gusher authenticates clients with an **RS256 JWT**. This page covers the whole
key lifecycle: generate the key pair, sign a token, hand the public key to
gusher, and rotate.

## Who signs, who verifies

gusher **never signs** tokens — it only **verifies** them. Signing is done by
**your own auth service**.

```
your Auth service ──(sign RS256 with private.pem)──▶ JWT ──▶ player browser
                                                              │
player presents JWT to gusher slave /auth, /ws ◀──────────────┘
gusher slave/master ──(verify with public.pem)──▶ allow only if the signature is valid
```

| Key | Lives where | Secrecy |
|---|---|---|
| `private.pem` | **only** in your auth service | **secret — never share, never ship to gusher** |
| `public.pem` | gusher master & slave (`GUSHER_PUBLIC_PEM_FILE`) | public — safe to distribute |

The algorithm is fixed to **RS256**. Every token must carry a `gusher` claim:
`{"app_key", "user_id", "channels"}`.

## 1. Generate the key pair (one-off)

```sh
make rsakey
# equivalent to:
#   openssl genrsa -out private.pem 2048
#   openssl rsa -in private.pem -pubout -out public.pem
```

This writes `private.pem` (keep it in your auth service) and `public.pem` (give
it to gusher). 2048-bit RSA is the minimum; 4096 is fine too.

## 2. Sign a JWT

**Production** — your auth service signs with `private.pem` using RS256, putting
the `gusher` claim in the payload. Pseudocode:

```
claims = { "gusher": { "app_key": "TEST", "user_id": "U1", "channels": ["AA","BB"] } }
token  = jwt.sign(claims, private_pem, algorithm = "RS256")
```

**Development / testing** — use the bundled tool:

```sh
go run test/jwtgenerate/jwtgenerate.go gen \
  --private-key test/key/private.pem \
  --payload '{"gusher":{"user_id":"U1","channels":["AA","BB"],"app_key":"TEST"}}'
```

It prints a signed token you can paste into `POST /auth`.

## 3. Give the public key to gusher

Both master and slave need it:

```sh
GUSHER_PUBLIC_PEM_FILE=./public.pem ./gusher.cluster slave   # and master
```

With docker-compose, just place `public.pem` next to the compose file — it is
mounted into both containers.

## 4. Verify it end to end

```sh
# 1. sign a token (dev tool above) → TOKEN
# 2. exchange it at /v1/auth
curl -s -X POST http://127.0.0.1:8888/v1/auth -d "{\"jwt\":\"$TOKEN\"}"
#    → {"token":"<JWT>"}
# 3. open the WebSocket: GET /v1/apps/{app}/ws?token=<token>
```

`make smoke` runs exactly this flow against a live compose stack.

## Rotating keys

The public key is read at startup, so rotation is a restart, not a live reload:

1. Generate a new pair.
2. Roll the new `private.pem` out to the auth service so new tokens are signed
   with it.
3. Update `GUSHER_PUBLIC_PEM_FILE` (or replace the mounted `public.pem`) and
   restart master & slave.

gusher trusts exactly one public key at a time. To avoid invalidating tokens
already in flight, drain or accept a short window where old tokens fail
verification during the swap.

## See also

- [doc/protocal.md](../doc/protocal.md) — full WebSocket protocol and the JWT claim shape.
- [doc/api.md](../doc/api.md) — REST API including `/auth`.
- [ARCHITECTURE.md](ARCHITECTURE.md) — where auth sits in the overall design.

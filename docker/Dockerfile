# build stage（本地原始碼 + vendor;Go 1.25，無 redis、NATS 後端）
FROM golang:1.25-alpine AS builder
WORKDIR /app
# VERSION is stamped into main.version (shown by GET /version); passed from CI as
# the release tag, empty for non-tag (:latest) builds.
ARG VERSION
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor \
    -ldflags "-X main.name=gusher -X main.version=${VERSION}" -o /gusher.cluster .

# final stage
FROM alpine:3.20
WORKDIR /gusher
COPY --from=builder /gusher.cluster .
EXPOSE 7777 8888
ENTRYPOINT ["./gusher.cluster"]

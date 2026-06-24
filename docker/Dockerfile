# build stage（本地原始碼 + vendor;Go 1.25，無 redis、NATS 後端）
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor \
    -ldflags "-X main.name=gusher" -o /gusher.cluster .

# final stage
FROM alpine:3.20
WORKDIR /gusher
COPY --from=builder /gusher.cluster .
EXPOSE 7777 8888
ENTRYPOINT ["./gusher.cluster"]

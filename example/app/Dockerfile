FROM golang:1.27.3-alpine AS build
WORKDIR /src
COPY main.go .
RUN go mod init gusher-example && CGO_ENABLED=0 go build -o /demo .

FROM alpine:3.24
WORKDIR /app
COPY --from=build /demo ./demo
COPY static ./static
EXPOSE 8080
ENTRYPOINT ["./demo"]

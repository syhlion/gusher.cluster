# build stage
FROM golang:1.19.2-alpine3.16 AS builder

WORKDIR /app
RUN apk update && apk add git && apk add make
RUN apk add --update gcc g++
RUN git clone https://github.com/syhlion/gusher.cluster.git  &&\
    cd gusher.cluster && \
    make docker-build

# final stage
FROM alpine:3.16.2
WORKDIR /gusher
COPY --from=builder /app/gusher.cluster/gusher.cluster .

EXPOSE 8888

ENTRYPOINT ["./gusher.cluster"]

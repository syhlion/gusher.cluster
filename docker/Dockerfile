# build stage
FROM golang:1.12.2-alpine3.9 AS builder

WORKDIR /app
RUN apk update && apk add git && apk add make
RUN apk add --update gcc g++
RUN git clone https://github.com/syhlion/gusher.cluster.git  &&\
    cd gusher.cluster && \
    make docker-build

# final stage
FROM scratch
WORKDIR /gusher
COPY --from=builder /app/gusher.cluster/gusher.cluster .

EXPOSE 8888

ENTRYPOINT ["./gusher.cluster"]

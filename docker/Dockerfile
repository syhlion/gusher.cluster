FROM golang:1.11.2-alpine3.7

RUN apk update && apk add git && apk add make
RUN apk add --update gcc g++
RUN git clone https://github.com/syhlion/gusher.cluster.git /go/src/gusher.cluster &&\
    cd /go/src/gusher.cluster && \
    make docker-build

WORKDIR /go/src/gusher.cluster

EXPOSE 8888

ENTRYPOINT ["./gusher.cluster"]

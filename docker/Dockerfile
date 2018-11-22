FROM golang:1.11.2

RUN git clone https://github.com/syhlion/gusher.cluster.git /go/src/gusher.cluster &&\
    cd /go/src/gusher.cluster && \
    make docker-build

WORKDIR /go/src/gusher.cluster

EXPOSE 8888

ENTRYPOINT ["./gusher.cluster"]

FROM golang:1.7.1

RUN git clone --branch v0.4.0  https://github.com/syhlion/gusher.cluster.git /go/src/gusher.cluster &&\
    cd /go/src/gusher.cluster && \
    make build 

WORKDIR /go/src/gusher.cluster

EXPOSE 8000

ENTRYPOINT ["./gusher.cluster"]

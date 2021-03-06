# Gusher.Cluster

[![Build Status](https://drone.syhlion.tw/api/badges/syhlion/gusher.cluster/status.svg)](https://drone.syhlion.tw/syhlion/gusher.cluster)
 [![Release Status](https://img.shields.io/badge/release-1.8.2-blue.svg)](https://github.com/syhlion/gusher.cluster/releases/tag/v1.8.2)
 [![Stars](https://img.shields.io/github/stars/syhlion/gusher.cluster.svg)](https://github.com/syhlion/gusher.cluster)

 [gusher](https://github.com/syhlion/gusher) plus version ,support cluster

## Changelog

[CHANGELOG](./CHANGELOG.md)

## Requirements

* redis

## Usage

Docker example use [doc](./docker)

```
docker pull syhlion/gusher.cluster
docker run --name docker-redis -d redis
docker run --env-file env.example --link docker-redis --name gusher-master -p 7999:8888 -d syhlion/gusher.cluster master //master mode
docker run --env-file env.example --link docker-redis --link gusher-master --name gusher-slave -p 8000:8888 -d syhlion/gusher.cluster slave //slave mode
//note env & link hostname
```

docker-compose use [doc](./docker-compose) 

Build from source:

Package Management use [govendor](https://github.com/kardianos/govendor)

```
$ go get github.com/syhlion/gusher.cluster && cd $GOPATH/github.com/syhlion/gusher.cluster
$ make build/linux

```

Download:

[release](./releases)



And Set ENV  like [example](./env.example)


Than Use

master mode:

`$ ./gusher.cluster master` or `./gusher.cluster master --env-file env.example`

slave mode:

`$ ./gusher.cluster slave` or `./gusher.cluster slave --env-file env.example`



## Third party lib

client js:

[gusher-js](https://github.com/cswleocsw/gusher-js)

backend php:

[gusher-php](https://github.com/benjaminchen/gusher-php)

## Api

[Api Doc](./doc/api.md)


## Internal Protocal

[Protocal Doc](./doc/protocal.md)

## Thanks

* [@leo](https://github.com/cswleocsw) , [@benjamin](https://github.com/benjaminchen) support api lib
* [pusher](https://pusher.com) inspiration
* [gorilla/websocket](https://github.com/gorilla/websocket)

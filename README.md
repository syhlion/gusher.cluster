# Gusher.Cluster

 [gusher](https://github.com/syhlion/gusher) Plus version ,support cluster

## Requirements

* redis

## Usage

Docker example use

```
docker pull syhlion/gusher.cluster
docker run --name docker-redis -d redis
docker run --env-file env.example --link docker-redis --name gusher-master -p 7999:8000 -d syhlion/gusher.cluster master //master mode
docker run --env-file env.example --link docker-redis --link gusher-master --name gusher-slave -p 8000:8000 -d syhlion/gusher.cluster slave //slave mode
```

Install from source:

Package Management use [glide](https://github.com/Masterminds/glide)

```
$ git clone github.com/syhlion/gusher.cluster && cd gusher.cluster
$ make build

```

Download:

[Debian & Ubuntu use](https://github.com/syhlion/gusher.cluster/releases)



And Set ENV  like [example](https://github.com/syhlion/gusher.cluster/blob/master/env.example)


Than Use

master mode:

`./gusher.cluster master` or `./gusher.cluster master --env-file env.example`

slave mode:

`./gusher.cluster slave` or `./gusher.cluster slave --env-file env.example`



## Third party lib

client js:

[gusher-js](https://github.com/cswleocsw/gusher-js)

backend php:

[gusher-php](https://github.com/benjaminchen/gusher-php)

## Api

[Api Doc](https://github.com/syhlion/gusher.cluster/blob/master/doc/api.md)


## Internal Protocal

[Protocal Doc](https://github.com/syhlion/gusher.cluster/blob/master/doc/protocal.md)

## Thanks

* [@leo](https://github.com/cswleocsw) , [@benjamin](https://github.com/benjaminchen) support api lib
* [pusher](https://pusher.com) inspiration

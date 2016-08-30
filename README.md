# Gusher.Cluster

 [gusher](https://github.com/syhlion/gusher) Plus version ,support cluster

## Requirements

* redis

## Usage

Install from source:

`go get -u github.com/syhlion/gusher.cluster`

And Set .env like [example](https://github.com/syhlion/gusher.cluster/blob/master/.env.example)

master mode:

`./gusher.cluster master`

slave mode:

`./gusher.cluster slave`

## Slave  Api

Connect:

`GET /ws/{app_key}?auth={auth}`

## Master Api

Check app_key exist:

`GET /api/check/{app_key}`


Scuess Response:


```
{
    "app_key":""
}
```

Push Message:

`POST /api/push/{app_key}/{channel}/{event}?data={data}`


Scuess Response:
```
{
    "channel":"",
    "event":"",
    "data":""
}
```

Slave Server Info:

`GET /api/system/slaveinfos`


Scuess Response:

```
{
    "'{ip}'+'@'+'{listen_port}'":{
        "ip":"",
        "local_listen":"",
        "version":"",
        "runtime_version":"",
        "cpu":,
        "usage-memory":,
        "goroutines":6,
        "connections":,
        "send_interval":"",
        "update_time":
    }
}
```

## Client Request Protocol

### Event list:

Events|Discription
---|---
gusher.subscribe|subscribe event
gusher.unsubscribe|unsubscribe event
gusher.subscribe_succeeded|subscribe sucess
gusher.subscribe_error|subscribe error
gusher.unsubscribe_succeeded|unsubscribe sucess
gusher.unsubscribe_error|unsubscribe error

#### Common Receive Message:

```
{
    "channel":"",
    "event":"",
    "data":
}
```

#### Subscribe Command:

command:
```
{
    "event":"gusher.subscribe",
    "data":{
        "id":"",
        "channel":""
    }
}
```

reply scuess:
```
{
    "event":"gusher.subscribe_succeeded",
    "data":{
        "id":"",
        "channel":""
    }
}
```
reply error:
```
{
    "event":"gusher.subscribe_error",
    "data":{
        "id":"",
        "channel":""
    }
}
```

#### Unsubscribe Command:

command:
```
{
    "event":"gusher.unsubscribe",
    "data":{
        "id":"",
        "channel":""
    }
}
```

reply scuess:
```
{
    "event":"gusher.unsubscribe_succeeded",
    "data":{
        "id":"",
        "channel":""
    }
}
```

reply error:
```
{
    "event":"gusher.unsubscribe_error",
    "data":{
        "id":""
        "channel":"",
    }
}
```

## WebHook Response Protocol

```
{
    "user_id":"BlackJack....",
    "channels":["channel1","channel2"...]
}
```

## Admin Protocol

Use Redis Hashes to stored

WebHook Storage 

Key|field|value
---|---|---
{app_key}|url|http://hook-domain/






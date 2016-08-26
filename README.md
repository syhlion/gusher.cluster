# Gusher.Cluster

 [gusher](https://github.com/syhlion/gusher) Plus version ,support cluster

## Requirements

* redis

## Usage

Install from source

`go get -u github.com/syhlion/gusher.cluster`

And Set .env like [example](https://github.com/syhlion/gusher.cluster/blob/master/.env.example)

`./gusher.cluster start`

## Client Connect

`GET /ws/{app_key}?auth={auth}`

## Client Request Protocol

Subscribe Command:

command:
```
gusher:subscribe:{id}:{channel:'xxx'}
```

reply scuess:
```
gushe_internal:subscribe_scuessed:{id}:{channel:'xxx'}
```
reply error:
```
gusher_internal:subscribe_error:{id}:{channel:'xxx'}
```

Unsubscribe Command:

command:
```
gusher:unsubscribe:{id}:{channel:'xxx'}
```

reply scuess:
```
gusher_internal:unsubscribe_scuessed:{id}:{channel:'xxx'}
```

reply error:
```
gusher_internal:unsubscribe_error:{id}:{channel:'xxx'}
```

## WebHook Response Protocol

```
{
    "user_id":"BlackJack....",
    "channel":["channel1","channel2"...]
}
```

## Admin Protocol

Use Redis Hashes to stored

WebHook Storage 

Key|field|value
---|---|---
{app_key}|url|http://hook-domain/


Push schema

```
gusher:backpush:{ channel: 'xxx', data: {}}
```



## TODO

* gusher api implement
* <del>support gracefulll shutdown</del>
* <del>decide auth pattern</del>


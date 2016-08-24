# Gusher.Cluster

 [gusher](https://github.com/syhlion/gusher) Plus version ,support cluster

## Requirements

* redis

## Usage

Install from source

`go get -u github.com/syhlion/gusher.cluster`

And Set .env like [example](https://github.com/syhlion/gusher.cluster/blob/master/.env.example)

```
$ ./gusher.cluster start 

```

## Client Connect

`GET /ws/{app_key}?auth={auth}`

## Client Protocol

Subscribe Command

```
{
 "action":"Sub",
 "content":["Channel1","Channel2"]
}
```

Unsubscribe Command

```
{
 "action":"UnSub",
 "content":["Channel1","Channel2"...]
}
```

## WebHook Protocol

```
{
    "user_id":"BlackJack....",
    "channel":["channel1","channel2"...]
}
``




## TODO

* gusher api implement
* <del>support gracefulll shutdown</del>
* <del>decide auth pattern</del>


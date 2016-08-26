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
    channel:"{channel1}",
    event:"{custom event}",
    data:{
        data:{},
    }
}
```

#### Subscribe Command:

command:
```
{
    event:"gusher.subscribe",
    data:{
        id:"{customId}"
        channel:"{channel}",
    }
}
```

reply scuess:
```
{
    event:"gusher.subscribe_succeeded",
    data:{
        id:"{customId}"
        channel:"{channel}",
    }
}
```
reply error:
```
{
    event:"gusher.subscribe_error",
    data:{
        id:"{customId}"
        channel:"{channel}",
    }
}
```

#### Unsubscribe Command:

command:
```
{
    event:"gusher.unsubscribe",
    data:{
        id:"{customId}"
        channel:"{channel}",
    }
}
```

reply scuess:
```
{
    event:"gusher.unsubscribe_succeeded",
    data:{
        id:"{customId}"
        channel:"{channel}",
    }
}
```

reply error:
```
{
    event:"gusher.unsubscribe_error",
    data:{
        id:"{customId}"
        channel:"{channel}",
    }
}
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


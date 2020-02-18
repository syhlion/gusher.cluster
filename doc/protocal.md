
## Client Request Protocol

### Event list:

Events|Discription
---|---
gusher.ping|ping event
gusher.pong_succeeded|pong success event
gusher.remote|remote event
gusher.remote_succeeded|remote sucess event
gusher.remote_error|remote error event
gusher.remote|remote event
gusher.subscribe|subscribe event
gusher.multi_subscribe|subscribe event
gusher.unsubscribe|unsubscribe event
gusher.subscribe_succeeded|subscribe sucess
gusher.multi_subscribe_succeeded|subscribe sucess
gusher.subscribe_error|subscribe error
gusher.multi_subscribe_error|subscribe error
gusher.unsubscribe_succeeded|unsubscribe sucess
gusher.unsubscribe_error|unsubscribe error
gusher.querychannel|query channel event
gusher.querychannel_succeeded|query channel sucess
gusher.unsubscribe_error| query channel error

#### Common Receive Message:

```
{
    "channel":"",
    "event":"",
    "data":
}
```

#### Query Channel Command:

command:
```
{
    "event":"gusher.querychannel",
    "data":{}
}
```

reply scuess:
```
{
    "event":"gusher.querychannel_succeeded",
    "data":{
        "channels":["AA","BB"]
    }
}
```

reply error:
```
{
    "event":"gusher.querychannel_error",
    "data":{}
}
```

#### Multi Subscribe Command:

command:
```
{
    "event":"gusher.multi_subscribe",
    "data":{
        "multi_channel":[]
    }
}
```

reply scuess:
```
{
    "event":"gusher.multi_subscribe_succeeded",
    "data":{
        "multi_channel":[]
    }
}
```

reply error:
```
{
    "event":"gusher.multi_subscribe_error",
    "data":{
        "multi_channel":[]
    }
}
```
#### Subscribe Command:

command:
```
{
    "event":"gusher.subscribe",
    "data":{
        "channel":""
    }
}
```

reply scuess:
```
{
    "event":"gusher.subscribe_succeeded",
    "data":{
        "channel":""
    }
}
```

reply error:
```
{
    "event":"gusher.subscribe_error",
    "data":{
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
        "channel":""
    }
}
```

reply scuess:
```
{
    "event":"gusher.unsubscribe_succeeded",
    "data":{
        "channel":""
    }
}
```

reply error:
```
{
    "event":"gusher.unsubscribe_error",
    "data":{
        "channel":"",
    }
}
```

#### Remote Command:

command:
```
{
    "event":"gusher.remote",
    "data":{
        "remote":"chat"
        "uid":"",
        "payload":{"msg":1},
    }
}
```

reply scuess:
```
{
    "event":"gusher.remote_succeeded",
    "data":{
        "remote":"test"
        "uid":"",
        "payload":{},
    }
}
```

reply error:
```
{
    "event":"gusher.remote_error",
    "data":{
        "remote":"test",
        "uid":"",
        "payload":{}
    }
}
```

this command use redis RPUSH {app_key}@{remote} {data}

data:
```
{
    "user_id":"Test_User",
    "uid":"abc",
    "socket_id":"abc-bbb-ccc-ddd-eee", //unique socket id
    "app_key":"TEST",
    "data":""
}
```


#### Ping Command:
command:
```
{
    "event":"gusher.ping",
    "data":{
        //custom
    }
}
```

reply :
```
{
    "event":"gusher.pong_succeeded",
    "data":{
        //custom
    }
}
```


## JWT Protocol

this is default test jwt look [this](https://github.com/syhlion/gusher.cluster/blob/master/test/jwt/jwt.go)
```
{
    "gusher":{
        "user_id":"Test_User",
        "channels":["AA","BB"],
        "app_key":"TEST"
        "remotes":{
            "cmd1":true,
            "cmd2":true
        }
        
    }
}
```



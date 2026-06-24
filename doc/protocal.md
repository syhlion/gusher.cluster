
## Client Request Protocol

### Event list:

Events|Discription
---|---
gusher.ping|ping event
gusher.pong_succeeded|pong success event
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
    "socket_id":"cd19cdaa-44f1-11eb-80c2-784f43873ba3",
    "data":{
        "channels":["AA","BB"]
    }
}
```

reply error:
```
{
    "event":"gusher.querychannel_error",
    "socket_id":"cd19cdaa-44f1-11eb-80c2-784f43873ba3",
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
    "socket_id":"cd19cdaa-44f1-11eb-80c2-784f43873ba3",
    "data":{
        "multi_channel":[]
    }
}
```

reply error:
```
{
    "event":"gusher.multi_subscribe_error",
    "socket_id":"cd19cdaa-44f1-11eb-80c2-784f43873ba3",
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
    "socket_id":"cd19cdaa-44f1-11eb-80c2-784f43873ba3",
    "data":{
        "channel":""
    }
}
```

reply error:
```
{
    "event":"gusher.subscribe_error",
    "socket_id":"cd19cdaa-44f1-11eb-80c2-784f43873ba3",
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
    "socket_id":"cd19cdaa-44f1-11eb-80c2-784f43873ba3",
    "data":{
        "channel":""
    }
}
```

reply error:
```
{
    "event":"gusher.unsubscribe_error",
    "socket_id":"cd19cdaa-44f1-11eb-80c2-784f43873ba3",
    "data":{
        "channel":"",
    }
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
    "socket_id":"cd19cdaa-44f1-11eb-80c2-784f43873ba3",
    "data":{
        //custom
    }
}
```


## JWT Protocol

The JWT carries a `gusher` claim, signed RS256. Generate a test token with
[`test/jwtgenerate`](https://github.com/syhlion/gusher.cluster/tree/master/test/jwtgenerate).

```
{
    "gusher":{
        "user_id":"Test_User",
        "channels":["AA","BB"],
        "app_key":"TEST"
    }
}
```



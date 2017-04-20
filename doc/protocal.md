
## Client Request Protocol

### Event list:

Events|Discription
---|---
gusher.remote|remote event
gusher.remote_succeeded|remote sucess event
gusher.remote_error|remote error event
gusher.remote|remote event
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
    "app_key":"TEST",
    "data":""
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



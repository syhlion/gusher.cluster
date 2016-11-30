
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

## JWT Protocol

this is default test jwt look [this](https://github.com/syhlion/gusher.cluster/blob/master/test/jwt/jwt.go)
```
{
    "gusher":{
        "user_id":"Test_User",
        "channels":["AA","BB"],
        "app_key":"TEST"
    }
}
```



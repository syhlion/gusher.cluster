
## Client Request Protocol

### Event list:

Events|Discription
---|---
gusher.login|login event
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

#### Login Command:

command:
```
{
    "event":"gusher.login",
    "data":{
        "jwt":""
    }
}
```

jwt [ref](https://jwt.io)

jwt [example](https://github.com/syhlion/gusher.cluster/blob/master/jwt.example)

example payload decode
```
{
    "user_id":"Test_User",
    "channels":["AA","BB"],
    "app_key":"TEST"
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
    "user_id":"Test_User",
    "channels":["AA","BB"],
    "app_key":"TEST"
}
```


## Admin Protocol

Use Redis Hashes to stored

WebHook Storage 

Key|field|value
---|---|---
{app_key}|url|http://hook-domain/
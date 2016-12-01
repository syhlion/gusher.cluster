## Slave  Api

#### Auth:

`POST /{prefix}/auth`

fields: jwt={jwt}

Success Response:
```
{
    "token":""
}
```

jwt [ref](https://jwt.io)

jwt [example](https://github.com/syhlion/gusher.cluster/blob/master/jwt.example)


#### Connect:

`GET /{prefix}/ws/{app_key}?token={token}`


## Master Api


### Push Message:

`POST /{api}/push/{app_key}/{channel}/{event}?data={data}`


Sucess Response:

```
{
    "channel":"",
    "event":"",
    "data":""
}
```


### Decode:

`POST /{api}/decode?data={jwt}`


Sucess Response:

```
{
    "gusher":{
        "channels":[],
        "user_id":"",
        "app_key":""
    }
}
```








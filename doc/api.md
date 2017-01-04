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

`POST /{api}/push/{app_key}/{channel}/{event}`

|key|value|description|
|----|----|----|
|data|{"key":"value"}|string or json|

Sucess Response:

```
{
    "channel":"",
    "event":"",
    "data":""
}
```

`POST /{api}/push_batch/{app_key}?data={data}`

|key|value|description|
|----|----|----|
|batch|[{"channel":"public","event":"notify","data":"test"},{"channel":"public","event":"notify","data":{"username":"test"}}]|json|


Sucess Response:

```
{
    "total":2,
    "cap":102456 //byte
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








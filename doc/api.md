## Slave  Api

### Auth:

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


### Connect:

`GET /{prefix}/ws/{app_key}?token={token}`

### Ping:

`GET /{prefix}/ping`

Success Response:

```
pong
```

## Master Api


### Get Channels:

`GET /{app_key}/channels`
Sucess Response:

```

["channel1","channel2","channel3"...]

```

### Get Online:

`GET /{app_key}/online`
Sucess Response:

```

["user_id","user_id","user_id"...]

```

### Get Online by channel:

`GET /{app_key}/online/{channel}`
Sucess Response:

```

["user_id","user_id","user_id"...]

```

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

`POST /{api}/push_batch/{app_key}`

|key|value|description|
|----|----|----|
|batch_data|[{"channel":"public","event":"notify","data":"test"},{"channel":"public","event":"notify","data":{"username":"test"}}]|json|


Sucess Response:

```
{
    "total":2,
    "cap":102456 //byte
}
```

### Push Message to User:

`POST /{api}/push/user/{app_key}/{user_id}`

|key|value|description|
|----|----|----|
|data|{"key":"value"}|string or json|

Sucess Response:

```
{
    "user_id":"",
    "data":""
}
```

### Push Message to Socket:

`POST /{api}/push/socket/{app_key}/{socket_id}`

|key|value|description|
|----|----|----|
|data|{"key":"value"}|string or json|

Sucess Response:

```
{
    "socket_id":"",
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
        "app_key":"",
        "remotes":{
            "cmd1":true,
            "cmd2":true
        }
    }
}
```

### Ping:

`GET /{prefix}/ping`

Success Response:

```
pong
```

* note1: if channels slice have "*" char that user can sub all channels
* note2: support *  like test = t*st or app* = apple








## Slave  Api

Connect:

`GET /{ws}/{app_key}?auth={auth}`

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

`GET /{api}/decode`


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








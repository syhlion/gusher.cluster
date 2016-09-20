## Slave  Api

Connect:

`GET /ws/{app_key}?auth={auth}`

## Master Api


### Push Message:

`POST /api/push/{app_key}/{channel}/{event}?data={data}`


Sucess Response:

```
{
    "channel":"",
    "event":"",
    "data":""
}
```


### Slave Server Info:

`GET /api/system/slaveinfos`


Sucess Response:

```
{
    "{ip}@{listen_port}":{
        "ip":"",
        "local_listen":"",
        "version":"",
        "runtime_version":"",
        "cpu":,
        "usage-memory":,
        "goroutines":6,
        "connections":,
        "send_interval":"",
        "update_time":
    }
}
```








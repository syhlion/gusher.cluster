## Slave  Api

Connect:

`GET /ws/{app_key}?auth={auth}`

## Master Api

### Check app_key exist:

`GET /api/exist/{app_key}`

Sucess Response:

```
{
    "app_key":""
}
```

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

### Query AppKey:

`GET /api/query/{app_key}`


Sucess Response:
```
{
    "app_key":"",
    "url":""
}
```

### Register AppKey:

`POST /api/register/{app_key}?url={url}`

Sucess Response:

```
{
    "app_key":"",
    "url":""
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








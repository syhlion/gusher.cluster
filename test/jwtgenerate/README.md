# jwtgenerate


## Install

```
$ go get -u github.com/syhlion/gusher.cluster/test/jwtgenerate
```

## Usage

Crypto method use RSA256

```
$ ./jwtgenerate start --private-key private.pem  --payload "{\"gusher\":{\"user_id\":\"Test_User\",\"channels\":[\"AA\",\"BB\"],\"app_key\":\"TEST\"}}"

```

--payload has default

default value
```
{
    "gusher":{
        "user_id":"Test_User",
        "channesl":["AA","BB"],
        "app_key":"TEST"
    }
}
```



# restful web service client

restful web service reqeust tool

## Requirements

* [requestwork](https://github.com/syhlion/requestwork.v2)


## Install

`go get github.com/syhlion/restclient`


## Usage

```
func main(){
    worker:=requestwork.New(50)
    client:=restclient.New(worker,15*time.Second)
    data,httpstatus,err:=client.Get("https://tw.yahoo.com",nil)
}
```

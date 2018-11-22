# restful web service client

[![Go Report Card](https://goreportcard.com/badge/github.com/syhlion/greq)](https://goreportcard.com/report/github.com/syhlion/greq)
[![Build Status](https://travis-ci.org/syhlion/greq.svg?branch=master)](https://travis-ci.org/syhlion/greq)

restful web service reqeust tool

## Requirements

* [requestwork](https://github.com/syhlion/requestwork.v2)


## Install

`go get github.com/syhlion/greq`


## Usage

```
func main(){

    //need import https://github.com/syhlion/requestwork.v2
    worker:=requestwork.New(50)

    client:=greq.New(worker,15*time.Second)

    //GET
    data,httpstatus,err:=client.Get("https://tw.yahoo.com",nil)

    //POST
    v := url.Values{}
    v.Add("data", string(data))
    data,httpstatus,err:=client.Post("https://tw.yahoo.com",bytes.NewBufferString(v.Encode()))

}
```

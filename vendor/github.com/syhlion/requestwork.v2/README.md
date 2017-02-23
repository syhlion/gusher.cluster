# requestwork
[![Build Status](https://travis-ci.org/syhlion/requestwork.svg?branch=master)](https://travis-ci.org/syhlion/requestwork)

a lib for go to batch processing send web request

## Install

`go get github.com/syhlion/requestwork.v2`

### Usage

```

func main() {

    // Init request
    resq, err := http.NewRequest("GET", "http://tw.yahoo.com", nil)
    if err != nil {
        panic(err)
    }
    resp,err:=worker:=requestwork.New(resq)

    if err != nil {
        panic(err)
    }

    defer resp.Body.Close()
    fmt.Println("end")
}

```

# requestwork

[![Go Report Card](https://goreportcard.com/badge/github.com/syhlion/requestwork.v2)](https://goreportcard.com/report/github.com/syhlion/requestwork.v2)
[![Build Status](https://drone.syhlion.tw/api/badges/syhlion/requestwork.v2/status.svg)](https://drone.syhlion.tw/syhlion/requestwork.v2)

a lib for go to batch processing send web request

## Install

`go get github.com/syhlion/requestwork.v2`

### Usage

```

func main() {

    // Init request
	req, err := http.NewRequest("GET", "http://tw.yahoo.com", nil)
	if err != nil {
		t.Error("request error: ", err)
	}

	// Init worker
	a := New(5)
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	err = a.Execute(ctx, req, func(resp *http.Response, err error) error {

		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return nil

	})
}

```

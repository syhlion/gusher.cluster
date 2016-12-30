# httplog middleware

[negroni](https://github.com/urfave/negroni) middleware


## Install

`go get -u github.com/syhlion/httplog`


## Example

```
package main

import (
  "fmt"
  "net/http"

  "github.com/urfave/negroni"
)

func main() {
  mux := http.NewServeMux()
  mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
    fmt.Fprintf(w, "Welcome to the home page!")
  })

  n := negroni.New()
  n.Use(httplog.NewLogger())
  n.UseHandler(mux)

  http.ListenAndServe(":3004", n)
}
```

Will print a log similar to:

```

[http log] 127.0.0.1:46945 - [2015-12-30 09:05:27.502464588 +0800 CST] "POST /register" 200 37.96608ms  "username=syhlion&email=xxx@gmail.com"
[http log] 127.0.0.1:46947 - [2015-12-30 10:06:29.502464588 +0800 CST] "GET /feeds" 200 37.96608ms  ""
[http log] 127.0.0.1:46956 - [2015-12-31 12:15:33.502464588 +0800 CST] "GET /news" 200 37.96608ms  "offset=0&limit=10"

```



package greq

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	requestwork "github.com/syhlion/requestwork.v2"
)

//New return http client
func New(worker *requestwork.Worker, timeout time.Duration) *Client {
	return &Client{
		Worker:  worker,
		Timeout: timeout,
		Headers: make(map[string]string),
	}
}

//Client instance
type Client struct {
	Worker  *requestwork.Worker
	Timeout time.Duration
	Headers map[string]string
}

//SetHeader set http header
func (c *Client) SetHeader(key, value string) *Client {
	key = strings.Title(key)
	c.Headers[key] = value
	return c
}

//Get http method get
func (c *Client) Get(url string, params url.Values) (data []byte, httpstatus int, err error) {
	if params != nil {
		url += "?" + params.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	return c.resolveRequest(req, err)

}

//Post http method post
func (c *Client) Post(url string, params url.Values) (data []byte, httpstatus int, err error) {
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(params.Encode()))
	return c.resolveRequest(req, err)
}

//Put http method put
func (c *Client) Put(url string, params url.Values) (data []byte, httpstatus int, err error) {
	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(params.Encode()))
	return c.resolveRequest(req, err)
}

//Delete http method Delete
func (c *Client) Delete(url string, params url.Values) (data []byte, httpstatus int, err error) {
	req, err := http.NewRequest(http.MethodDelete, url, strings.NewReader(params.Encode()))
	return c.resolveRequest(req, err)
}

func (c *Client) resolveHeaders(req *http.Request) {
	for key, value := range c.Headers {
		req.Header.Set(key, value)
	}
}

func (c *Client) resolveRequest(req *http.Request, e error) (data []byte, httpstatus int, err error) {
	var (
		body   []byte
		status int
	)
	if e != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)

	defer cancel()
	c.resolveHeaders(req)

	switch req.Method {
	case "PUT", "POST", "DELETE":
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	}

	err = c.Worker.Execute(ctx, req, func(resp *http.Response, err error) (e error) {
		if err != nil {
			return
		}
		defer resp.Body.Close()
		status = resp.StatusCode
		body, e = ioutil.ReadAll(resp.Body)
		if e != nil {
			return
		}
		return
	})
	if err != nil {
		cancel()
		return
	}
	data = body
	httpstatus = status
	return

}

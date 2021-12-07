package greq

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	requestwork "github.com/syhlion/requestwork.v2"
)

type Trace struct {
	Url              string        `json:"url"`
	Method           string        `json:"method"`
	Body             string        `json:"body"`
	Param            string        `json:"param"`
	DNSLookup        time.Duration `json:"dns_lookup"`
	TCPConnection    time.Duration `json:"tcp_connection"`
	TLSHandshake     time.Duration `json:"tls_handshake"`
	ServerProcessing time.Duration `json:"server_prcoessing"`
	ContentTransfer  time.Duration `json:"content_transfer"`
	NameLookup       time.Duration `json:"name_lookup"`
	Connect          time.Duration `json:"connect"`
	PreTransfer      time.Duration `json:"pre_transfer"`
	StartTransfer    time.Duration `json:"start_transfer"`
	Total            time.Duration `json:"total"`
}

var ip string

func getExternalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}

func init() {
	log.SetFormatter(&log.JSONFormatter{})

	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	ip, _ = getExternalIP()
}

//New return http client
func New(worker *requestwork.Worker, timeout time.Duration, debug bool) *Client {
	return &Client{
		worker:  worker,
		timeout: timeout,
		headers: make(map[string]string),
		lock:    &sync.RWMutex{},
		debug:   debug,
	}
}

//Client instance
type Client struct {
	worker  *requestwork.Worker
	timeout time.Duration
	headers map[string]string
	host    string
	lock    *sync.RWMutex
	debug   bool
}

func (c *Client) CheckRedircet(f func(req *http.Request, via []*http.Request) error) {
	c.worker.CheckRedirect(f)
}

//SetBasicAuth  set Basic auth
func (c *Client) SetBasicAuth(username, password string) *Client {
	auth := username + ":" + password
	hash := base64.StdEncoding.EncodeToString([]byte(auth))
	c.lock.Lock()
	defer c.lock.Unlock()
	c.headers["Authorization"] = "Basic " + hash
	return c
}

//SetHeader set http header
func (c *Client) SetHeader(key, value string) *Client {
	key = strings.Title(key)
	c.lock.Lock()
	defer c.lock.Unlock()
	c.headers[key] = value
	return c
}

//SetHost set host
func (c *Client) SetHost(host string) *Client {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.host = host
	return c
}

//Get http method get
func (c *Client) Get(url string, params url.Values) (data []byte, httpstatus int, err error) {
	if params != nil {
		url += "?" + params.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	return c.resolveRequest(req, params, err)

}
func (c *Client) GetWithOnceHeader(url string, params url.Values, headers map[string]string) (data []byte, httpstatus int, err error) {
	if params != nil {
		url += "?" + params.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return c.resolveRequest(req, params, err)
}

//Post http method post
func (c *Client) Post(url string, params url.Values) (data []byte, httpstatus int, err error) {
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(params.Encode()))
	return c.resolveRequest(req, params, err)
}
func (c *Client) PostRaw(url string, body io.Reader) (data []byte, httpstatus int, err error) {
	req, err := http.NewRequest(http.MethodPost, url, body)
	return c.resolveRawRequest(req, body, err)
}
func (c *Client) PostRawWithOnceHeader(url string, body io.Reader, headers map[string]string) (data []byte, httpstatus int, err error) {
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return c.resolveRawRequest(req, body, err)
}

func (c *Client) PostWithOnceHeader(url string, params url.Values, headers map[string]string) (data []byte, httpstatus int, err error) {
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(params.Encode()))
	if err != nil {
		return
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return c.resolveRequest(req, params, err)
}

//Put http method put
func (c *Client) Put(url string, params url.Values) (data []byte, httpstatus int, err error) {
	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(params.Encode()))
	return c.resolveRequest(req, params, err)
}
func (c *Client) PutRaw(url string, body io.Reader) (data []byte, httpstatus int, err error) {
	req, err := http.NewRequest(http.MethodPut, url, body)
	return c.resolveRawRequest(req, body, err)
}
func (c *Client) PutWithOnceHeader(url string, params url.Values, headers map[string]string) (data []byte, httpstatus int, err error) {
	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(params.Encode()))
	if err != nil {
		return
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return c.resolveRequest(req, params, err)
}
func (c *Client) PutRawWithOnceHeader(url string, body io.Reader, headers map[string]string) (data []byte, httpstatus int, err error) {
	req, err := http.NewRequest(http.MethodPut, url, body)
	if err != nil {
		return
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return c.resolveRawRequest(req, body, err)
}

//Delete http method Delete
func (c *Client) Delete(url string, params url.Values) (data []byte, httpstatus int, err error) {
	req, err := http.NewRequest(http.MethodDelete, url, strings.NewReader(params.Encode()))
	return c.resolveRequest(req, params, err)
}
func (c *Client) DeleteRaw(url string, body io.Reader) (data []byte, httpstatus int, err error) {
	req, err := http.NewRequest(http.MethodDelete, url, body)
	return c.resolveRawRequest(req, body, err)
}
func (c *Client) DeleteWithOnceHeader(url string, params url.Values, headers map[string]string) (data []byte, httpstatus int, err error) {
	req, err := http.NewRequest(http.MethodDelete, url, strings.NewReader(params.Encode()))
	if err != nil {
		return
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return c.resolveRequest(req, params, err)
}
func (c *Client) DeleteRawWithOnceHeader(url string, body io.Reader, headers map[string]string) (data []byte, httpstatus int, err error) {
	req, err := http.NewRequest(http.MethodDelete, url, body)
	if err != nil {
		return
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return c.resolveRawRequest(req, body, err)
}

func (c *Client) resolveHeaders(req *http.Request) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}
	if c.host != "" {
		req.Host = c.host
	}
}

func (c *Client) resolveRawRequest(req *http.Request, bb io.Reader, e error) (data []byte, httpstatus int, err error) {
	var (
		body           []byte
		status         int
		trace          *httptrace.ClientTrace
		t0, t3, t4, t5 time.Time
	)
	req.Close = true
	t0 = time.Now()
	if c.debug {
		var stat Trace
		defer func() {
			buf := new(bytes.Buffer)
			buf.ReadFrom(bb)
			stat.Param = buf.String()
			stat.Url = req.URL.String()
			stat.Method = req.Method
			stat.Body = string(data)
			stat.TCPConnection = t3.Sub(t0)
			stat.ServerProcessing = t4.Sub(t3)
			stat.ContentTransfer = t5.Sub(t4)
			stat.Connect = t3.Sub(t0)
			stat.StartTransfer = t4.Sub(t0)
			stat.Total = t5.Sub(t0)
			log.WithFields(log.Fields{
				"ip":     ip,
				"name":   "syhlion/greq",
				"param":  stat.Param,
				"url":    stat.Url,
				"method": stat.Method,
				//	"body":              stat.Body,
				"tcp_connection":    stat.TCPConnection.String(),
				"server_processing": stat.ServerProcessing.String(),
				"content_transfer":  stat.ContentTransfer.String(),
				"connect":           stat.Connect.String(),
				"start_transfer":    stat.StartTransfer.String(),
				"total":             stat.Total.String(),
			}).Debug("http trace")

		}()
		trace = &httptrace.ClientTrace{
			GotConn:              func(_ httptrace.GotConnInfo) { t3 = time.Now() },
			GotFirstResponseByte: func() { t4 = time.Now() },
		}
		req = req.WithContext(httptrace.WithClientTrace(context.Background(), trace))
	}
	if e != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)

	defer cancel()
	c.resolveHeaders(req)
	switch req.Method {
	case "PUT", "POST", "DELETE":
		tmp := req.Header.Get("Content-Type")
		if tmp == "" {
			req.Header.Set("Content-Type", "application/json")
		}
	}
	req = req.WithContext(ctx)

	err = c.worker.Execute(req, func(resp *http.Response, err error) (er error) {
		if err != nil {
			return err
		}
		var readErr error
		defer func() {
			resp.Body.Close()
			t5 = time.Now()
		}()
		status = resp.StatusCode
		switch resp.Header.Get("Content-Encoding") {
		case "gzip":
			reader, err := gzip.NewReader(resp.Body)
			if err != nil {
				return err
			}
			defer reader.Close()
			body, readErr = ioutil.ReadAll(reader)
			if readErr != nil {
				return readErr
			}
		default:
			body, readErr = ioutil.ReadAll(resp.Body)
			if readErr != nil {
				return readErr
			}
		}
		return
	})
	if err != nil {
		return
	}
	data = body
	httpstatus = status
	return

}
func (c *Client) resolveRequest(req *http.Request, params url.Values, e error) (data []byte, httpstatus int, err error) {
	var (
		body           []byte
		status         int
		trace          *httptrace.ClientTrace
		t0, t3, t4, t5 time.Time
	)
	req.Close = true
	t0 = time.Now()
	if c.debug {
		var stat Trace
		defer func() {
			stat.Param = params.Encode()
			stat.Url = req.URL.String()
			stat.Method = req.Method
			stat.Body = string(data)
			stat.TCPConnection = t3.Sub(t0)
			stat.ServerProcessing = t4.Sub(t3)
			stat.ContentTransfer = t5.Sub(t4)
			stat.Connect = t3.Sub(t0)
			stat.StartTransfer = t4.Sub(t0)
			stat.Total = t5.Sub(t0)
			log.WithFields(log.Fields{
				"ip":     ip,
				"name":   "syhlion/greq",
				"param":  stat.Param,
				"url":    stat.Url,
				"method": stat.Method,
				//	"body":              stat.Body,
				"tcp_connection":    stat.TCPConnection.String(),
				"server_processing": stat.ServerProcessing.String(),
				"content_transfer":  stat.ContentTransfer.String(),
				"connect":           stat.Connect.String(),
				"start_transfer":    stat.StartTransfer.String(),
				"total":             stat.Total.String(),
			}).Debug("http trace")

		}()
		trace = &httptrace.ClientTrace{
			GotConn:              func(_ httptrace.GotConnInfo) { t3 = time.Now() },
			GotFirstResponseByte: func() { t4 = time.Now() },
		}
		req = req.WithContext(httptrace.WithClientTrace(context.Background(), trace))
	}
	if e != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)

	defer cancel()
	c.resolveHeaders(req)

	switch req.Method {
	case "PUT", "POST", "DELETE":
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	}
	req = req.WithContext(ctx)

	err = c.worker.Execute(req, func(resp *http.Response, err error) (er error) {
		if err != nil {
			return err
		}
		var readErr error
		defer func() {
			resp.Body.Close()
			t5 = time.Now()
		}()
		status = resp.StatusCode
		switch resp.Header.Get("Content-Encoding") {
		case "gzip":
			reader, err := gzip.NewReader(resp.Body)
			if err != nil {
				return err
			}
			defer reader.Close()
			body, readErr = ioutil.ReadAll(reader)
			if readErr != nil {
				return readErr
			}
		default:
			body, readErr = ioutil.ReadAll(resp.Body)
			if readErr != nil {
				return readErr
			}
		}
		return
	})
	if err != nil {
		return
	}
	data = body
	httpstatus = status
	return

}

func (c *Client) ResolveRequest(req *http.Request, params url.Values, e error) (data []byte, httpstatus int, err error) {
	return c.resolveRequest(req, params, err)
}

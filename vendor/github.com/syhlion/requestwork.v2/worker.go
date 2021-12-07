package requestwork

import (
	"net"
	"net/http"
	"net/url"
	"time"
)

type job struct {
	req     *http.Request
	handler func(resp *http.Response, err error) error

	end chan error
}

type result struct {
	resp *http.Response
	err  error
}

//DefaultMaxIdleConnPerHost max idle
const DefaultMaxIdleConnPerHost = 20

//New return http worker
func New(threads int) *Worker {
	tr := &http.Transport{
		Proxy: NoProxyAllowed,
		Dial: (&net.Dialer{
			Timeout:   60 * time.Second,
			KeepAlive: 60 * time.Second,
		}).Dial,
		DisableKeepAlives:     true,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   60 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   time.Second * 120,
	}
	w := &Worker{
		jobQuene: make(chan *job),
		threads:  threads,
		client:   client,
	}

	go w.start()
	return w

}

//NoProxyAllowed no proxy
func NoProxyAllowed(request *http.Request) (*url.URL, error) {
	return nil, nil
}

//Worker instance
type Worker struct {
	jobQuene chan *job
	threads  int
	client   *http.Client
}

func (w *Worker) SetTransport(tr *http.Transport) {
	w.client.Transport = tr
}

func (w *Worker) CheckRedirect(f func(req *http.Request, via []*http.Request) error) {
	w.client.CheckRedirect = f
}

//Execute exec http request
func (w *Worker) Execute(req *http.Request, h func(resp *http.Response, err error) error) (err error) {

	j := &job{req, h, make(chan error)}
	w.jobQuene <- j
	return <-j.end

}

func (w *Worker) run() {
	for j := range w.jobQuene {
		c := make(chan error, 1)
		go func() {
			c <- j.handler(w.client.Do(j.req))
		}()
		select {
		case <-j.req.Context().Done():

			j.end <- j.req.Context().Err()
		case err := <-c:
			j.end <- err
		}
	}

}

func (w *Worker) start() {

	for i := 0; i < w.threads; i++ {
		go w.run()
	}

}

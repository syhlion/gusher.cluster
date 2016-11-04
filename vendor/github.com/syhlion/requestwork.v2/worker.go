package requestwork

import (
	"context"
	"net/http"
	"net/url"
)

type job struct {
	ctx context.Context
	req *http.Request
	h   func(resp *http.Response, err error) error
	end chan error
}

type result struct {
	resp *http.Response
	err  error
}

const DefaultMaxIdleConnPerHost = 20

func New(threads int) *Worker {

	w := &Worker{
		jobQuene: make(chan *job),
		threads:  threads,
	}

	go w.start()
	return w

}

func NoProxyAllowed(request *http.Request) (*url.URL, error) {
	return nil, nil
}

type Worker struct {
	jobQuene chan *job
	threads  int
}

func (w *Worker) Execute(ctx context.Context, req *http.Request, h func(resp *http.Response, err error) error) (err error) {

	j := &job{ctx, req, h, make(chan error)}
	w.jobQuene <- j
	return <-j.end

}

func (w *Worker) run() {
	tr := &http.Transport{
		Proxy:               NoProxyAllowed,
		Dial:                Dial,
		MaxIdleConnsPerHost: w.threads * DefaultMaxIdleConnPerHost,
	}
	client := &http.Client{
		Transport: tr,
	}
	for j := range w.jobQuene {
		c := make(chan error, 1)
		go func() {
			c <- j.h(client.Do(j.req))
		}()
		select {
		case <-j.ctx.Done():
			tr.CancelRequest(j.req)
			j.end <- j.ctx.Err()
			close(j.end)
		case err := <-c:
			j.end <- err
			close(j.end)
		}
	}

}

func (w *Worker) start() {

	for i := 0; i < w.threads; i++ {
		go w.run()
	}

}

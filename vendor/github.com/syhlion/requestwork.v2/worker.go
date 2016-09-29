package requestwork

import (
	"context"
	"net/http"
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

func New(threads int) *Worker {

	w := &Worker{
		jobQuene: make(chan *job),
		threads:  threads,
	}
	go w.start()
	return w

}

type Worker struct {
	jobQuene   chan *job
	threads    int
	httpClient *http.Client
}

func (w *Worker) Execute(ctx context.Context, req *http.Request, h func(resp *http.Response, err error) error) (err error) {

	j := &job{ctx, req, h, make(chan error)}
	w.jobQuene <- j
	return <-j.end

}

func (w *Worker) run() {
	for j := range w.jobQuene {
		tr := &http.Transport{}
		client := &http.Client{Transport: tr}
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

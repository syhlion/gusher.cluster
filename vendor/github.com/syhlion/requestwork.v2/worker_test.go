package requestwork

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestExecute(t *testing.T) {
	req, err := http.NewRequest("GET", "http://tw.yahoo.com", nil)
	if err != nil {
		t.Error("request error: ", err)
	}
	a := New(5)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = a.Execute(ctx, req, func(resp *http.Response, err error) error {

		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return nil

	})
	if err != nil {
		t.Error(err)
		return
	}
	err = a.Execute(context.Background(), req, func(resp *http.Response, err error) error {

		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return nil

	})
	if err != nil {
		t.Error(err)
	}

}

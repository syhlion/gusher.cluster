package greq

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	requestwork "github.com/syhlion/requestwork.v2"
)

var worker *requestwork.Worker

func postHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("method error"))
		return
	}
	a := r.FormValue("key")

	if a != "TEST_HELLO" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("param error ,request:" + a))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("success"))
	return

}
func getHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("method error"))
		return
	}

	a := r.FormValue("key")
	if a != "TEST_HELLO" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("param error ,request:" + a))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("success"))
	return

}
func putHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("method error"))
		return
	}
	a := r.FormValue("key")
	if a != "TEST_HELLO" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("param error ,request:" + a))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("success"))
	return

}
func deleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("method error"))
		return
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("read body error: " + err.Error()))
		return
	}
	v, err := url.ParseQuery(string(b))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("parser query error: " + err.Error()))
		return
	}
	a := v.Get("key")
	if a != "TEST_HELLO" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("param error ,request:" + a))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("success"))
	return

}
func init() {
	worker = requestwork.New(10)
	http.HandleFunc("/get", getHandler)
	http.HandleFunc("/post", postHandler)
	http.HandleFunc("/put", putHandler)
	http.HandleFunc("/delete", deleteHandler)
	go func() {
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

}

func TestGet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(getHandler))
	defer ts.Close()

	client := New(worker, 15*time.Second)
	v := url.Values{}
	v.Set("key", "TEST_HELLO")
	data, s, err := client.Get(ts.URL, v)
	if err != nil {
		t.Fatal(err)
	}
	if s != http.StatusOK {
		t.Fatalf("status fatal:%d ,body:%s", s, string(data))
	}
	if string(data) != "success" {
		t.Fatal("body fatal :", string(data))
	}

}
func TestPost(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(postHandler))
	defer ts.Close()
	client := New(worker, 15*time.Second)
	v := url.Values{}
	v.Set("key", "TEST_HELLO")
	data, s, err := client.Post(ts.URL, v)
	if err != nil {
		t.Fatal(err)
	}
	if s != http.StatusOK {
		t.Fatalf("status fatal:%d ,body:%s", s, string(data))
	}
	if string(data) != "success" {
		t.Fatal("body fatal :", string(data))
	}
}
func TestPut(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(putHandler))
	defer ts.Close()
	client := New(worker, 15*time.Second)
	v := url.Values{}
	v.Set("key", "TEST_HELLO")
	data, s, err := client.Put(ts.URL, v)
	if err != nil {
		t.Fatal(err)
	}
	if s != http.StatusOK {
		t.Fatalf("status fatal:%d ,body:%s", s, string(data))
	}
	if string(data) != "success" {
		t.Fatal("body fatal :", string(data))
	}
}
func TestDelete(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(deleteHandler))
	defer ts.Close()
	client := New(worker, 15*time.Second)
	v := url.Values{}
	v.Set("key", "TEST_HELLO")
	data, s, err := client.Delete(ts.URL, v)
	if err != nil {
		t.Fatal(err)
	}
	if s != http.StatusOK {
		t.Fatalf("status fatal:%d ,body:%s", s, string(data))
	}
	if string(data) != "success" {
		t.Fatal("body fatal :", string(data))
	}
}

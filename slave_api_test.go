package main

import (
	"testing"

	redisocket "github.com/syhlion/redisocket.v2"
)

func mockNoStarData() (a redisocket.Auth, d []byte) {
	a = redisocket.Auth{
		Channels: []string{"AA"},
		UserId:   "AAA",
		AppKey:   "TEST",
	}
	d = []byte(`{"channel":"AA"}}`)
	return
}
func mockNoMatchData() (a redisocket.Auth, d []byte) {
	a = redisocket.Auth{
		Channels: []string{"AA"},
		UserId:   "AAA",
		AppKey:   "TEST",
	}
	d = []byte(`{"channel":"BB"}}`)
	return
}
func mockAdminData() (a redisocket.Auth, d []byte) {
	a = redisocket.Auth{
		Channels: []string{"*"},
		UserId:   "AAA",
		AppKey:   "TEST",
	}
	d = []byte(`{"channel":"DD"}}`)
	return
}
func mockStarData() (a redisocket.Auth, d []byte) {
	a = redisocket.Auth{
		Channels: []string{"@^WTF*"},
		UserId:   "AAA",
		AppKey:   "TEST",
	}
	d = []byte(`{"channel":"@^WTFDD"}}`)
	return
}
func TestSubscribeCommand(t *testing.T) {
	a, d := mockNoStarData()
	m, err := SubscribeCommand("TEST", a, d, false)
	if err != nil {
		t.Errorf("%s err:%v", "nostar", err)
		return
	}
	if m.data != "AA" {
		t.Error("subscribe AA error ", m.data)
		return
	}
	a, d = mockNoMatchData()
	m, err = SubscribeCommand("TEST", a, d, false)
	if err != nil {
		t.Errorf("%s err:%v", "nostar", err)
		return
	}
	if m.cmdType != "" {
		t.Errorf("%s err", "nomatch")
		return
	}
	a, d = mockAdminData()
	m, err = SubscribeCommand("TEST", a, d, false)
	if err != nil {
		t.Errorf("%s err:%v", "admin", err)
		return
	}
	if m.data != "DD" {
		t.Error("subscribe DD error")
		return
	}
	a, d = mockStarData()
	m, err = SubscribeCommand("TEST", a, d, false)
	if m.data != "@^WTFDD" {
		t.Error("subscribe @^WTFDD error")
		return
	}
	if err != nil {
		t.Errorf("%s err:%v", "star", err)
		return
	}
}

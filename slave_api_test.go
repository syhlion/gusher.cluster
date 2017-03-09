package main

import "testing"

func mockNoStarData() (a Auth, d []byte) {
	a = Auth{
		Channels: []string{"AA"},
		UserId:   "AAA",
		AppKey:   "TEST",
	}
	d = []byte(`{"channel":"AA"}}`)
	return
}
func mockAdminData() (a Auth, d []byte) {
	a = Auth{
		Channels: []string{"*"},
		UserId:   "AAA",
		AppKey:   "TEST",
	}
	d = []byte(`{"channel":"DD"}}`)
	return
}
func mockStarData() (a Auth, d []byte) {
	a = Auth{
		Channels: []string{"@^WTF*"},
		UserId:   "AAA",
		AppKey:   "TEST",
	}
	d = []byte(`{"channel":"@^WTFDD"}}`)
	return
}

func TestSubscribeCommand(t *testing.T) {
	a, d := mockNoStarData()
	m, err := SubscribeCommand("TEST", a, d)
	if err != nil {
		t.Errorf("%s err:%v", "nostar", err)
		return
	}
	if m.Event != "AA" {
		t.Error("subscribe AA error")
		return
	}
	a, d = mockAdminData()
	m, err = SubscribeCommand("TEST", a, d)
	if err != nil {
		t.Errorf("%s err:%v", "admin", err)
		return
	}
	if m.Event != "DD" {
		t.Error("subscribe DD error")
		return
	}
	a, d = mockStarData()
	m, err = SubscribeCommand("TEST", a, d)
	if m.Event != "@^WTFDD" {
		t.Error("subscribe @^WTFDD error")
		return
	}
	if err != nil {
		t.Errorf("%s err:%v", "star", err)
		return
	}
}

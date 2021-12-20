package main

import (
	"crypto/rsa"
	"io/ioutil"
	"testing"

	"github.com/golang-jwt/jwt"
)

func makePrivateKey() (pkey *rsa.PrivateKey, err error) {
	private, err := ioutil.ReadFile("test/key/private.pem")
	if err != nil {
		return
	}
	pkey, err = jwt.ParseRSAPrivateKeyFromPEM(private)
	if err != nil {
		return
	}
	return
}
func makePublicKey() (pkey *rsa.PublicKey, err error) {
	public, err := ioutil.ReadFile("test/key/public.pem")
	if err != nil {
		return
	}
	pkey, err = jwt.ParseRSAPublicKeyFromPEM(public)
	if err != nil {
		return
	}
	return
}

func getJwtToken() (t string, err error) {
	privateKey, err := makePrivateKey()
	if err != nil {
		return
	}
	type GusherData struct {
		UserId   string   `json:"user_id"`
		Channels []string `json:"channels"`
		AppKey   string   `json:"app_key"`
	}
	type MyCustomClaims struct {
		Gusher GusherData `json:"gusher"`
		jwt.StandardClaims
	}
	gd := GusherData{
		"test", []string{"AA", "BB"}, "TEST1",
	}
	claims := MyCustomClaims{
		gd, jwt.StandardClaims{},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	t, err = token.SignedString(privateKey)
	if err != nil {
		return
	}
	return
}

func TestDecode(t *testing.T) {
	token, err := getJwtToken()
	if err != nil {
		t.Error(err)
	}
	publicKey, err := makePublicKey()
	if err != nil {
		t.Error(err)
	}
	a, err := Decode(publicKey, token)
	if err != nil {
		t.Error(err)
	}

	if a.Gusher.UserId != "test" {
		t.Errorf("user_id parse error %s", a.Gusher.UserId)
	}
	if a.Gusher.AppKey != "TEST1" {
		t.Errorf("app_key parse error %s", a.Gusher.AppKey)
	}
	if a.Gusher.Channels[0] != "AA" && a.Gusher.Channels[1] != "BB" {
		t.Errorf("channels parse error %v", a.Gusher.Channels)
	}

}

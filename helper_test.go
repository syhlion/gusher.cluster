package main

import (
	"crypto/rsa"
	"io/ioutil"
	"testing"

	"github.com/dgrijalva/jwt-go"
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
	type MyCustomClaims struct {
		UserId   string   `json:"user_id"`
		Channels []string `json:"channels"`
		AppKey   string   `json:"app_key"`
		jwt.StandardClaims
	}
	claims := MyCustomClaims{
		"test", []string{"AA", "BB"}, "TEST1", jwt.StandardClaims{},
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
	if a.UserId != "test" {
		t.Errorf("user_id parse error %s", a.UserId)
	}
	if a.AppKey != "TEST1" {
		t.Errorf("app_key parse error %s", a.AppKey)
	}
	if a.Channels[0] != "AA" && a.Channels[1] != "BB" {
		t.Errorf("channels parse error %v", a.Channels)
	}

}

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	jwt "github.com/dgrijalva/jwt-go"
)

var tokenString = ""

func main() {
	pwd, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	private, err := ioutil.ReadFile(pwd + "/test/key/private.pem")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	priKey, err := jwt.ParseRSAPrivateKeyFromPEM(private)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	type MyCustomClaims struct {
		UserId   string   `json:"user_id"`
		Channels []string `json:"channels"`
		AppKey   string   `json:"app_key"`
		jwt.StandardClaims
	}
	claims := MyCustomClaims{
		"Test_User", []string{"AA", "BB"}, "TEST", jwt.StandardClaims{},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(priKey)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(tokenString)

}

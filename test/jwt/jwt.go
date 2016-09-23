package main

import (
	"fmt"
	"io/ioutil"
	"os"

	jwt "github.com/dgrijalva/jwt-go"
)

var tokenString = ""

func main() {
	private, err := ioutil.ReadFile("../key/private.pem")
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
		channels []string `json:"channels"`
		jwt.StandardClaims
	}
	claims := MyCustomClaims{
		"TEST", []string{"AA", "BB"}, jwt.StandardClaims{},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(priKey)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(tokenString)

}

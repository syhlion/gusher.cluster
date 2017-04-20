package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/SermoDigital/jose/crypto"
	"github.com/SermoDigital/jose/jws"
	"github.com/urfave/cli"
)

var tokenString = ""

var (
	name     string
	version  string
	cmdStart = cli.Command{
		Name:    "gen",
		Aliases: []string{"g"},
		Usage:   "generate jwt token (only support RSA256)",
		Action:  start,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "payload",
				Usage: "You want to hash payload",
				Value: "{\"gusher\":{\"user_id\":\"Test_User\",\"channels\":[\"AA\",\"BB\"],\"app_key\":\"TEST\",\"remotes\":{\"cmd1\":true}}}",
			},
			cli.StringFlag{
				Name:  "private-key",
				Usage: "Assign rsa256 private key",
			},
		},
	}
	payload       string
	privateKeyPwd string
)

func start(c *cli.Context) {
	payload = c.String("payload")

	var claims map[string]interface{}
	err := json.Unmarshal([]byte(payload), &claims)
	if err != nil {
		log.Println(err, payload)
		os.Exit(1)
	}

	pwd, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	if c.String("private-key") == "" {
		privateKeyPwd = pwd + "/test/key/private.pem"
	} else {
		privateKeyPwd = c.String("private-key")
	}
	privateKey, err := ioutil.ReadFile(privateKeyPwd)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	rsaPrivate, err := crypto.ParseRSAPrivateKeyFromPEM(privateKey)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	jwt := jws.NewJWT(jws.Claims(claims), crypto.SigningMethodRS256)
	token, err := jwt.Serialize(rsaPrivate)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	fmt.Printf("\033[40;31mprivate key \033[0m : \n %s \n\n \033[40;31mpayload\033[0m: \n %s \n\n \033[40;31mtoken\033[0m: \n %s \n", privateKey, payload, token)

}

func main() {
	cli.AppHelpTemplate += "\nWEBSITE:\n\t\thttps://github.com/syhlion/gusher.cluster/tree/master/test/jwtgenerate\n\n"
	gusher := cli.NewApp()
	gusher.Author = "Scott (syhlion)"
	gusher.Usage = "simple jwt generate (only support RSA256)"
	gusher.Name = name
	gusher.Version = version
	gusher.Compiled = time.Now()
	gusher.Commands = []cli.Command{
		cmdStart,
	}
	gusher.Run(os.Args)

}

GOROOT := $(GOROOT)
GOPATH := $(GOPATH)
GO := $(GOROOT)/bin/go
PWD := $(PWD)
NAME := gusher.cluster
TAG := `git describe --tags | cut -d '-' -f 1 `.`git rev-parse --short HEAD`
#TAG := "DEV"
TZ := Asia/Taipei
DATETIME := `TZ=$(TZ) date +%Y%m%d.%H%M%S`
show-tag:
	echo $(TAG)
build: 
	go build test/jwt/jwtgenerate.go
	./jwtgenerate >> jwt
	go build -ldflags "-X main.name=$(NAME) -X main.version=$(TAG) -X main.compileDate=$(DATETIME)($(TZ)) " -a -o ./$(NAME);
run: build
	./$(NAME)
tar: build
	tar zcvf $(NAME).$(TAG).linux-amd64.tar.gz $(NAME) env.example test/key jwt
todo:
	find -type f \( -iname '*.go' ! -wholename './vendor/*' \) -exec grep -Hn 'TODO' {} \;
rsakey:
	openssl genrsa -out private.pem 2048
	openssl rsa -in private.pem -pubout -out public.pem

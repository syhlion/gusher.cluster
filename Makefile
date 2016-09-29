GOROOT := $(GOROOT)
OS:=linux-amd64
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
verify-glide:
	if [ ! -e `which glide` ] ; then\
		echo 'please install "https://github.com/Masterminds/glide"';\
		exit 1;\
	fi
build: 
	go build test/jwt/jwtgenerate.go
	./jwtgenerate > jwt.example
	go build -ldflags "-X main.name=$(NAME) -X main.version=$(TAG) -X main.compileDate=$(DATETIME)($(TZ)) " -a -o ./$(NAME);
run: build
	./$(NAME)
tar: build
	tar zcvf $(NAME).$(TAG).$(OS).tar.gz $(NAME) env.example test/key jwt.example
todo:
	find -type f \( -iname '*.go' ! -wholename './vendor/*' \) -exec grep -Hn 'TODO' {} \;
rsakey:
	openssl genrsa -out private.pem 2048
	openssl rsa -in private.pem -pubout -out public.pem

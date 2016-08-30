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
build-dev: 
	go build -ldflags "-X main.name=$(NAME) -X main.version=$(TAG) -X main.compileDate=$(DATETIME)($(TZ)) -X main.gusherDevState=DEV" -a -o ./$(NAME);
build-production: 
	go build -ldflags "-X main.name=$(NAME) -X main.version=$(TAG) -X main.compileDate=$(DATETIME)($(TZ)) -X main.gusherDevState=PRODUCTION" -a -o ./$(NAME);
run: build
	./$(NAME)
todo:
	find -type f \( -iname '*.go' ! -wholename './vendor/*' \) -exec grep -Hn 'TODO' {} \;


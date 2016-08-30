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
	go build -ldflags "-X main.name=$(NAME) -X main.version=$(TAG) -X main.compileDate=$(DATETIME)($(TZ)) " -a -o ./$(NAME);
run: build
	./$(NAME)
todo:
	find -type f \( -iname '*.go' ! -wholename './vendor/*' \) -exec grep -Hn 'TODO' {} \;


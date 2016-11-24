OS:=linux-amd64
GUSHER:= gusher.cluster
CONNTEST:= conn-test
JWTGENERATE:=jwt-generate
TAG := `git describe --tags | cut -d '-' -f 1 `.`git rev-parse --short HEAD`
show-tag:
	echo $(TAG)
verify-glide:
	if [ ! -e `which glide` ] ; then\
		echo 'please install "https://github.com/Masterminds/glide"';\
		exit 1;\
	fi
build: 
	go test
	go build -ldflags "-X main.version=$(TAG) -X main.name=$(JWTGENERATE)" -a -o $(JWTGENERATE) test/jwtgenerate/jwtgenerate.go 
	go build -ldflags "-X main.version=$(TAG) -X main.name=$(CONNTEST)" -a -o test/conn-test/$(CONNTEST) test/conn-test/conn-test.go
	./jwt-generate gen --private-key test/key/private.pem > jwt.example
	go build -ldflags "-X main.name=$(GUSHER) -X main.version=$(TAG) " -a -o ./$(GUSHER);
docker-build:
	go build -ldflags "-X main.name=$(GUSHER) -X main.version=$(TAG) " -a -o ./$(GUSHER);
run: build
	./$(NAME)
tar: build
	tar zcvf $(GUSHER).$(TAG).$(OS).tar.gz $(GUSHER) env.example LICENSE test/key jwt.example test/conn-test --exclude=test/conn-test/conn-test.go docker-compose docker
todo:
	find -type f \( -iname '*.go' ! -wholename './vendor/*' \) -exec grep -Hn 'TODO' {} \;
rsakey:
	openssl genrsa -out private.pem 2048
	openssl rsa -in private.pem -pubout -out public.pem

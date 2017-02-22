OS:=linux-amd64
GUSHER:= gusher.cluster
CONNTEST:= conn-test
JWTGENERATE:=jwt-generate
TAG := `git describe --tags | cut -d '-' -f 1 `.`git rev-parse --short HEAD`
TZ := Asia/Taipei
DATETIME := `TZ=$(TZ) date +%Y/%m/%d.%T`
show-tag:
	echo $(TAG)
verify-glide:
	if [ ! -e `which glide` ] ; then\
		echo 'please install "https://github.com/Masterminds/glide"';\
		exit 1;\
	fi
buildjwt = GOOS=$(1) GOARCH=$(2) go build -ldflags "-X main.version=$(TAG) -X main.name=$(JWTGENERATE)" -a -o build/$(JWTGENERATE)$(3) test/jwtgenerate/jwtgenerate.go
buildconntest = GOOS=$(1) GOARCH=$(2) go build -ldflags "-X main.version=$(TAG) -X main.name=$(CONNTEST)" -a -o build/$(CONNTEST)$(3) test/conn-test/conn-test.go
buildgusher = GOOS=$(1) GOARCH=$(2) go build -ldflags "-X main.version=$(TAG) -X main.name=$(GUSHER)" -a -o build/$(GUSHER)$(3) 
tar = cp env.example ./build && cp test/conn-test/conn-test.env.example ./build &&cd build && tar -zcvf $(GUSHER)_$(TAG)_$(1)_$(2).tar.gz $(JWTGENERATE)$(3) $(CONNTEST)$(3) $(GUSHER)$(3) env.example conn-test.env.example  test/ && rm $(JWTGENERATE)$(3) $(CONNTEST)$(3) $(GUSHER)$(3) conn-test.env.example env.example  && rm -rf test/

build/linux: 
	go test
	$(call buildjwt,linux,amd64,)
	$(call buildconntest,linux,amd64,)
	$(call buildgusher,linux,amd64,)
	cp env.example build/env.example && cp test/conn-test/conn-test.env.example build/conn-test.env.example && cp -R --parents test/key/ build/
build/linux_amd64.tar.gz: build/linux
	$(call tar,linux,amd64,)
build/windows: 
	go test
	$(call buildjwt,windows,amd64,.exe)
	$(call buildconntest,windows,amd64,.exe)
	$(call buildgusher,windows,amd64,.exe)
	cp env.example build/env.example && cp test/conn-test/conn-test.env.example build/conn-test.env.example && cp -R --parents test/key/ build/
build/windows_amd64.tar.gz: build/windows
	$(call tar,windows,amd64,.exe)
build/darwin: 
	go test
	$(call buildjwt,darwin,amd64,)
	$(call buildconntest,darwin,amd64,)
	$(call buildgusher,darwin,amd64,)
	cp env.example build/env.example && cp test/conn-test/conn-test.env.example build/conn-test.env.example && cp -R --parents test/key/ build/
build/darwin_amd64.tar.gz: build/darwin
	$(call tar,darwin,amd64,)
clean:
	rm -rf build/
docker-build:
	go build -ldflags "-X main.name=$(GUSHER) -X main.version=$(TAG) " -a -o ./$(GUSHER);
rsakey:
	openssl genrsa -out private.pem 2048
	openssl rsa -in private.pem -pubout -out public.pem

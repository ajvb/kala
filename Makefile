APP := kala
VERSION := $(shell git describe --tags --always --dirty)
GOPATH := $(CURDIR)/Godeps/_workspace:$(GOPATH)
PATH := $(GOPATH)/bin:$(PATH)

bin/$(APP): bin
	go build -v -o $@ -ldflags "-X main.Version='${VERSION}'"

bin: clean
	mkdir -p bin

clean:
	rm -rf bin

start: bin/$(APP)
	./bin/$(APP) serve -v

start-consul: bin/$(APP)
	./bin/$(APP) serve --jobdb=consul -v

test:
	go test -v ./...

gen: tools
	go-bindata-assetfs -pkg api -o api/webui_bindata.go webui/...

tools:
	go install github.com/go-bindata/go-bindata/...
	go install github.com/elazarl/go-bindata-assetfs/...


.PHONY: bin/$(APP) bin clean start test gen tools

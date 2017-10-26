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
	./bin/$(APP) run -v

start-consul: bin/$(APP)
	./bin/$(APP) run --jobDB=consul -v

test:
	go test -v ./...

.PHONY: bin/$(APP) bin clean start test

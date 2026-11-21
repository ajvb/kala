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

# Run vet and the linters, then all tests, including the integration tests
# (build tag "integration"), which need Docker. `go test ./...` runs only the
# unit tests.
test: vet lint
	go test -v -tags integration ./...
	cd client && go test -v -tags integration ./...

# Run only the integration tests.
test-integration:
	go test -v -tags integration ./job/storage/... ./internal/...

# Run go vet, including the files behind the integration build tag.
vet:
	go vet -tags integration ./...
	cd client && go vet -tags integration ./...

# Run golangci-lint v2 on both modules.
lint:
	golangci-lint run
	cd client && golangci-lint run

gen: tools
	go-bindata-assetfs -pkg api -o api/webui_bindata.go webui/...

tools:
	go install github.com/go-bindata/go-bindata/...
	go install github.com/elazarl/go-bindata-assetfs/...


.PHONY: bin/$(APP) bin clean start test test-integration vet lint gen tools

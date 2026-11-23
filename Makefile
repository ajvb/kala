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

# Install the web interface's libraries with npm (versions and integrity
# hashes from webui/package-lock.json) and copy the files the interface loads
# into webui/vendor, where they are embedded. Needs npm.
webui-libs:
	cd webui && npm ci --ignore-scripts --no-audit --no-fund
	find webui/vendor -mindepth 1 ! -name README.md -exec rm -rf {} +
	install -D -m 0644 webui/node_modules/@fortawesome/fontawesome-free/js/all.min.js webui/vendor/fontawesome/all.min.js
	install -D -m 0644 webui/node_modules/bulma/css/bulma.css webui/vendor/bulma/bulma.css
	install -D -m 0644 webui/node_modules/reefjs/dist/reef.polyfills.min.js webui/vendor/reef/reef.polyfills.min.js
	install -D -m 0644 webui/node_modules/reefjs/dist/router.min.js webui/vendor/reef/router.min.js

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

.PHONY: bin/$(APP) bin clean webui-libs start test test-integration vet lint

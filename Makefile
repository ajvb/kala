APP := kala
# Override on the command line, for example "make bin/kala VERSION=v2.0.0".
# Values from the environment are ignored.
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
VERSION_PKG := github.com/ajvb/kala/v2/internal/version
LDFLAGS := -X $(VERSION_PKG).Version=$(VERSION) -X $(VERSION_PKG).Commit=$(COMMIT) -X $(VERSION_PKG).BuildDate=$(BUILD_DATE)

# Build the release binary: the version, commit and build date are set at
# build time, -trimpath keeps local paths out of the binary, and the web
# interface's libraries are downloaded first so they are embedded.
bin/$(APP): bin webui-libs
	go build -trimpath -v -o $@ -ldflags "$(LDFLAGS)" .

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

bin: clean
	mkdir -p bin

clean:
	rm -rf bin

# Build the release binary and fail if "kala version" does not report the
# version, commit and build date it was built with.
check-version: bin/$(APP)
	./bin/$(APP) version | tee bin/version.txt
	grep -qx 'kala $(VERSION)' bin/version.txt
	grep -qx 'commit:     $(COMMIT)' bin/version.txt
	grep -qx 'build date: $(BUILD_DATE)' bin/version.txt

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
	go test -v -tags integration ./job ./job/storage/... ./internal/...

# Run the benchmarks once each.
bench:
	go test -run '^$$' -bench . -benchmem ./...

# Run go vet, including the files behind the integration build tag.
vet:
	go vet -tags integration ./...
	cd client && go vet -tags integration ./...

# Run golangci-lint v2 on both modules.
lint:
	golangci-lint run
	cd client && golangci-lint run

.PHONY: bin/$(APP) bin clean webui-libs check-version start test test-integration bench vet lint

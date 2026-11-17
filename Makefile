VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

.PHONY: build install test vet fmt check dist clean

build:            ## build ./bin/burrow-g
	go build -ldflags "$(LDFLAGS)" -o bin/burrow-g ./cmd/burrow-g

install:          ## install burrow-g into $(go env GOPATH)/bin
	go install -ldflags "$(LDFLAGS)" ./cmd/burrow-g

test:
	go test ./...

vet:
	go vet ./...

fmt:              ## fail if any file needs gofmt
	@test -z "$$(gofmt -l .)" || (gofmt -l . && echo "run: gofmt -w ." && exit 1)

check: fmt vet test

dist: check     ## release archives + checksums in ./dist (VERSION=x.y.z)
	rm -rf dist && mkdir -p dist
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; name=burrow-g_$(VERSION)_$${os}_$${arch}; dir=dist/$$name; \
		bin=burrow-g; if [ $$os = windows ]; then bin=burrow-g.exe; fi; \
		echo "building $$os/$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" -o $$dir/$$bin ./cmd/burrow-g || exit 1; \
		cp LICENSE README.md $$dir/; \
		if [ $$os = windows ]; then (cd dist && zip -qr $$name.zip $$name); \
		else tar -C dist -czf $$dir.tar.gz $$name; fi; \
	done
	cd dist && sha256sum *.tar.gz *.zip > checksums.txt

clean:
	rm -rf bin dist npm/dist

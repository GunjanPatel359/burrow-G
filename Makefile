VERSION := $(shell sed -n 's/^const version = "\(.*\)"/\1/p' cmd/burrow-g/commands.go)
LDFLAGS := -s -w

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

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

dist: check     ## release archives + checksums in ./dist (version from cmd/burrow-g/commands.go)
	rm -rf dist && mkdir -p dist
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; dir=dist/burrow-g_$(VERSION)_$${os}_$${arch}; \
		echo "building $$os/$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" -o $$dir/burrow-g ./cmd/burrow-g || exit 1; \
		cp LICENSE README.md $$dir/; \
		tar -C dist -czf $$dir.tar.gz $$(basename $$dir); \
	done
	cd dist && sha256sum *.tar.gz > checksums.txt

clean:
	rm -rf bin dist npm/dist

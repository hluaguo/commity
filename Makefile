.PHONY: build install clean test lint release

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o commity ./cmd/commity

install:
	go install $(LDFLAGS) ./cmd/commity

clean:
	rm -f commity
	rm -rf dist/

test:
	go test -v ./...

lint:
	golangci-lint run

release:
	goreleaser release --clean

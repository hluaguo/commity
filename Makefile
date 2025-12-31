.PHONY: build install clean test coverage lint release

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o commity ./cmd/commity

install:
	go install $(LDFLAGS) ./cmd/commity

clean:
	rm -f commity
	rm -rf dist/
	rm -f coverage.out

test:
	go test ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint:
	golangci-lint run

release:
	goreleaser release --clean

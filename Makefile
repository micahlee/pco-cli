VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GOBIN ?= $(shell go env GOBIN)
ifeq ($(GOBIN),)
GOBIN := $(shell go env GOPATH)/bin
endif

.PHONY: build test lint install install-skill clean

build:
	go build -ldflags "-X main.version=$(VERSION)" -o pco .

test:
	go test ./...

lint:
	golangci-lint run

install:
	go build -ldflags "-X main.version=$(VERSION)" -o "$(GOBIN)/pco" .

install-skill:
	scripts/install-skill.sh

clean:
	rm -f pco

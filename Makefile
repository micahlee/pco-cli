VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GOBIN ?= $(shell go env GOBIN)
ifeq ($(GOBIN),)
GOBIN := $(shell go env GOPATH)/bin
endif

.PHONY: build test lint install install-skill install-skill-codex install-skill-claude install-cursor-rule package-skill clean

build:
	go build -ldflags "-X main.version=$(VERSION)" -o pco .

test:
	go test ./...

lint:
	golangci-lint run

install:
	go build -ldflags "-X main.version=$(VERSION)" -o "$(GOBIN)/pco" .

install-skill: install-skill-codex

install-skill-codex:
	scripts/install-skill.sh --tool codex

install-skill-claude:
	scripts/install-skill.sh --tool claude

install-cursor-rule:
	scripts/install-skill.sh --tool cursor --project-dir "$(or $(CURSOR_PROJECT_DIR),$(CURDIR))"

package-skill:
	scripts/package-skill.sh "$(VERSION)"

clean:
	rm -f pco

SHELL := /bin/bash

build:
	go build -o ./bin/status-term-client .
.PHONY: build

run: ARGS ?=
run: build
	./bin/status-term-client $(ARGS) 2>/tmp/status-term-client.log
.PHONY: run

test:
	go test ./...
.PHONY: test

test-race:
	go test -race ./...
.PHONY: test

lint:
	golangci-lint run
.PHONY: lint

install-dev:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.12.5
.PHONY: install-dev

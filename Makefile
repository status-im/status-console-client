SHELL := /bin/bash

GO111MODULE = on

build: GOFLAGS ?= "-mod=vendor"
build:
	GOFLAGS=$(GOFLAGS) go build -o ./bin/status-term-client .
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
.PHONY: test-race

lint:
	golangci-lint run -v
.PHONY: lint

lint-v110:
	golangci-lint run -v --config .golangci-v110.yml
.PHONY: lint-v110

vendor:
	go mod vendor
	modvendor -copy="**/*.c **/*.h" -v
.PHONY: vendor

install-linter:
	# install linter
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.12.5
.PHONY: install-linter

install-dev:
	# a tool to vendor non-go files
	go get -u github.com/goware/modvendor
	go get -u github.com/golang/mock/gomock
	go install github.com/golang/mock/mockgen
	go get -u github.com/jteeuwen/go-bindata/...
.PHONY: install-dev

mock:
	mockgen \
		-destination ./protocol/v1/mock/protocol_mock.go \
		-package protocol_mock \
		github.com/status-im/status-console-client/protocol/v1 Protocol
.PHONY: mock

gen-migrations:
	pushd protocol/client/migrations/ && rm bindata.go && go-bindata -pkg migrations ./ && popd
.PHONY: gen-migrations

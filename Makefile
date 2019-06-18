GO111MODULE = on

ENABLE_METRICS ?= true
BUILD_FLAGS ?= $(shell echo "-ldflags '\
	-X github.com/status-im/status-console-client/vendor/github.com/ethereum/go-ethereum/metrics.EnabledStr=$(ENABLE_METRICS)'")


DOCKER_IMAGE_NAME ?= statusteam/status-client
DOCKER_CUSTOM_TAG ?= $(shell git rev-parse --short HEAD)

build: GOFLAGS ?= "-mod=vendor"
build:
	GOFLAGS=$(GOFLAGS) go build $(BUILD_FLAGS) -o ./bin/status-term-client .
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
	go mod tidy
	go mod vendor
	modvendor -copy="**/*.c **/*.h" -v
.PHONY: vendor

install-linter:
	# install linter
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.17.1
.PHONY: install-linter

install-dev:
	# a tool to vendor non-go files
	go get -u github.com/goware/modvendor
	go get -u github.com/golang/mock/gomock
	go install github.com/golang/mock/mockgen
	go get -u github.com/jteeuwen/go-bindata/go-bindata@v3.0.7
	go mod tidy || echo 'ignore mod tidy'
.PHONY: install-dev

mock:
	mockgen \
		-destination ./protocol/v1/mock/protocol_mock.go \
		-package protocol_mock \
		github.com/status-im/status-console-client/protocol/v1 Protocol
.PHONY: mock

gen-migrations:
	pushd protocol/client/migrations/ && rm -f bindata.go && go-bindata -pkg migrations ./ && popd
.PHONY: gen-migrations

image:
	docker build . -t $(DOCKER_IMAGE_NAME):latest -t $(DOCKER_IMAGE_NAME):$(DOCKER_CUSTOM_TAG)
.PHONY: image

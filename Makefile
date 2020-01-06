GO111MODULE = on

ENABLE_METRICS ?= true
BUILD_FLAGS ?= $(shell echo "-ldflags '\
	-X github.com/status-im/status-console-client/vendor/github.com/ethereum/go-ethereum/metrics.EnabledStr=$(ENABLE_METRICS)'")


DOCKER_IMAGE_NAME ?= statusteam/status-client
DOCKER_CUSTOM_TAG ?= $(shell git rev-parse --short HEAD)

gofmt:
	find . -name '*.go' -and -not -name 'bindata*' -and -not -name 'migrations.go' -and -not -wholename '*/vendor/*' -exec goimports -local 'github.com/ethereum/go-ethereum,github.com/status-im/status-go,github.com/status-im/status-console-client' -w {} \;

build: GOFLAGS ?= "-mod=vendor"
build:
	GOFLAGS=$(GOFLAGS) go build $(BUILD_FLAGS) -tags geth -o ./bin/status-term-client .
.PHONY: build

# XXX: Multiple ldflags a bit brittle, keeping it simple by having separate build target for now.
# See https://github.com/golang/go/issues/29053
build-nimbus: GOFLAGS ?= "-mod=vendor"
build-nimbus: _NIMBUS_DIR := "./vendor/github.com/status-im/status-go/eth-node/bridge/nimbus"
build-nimbus:
	chmod u+x $(_NIMBUS_DIR)/build-nimbus.sh
	$(_NIMBUS_DIR)/build-nimbus.sh
	chmod u-x $(_NIMBUS_DIR)/build-nimbus.sh
	GOFLAGS=$(GOFLAGS) go build -ldflags="-r $(_NIMBUS_DIR)" -tags "nimbus geth" -o ./bin/status-term-client .
.PHONY: build-nimbus

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
	modvendor -copy="**/*.c **/*.h **/build-nimbus.sh" -v
.PHONY: vendor

install-linter:
	# install linter
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.21.0
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
	echo "no mocks"
.PHONY: mock

generate:
	go generate ./...
.PHONY: gen-migrations

image:
	docker build . -t $(DOCKER_IMAGE_NAME):latest -t $(DOCKER_IMAGE_NAME):$(DOCKER_CUSTOM_TAG)
.PHONY: image

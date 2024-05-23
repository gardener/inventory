.DEFAULT_GOAL := build

IMAGE ?= europe-docker.pkg.dev/gardener-project/releases/gardener/inventory
LOCAL_BIN ?= $(shell pwd)/bin
BINARY ?= $(LOCAL_BIN)/inventory

VERSION := $(shell cat VERSION)
EFFECTIVE_VERSION ?= $(VERSION)-$(shell git rev-parse --short HEAD)

ifneq ($(strip $(shell git status --porcelain 2>/dev/null)),)
	EFFECTIVE_VERSION := $(EFFECTIVE_VERSION)-dirty
endif

IMAGE_TAG ?= $(EFFECTIVE_VERSION)

$(LOCAL_BIN):
	mkdir -p $(LOCAL_BIN)

$(BINARY): $(LOCAL_BIN)
	go build \
		-o $(BINARY) \
		-ldflags="-X 'github.com/gardener/inventory/pkg/version.Version=${EFFECTIVE_VERSION}'" \
		./cmd/inventory

build: $(BINARY)

get:
	go mod download

test:
	go test -v -race ./...

test-cover:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

docker-build:
	docker build -t ${IMAGE}:${IMAGE_TAG} .

.PHONY: get test test-cover build docker-build

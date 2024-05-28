.DEFAULT_GOAL := build
REPO_ROOT                    := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
IMAGE                        ?= europe-docker.pkg.dev/gardener-project/releases/gardener/inventory
LOCAL_BIN                    ?= $(REPO_ROOT)/bin
TOOLS_BIN 				     ?= $(REPO_ROOT)/tools
BINARY                       ?= $(LOCAL_BIN)/inventory
SRC_DIRS                     := $(shell find . -name '*.go' -exec dirname {} \; | sort | uniq)
VERSION                      := $(shell cat VERSION)
EFFECTIVE_VERSION            ?= $(VERSION)-$(shell git rev-parse --short HEAD)

ifneq ($(strip $(shell git status --porcelain 2>/dev/null)),)
	EFFECTIVE_VERSION := $(EFFECTIVE_VERSION)-dirty
endif

IMAGE_TAG                    ?= $(EFFECTIVE_VERSION)

GOIMPORTS                    := $(TOOLS_BIN)/goimports
GOLANGCI_LINT                := $(TOOLS_BIN)/golangci-lint
GOIMPORTS_REVISER            := $(TOOLS_BIN)/goimports-reviser

GOIMPORTS_VERSION            ?= $(call version_gomod,golang.org/x/tools)
GOIMPORTS_REVISER_VERSION    ?= v3.6.5
GOLANGCI_LINT_VERSION        ?= v1.59.0

# Fetch the version of a go module from go.mod
version_gomod = $(shell go list -mod=mod -f '{{ .Version }}' -m $(1))
tool_version_file = $(TOOLS_BIN)/.version_$(subst $(TOOLS_BIN)/,,$(1))_$(2)


$(LOCAL_BIN):
	mkdir -p $(LOCAL_BIN)
$(TOOLS_BIN):
	mkdir -p $(TOOLS_BIN)

$(TOOLS_BIN)/.version_%: $(TOOLS_BIN)
	@version_file=$@; rm -f $${version_file%_*}*
	@touch $@


#########################################
# Tools                                 #
#########################################
$(GOLANGCI_LINT): $(call tool_version_file,$(GOLANGCI_LINT),$(GOLANGCI_LINT_VERSION))
	@GOBIN=$(abspath $(TOOLS_BIN)) CGO_ENABLED=1 go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

$(GOIMPORTS): $(call tool_version_file,$(GOIMPORTS),$(GOIMPORTS_VERSION))
	@GOBIN=$(abspath $(TOOLS_BIN)) go install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION)

$(GOIMPORTS_REVISER): $(call tool_version_file,$(GOIMPORTS_REVISER),$(GOIMPORTS_REVISER_VERSION))
	@GOBIN=$(abspath $(TOOLS_BIN)) go install github.com/incu6us/goimports-reviser/v3@$(GOIMPORTS_REVISER_VERSION)

.PHONY: clean-tools-bin
clean-tools-bin:
	@rm -f $(TOOLS_BIN)/.version_*
	@rm -rf $(TOOLS_BIN)/*

#########################################
# Makefile targets                      #
#########################################

.PHONY: goimports
goimports: $(GOIMPORTS)
	@for dir in $(SRC_DIRS); do \
  		$(GOIMPORTS) -w $$dir/; \
	done

.PHONY: goimports-reviser
goimports-reviser: $(GOIMPORTS_REVISER)
	@for dir in $(SRC_DIRS); do \
  		GOIMPORTS_REVISER_OPTIONS="-imports-order std,project,general,company" \
  		$(GOIMPORTS_REVISER) -recursive $$dir/; \
	done

.PHONY: lint
lint: $(GOLANGCI_LINT)
	@for dir in $(SRC_DIRS); do \
		$(GOLANGCI_LINT) run --config=$(REPO_ROOT)/.golangci.yaml $$dir/ ; \
	done

$(BINARY): $(LOCAL_BIN) goimports lint
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

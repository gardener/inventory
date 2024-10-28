.DEFAULT_GOAL := build
REPO_ROOT                    := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
IMAGE                        ?= europe-docker.pkg.dev/gardener-project/releases/gardener/inventory
LOCAL_BIN                    ?= $(REPO_ROOT)/bin
TOOLS_BIN		     ?= $(REPO_ROOT)/bin/tools
BINARY                       ?= $(LOCAL_BIN)/inventory
SRC_DIRS                     := $(shell go list -f '{{.Dir}}' ./...)
VERSION                      := $(shell cat VERSION)
EFFECTIVE_VERSION            ?= $(VERSION)-$(shell git rev-parse --short HEAD)

ifneq ($(strip $(shell git status --porcelain 2>/dev/null)),)
	EFFECTIVE_VERSION := $(EFFECTIVE_VERSION)-dirty
endif

IMAGE_TAG                    ?= $(EFFECTIVE_VERSION)

GOIMPORTS                    := $(TOOLS_BIN)/goimports
GOLANGCI_LINT                := $(TOOLS_BIN)/golangci-lint
GOIMPORTS_REVISER            := $(TOOLS_BIN)/goimports-reviser
KUSTOMIZE                    := $(TOOLS_BIN)/kustomize
MINIKUBE                     := $(TOOLS_BIN)/minikube

GOIMPORTS_VERSION            ?= $(call version_gomod,golang.org/x/tools)
GOIMPORTS_REVISER_VERSION    ?= v3.6.5
GOLANGCI_LINT_VERSION        ?= v1.60.1
KUSTOMIZE_VERSION            ?= v5.4.2
MINIKUBE_VERSION 	     ?= v1.33.1

# Minikube settings
MINIKUBE_PROFILE ?= inventory
MINIKUBE_DRIVER ?= docker
KUSTOMIZE_OVERLAY ?= local

GOOS = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)

export PATH := $(abspath $(TOOLS_BIN)):$(PATH)

# Fetch the version of a go module from go.mod
version_gomod = $(shell go list -mod=mod -f '{{ .Version }}' -m $(1))
tool_version_file = $(TOOLS_BIN)/.version_$(subst $(TOOLS_BIN)/,,$(1))_$(2)

# download-tool will download a binary package from the given URL.
#
# $1 - name of the tool
# $2 - HTTP URL to download the tool from
define download-tool
@set -e; \
tool=$(1) ;\
echo "Downloading $${tool}" ;\
curl -o $(TOOLS_BIN)/$(1) -sSfL $(2) ;\
chmod +x $(TOOLS_BIN)/$(1)
endef

$(LOCAL_BIN):
	mkdir -p $(LOCAL_BIN)
$(TOOLS_BIN):
	mkdir -p $(TOOLS_BIN)

$(TOOLS_BIN)/.version_%: | $(TOOLS_BIN)
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

$(KUSTOMIZE): $(call tool_version_file,$(KUSTOMIZE),$(KUSTOMIZE_VERSION))
	@GOBIN=$(abspath $(TOOLS_BIN)) go install sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION)

$(MINIKUBE): $(call tool_version_file,$(MINIKUBE),$(MINIKUBE_VERSION))
	$(call download-tool,minikube,https://github.com/kubernetes/minikube/releases/download/$(MINIKUBE_VERSION)/minikube-$(GOOS)-$(GOARCH))


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
	@set -e && \
	for dir in $(SRC_DIRS); do \
		GOIMPORTS_REVISER_OPTIONS="-imports-order std,project,general,company" \
		$(GOIMPORTS_REVISER) -set-exit-status -recursive $$dir/; \
	done

.PHONY: lint
lint: $(GOLANGCI_LINT)
	@$(GOLANGCI_LINT) run --config=$(REPO_ROOT)/.golangci.yaml ./...

$(BINARY): $(SRC_DIRS) | $(LOCAL_BIN)
	go build \
		-o $(BINARY) \
		-ldflags="-X 'github.com/gardener/inventory/pkg/version.Version=${EFFECTIVE_VERSION}'" \
		./cmd/inventory

.PHONY: build
build: $(BINARY)

.PHONY: get
get:
	go mod download

.PHONY: test
test:
	go test -v -race ./...

.PHONY: test-cover
test-cover:
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

.PHONY: docker-build
docker-build:
	docker build -t $(IMAGE):$(IMAGE_TAG) -t $(IMAGE):latest .

.PHONY: docker-compose-up
docker-compose-up:
	docker compose up --build --remove-orphans

.PHONY: kustomize-build
kustomize-build: $(KUSTOMIZE)
	@$(KUSTOMIZE) build deployment/kustomize/$(KUSTOMIZE_OVERLAY)

.PHONY: minikube-up
minikube-up: $(MINIKUBE)
	$(MINIKUBE) -p $(MINIKUBE_PROFILE) start --driver $(MINIKUBE_DRIVER)
	$(MAKE) minikube-load-image
	@$(MAKE) -s kustomize-build | $(MINIKUBE) -p $(MINIKUBE_PROFILE) kubectl -- apply -f -

.PHONY: minikube-down
minikube-down: $(MINIKUBE)
	$(MINIKUBE) delete -p $(MINIKUBE_PROFILE)

.PHONY: minikube-load-image
minikube-load-image: $(MINIKUBE)
	$(MAKE) docker-build
	docker image save -o image.tar $(IMAGE):latest
	$(MINIKUBE) -p $(MINIKUBE_PROFILE) image load --overwrite=true image.tar
	rm -f image.tar

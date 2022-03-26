# Set the shell to bash always
SHELL := /bin/bash

ORG_NAME := lucasepe
PROJECT_NAME := expression-resolver
VENDOR := Luca Sepe

# Github Container Registry
DOCKER_REGISTRY := ghcr.io/$(ORG_NAME)

TARGET_OS := linux
TARGET_ARCH := amd64

# Tools
KIND=$(shell which kind)
LINT=$(shell which golangci-lint)
KUBECTL=$(shell which kubectl)
DOCKER=$(shell which docker)

KIND_CLUSTER_NAME ?= local-dev
KUBECONFIG ?= $(HOME)/.kube/config

VERSION := $(shell git describe --dirty --always --tags | sed 's/-/./2' | sed 's/-/./2')
ifndef VERSION
VERSION := 0.0.0
endif

BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
REPO_URL := $(shell git config --get remote.origin.url | sed "s/git@/https\:\/\//; s/\.com\:/\.com\//; s/\.git//")
LAST_COMMIT := $(shell git log -1 --pretty=%h)

UNAME := $(uname -s)

.PHONY: help
help:	### Show targets documentation
ifeq ($(UNAME), Linux)
	@grep -P '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
else
	@awk -F ':.*###' '$$0 ~ FS {printf "%15s%s\n", $$1 ":", $$2}' \
		$(MAKEFILE_LIST) | grep -v '@awk' | sort
endif

.PHONY: print.vars
print.vars: ### Print all the build variables
	@echo VENDOR=$(VENDOR)
	@echo ORG_NAME=$(ORG_NAME)
	@echo PROJECT_NAME=$(PROJECT_NAME)
	@echo REPO_URL=$(REPO_URL)
	@echo LAST_COMMIT=$(LAST_COMMIT)
	@echo VERSION=$(VERSION)
	@echo BUILD_DATE=$(BUILD_DATE)
	@echo TARGET_OS=$(TARGET_OS)
	@echo TARGET_ARCH=$(TARGET_ARCH)
	@echo DOCKER_REGISTRY=$(DOCKER_REGISTRY)

.PHONY: kind.up
kind.up: ### Starts a KinD cluster for local development
	@$(KIND) get kubeconfig --name $(KIND_CLUSTER_NAME) >/dev/null 2>&1 || $(KIND) create cluster --name=$(KIND_CLUSTER_NAME)


.PHONY: kind.down
kind.down: ### Shuts down the KinD cluster
	@$(KIND) delete cluster --name=$(KIND_CLUSTER_NAME)

.PHONY: image.build
image.build: ### Build the Docker image
	@$(DOCKER) build -t "$(DOCKER_REGISTRY)/$(PROJECT_NAME):$(VERSION)" \
	--build-arg VERSION="$(VERSION)" \
	--build-arg BUILD_DATE="$(BUILD_DATE)" \
	--build-arg REPO_URL="$(REPO_URL)" \
	--build-arg LAST_COMMIT="$(LAST_COMMIT)" \
	--build-arg PROJECT_NAME="$(PROJECT_NAME)" \
	--build-arg VENDOR="$(VENDOR)" .
	@$(DOCKER) rmi -f $$(docker images -f "dangling=true" -q)

.PHONY: image.push
image.push: ### Push the image to the Docker Registry
	@$(DOCKER) push "$(DOCKER_REGISTRY)/$(PROJECT_NAME):$(VERSION)"

.PHONY: vendor
vendor: deps ### Vendor dependencies
	@go mod vendor

.PHONY: deps
deps:	### Optimize dependencies
	@go mod tidy

.PHONY: fmt
fmt: ### Format
	@gofmt -s -w .

.PHONY: vet
vet: ### Vet
	@go vet ./...

### Lint
.PHONY: lint
lint: fmt vet

.PHONY: generate
generate: vendor ### Generate code
	./hack/hack.sh

.PHONY: install
install: ## Install CRDs
	$(KUBECTL) apply -f manifests/crds

.PHONY: clean
clean: ### Clean build files
	@rm -rf ./pkg/generated
	@rm -rf ./bin
	@go clean

.PHONY: build
build: generate ### Build binary
	@go build -tags netgo -a -v -ldflags "${LD_FLAGS}" -o ./bin/$(PROJECT_NAME) ./cmd/*.go
	@chmod +x ./bin/*


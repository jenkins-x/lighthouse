# Make does not offer a recursive wildcard function, so here's one:
rwildcard=$(wildcard $1$2) $(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2))

SHELL := /bin/bash
PROJECT := github.com/jenkins-x/lighthouse
WEBHOOKS_EXECUTABLE := lighthouse
KEEPER_EXECUTABLE := keeper
FOGHORN_EXECUTABLE := foghorn
GCJOBS_EXECUTABLE := gc-jobs
JXCONTROLLER_EXECUTABLE := lighthouse-jx-controller
TEKTONCONTROLLER_EXECUTABLE := lighthouse-tekton-controller
DOCKER_REGISTRY := jenkinsxio
DOCKER_IMAGE_NAME := lighthouse
WEBHOOKS_MAIN_SRC_FILE=cmd/webhooks/main.go
KEEPER_MAIN_SRC_FILE=cmd/keeper/main.go
FOGHORN_MAIN_SRC_FILE=cmd/foghorn/main.go
GCJOBS_MAIN_SRC_FILE=cmd/gc/main.go
JXCONTROLLER_MAIN_SRC_FILE=cmd/jxcontroller/main.go
TEKTONCONTROLLER_MAIN_SRC_FILE=cmd/tektoncontroller/main.go
GO := GO111MODULE=on go
GO_NOMOD := GO111MODULE=off go
VERSION ?= $(shell echo "$$(git describe --abbrev=0 --tags 2>/dev/null)-dev+$(REV)" | sed 's/^v//')
GO_LDFLAGS :=  -X $(PROJECT)/pkg/version.Version='$(VERSION)'
GO_DEPENDENCIES := $(call rwildcard,pkg/,*.go) $(call rwildcard,cmd/,*.go)

GOTEST := $(GO) test

CLIENTSET_GENERATOR_VERSION := kubernetes-1.15.12

all: check test build

.PHONY: test
test: 
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -short ./pkg/... ./cmd/...

.PHONY: check
check: fmt lint sec

get-fmt-deps: ## Install test dependencies
	$(GO_NOMOD) get golang.org/x/tools/cmd/goimports

.PHONY: importfmt
importfmt: get-fmt-deps
	@echo "Formatting the imports..."
	goimports -w $(GO_DEPENDENCIES)

.PHONY: fmt
fmt: importfmt
	@echo "FORMATTING"
	@FORMATTED=`$(GO) fmt ./...`
	@([[ ! -z "$(FORMATTED)" ]] && printf "Fixed unformatted files:\n$(FORMATTED)") || true

GOLINT := $(GOPATH)/bin/golint
$(GOLINT):
	$(GO_NOMOD) get -u golang.org/x/lint/golint

.PHONY: lint
lint: $(GOLINT)
	@echo "VETTING"
	$(GO) vet ./...
	@echo "LINTING"
	$(GOLINT) -set_exit_status ./...

GOSEC := $(GOPATH)/bin/gosec
$(GOSEC):
	$(GO_NOMOD) get -u github.com/securego/gosec/cmd/gosec

.PHONY: sec
sec: $(GOSEC)
	@echo "SECURITY SCANNING"
	$(GOSEC) -fmt=csv ./...

.PHONY: clean
clean:
	rm -rf bin build release

.PHONY: build
build: webhooks keeper foghorn jx-controller tekton-controller gc-jobs

.PHONY: webhooks
webhooks:
	$(GO) build -i -ldflags "$(GO_LDFLAGS)" -o bin/$(WEBHOOKS_EXECUTABLE) $(WEBHOOKS_MAIN_SRC_FILE)

.PHONY: keeper
keeper:
	$(GO) build -i -ldflags "$(GO_LDFLAGS)" -o bin/$(KEEPER_EXECUTABLE) $(KEEPER_MAIN_SRC_FILE)

.PHONY: foghorn
foghorn:
	$(GO) build -i -ldflags "$(GO_LDFLAGS)" -o bin/$(FOGHORN_EXECUTABLE) $(FOGHORN_MAIN_SRC_FILE)

.PHONY: gc-jobs
gc-jobs:
	$(GO) build -i -ldflags "$(GO_LDFLAGS)" -o bin/$(GCJOBS_EXECUTABLE) $(GCJOBS_MAIN_SRC_FILE)

.PHONY: jx-controller
jx-controller:
	$(GO) build -i -ldflags "$(GO_LDFLAGS)" -o bin/$(JXCONTROLLER_EXECUTABLE) $(JXCONTROLLER_MAIN_SRC_FILE)

.PHONY: tekton-controller
tekton-controller:
	$(GO) build -i -ldflags "$(GO_LDFLAGS)" -o bin/$(TEKTONCONTROLLER_EXECUTABLE) $(TEKTONCONTROLLER_MAIN_SRC_FILE)

.PHONY: compile-e2e
compile-e2e:
	$(GO) build -i -ldflags "$(GO_LDFLAGS)" ./test/...

.PHONY: run-e2e-tests
run-e2e-tests:
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -short ./test/...

.PHONY: mod
mod: build
	echo "tidying the go module"
	$(GO) mod tidy

.PHONY: build-linux
build-linux: build-webhooks-linux build-foghorn-linux build-gc-jobs-linux build-keeper-linux build-jx-controller-linux build-tekton-controller-linux

.PHONY: build-webhooks-linux
build-webhooks-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(WEBHOOKS_EXECUTABLE) $(WEBHOOKS_MAIN_SRC_FILE)

.PHONY: build-keeper-linux
build-keeper-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(KEEPER_EXECUTABLE) $(KEEPER_MAIN_SRC_FILE)

.PHONY: build-foghorn-linux
build-foghorn-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(FOGHORN_EXECUTABLE) $(FOGHORN_MAIN_SRC_FILE)

.PHONY: build-gc-jobs-linux
build-gc-jobs-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(GCJOBS_EXECUTABLE) $(GCJOBS_MAIN_SRC_FILE)

.PHONY: build-jx-controller-linux
build-jx-controller-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(JXCONTROLLER_EXECUTABLE) $(JXCONTROLLER_MAIN_SRC_FILE)

.PHONY: build-tekton-controller-linux
build-tekton-controller-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(TEKTONCONTROLLER_EXECUTABLE) $(TEKTONCONTROLLER_MAIN_SRC_FILE)

.PHONY: container
container: 
	docker-compose build $(DOCKER_IMAGE_NAME)

.PHONY: production-container
production-container:
	docker build --rm -t $(DOCKER_REGISTRY)/$(DOCKER_IMAGE_NAME) .

.PHONY: push-container
push-container: production-container
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE_NAME)

CODEGEN_BIN := $(GOPATH)/bin/codegen
$(CODEGEN_BIN):
	$(GO_NOMOD) get github.com/jenkins-x/jx/cmd/codegen

generate-client: codegen-clientset fmt ## Generate the client

codegen-clientset: $(CODEGEN_BIN) ## Generate the k8s types and clients
	@echo "Generating Kubernetes Clients for pkg/apis/lighthouse/v1alpha1 in pkg/client for lighthouse.jenkins.io:v1alpha1"
	$(CODEGEN_BIN) --generator-version $(CLIENTSET_GENERATOR_VERSION) clientset --output-package=pkg/client --input-package=pkg/apis --group-with-version=lighthouse:v1alpha1


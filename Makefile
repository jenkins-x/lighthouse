SHELL := /bin/bash
PROJECT := github.com/jenkins-x/lighthouse
EXECUTABLE := lighthouse
TIDE_EXECUTABLE := tide
DOCKER_REGISTRY := jenkinsxio
DOCKER_IMAGE_NAME := lighthouse
MAIN_SRC_FILE=pkg/main/main.go
TIDE_MAIN_SRC_FILE=cmd/tide/main.go
GO := GO111MODULE=on go
GO_NOMOD := GO111MODULE=off go
VERSION ?= $(shell echo "$$(git describe --abbrev=0 --tags 2>/dev/null)-dev+$(REV)" | sed 's/^v//')
GO_LDFLAGS :=  -X $(PROJECT)/pkg/version.Version='$(VERSION)'

GOTEST := $(GO) test

CLIENTSET_GENERATOR_VERSION := kubernetes-1.12.9

all: check test build

.PHONY: test
test: 
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -short ./...

.PHONY: check
check: fmt lint sec

.PHONY: fmt
fmt:
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
build:
	$(GO) build -i -ldflags "$(GO_LDFLAGS)" -o bin/$(EXECUTABLE) $(MAIN_SRC_FILE) 

.PHONY: tide
tide:
	$(GO) build -i -ldflags "$(GO_LDFLAGS)" -o bin/$(TIDE_EXECUTABLE) $(TIDE_MAIN_SRC_FILE)

.PHONY: all
all: build tide

.PHONY: mod
mod: build
	echo "tidying the go module"
	$(GO) mod tidy

.PHONY: build-linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(EXECUTABLE) $(MAIN_SRC_FILE)

.PHONY: build-tide-linux
build-tide-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(TIDE_EXECUTABLE) $(TIDE_MAIN_SRC_FILE)

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


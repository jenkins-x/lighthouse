SHELL := /bin/bash
PROJECT := github.com/jenkins-x/lighthouse
PKGS := $(shell go list ./...)
EXECUTABLE := lighthouse
DOCKER_REGISTRY := jenkinsxio
DOCKER_IMAGE_NAME := lighthouse
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null)
MAIN_SRC_FILE=pkg/main/main.go

.PHONY: build test golint docs $(PROJECT) $(PKGS) vendor

GOVERSION := $(shell go version | grep 1.12)
ifeq "$(GOVERSION)" ""
#    $(error must be running Go version 1.12)
endif
export GO111MODULE=on
#export GO15VENDOREXPERIMENT=1

all: test build

FGT := $(GOPATH)/bin/fgt
$(FGT):
	go get github.com/GeertJohan/fgt

GOLINT := $(GOPATH)/bin/golint
$(GOLINT):
	go get github.com/golang/lint/golint

GOVENDOR := $(GOPATH)/bin/govendor
$(GOVENDOR):
	go get -u github.com/kardianos/govendor

GO_LDFLAGS := -X $(shell go list ./$(PACKAGE)).GitCommit=$(GIT_COMMIT)

test: $(PKGS)

$(PKGS): $(GOLINT) $(FGT)
	@echo "FORMATTING"
	#@$(FGT) gofmt -l=true pkg/$@/*.go
	#@echo "LINTING"
	#@$(FGT) $(GOLINT) pkg/$@/*.go
	#@echo "VETTING"
	#@go vet -v $@
	@echo "TESTING with go $(GOVERSION)"
	@go test -v $@

vendor: $(GOVENDOR)
	$(GOVENDOR) add +external

build:
	go build -i -ldflags "$(GO_LDFLAGS)" -o bin/$(EXECUTABLE) $(MAIN_SRC_FILE)
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(GO_LDFLAGS)" -o bin/$(EXECUTABLE) $(MAIN_SRC_FILE)
container: build-linux
	docker-compose build $(DOCKER_IMAGE_NAME)
production-container: build-linux
	docker build --rm -t $(DOCKER_REGISTRY)/$(DOCKER_IMAGE_NAME) .
push-container: production-container
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE_NAME)
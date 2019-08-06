SHELL := /bin/bash
PROJECT := github.com/jenkins-x/lighthouse
EXECUTABLE := lighthouse
DOCKER_REGISTRY := jenkinsxio
DOCKER_IMAGE_NAME := lighthouse
MAIN_SRC_FILE=pkg/main/main.go
GO := GO111MODULE=on go
GO_NOMOD := GO111MODULE=off go
VERSION ?= $(shell echo "$$(git describe --abbrev=0 --tags 2>/dev/null)-dev+$(REV)" | sed 's/^v//')
GO_LDFLAGS :=  -X $(PROJECT)/pkg/version.Version='$(VERSION)'

all: fmt lint sec test build

.PHONY: test
test: 
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test -test.v ./...

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

.PHONY: build-linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(EXECUTABLE) $(MAIN_SRC_FILE)

.PHONY: container
container: build-linux
	docker-compose build $(DOCKER_IMAGE_NAME)

.PHONY: production-container
production-container: build-linux
	docker build --rm -t $(DOCKER_REGISTRY)/$(DOCKER_IMAGE_NAME) .

.PHONY: push-container
push-container: production-container
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE_NAME)

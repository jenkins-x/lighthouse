SHELL := /bin/bash
PROJECT_ROOT := $(GOPATH)/src/github.com/jenkins-x/lighthouse
MODULE := webhook
PKGS := $(shell go list ./...)

GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null)

.PHONY: build test golint docs $(PROJECT) $(PKGS) vendor

GOVERSION := $(shell go version | grep 1.12)
ifeq "$(GOVERSION)" ""
    $(error must be running Go version 1.12)
endif
export GO111MODULE=on

all: test webhook

test: $(PKGS)

webhook:
	go build -i -ldflags "$(GO_LDFLAGS)" -o $(PROJECT_ROOT)/bin/webhook cmd/webhook/main.go
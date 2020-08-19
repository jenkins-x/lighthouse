# Make does not offer a recursive wildcard function, so here's one:
rwildcard=$(wildcard $1$2) $(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2))

PROJECT := github.com/jenkins-x/lighthouse

WEBHOOKS_EXECUTABLE := lighthouse
KEEPER_EXECUTABLE := keeper
FOGHORN_EXECUTABLE := foghorn
GCJOBS_EXECUTABLE := gc-jobs
TEKTONCONTROLLER_EXECUTABLE := lighthouse-tekton-controller

WEBHOOKS_MAIN_SRC_FILE=cmd/webhooks/main.go
KEEPER_MAIN_SRC_FILE=cmd/keeper/main.go
FOGHORN_MAIN_SRC_FILE=cmd/foghorn/main.go
GCJOBS_MAIN_SRC_FILE=cmd/gc/main.go
TEKTONCONTROLLER_MAIN_SRC_FILE=cmd/tektoncontroller/main.go

DOCKER_REGISTRY := jenkinsxio
DOCKER_IMAGE_NAME := lighthouse

GO := GO111MODULE=on go
GO_NOMOD := GO111MODULE=off go
GOTEST := $(GO) test

REV := $(shell git rev-parse --short HEAD 2> /dev/null || echo 'unknown')
VERSION ?= $(shell echo "$$(git for-each-ref refs/tags/ --count=1 --sort=-version:refname --format='%(refname:short)' 2>/dev/null)-dev+$(REV)" | sed 's/^v//')
GO_LDFLAGS :=  -X $(PROJECT)/pkg/version.Version='$(VERSION)'
GO_DEPENDENCIES := $(call rwildcard,pkg/,*.go) $(call rwildcard,cmd/,*.go)

.PHONY: all
all: build test check

.PHONY: test
test: 
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -short ./pkg/... ./cmd/...

.PHONY: check
check: fmt lint sec

get-fmt-deps: ## Install test dependencies
	$(GO_NOMOD) get golang.org/x/tools/cmd/goimports

.PHONY: importfmt
importfmt: get-fmt-deps
	@echo "FORMATTING IMPORTS"
	@goimports -w $(GO_DEPENDENCIES)

.PHONY: fmt
fmt: importfmt
	@echo "FORMATTING SOURCE"
	FORMATTED=`$(GO) fmt ./...`
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
	$(GOSEC) -quiet -fmt=csv ./...

.PHONY: clean
clean:
	rm -rf bin build release

.PHONY: build
build: webhooks keeper foghorn tekton-controller gc-jobs

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

.PHONY: tekton-controller
tekton-controller:
	$(GO) build -i -ldflags "$(GO_LDFLAGS)" -o bin/$(TEKTONCONTROLLER_EXECUTABLE) $(TEKTONCONTROLLER_MAIN_SRC_FILE)

.PHONY: compile-e2e
compile-e2e:
	$(GOTEST) -run=nope -failfast -short -ldflags "$(GO_LDFLAGS)" ./test/...

.PHONY: run-e2e-tests
run-e2e-tests:
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) --count=1 -v ./test/...

.PHONY: mod
mod: build
	echo "tidying the go module"
	$(GO) mod tidy

.PHONY: build-linux
build-linux: build-webhooks-linux build-foghorn-linux build-gc-jobs-linux build-keeper-linux build-tekton-controller-linux

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

generate-client: codegen-clientset fmt ## Generate the client

codegen-clientset: ## Generate the k8s types and clients
	@echo "Generating Kubernetes Clients for pkg/apis/lighthouse/v1alpha1 in pkg/client for lighthouse.jenkins.io:v1alpha1"
	./hack/update-codegen.sh

verify-code-unchanged: ## Verify the generated/formatting of code is up to date
	$(eval CHANGED = $(shell git ls-files --modified --others --exclude-standard))
	@if [ "$(CHANGED)" == "" ]; \
      	then \
      	    echo "All generated and formatted files up to date"; \
      	else \
      		echo "Code generation and/or formatting is out of date"; \
      		echo "$(CHANGED)"; \
			git diff; \
      		exit 1; \
      	fi

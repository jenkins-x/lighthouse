# Make does not offer a recursive wildcard function, so here's one:
rwildcard=$(wildcard $1$2) $(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2))

PROJECT := github.com/jenkins-x/lighthouse

WEBHOOKS_EXECUTABLE := webhooks
POLLER_EXECUTABLE := poller
KEEPER_EXECUTABLE := keeper
FOGHORN_EXECUTABLE := foghorn
GC_JOBS_EXECUTABLE := gc-jobs
TEKTON_CONTROLLER_EXECUTABLE := lighthouse-tekton-controller
JENKINS_CONTROLLER_EXECUTABLE := jenkins-controller

WEBHOOKS_MAIN_SRC_FILE=cmd/webhooks/main.go
POLLER_MAIN_SRC_FILE=cmd/poller/main.go
KEEPER_MAIN_SRC_FILE=cmd/keeper/main.go
FOGHORN_MAIN_SRC_FILE=cmd/foghorn/main.go
GC_JOBS_MAIN_SRC_FILE=cmd/gc/main.go
TEKTON_CONTROLLER_MAIN_SRC_FILE=cmd/tektoncontroller/main.go
JENKINS_CONTROLLER_MAIN_SRC_FILE=cmd/jenkins/main.go

GO := GO111MODULE=on go
GO_NOMOD := GO111MODULE=off go
GOTEST := $(GO) test

REV := $(shell git rev-parse --short HEAD 2> /dev/null || echo 'unknown')
VERSION ?= $(shell echo "$$(git for-each-ref refs/tags/ --count=1 --sort=-version:refname --format='%(refname:short)' 2>/dev/null)-dev+$(REV)" | sed 's/^v//')
GO_LDFLAGS :=  -X $(PROJECT)/pkg/version.Version='$(VERSION)'

.PHONY: all
all: build test check docs ## Default rule, builds all binaries, runs tests and format checks

.PHONY: build
build: build-webhooks build-poller build-keeper build-foghorn build-tekton-controller build-gc-jobs build-jenkins-controller ## Builds all Lighthouse binaries native to your machine

.PHONY: build-webhooks
build-webhooks: ## Build the webhooks controller binary for the native OS
	$(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(WEBHOOKS_EXECUTABLE) $(WEBHOOKS_MAIN_SRC_FILE)

.PHONY: build-poller
build-poller: ## Build the poller controller binary for the native OS
	$(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(POLLER_EXECUTABLE) $(POLLER_MAIN_SRC_FILE)

.PHONY: build-keeper
build-keeper: ## Build the keeper controller binary for the native OS
	$(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(KEEPER_EXECUTABLE) $(KEEPER_MAIN_SRC_FILE)

.PHONY: build-foghorn
build-foghorn: ## Build the foghorn controller binary for the native OS
	$(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(FOGHORN_EXECUTABLE) $(FOGHORN_MAIN_SRC_FILE)

.PHONY: build-gc-jobs
build-gc-jobs: ## Build the GC jobs binary for the native OS
	$(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(GC_JOBS_EXECUTABLE) $(GC_JOBS_MAIN_SRC_FILE)

.PHONY: build-tekton-controller
build-tekton-controller: ## Build the Tekton controller binary for the native OS
	$(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(TEKTON_CONTROLLER_EXECUTABLE) $(TEKTON_CONTROLLER_MAIN_SRC_FILE)

.PHONY: build-jenkins-controller
build-jenkins-controller: ## Build the Jenkins controller binary for the native OS
	$(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(JENKINS_CONTROLLER_EXECUTABLE) $(JENKINS_CONTROLLER_MAIN_SRC_FILE)

.PHONY: release
release: linux

.PHONY: linux
linux: build-linux

.PHONY: build-linux
build-linux: build-webhooks-linux build-poller-linux build-foghorn-linux build-gc-jobs-linux build-keeper-linux build-tekton-controller-linux build-jenkins-controller-linux ## Build all binaries for Linux

.PHONY: build-webhooks-linux ## Build the webhook controller binary for Linux
build-webhooks-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(WEBHOOKS_EXECUTABLE) $(WEBHOOKS_MAIN_SRC_FILE)

.PHONY: build-poller-linux ## Build the webhook controller binary for Linux
build-poller-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(POLLER_EXECUTABLE) $(POLLER_MAIN_SRC_FILE)

.PHONY: build-keeper-linux
build-keeper-linux: ## Build the keeper controller binary for Linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(KEEPER_EXECUTABLE) $(KEEPER_MAIN_SRC_FILE)

.PHONY: build-foghorn-linux
build-foghorn-linux: ## Build the foghorn controller binary for Linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(FOGHORN_EXECUTABLE) $(FOGHORN_MAIN_SRC_FILE)

.PHONY: build-gc-jobs-linux
build-gc-jobs-linux: ## Build the GC jobs binary for Linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(GC_JOBS_EXECUTABLE) $(GC_JOBS_MAIN_SRC_FILE)

.PHONY: build-tekton-controller-linux
build-tekton-controller-linux: ## Build the Tekton controller binary for Linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(TEKTON_CONTROLLER_EXECUTABLE) $(TEKTON_CONTROLLER_MAIN_SRC_FILE)

.PHONY: build-jenkins-controller-linux
build-jenkins-controller-linux: ## Build the Jenkins controller binary for Linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(GO_LDFLAGS)" -o bin/$(JENKINS_CONTROLLER_EXECUTABLE) $(JENKINS_CONTROLLER_MAIN_SRC_FILE)

.PHONY: test
test: ## Runs the unit tests
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -short ./pkg/... ./cmd/...

.PHONY: compile-e2e
compile-e2e:
	$(GOTEST) -run=nope -failfast -short -ldflags "$(GO_LDFLAGS)" ./test/...

.PHONY: run-tekton-e2e-tests
run-tekton-e2e-tests: ## Runs Tekton E2E tests
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -v --count=1 -run TestTekton ./test/e2e/tekton

.PHONY: run-jenkins-e2e-tests
run-jenkins-e2e-tests: ## Runs Jenkins E2E tests
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -v --count=1 -run TestJenkins ./test/e2e/jenkins

.PHONY: clean
clean: ## Deletes the generated build directories
	rm -rf bin build

.PHONY: check
check: fmt lint sec ## Runs Go format check as well as security checks

get-fmt-deps:
	$(GO_NOMOD) install golang.org/x/tools/cmd/goimports

.PHONY: importfmt
importfmt: get-fmt-deps ## Checks the import format of the Go source files
	@echo "FORMATTING IMPORTS"
	@goimports -w $(call rwildcard,,*.go)

.PHONY: fmt ## Checks Go source files are formatted properly
fmt: importfmt
	@echo "FORMATTING SOURCE"
	FORMATTED=`$(GO) fmt ./...`
	@([[ ! -z "$(FORMATTED)" ]] && printf "Fixed un-formatted files:\n$(FORMATTED)") || true

GOLINT := $(GOPATH)/bin/golint
$(GOLINT):
	$(GO_NOMOD) get -u golang.org/x/lint/golint

.PHONY: lint
lint: ## Lint the code
	./hack/gofmt.sh
	./hack/linter.sh


GOSEC := $(GOPATH)/bin/gosec
$(GOSEC):
	$(GO_NOMOD) get -u github.com/securego/gosec/cmd/gosec

.PHONY: sec
sec: $(GOSEC) ## Runs gosec to check for potential security issues in the Go source
	@echo "SECURITY SCANNING"
	$(GOSEC) -quiet -fmt=csv ./...

.PHONY: mod
mod: build ## Run 'go mod tidy' to clean up the modules file
	echo "tidying the go module"
	$(GO) mod tidy

generate-client: codegen-clientset fmt ## Generate the K8s types and clients

codegen-clientset:
	@echo "Generating Kubernetes Clients for pkg/apis/lighthouse/v1alpha1 in pkg/client for lighthouse.jenkins.io:v1alpha1"
	./hack/update-codegen.sh

verify-code-unchanged:
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

CONTROLLER_GEN := $(GOPATH)/bin/controller-gen
$(CONTROLLER_GEN):
	$(GO) install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.16.5

crd-manifests: $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) crd:maxDescLen=0 paths="./pkg/apis/lighthouse/v1alpha1/..." output:crd:artifacts:config=crds

.PHONY: docs
docs: job-docs plugins-docs config-docs trigger-docs crds-docs ## Builds generated docs

DOCS_GEN := bin/gen-docs
$(DOCS_GEN):
	cd hack && $(GO) build -o ../$(DOCS_GEN) struct-docs.go

.PHONY: crds-docs
crds-docs: $(DOCS_GEN)
	rm -rf ./docs/crds
	$(DOCS_GEN) --input=./pkg/apis/lighthouse/v1alpha1/... --root=LighthouseBreakpoint --root=LighthouseJob --output=./docs/crds

.PHONY: config-docs
config-docs: $(DOCS_GEN)
	rm -rf ./docs/config/lighthouse
	$(DOCS_GEN) --input=./pkg/config/lighthouse/... --root=Config --output=./docs/config/lighthouse

.PHONY: trigger-docs
trigger-docs: $(DOCS_GEN)
	rm -rf ./docs/trigger
	$(DOCS_GEN) --input=./pkg/triggerconfig/... --root=Config --output=./docs/trigger

.PHONY: plugins-docs
plugins-docs: $(DOCS_GEN)
	rm -rf ./docs/config/plugins
	$(DOCS_GEN) --input=./pkg/plugins/... --root=Configuration --output=./docs/config/plugins

.PHONY: job-docs
job-docs: $(DOCS_GEN)
	rm -rf ./docs/config/jobs
	$(DOCS_GEN) --input=./pkg/config/job/... --root=Config --output=./docs/config/jobs

.PHONY: help
help: ## Prints this help
	@grep -E '^[^.]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-40s\033[0m %s\n", $$1, $$2}'

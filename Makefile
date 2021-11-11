# SPDX-License-Identifier: MIT


# Image URL to use all building/pushing image targets
IMG ?= controller:latest

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif
LD_BUILDDATE="$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')"
LDFLAGS	:= "-X 'main.user=${USER}' -X 'main.buildDate=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')' -X 'main.gitBranch=$(shell git rev-parse --abbrev-ref HEAD)' -X 'main.gitCommit=$(shell git rev-parse HEAD)'"

REGISTRY ?= ghcr.io/daimler/cluster-api-state-metrics
IMAGE ?= $(REGISTRY)/cluster-api-state-metrics
GIT_TAG_OR_COMMIT ?= $(shell git describe --tags --exact-match 2>/dev/null || git rev-parse HEAD)

# buildDate string

# echo "GIT_BRANCH $(git rev-parse --abbrev-ref HEAD)"
# echo "GIT_COMMIT $(git rev-parse HEAD)"


# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

all: test manifests build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help-nocolor:
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make <target>\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  %-15s %s\n", $$1, $$2 } /^##@/ { printf "\n%s\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..."

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

lint: # Run golangci-lint against code.
	@hack/check_golangci-lint.sh

spdxcheck: ## Run spdx check against all files.
	@hack/check_spdx.sh

doccheck: docs ## Run docs specific checks
	@echo "- Checking if the generated documentation is up to date..."
	@git diff --exit-code || (echo "ERROR: dirty git detected"; exit 1)
	@echo "- Checking if the documentation is in sync with the code..."
	@grep -hoE -d skip '\| capi_[^ |]+' docs/* --exclude=README.md | sed -E 's/\| //g' | sort -u > documented_metrics
	@find pkg/store -type f -not -name '*_test.go' -exec sed -nE 's/.*"(capi_[^"]+)".*/\1/p' {} \; | sort -u > code_metrics
	@diff -u0 code_metrics documented_metrics || (echo "ERROR: Metrics with - are present in code but missing in documentation, metrics with + are documented but not found in code."; exit 1)
	@echo OK
	@rm -f code_metrics documented_metrics
	@echo "- Checking for orphan documentation files"
	@cd docs; for doc in *.md; do if [ "$$doc" != "README.md" ] && ! grep -q "$$doc" *.md; then echo "ERROR: No link to documentation file $${doc} detected"; exit 1; fi; done
	@echo OK

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: manifests generate fmt vet spdxcheck docs doccheck ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

##@ Build

docs: embedmd build ## Regenerate docs
	@echo ">> generating docs"
	echo "$ cluster-api-state-metrics -h" > help.txt
	@./bin/cluster-api-state-metrics -h 2>> help.txt
	@make help-nocolor | grep -v '^make' > make-help.txt
	$(EMBEDMD) -w `find . -path ./vendor -prune -o -name "*.md" -print`

build: generate fmt vet ## Build manager binary.
	CGO_ENABLED=0 go build \
		-ldflags $(LDFLAGS) \
		-o bin/cluster-api-state-metrics \
		main.go

image: build ## Build container image
	docker build -t $(IMAGE):$(GIT_TAG_OR_COMMIT) .

push: image ## Push container image
	docker push $(IMAGE):$(GIT_TAG_OR_COMMIT)

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

##@ Deployment

template: manifests kustomize ## Create kustomized deployment yaml.
	$(KUSTOMIZE) build config/default > bin/deploy.yaml

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.5.0
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

KUSTOMIZE = $(shell pwd)/bin/kustomize
# Download kustomize locally if necessary.
kustomize:
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

EMBEDMD = $(shell pwd)/bin/embedmd
# Download embedmd locally if necessary.
embedmd:
	$(call go-get-tool,$(EMBEDMD),github.com/campoy/embedmd@latest)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

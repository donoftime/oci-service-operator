#
# Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
# Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
#


# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=1.0.0)
# - use environment variables to overwrite this value (e.g export VERSION=1.0.0)
VERSION ?= 1.0.0

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "preview,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=preview,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="preview,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# oraclecloud/upgradeoperatorsdk-bundle:$VERSION and oraclecloud/upgradeoperatorsdk-catalog:$VERSION.
IMAGE_TAG_BASE ?= iad.ocir.io/oracle/oci-service-operator

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:$(VERSION)

# Image URL to use all building/pushing image targets
IMG ?= $(IMAGE_TAG_BASE):$(VERSION)
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:generateEmbeddedObjectMeta=true,allowDangerousTypes=true"
API_GEN_PATHS ?= "./api/..."
CONTROLLER_GEN_PATHS ?= "./controllers/..."

# Keep Go and XDG caches under the repo's .tmp directory.
TMP_DIR ?= $(CURDIR)/.tmp
TMP_WORK_DIR ?= $(TMP_DIR)/tmp
GOCACHE ?= $(TMP_DIR)/go-cache
GOMODCACHE ?= $(TMP_DIR)/go-mod-cache
GOTMPDIR ?= $(TMP_DIR)/go-tmp
TMPDIR ?= $(TMP_WORK_DIR)
XDG_CACHE_HOME ?= $(TMP_DIR)/xdg-cache

export GOCACHE
export GOMODCACHE
export GOTMPDIR
export TMPDIR
export XDG_CACHE_HOME

MODULE_CACHE_STAMP ?= $(GOMODCACHE)/.download.stamp

# Use bash with pipefail for recipes where every stage must propagate failures.
BASH_PIPEFAIL ?= /usr/bin/env bash -e -o pipefail -c

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: build

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

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: module-cache cache-dirs controller-gen ## Generate ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) paths=$(API_GEN_PATHS) output:crd:artifacts:config=config/crd/bases
	$(CONTROLLER_GEN) rbac:roleName=manager-role paths=$(CONTROLLER_GEN_PATHS)

generate: module-cache cache-dirs controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths=$(API_GEN_PATHS)

fmt: module-cache cache-dirs ## Run go fmt against code.
	go fmt ./...

vet: module-cache cache-dirs ## Run go vet against code.
	go vet ./...

lint: module-cache cache-dirs golangci-lint ## Run lint and complexity checks.
	@mkdir -p "$(TMP_DIR)" "$(LINT_GOCACHE)" "$(LINT_CACHE)"
	GOCACHE="$(LINT_GOCACHE)" GOLANGCI_LINT_CACHE="$(LINT_CACHE)" $(GOLANGCI_LINT) run ./...

ENVTEST_K8S_VERSION ?= 1.28.0

test: module-cache cache-dirs manifests generate fmt vet setup-envtest ## Run tests.
	$(BASH_PIPEFAIL) 'assets="$$($(SETUP_ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)"; KUBEBUILDER_ASSETS="$$assets" go test ./... -coverprofile cover.out | tee unittests.cover'
	$(BASH_PIPEFAIL) "go tool cover -func cover.out | tail -n 1 | xargs | cut -d ' ' -f 3 | tr -d '%' > unittests.percent"

functionaltest: ## Run functionaltest (placeholder — no functional tests yet).
	@echo "No functional tests available."

##@ Build Service

test-sample: module-cache cache-dirs fmt vet setup-envtest ## Run tests.
	$(BASH_PIPEFAIL) 'assets="$$($(SETUP_ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)"; KUBEBUILDER_ASSETS="$$assets" go test -v ./... -coverprofile cover.out -args -ginkgo.v'

docker-build-sample: ## Build docker image with the manager.
	docker build -t ${IMG} .

##@ Build

build: module-cache cache-dirs generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

run: module-cache cache-dirs manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

docker-build: test bundle ## Build docker image with the manager and CRDs
	docker build -t ${IMG} .

docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -


CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: cache-dirs ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.17.0)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: cache-dirs ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5@v5.4.2)

GOLANGCI_LINT_VERSION ?= v2.6.2
GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
GOLANGCI_LINT_PKG = github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
LINT_GOCACHE ?= $(GOCACHE)
LINT_CACHE ?= $(TMP_DIR)/golangci-lint-cache
golangci-lint: cache-dirs ## Download golangci-lint locally if necessary.
	$(call go-get-tool,$(GOLANGCI_LINT),$(GOLANGCI_LINT_PKG))

SETUP_ENVTEST = $(shell pwd)/bin/setup-envtest
setup-envtest: cache-dirs ## Download setup-envtest locally if necessary.
	$(call go-get-tool,$(SETUP_ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

.PHONY: cache-dirs
cache-dirs:
	@mkdir -p "$(TMP_DIR)" "$(TMPDIR)" "$(GOCACHE)" "$(GOMODCACHE)" "$(GOTMPDIR)" "$(XDG_CACHE_HOME)" "$(LINT_CACHE)"

.PHONY: module-cache
module-cache: $(MODULE_CACHE_STAMP)

$(MODULE_CACHE_STAMP): go.mod go.sum | cache-dirs
	@echo "Downloading module graph into $(GOMODCACHE)"
	go mod download all
	@touch "$(MODULE_CACHE_STAMP)"

# go-get-tool will 'go install' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
}
endef

FORMAL_DIR ?= $(PROJECT_DIR)/formal
FORMAL_TOOLS_DIR ?= $(PROJECT_DIR)/bin/formal
FORMAL_TMP_DIR ?= $(TMP_DIR)/formal
TLA2TOOLS_VERSION ?= 1.8.0
PLANTUML_VERSION ?= 1.2024.6
TLA2TOOLS_JAR ?= $(FORMAL_TOOLS_DIR)/tla2tools-$(TLA2TOOLS_VERSION).jar
PLANTUML_JAR ?= $(FORMAL_TOOLS_DIR)/plantuml-$(PLANTUML_VERSION).jar

.PHONY: formal-tools
formal-tools: ## Download TLC and PlantUML locally if necessary.
	@mkdir -p "$(FORMAL_TMP_DIR)"
	TMPDIR="$(FORMAL_TMP_DIR)" FORMAL_TMP_DIR="$(FORMAL_TMP_DIR)" ./tools/formal/bootstrap.sh "$(FORMAL_TOOLS_DIR)" "$(TLA2TOOLS_VERSION)" "$(PLANTUML_VERSION)"

.PHONY: formal
formal: formal-tools ## Run TLC for every controller spec under formal/controllers.
	TMPDIR="$(FORMAL_TMP_DIR)" FORMAL_TMP_DIR="$(FORMAL_TMP_DIR)" ./tools/formal/run_all.sh "$(TLA2TOOLS_JAR)" "$(FORMAL_DIR)/controllers"

formal-%: formal-tools ## Run TLC for a single controller slug from formal/controllers/<slug>.
	TMPDIR="$(FORMAL_TMP_DIR)" FORMAL_TMP_DIR="$(FORMAL_TMP_DIR)" ./tools/formal/run_controller.sh "$(TLA2TOOLS_JAR)" "$(FORMAL_DIR)/controllers/$*"

.PHONY: diagrams
diagrams: formal-tools ## Render all PlantUML diagrams under formal/controllers.
	TMPDIR="$(FORMAL_TMP_DIR)" FORMAL_TMP_DIR="$(FORMAL_TMP_DIR)" ./tools/formal/render_all.sh "$(PLANTUML_JAR)" "$(FORMAL_DIR)/controllers"

diagrams-%: formal-tools ## Render PlantUML diagrams for a single controller slug.
	TMPDIR="$(FORMAL_TMP_DIR)" FORMAL_TMP_DIR="$(FORMAL_TMP_DIR)" ./tools/formal/render_controller.sh "$(PLANTUML_JAR)" "$(FORMAL_DIR)/controllers/$*"

.PHONY: bundle
bundle: manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)

.PHONY: update-bundle-image-version
update-bundle-image-version: ## Updates versioning info in bundle/manifests/oci-service-operator.clusterserviceversion.yaml.
	sed -i "s/name: oci-service-operator.v1.0.0/name: oci-service-operator.v${VERSION}/g" bundle/manifests/oci-service-operator.clusterserviceversion.yaml
	sed -i "s#iad.ocir.io/oracle/oci-service-operator:1.0.0#${IMAGE_TAG_BASE}:${VERSION}#g" bundle/manifests/oci-service-operator.clusterserviceversion.yaml
	sed -i "s/version: 1.0.0/version: ${VERSION}/g" bundle/manifests/oci-service-operator.clusterserviceversion.yaml

.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.15.1/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:v$(VERSION)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool docker --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)

delete-crds:
	kubectl delete crd autonomousdatabases.oci.oracle.com &
	kubectl delete crd mysqldbsystems.oci.oracle.com &
	kubectl delete crd streams.oci.oracle.com &
	kubectl delete crd apigatewaydeployments.oci.oracle.com &
	kubectl delete crd apigateways.oci.oracle.com &
	kubectl delete crd containerinstances.oci.oracle.com &
	kubectl delete crd functionsapplications.oci.oracle.com &
	kubectl delete crd functionsfunctions.oci.oracle.com &
	kubectl delete crd nosqldatabases.oci.oracle.com &
	kubectl delete crd ociqueues.oci.oracle.com &
	kubectl delete crd opensearchclusters.oci.oracle.com &
	kubectl delete crd postgresdbsystems.oci.oracle.com &
	kubectl delete crd redisclusters.oci.oracle.com &
	kubectl delete crd ocivaults.oci.oracle.com &
	kubectl delete crd dataflowapplications.oci.oracle.com &
	kubectl delete crd objectstoragebuckets.oci.oracle.com &

delete-operator:
	kubectl delete ns $(OPERATOR_NAMESPACE)

.PHONY: delete-crds-force
delete-crds-force:
	kubectl patch crd/autonomousdatabases.oci.oracle.com -p '{"metadata":{"finalizers":[]}}' --type=merge &
	kubectl patch crd/mysqldbsystems.oci.oracle.com -p '{"metadata":{"finalizers":[]}}' --type=merge &
	kubectl patch crd/streams.oci.oracle.com -p '{"metadata":{"finalizers":[]}}' --type=merge &
	kubectl patch crd/apigatewaydeployments.oci.oracle.com -p '{"metadata":{"finalizers":[]}}' --type=merge &
	kubectl patch crd/apigateways.oci.oracle.com -p '{"metadata":{"finalizers":[]}}' --type=merge &
	kubectl patch crd/containerinstances.oci.oracle.com -p '{"metadata":{"finalizers":[]}}' --type=merge &
	kubectl patch crd/functionsapplications.oci.oracle.com -p '{"metadata":{"finalizers":[]}}' --type=merge &
	kubectl patch crd/functionsfunctions.oci.oracle.com -p '{"metadata":{"finalizers":[]}}' --type=merge &
	kubectl patch crd/nosqldatabases.oci.oracle.com -p '{"metadata":{"finalizers":[]}}' --type=merge &
	kubectl patch crd/ociqueues.oci.oracle.com -p '{"metadata":{"finalizers":[]}}' --type=merge &
	kubectl patch crd/opensearchclusters.oci.oracle.com -p '{"metadata":{"finalizers":[]}}' --type=merge &
	kubectl patch crd/postgresdbsystems.oci.oracle.com -p '{"metadata":{"finalizers":[]}}' --type=merge &
	kubectl patch crd/redisclusters.oci.oracle.com -p '{"metadata":{"finalizers":[]}}' --type=merge &
	kubectl patch crd/ocivaults.oci.oracle.com -p '{"metadata":{"finalizers":[]}}' --type=merge

 
# Include various params
include $(shell pwd)/env

# Default bundle image tag
BUNDLE_IMG ?= controller-bundle:$(VERSION)
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)
ifneq ($(origin CUSTOM_OLD_VERSION),undefined)
	OLD_VERSION=${CUSTOM_OLD_VERSION}
endif

# Image URL to use all building/pushing image targets
IMG=quay.io/jooholee/${OP_NAME}:${TAG}
INDEX_IMG=quay.io/jooholee/${OP_NAME}-index:${TAG}
OLD_INDEX_IMG=quay.io/jooholee/${OP_NAME}-index:${OLD_VERSION}
BUNDLE_IMG=quay.io/jooholee/${OP_NAME}-bundle:${TAG}

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# example.com/operatorsdk-bundle:$VERSION and example.com/operatorsdk-catalog:$VERSION.
IMAGE_TAG_BASE ?= example.com/operatorsdk

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)

# BUNDLE_GEN_FLAGS are the flags passed to the operator-sdk generate bundle command
BUNDLE_GEN_FLAGS ?= -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)

# USE_IMAGE_DIGESTS defines if images are resolved via tags or digests
# You can enable this value if you would like to use SHA Based Digests
# To enable set flag to true
USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
	BUNDLE_GEN_FLAGS += --use-image-digests
endif

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.30.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
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

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_TOOLS_VERSION ?= v0.15.0

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
ifeq (,$(wildcard ./bin/kustomize))
	curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN)
endif

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: bundle
bundle: manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=quay.io/jooholee/$(OP_NAME)@$(NFS_OPERATOR_PINNED_DIGESTS)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle $(BUNDLE_GEN_FLAGS)
	operator-sdk bundle validate ./bundle

	
.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.23.0/$${OS}-$${ARCH}-opm ;\
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


##@ Custom
# Build the podman image
podman-build: test
	sudo docker build . -t ${IMG}
  
  

# Push the podman image
podman-push: update-digests
	sudo docker push ${IMG}
# Generate bundle manifests and metadata, then validate generated files.

.PHONY: update-digests
update-digests:
	./hack/scripts/update_pinned_digests.sh

.PHONY: digests
digests: 
	sed "s/.*containerImage.*/    containerImage: quay.io\/jooholee\/nfs-provisioner-operator@${NFS_OPERATOR_PINNED_DIGESTS}/g" -i ./config/manifests/bases/nfs-provisioner-operator.clusterserviceversion.yaml
	sed -i "s/.*newTag.*/  newTag: ${NFS_OPERATOR_PINNED_DIGESTS}/g" ./config/manager/kustomization.yaml
	sed -i "s/k8s.gcr.io\/sig-storage\/nfs-provisioner\([^,]\+\)/k8s.gcr.io\/sig-storage\/nfs-provisioner@${NFS_SERVER_PINNED_DIGESTS}/g" ./config/samples/cache_v1alpha1_nfsprovisioner.yaml 
	sed -i "s/k8s.gcr.io\/sig-storage\/nfs-provisioner\([^,]\+\)/k8s.gcr.io\/sig-storage\/nfs-provisioner@${NFS_SERVER_PINNED_DIGESTS}/g" ./config/samples/cache_v1alpha1_nfsprovisioner_pvc.yaml 
	sed -i "s/k8s.gcr.io\/sig-storage\/nfs-provisioner\([^,]\+\)/k8s.gcr.io\/sig-storage\/nfs-provisioner@${NFS_SERVER_PINNED_DIGESTS}/g" ./config/samples/cache_v1alpha1_nfsprovisioner_hostPath.yaml 
	sed -i "s/gcr.io\/kubebuilder\/kube-rbac-proxy.*/gcr.io\/kubebuilder\/kube-rbac-proxy@${RBAC_PROXY_PINNED_DIGESTS}/g" ./config/default/manager_auth_proxy_patch.yaml
	sed -i "s/k8s.gcr.io\/sig-storage\/nfs-provisioner.*/k8s.gcr.io\/sig-storage\/nfs-provisioner@${NFS_SERVER_PINNED_DIGESTS}/g" ./hack/templates/nfs.yaml
	sed -i "s/k8s.gcr.io\/sig-storage\/nfs-provisioner.*/k8s.gcr.io\/sig-storage\/nfs-provisioner@${NFS_SERVER_PINNED_DIGESTS}/g" ./hack/templates/nfs-hostpath.yaml

## Build the bundle image.
.PHONY: bundle-build
bundle-build: digests bundle
	podman build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

## Push the bundle image.
.PHONY: bundle-push
bundle-push:
	podman push $(BUNDLE_IMG) 

# Index
## Build the Index image
index-build:
	opm index add -c podman  --bundles ${BUNDLE_IMG} --from-index ${OLD_INDEX_IMG} --tag ${INDEX_IMG}


## Push the Index image
index-push:
	podman push ${INDEX_IMG}

# Last Job for push a new version 
push-new-images:
	./hack/scripts/push-new-images.sh


# Test
.PHONY: test-op-local test-op-cluster test-bundle test-index 
deploy-op-local: install 
	oc project ${NAMESPACE} || oc new-project ${NAMESPACE} 
	make run ENABLE_WEBHOOKS=false

deploy-nfs-cr: install
	./hack/scripts/deploy-nfs-cr.sh

deploy-nfs-cluster: install
	./hack/scripts/deploy-nfs-cluster.sh

deploy-nfs-cluster-olm: install
	./hack/scripts/deploy-nfs-cluster-olm.sh

deploy-nfs-cluster-olm-upgrade: 
	./hack/scripts/deploy-nfs-cluster-olm-upgrade.sh

test-pvc:
	./hack/scripts/test-pvc.sh

test-pod:
	./hack/scripts/test-pod.sh

test-rw:
	./hack/scripts/test-rw.sh

test-cleanup:
	./hack/scripts/test-cleanup.sh

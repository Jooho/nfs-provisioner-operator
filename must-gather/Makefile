# Must Gather
OPERATOR_NAME ?=nfs-provisioner-operator
DEFAULT_IMAGE_REGISTRY=quay.io
DEFAULT_REGISTRY_NAMESPACE=jooholee
DEFAULT_IMAGE_TAG=0.0.1
DEFAULT_MUST_GATHER_IMAGE_NAME=$(OPERATOR_NAME)-must-gather
IMAGE_REGISTRY ?=$(DEFAULT_IMAGE_REGISTRY)
REGISTRY_NAMESPACE ?=$(DEFAULT_REGISTRY_NAMESPACE)
MUST_GATHER_IMAGE_NAME ?=$(DEFAULT_MUST_GATHER_IMAGE_NAME)
IMAGE_TAG ?=$(DEFAULT_IMAGE_TAG)
MUST_GATHER_FULL_IMAGE_NAME=$(IMAGE_REGISTRY)/$(REGISTRY_NAMESPACE)/$(MUST_GATHER_IMAGE_NAME):$(IMAGE_TAG)

.PHONY: check-must-gather build-must-gather push-must-gather deploy-must-gather must-gather

check-must-gather:
	@echo "Checking the Shellscript"
	./hack/shellcheck-test.sh
       
build-must-gather:
	@echo "Building the $(OPERATOR_NAME)-must-gather image"
	podman build --format docker -t $(MUST_GATHER_FULL_IMAGE_NAME) -f $(shell pwd)/Dockerfile.rhel .

push-must-gather:
	@echo "Pushing the $(OPERATOR_NAME)-must-gather image to $(IMAGE_REGISTRY)/$(REGISTRY_NAMESPACE)"
	podman push $(MUST_GATHER_FULL_IMAGE_NAME)

deploy-must-gather:
	@echo "Deploying the $(OPERATOR_NAME) must-gather image"
	isv-install must-gather $(MUST_GATHER_FULL_IMAGE_NAME)

must-gather:  build-must-gather push-must-gather

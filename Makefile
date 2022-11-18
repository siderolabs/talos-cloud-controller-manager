REGISTRY ?= ghcr.io
USERNAME ?= sergelogvinov
PROJECT ?= talos-cloud-controller-manager
IMAGE ?= $(REGISTRY)/$(USERNAME)/$(PROJECT)

SHA ?= $(shell git describe --match=none --always --abbrev=8 --dirty)
TAG ?= $(shell git describe --tag --always --dirty --match v[0-9]\*)

OS ?= $(shell go env GOOS)
ARCH ?= $(shell go env GOARCH)
ARCHS = amd64 arm64

TESTARGS ?= "-v"

######

# Help Menu

define HELP_MENU_HEADER
# Getting Started

To build this project, you must have the following installed:

- git
- make
- golang 1.19

endef

export HELP_MENU_HEADER

help: ## This help menu.
	@echo "$$HELP_MENU_HEADER"
	@grep -E '^[a-zA-Z0-9%_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# Build Abstractions

build-all-archs:
	@for arch in $(ARCHS); do $(MAKE) ARCH=$${arch} build ; done

.PHONY: build
build:
	CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build \
		-o talos-cloud-controller-manager-$(ARCH) ./cmd/talos-cloud-controller-manager

.PHONY: run
run: build
	./talos-cloud-controller-manager-$(ARCH) --v=4 --kubeconfig=kubeconfig --cloud-config=hack/talos-config.yaml --controllers=cloud-node \
		--use-service-account-credentials --leader-elect=false --bind-address=127.0.0.1

.PHONY: lint
lint:
	golangci-lint run --config .golangci.yml

.PHONY: unit
unit:
	go test -tags=unit $(shell go list ./...) $(TESTARGS)

.PHONY: conformance
conformance:
	docker run --rm -it -v $(PWD):/src -w /src ghcr.io/siderolabs/conform:v0.1.0-alpha.27 enforce

.PHONY: docs
docs:
	helm template -n kube-system talos-cloud-controller-manager -f charts/talos-cloud-controller-manager/values-example.yaml \
		charts/talos-cloud-controller-manager > docs/deploy/cloud-controller-manager.yml

images-push: $(foreach arch,$(ARCHS),image-push-$(arch)) image-manifest
image-push-%:
	@docker build --build-arg=ARCH=$* --build-arg=IMAGE=$(IMAGE) -t $(IMAGE):$(SHA)-$* \
			--target=release -f Dockerfile .
	@docker push $(IMAGE):$(SHA)-$*

image-manifest: $(foreach arch,$(ARCHS),image-manifest-$(arch))
	@docker manifest push --purge $(IMAGE):$(SHA)
image-manifest-%:
	@docker manifest create $(IMAGE):$(SHA) --amend $(IMAGE):$(SHA)-$*
	@docker manifest annotate --os linux --arch $* $(IMAGE):$(SHA) $(IMAGE):$(SHA)-$*

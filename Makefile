# Image URL to use all building/pushing image targets
REGISTRY ?=harbor.bhidi.com/library
CONTROLLER_GEN = bin/controller-gen
NACOS_BRIDGE_VERSION = v1.0.0

# Set go env
GOOS = linux
GOARCH = arm64

CI_COMMIT = $(shell git rev-parse --short HEAD)
CI_TAG ?= $(shell git describe --tags)
LDFLAGS = -s -w -X 'main.version=$(CI_TAG)' -X 'main.commit=$(CI_COMMIT)' -X 'main.nacosbridge=$(NACOS_BRIDGE_VERSION)'

.PHONY: all
all: build

.PHONY: manifests
manifests: 
	$(CONTROLLER_GEN) rbac:roleName=nacosbridge-role crd webhook paths="./..."

.PHONY: generate
generate: 
	$(CONTROLLER_GEN) object paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: 
	go test ./... -ldflags "-s -w" -v

.PHONY: install
install: manifests
	kubectl kustomize config | sed "s|CI_TAG|$(CI_TAG)|g" | sed "s|CI_REGISTRY|$(REGISTRY)|g" | kubectl apply -f -

.PHONY: clean
clean:
	kubectl kustomize config | kubectl delete --ignore-not-found -f -

.PHONY: build
build: 
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -trimpath -ldflags "$(LDFLAGS)" -o bin/nacosbridge -v ./cmd/

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

.PHONY: docker
docker: ## Build docker image with the manager.
	docker build -t ${REGISTRY}/nacosbridge:${CI_TAG} -f config/docker/Dockerfile .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${REGISTRY}/nacosbridge:${CI_TAG}

$(CONTROLLER_GEN): ## Download controller-gen locally if necessary.
	@[ -f $@ ] || GOBIN=$(CURDIR)/bin go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.18.0
OUTPUT := bin/ingress-istio-controller
VERSION ?= $(shell git describe --dirty --exact-match --tags 2>/dev/null)
SHA ?= $(shell git rev-parse HEAD)
IMAGE := statcan/ingress-istio-controller
IMAGE_TAG := latest

ifeq ($(VERSION),)
VERSION := $(SHA)
endif

$(OUTPUT):
	mkdir -p bin
	CGO_ENABLED=0 go build -o $(OUTPUT) -ldflags "-X github.com/StatCan/ingress-istio-controller/pkg/controller.controllerAgentVersion=$(VERSION)"

.PHONY: build
build: $(OUTPUT)

.PHONY: docker-build
docker-build:
	docker build -t $(IMAGE):$(IMAGE_TAG) --build-arg VERSION=$(VERSION) .

.PHONY: clean
clean:
	rm -f $(OUTPUT)

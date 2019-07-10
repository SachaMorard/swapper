.PHONY: all
MAKEFLAGS += --silent

DOCKER_TAG_SWAPPER_PROXY = 1.0.0
DOCKER_REPO_SWAPPER_PROXY = gcr.io/docker-swapper/swapper-proxy
DOCKER_IMAGE_SWAPPER_PROXY = $(DOCKER_REPO_SWAPPER_PROXY):$(DOCKER_TAG_SWAPPER_PROXY)

GOPATH = $(shell pwd)
GOBIN = $(shell pwd)/bin
export GOPATH
export GOBIN

all: help

help:
	@grep -E '^[a-zA-Z1-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort \
		| sed -e "s/^Makefile://" -e "s///" \
		| awk 'BEGIN { FS = ":.*?## " }; { printf "\033[36m%-30s\033[0m %s\n", $$1, $$2 }'

build-swapper-proxy: ## Build docker image for swapper-proxy
	docker build -f docker/swapper-proxy/Dockerfile -t ${DOCKER_IMAGE_SWAPPER_PROXY} .

docker-push-swapper-proxy: ## Build and push swapper-proxy image to registry
	echo '--> Build image'
	make build-swapper-proxy
	echo '--> Push ${DOCKER_IMAGE_SWAPPER_PROXY}'
	docker push ${DOCKER_IMAGE_SWAPPER_PROXY}

go-install: ## go install
	go install

install: ## Install dev environment
	echo '--> Install dependencies'
	go get -u github.com/docopt/docopt-go
	go get -u gopkg.in/yaml.v2
	go get -u github.com/valyala/fasthttp
	echo '--> Install binary file in ${GOBIN}'
	make go-install

test: ## launch unit test
	go vet
	go test -covermode=count -coverprofile=profile.cov .

ctest: ## launch unit test
	go vet
	go test -run ${RUN}

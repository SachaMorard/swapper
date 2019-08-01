.PHONY: all
MAKEFLAGS += --silent

DOCKER_TAG_SWAPPER_PROXY = 1.0.0
DOCKER_REPO_SWAPPER_PROXY = gcr.io/docker-swapper/swapper-proxy
DOCKER_IMAGE_SWAPPER_PROXY = $(DOCKER_REPO_SWAPPER_PROXY):$(DOCKER_TAG_SWAPPER_PROXY)

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

install: ## go install
	go install

test: ## launch unit test
	go vet
	go test -covermode=count .

ctest: ## launch unit test
	go vet
	go test -run ${RUN}

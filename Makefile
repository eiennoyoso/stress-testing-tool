SHELL=bash

GOPATH=$(shell go env GOPATH)

default: build

deps:
	go get -v -t -d ./...

build: deps
	CGO_ENABLED=0 go build -v -x -a $(LDFLAGS) -o $(CURDIR)/bin/$(BINARY_NAME)
	chmod +x $(CURDIR)/bin/$(BINARY_NAME)

clean:
	rm -rf $(CURDIR)/bin/*
	rm -rf $(CURDIR)/ui/assets.go
	rm -rf $(CURDIR)/ui/assets/node_modules/*
	rm -rf $(CURDIR)/ui/assets/dist/*
	go clean -i -cache

# Install binary locally
install:
	cp $(CURDIR)/bin/$(BINARY_NAME) /usr/local/bin

docker-login:
    ifdef DOCKER_REGISTRY_USERNAME
		@echo "Logged in to Docker Hub as " $(DOCKER_REGISTRY_USERNAME)
    else
		docker login
    endif

# build docker image from latest github tag
docker-build:
	@echo "Building docker image version: " $(GITHUB_LATEST_VERSION)
	docker build \
		--tag sokil/ltt:latest \
		-f ./Dockerfile .

docker-publish: docker-login
	docker push sokil/ltt:latest
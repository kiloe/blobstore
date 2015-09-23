DOCKER_TAG_NAME ?= beta

GO := GOPATH=$(shell pwd) go

all: bin/blobstore

bin/blobstore: $(wildcard src/**/*.go)
	mkdir -p bin
	$(GO) build -tags netgo -installsuffix netgo -o $@ --ldflags "--extldflags '-static'" blobstore

node_modules: package.json
	npm install

public/index.js: $(wildcard src/**/*.js)
	npm run compile

build: bin/blobstore
	docker build -t kiloe/blobstore:$(DOCKER_TAG_NAME) .

release: build
	docker push kiloe/blobstore:$(DOCKER_TAG_NAME)

test:
	$(GO) test blob
	$(GO) test blobstore

clean:
	rm -f bin/blobstore

.PHONY: clean all test build release

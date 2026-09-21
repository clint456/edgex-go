#
# Copyright 2022-2023 Intel Corporation
# Copyright (c) 2018 Cavium
# Copyright (C) 2024 IOTech Ltd
#
# SPDX-License-Identifier: Apache-2.0
#

.PHONY: build clean unittest hadolint lint test docker run sbom docker-fuzz fuzz-test-command fuzz-test-data

# change the following boolean flag to include or exclude the delayed start libs for builds for core services
INCLUDE_DELAYED_START_BUILD_CORE:="false"

# change the following boolean flag to enable or disable the Full RELRO (RELocation Read Only) for linux ELF (Executable and Linkable Format) binaries
ENABLE_FULL_RELRO=true
# change the following boolean flag to enable or disable PIE for linux binaries which is needed for ASLR (Address Space Layout Randomization) on Linux, the ASLR support on Windows is enabled by default
ENABLE_PIE=true

GO=go

DOCKERS= \
	docker_core_data \
	docker_core_metadata \
	docker_core_command  \
	docker_core_common_config \
	docker_core_keeper

.PHONY: $(DOCKERS)

MICROSERVICES= \
	cmd/core-data/core-data \
	cmd/core-metadata/core-metadata \
	cmd/core-command/core-command \
	cmd/core-common-config-bootstrapper/core-common-config-bootstrapper \
	cmd/core-keeper/core-keeper

.PHONY: $(MICROSERVICES)

VERSION=$(shell cat ./VERSION 2>/dev/null || echo 0.0.0)
DOCKER_TAG=$(VERSION)-dev

ifeq ($(ENABLE_FULL_RELRO), true)
	ENABLE_FULL_RELRO_GOFLAGS = -bindnow
endif

GOFLAGS=-ldflags "-s -w -X github.com/edge-hy/edgex-go.Version=$(VERSION) $(ENABLE_FULL_RELRO_GOFLAGS)" -trimpath -mod=readonly
GOTESTFLAGS?=-race

ifeq ($(ENABLE_PIE), true)
	GOFLAGS += -buildmode=pie
endif

GIT_SHA=$(shell git rev-parse HEAD)

ARCH=$(shell uname -m)

GO_VERSION=$(shell grep '^go [0-9].[0-9]*' go.mod | cut -d' ' -f 2)

# DO NOT change the following flag, as it is automatically set based on the boolean switch INCLUDE_DELAYED_START_BUILD_CORE
NON_DELAYED_START_GO_BUILD_TAG_FOR_CORE:=non_delayedstart
ifeq ($(INCLUDE_DELAYED_START_BUILD_CORE),"true")
	NON_DELAYED_START_GO_BUILD_TAG_FOR_CORE:=
endif

# Base docker image to speed up local builds
LOCAL_CACHE_IMAGE_BASE=edgex-go-local-cache-base
LOCAL_CACHE_IMAGE=edgex-go-local-cache

# Go module proxy / checksum database used *inside* the build containers.
# The Go default (proxy.golang.org) redirects module zips to
# storage.googleapis.com, which is often unreachable from mainland China and
# fails with "net/http: TLS handshake timeout" during `go mod download`.
# Override on the command line if you prefer another mirror, e.g.
#   make docker GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
GOPROXY ?= https://goproxy.cn,https://goproxy.io,direct
GOSUMDB ?= sum.golang.google.cn

# Alpine package mirror used inside the build containers (empty = distro default).
ALPINE_MIRROR ?= mirrors.aliyun.com

# Extra arguments for every `docker build` of the base cache image, e.g.
#   make docker DOCKER_BUILD_EXTRA="--network=host"
# which works around TLS handshake timeouts caused by a bridge MTU mismatch.
DOCKER_BUILD_EXTRA ?=

build: $(MICROSERVICES)

build-nats:
	make -e ADD_BUILD_TAGS=include_nats_messaging build

build-noziti:
	make -e ADD_BUILD_TAGS=no_openziti build

tidy:
	$(GO) mod tidy

core: metadata data command

metadata: cmd/core-metadata/core-metadata
cmd/core-metadata/core-metadata:
	$(GO) build -tags "$(ADD_BUILD_TAGS) $(NON_DELAYED_START_GO_BUILD_TAG_FOR_CORE)" $(GOFLAGS) -o $@ ./cmd/core-metadata

data: cmd/core-data/core-data
cmd/core-data/core-data:
	$(GO) build -tags "$(ADD_BUILD_TAGS) $(NON_DELAYED_START_GO_BUILD_TAG_FOR_CORE)" $(GOFLAGS) -o $@ ./cmd/core-data

command: cmd/core-command/core-command
cmd/core-command/core-command:
	$(GO) build -tags "$(ADD_BUILD_TAGS) $(NON_DELAYED_START_GO_BUILD_TAG_FOR_CORE)" $(GOFLAGS) -o $@ ./cmd/core-command

common-config: cmd/core-common-config-bootstrapper/core-common-config-bootstrapper
cmd/core-common-config-bootstrapper/core-common-config-bootstrapper:
	$(GO) build -tags "$(ADD_BUILD_TAGS) $(NON_DELAYED_START_GO_BUILD_TAG_FOR_CORE)" $(GOFLAGS) -o $@ ./cmd/core-common-config-bootstrapper

keeper: cmd/core-keeper/core-keeper
cmd/core-keeper/core-keeper:
	$(GO) build -tags "$(ADD_BUILD_TAGS) $(NON_DELAYED_START_GO_BUILD_TAG_FOR_CORE)" $(GOFLAGS) -o $@ ./cmd/core-keeper

clean:
	rm -f $(MICROSERVICES)

unittest:
	$(GO) test $(GOTESTFLAGS) -coverprofile=coverage.out ./...

hadolint:
	if which hadolint > /dev/null ; then hadolint --config .hadolint.yml `find * -type f -name 'Dockerfile*' -print` ; elif test "${ARCH}" = "x86_64" && which docker > /dev/null ; then docker run --rm -v `pwd`:/host:ro,z --entrypoint /bin/hadolint hadolint/hadolint:latest --config /host/.hadolint.yml `find * -type f -name 'Dockerfile*' | xargs -i echo '/host/{}'` ; fi
	
lint:
	@which golangci-lint >/dev/null || echo "WARNING: go linter not installed. To install, run make install-lint"
	@if [ "z${ARCH}" = "zx86_64" ] && which golangci-lint >/dev/null ; then echo "running golangci-lint"; golangci-lint version; go version; golangci-lint cache clean; golangci-lint run --verbose --config .golangci.yml ; else echo "WARNING: Linting skipped (not on x86_64 or linter not installed)"; fi

install-lint:
	sudo curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.5.0

test: unittest hadolint lint
	$(GO) vet ./...
	gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")
	[ "`gofmt -l $$(find . -type f -name '*.go'| grep -v "/vendor/")`" = "" ]
	./bin/test-attribution-txt.sh

docker: $(DOCKERS)

docker-nats:
	make -e ADD_BUILD_TAGS=include_nats_messaging docker

docker-noziti:
	make -e ADD_BUILD_TAGS=no_openziti docker

clean_docker_base:
	docker rmi -f $(LOCAL_CACHE_IMAGE) $(LOCAL_CACHE_IMAGE_BASE) 

docker_base:
	echo "Building local cache image";\
	DOCKERFILE=$$(mktemp); \
	{ \
		echo "FROM golang:$(GO_VERSION)-alpine"; \
		if [ -n "$(ALPINE_MIRROR)" ]; then \
			echo "RUN sed -i 's#dl-cdn.alpinelinux.org#$(ALPINE_MIRROR)#g' /etc/apk/repositories"; \
		fi; \
		echo "RUN apk add --update --no-cache make git"; \
		echo "WORKDIR /edgex-go"; \
		echo "COPY go.mod go.sum ./"; \
		echo "ENV GOPROXY=$(GOPROXY) GOSUMDB=$(GOSUMDB) GOTOOLCHAIN=local"; \
		echo "RUN go mod download || (echo 'retry 1'; sleep 5; go mod download) || (echo 'retry 2'; sleep 10; go mod download)"; \
	} > $$DOCKERFILE; \
	echo "Using Dockerfile:"; cat $$DOCKERFILE; \
	docker build $(DOCKER_BUILD_EXTRA) \
		--build-arg http_proxy --build-arg https_proxy \
		--build-arg HTTP_PROXY --build-arg HTTPS_PROXY \
		-t $(LOCAL_CACHE_IMAGE) -f $$DOCKERFILE .; \
	status=$$?; rm -f $$DOCKERFILE; exit $$status

dcore: dmetadata ddata dcommand

dmetadata: docker_core_metadata
docker_core_metadata: docker_base
	docker build \
		--build-arg ADD_BUILD_TAGS=$(ADD_BUILD_TAGS) \
		--build-arg http_proxy \
		--build-arg https_proxy \
		--build-arg BUILDER_BASE=$(LOCAL_CACHE_IMAGE) \
		--build-arg ALPINE_MIRROR=$(ALPINE_MIRROR) \
		-f cmd/core-metadata/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t edge-hy/core-metadata:$(GIT_SHA) \
		-t edge-hy/core-metadata:$(DOCKER_TAG) \
		.

ddata: docker_core_data
docker_core_data: docker_base
	docker build \
		--build-arg ADD_BUILD_TAGS=$(ADD_BUILD_TAGS) \
		--build-arg http_proxy \
		--build-arg https_proxy \
		--build-arg BUILDER_BASE=$(LOCAL_CACHE_IMAGE) \
		--build-arg ALPINE_MIRROR=$(ALPINE_MIRROR) \
		-f cmd/core-data/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t edge-hy/core-data:$(GIT_SHA) \
		-t edge-hy/core-data:$(DOCKER_TAG) \
		.

dcommand: docker_core_command
docker_core_command: docker_base
	docker build \
		--build-arg ADD_BUILD_TAGS=$(ADD_BUILD_TAGS) \
		--build-arg http_proxy \
		--build-arg https_proxy \
		--build-arg BUILDER_BASE=$(LOCAL_CACHE_IMAGE) \
		--build-arg ALPINE_MIRROR=$(ALPINE_MIRROR) \
		-f cmd/core-command/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t edge-hy/core-command:$(GIT_SHA) \
		-t edge-hy/core-command:$(DOCKER_TAG) \
		.

dcommon-config: docker_core_common_config
docker_core_common_config: docker_base
	docker build \
		--build-arg ADD_BUILD_TAGS=$(ADD_BUILD_TAGS) \
		--build-arg http_proxy \
		--build-arg https_proxy \
		--build-arg BUILDER_BASE=$(LOCAL_CACHE_IMAGE) \
		--build-arg ALPINE_MIRROR=$(ALPINE_MIRROR) \
		-f cmd/core-common-config-bootstrapper/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t edge-hy/core-common-config-bootstrapper:$(GIT_SHA) \
		-t edge-hy/core-common-config-bootstrapper:$(DOCKER_TAG) \
		.

dkeeper: docker_core_keeper
docker_core_keeper: docker_base
	docker build \
		--build-arg ADD_BUILD_TAGS=$(ADD_BUILD_TAGS) \
		--build-arg http_proxy \
		--build-arg https_proxy \
		--build-arg BUILDER_BASE=$(LOCAL_CACHE_IMAGE) \
		--build-arg ALPINE_MIRROR=$(ALPINE_MIRROR) \
		-f cmd/core-keeper/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t edge-hy/core-keeper:$(GIT_SHA) \
		-t edge-hy/core-keeper:$(DOCKER_TAG) \
		.

vendor:
	$(GO) mod vendor

sbom:
	docker run -it --rm \
		-v "$$PWD:/edgex-go" -v "$$PWD/sbom:/sbom" \
		spdx/spdx-sbom-generator -p /edgex-go/ -o /sbom/ --include-license-text true

docker-fuzz:
	docker build -t fuzz-edgex-go:latest -f fuzz_test/Dockerfile.fuzz .

fuzz-test-command:
# not joining the edgex-network due to swagger file url pointing to localhost for fuzz testing in the container
	docker run --net host --rm -v "$$PWD/fuzz_test/fuzz_results:/fuzz_results" fuzz-edgex-go:latest core-command /restler-fuzzer/openapi/core-command.yaml

fuzz-test-data:
# not joining the edgex-network due to swagger file url pointing to localhost for fuzz testing in the container
	docker run --net host --rm -v "$$PWD/fuzz_test/fuzz_results:/fuzz_results" fuzz-edgex-go:latest core-data /restler-fuzzer/openapi/core-data.yaml

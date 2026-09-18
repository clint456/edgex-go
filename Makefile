#
# Copyright 2022-2023 Intel Corporation
# Copyright (c) 2018 Cavium
# Copyright (C) 2024 IOTech Ltd
#
# SPDX-License-Identifier: Apache-2.0
#

.PHONY: build clean unittest hadolint lint test docker run sbom docker-fuzz fuzz-test-command fuzz-test-data fuzz-test-notifications

# change the following boolean flag to include or exclude the delayed start libs for builds for most of core services except support services
INCLUDE_DELAYED_START_BUILD_CORE:="false"
# change the following boolean flag to include or exclude the delayed start libs for builds for support services exculsively
INCLUDE_DELAYED_START_BUILD_SUPPORT:="true"

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
	docker_core_keeper \
	docker_support_notifications \
	docker_support_scheduler \
	docker_security_proxy_auth \
	docker_security_proxy_setup \
	docker_security_secretstore_setup \
	docker_security_bootstrapper \
	docker_security_spire_server \
	docker_security_spire_agent \
	docker_security_spire_config \
	docker_security_spiffe_token_provider

.PHONY: $(DOCKERS)

MICROSERVICES= \
	cmd/core-data/core-data \
	cmd/core-metadata/core-metadata \
	cmd/core-command/core-command \
	cmd/core-common-config-bootstrapper/core-common-config-bootstrapper \
	cmd/core-keeper/core-keeper \
	cmd/support-notifications/support-notifications \
	cmd/support-scheduler/support-scheduler \
	cmd/security-proxy-auth/security-proxy-auth \
	cmd/security-secretstore-setup/security-secretstore-setup \
	cmd/security-file-token-provider/security-file-token-provider \
	cmd/secrets-config/secrets-config \
	cmd/security-bootstrapper/security-bootstrapper \
	cmd/security-spiffe-token-provider/security-spiffe-token-provider

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
NON_DELAYED_START_GO_BUILD_TAG_FOR_SUPPORT:=
ifeq ($(INCLUDE_DELAYED_START_BUILD_SUPPORT),"false")
	NON_DELAYED_START_GO_BUILD_TAG_FOR_SUPPORT:=non_delayedstart
endif

NO_MESSAGEBUS_GO_BUILD_TAG:=no_messagebus

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

# SPIRE base images (used by the three security-spire-* images).
# ghcr.io serves image blobs from pkg-containers.githubusercontent.com, which is
# frequently reset from mainland China ("failed to resolve source metadata ... EOF").
# Point SPIRE_REGISTRY at a ghcr mirror to build those images, e.g.
#   make docker SPIRE_REGISTRY=ghcr.m.daocloud.io
# A mirror is a third party and can serve different content than ghcr.io, so only
# use one you trust (these images are referenced by tag, not by digest).
SPIRE_REGISTRY ?= ghcr.io
SPIRE_VERSION ?= 1.13.3
SPIRE_SERVER_IMAGE ?= $(SPIRE_REGISTRY)/spiffe/spire-server:$(SPIRE_VERSION)
SPIRE_AGENT_IMAGE ?= $(SPIRE_REGISTRY)/spiffe/spire-agent:$(SPIRE_VERSION)

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

support: notifications scheduler

notifications: cmd/support-notifications/support-notifications
cmd/support-notifications/support-notifications:
	$(GO) build -tags "$(ADD_BUILD_TAGS) $(NON_DELAYED_START_GO_BUILD_TAG_FOR_SUPPORT)" $(GOFLAGS) -o $@ ./cmd/support-notifications

scheduler: cmd/support-scheduler/support-scheduler
cmd/support-scheduler/support-scheduler:
	$(GO) build -tags "$(ADD_BUILD_TAGS) $(NON_DELAYED_START_GO_BUILD_TAG_FOR_SUPPORT)" $(GOFLAGS) -o $@ ./cmd/support-scheduler

proxy: cmd/security-proxy-setup/security-proxy-setup
cmd/security-proxy-setup/security-proxy-setup:
	$(GO) build -tags "$(NO_MESSAGEBUS_GO_BUILD_TAG) $(NON_DELAYED_START_GO_BUILD_TAG_FOR_CORE)" $(GOFLAGS) -o ./cmd/security-proxy-setup/security-proxy-setup ./cmd/security-proxy-setup

authproxy: cmd/security-proxy-auth/security-proxy-auth
cmd/security-proxy-auth/security-proxy-auth:
	$(GO) build -tags "$(NO_MESSAGEBUS_GO_BUILD_TAG) $(NON_DELAYED_START_GO_BUILD_TAG_FOR_CORE)" $(GOFLAGS) -o ./cmd/security-proxy-auth/security-proxy-auth ./cmd/security-proxy-auth

secretstore: cmd/security-secretstore-setup/security-secretstore-setup
cmd/security-secretstore-setup/security-secretstore-setup:
	$(GO) build -tags "$(NO_MESSAGEBUS_GO_BUILD_TAG) $(NON_DELAYED_START_GO_BUILD_TAG_FOR_CORE)" $(GOFLAGS) -o ./cmd/security-secretstore-setup/security-secretstore-setup ./cmd/security-secretstore-setup

token: cmd/security-file-token-provider/security-file-token-provider
cmd/security-file-token-provider/security-file-token-provider:
	$(GO) build -tags "$(NO_MESSAGEBUS_GO_BUILD_TAG) $(NON_DELAYED_START_GO_BUILD_TAG_FOR_CORE)" $(GOFLAGS) -o ./cmd/security-file-token-provider/security-file-token-provider ./cmd/security-file-token-provider

secrets-config: cmd/secrets-config/secrets-config
cmd/secrets-config/secrets-config:
	$(GO) build -tags "$(NO_MESSAGEBUS_GO_BUILD_TAG) $(NON_DELAYED_START_GO_BUILD_TAG_FOR_CORE)" $(GOFLAGS) -o ./cmd/secrets-config ./cmd/secrets-config

bootstrapper: cmd/security-bootstrapper/security-bootstrapper
cmd/security-bootstrapper/security-bootstrapper:
	$(GO) build -tags "$(NO_MESSAGEBUS_GO_BUILD_TAG) $(NON_DELAYED_START_GO_BUILD_TAG_FOR_CORE)" $(GOFLAGS) -o ./cmd/security-bootstrapper/security-bootstrapper ./cmd/security-bootstrapper

spiffetp: cmd/security-spiffe-token-provider/security-spiffe-token-provider
cmd/security-spiffe-token-provider/security-spiffe-token-provider:
	$(GO) build -tags "$(NO_MESSAGEBUS_GO_BUILD_TAG) $(NON_DELAYED_START_GO_BUILD_TAG_FOR_CORE)" $(GOFLAGS) -o $@ ./cmd/security-spiffe-token-provider

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

dsupport: dnotifications dscheduler dscheduler

dnotifications: docker_support_notifications
docker_support_notifications: docker_base
	docker build \
		--build-arg ADD_BUILD_TAGS=$(ADD_BUILD_TAGS) \
		--build-arg http_proxy \
		--build-arg https_proxy \
		--build-arg BUILDER_BASE=$(LOCAL_CACHE_IMAGE) \
		--build-arg ALPINE_MIRROR=$(ALPINE_MIRROR) \
		-f cmd/support-notifications/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t edge-hy/support-notifications:$(GIT_SHA) \
		-t edge-hy/support-notifications:$(DOCKER_TAG) \
		.

dscheduler: docker_support_scheduler
docker_support_scheduler: docker_base
	docker build \
		--build-arg ADD_BUILD_TAGS=$(ADD_BUILD_TAGS) \
		--build-arg http_proxy \
		--build-arg https_proxy \
		--build-arg BUILDER_BASE=$(LOCAL_CACHE_IMAGE) \
		--build-arg ALPINE_MIRROR=$(ALPINE_MIRROR) \
		-f cmd/support-scheduler/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t edge-hy/support-scheduler:$(GIT_SHA) \
		-t edge-hy/support-scheduler:$(DOCKER_TAG) \
		.

dproxya: docker_security_proxy_auth
docker_security_proxy_auth: docker_base
	docker build \
		--build-arg http_proxy \
		--build-arg https_proxy \
		--build-arg BUILDER_BASE=$(LOCAL_CACHE_IMAGE) \
		--build-arg ALPINE_MIRROR=$(ALPINE_MIRROR) \
		-f cmd/security-proxy-auth/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t edge-hy/security-proxy-auth:$(GIT_SHA) \
		-t edge-hy/security-proxy-auth:$(DOCKER_TAG) \
		.

dproxys: docker_security_proxy_setup
docker_security_proxy_setup: docker_base
	docker build \
		--build-arg http_proxy \
		--build-arg https_proxy \
		--build-arg BUILDER_BASE=$(LOCAL_CACHE_IMAGE) \
		--build-arg ALPINE_MIRROR=$(ALPINE_MIRROR) \
		-f cmd/security-proxy-setup/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t edge-hy/security-proxy-setup:$(GIT_SHA) \
		-t edge-hy/security-proxy-setup:$(DOCKER_TAG) \
		.
dsecretstore: docker_security_secretstore_setup
docker_security_secretstore_setup: docker_base
		docker build \
		--build-arg http_proxy \
		--build-arg https_proxy \
		--build-arg BUILDER_BASE=$(LOCAL_CACHE_IMAGE) \
		--build-arg ALPINE_MIRROR=$(ALPINE_MIRROR) \
		-f cmd/security-secretstore-setup/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t edge-hy/security-secretstore-setup:$(GIT_SHA) \
		-t edge-hy/security-secretstore-setup:$(DOCKER_TAG) \
		.

dbootstrapper: docker_security_bootstrapper
docker_security_bootstrapper: docker_base
	docker build \
		--build-arg http_proxy \
		--build-arg https_proxy \
		--build-arg BUILDER_BASE=$(LOCAL_CACHE_IMAGE) \
		--build-arg ALPINE_MIRROR=$(ALPINE_MIRROR) \
		-f cmd/security-bootstrapper/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t edge-hy/security-bootstrapper:$(GIT_SHA) \
		-t edge-hy/security-bootstrapper:$(DOCKER_TAG) \
		.

dspires: docker_security_spire_server
docker_security_spire_server: docker_base
	docker build \
		--build-arg http_proxy \
		--build-arg https_proxy \
		--build-arg BUILDER_BASE=$(LOCAL_CACHE_IMAGE) \
		--build-arg ALPINE_MIRROR=$(ALPINE_MIRROR) \
		--build-arg SPIRE_SERVER_IMAGE=$(SPIRE_SERVER_IMAGE) \
		-f cmd/security-spire-server/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t edge-hy/security-spire-server:$(GIT_SHA) \
		-t edge-hy/security-spire-server:$(DOCKER_TAG) \
		.

dspirea: docker_security_spire_agent
docker_security_spire_agent: docker_base
	docker build \
		--build-arg http_proxy \
		--build-arg https_proxy \
		--build-arg BUILDER_BASE=$(LOCAL_CACHE_IMAGE) \
		--build-arg ALPINE_MIRROR=$(ALPINE_MIRROR) \
		--build-arg SPIRE_SERVER_IMAGE=$(SPIRE_SERVER_IMAGE) \
		--build-arg SPIRE_AGENT_IMAGE=$(SPIRE_AGENT_IMAGE) \
		-f cmd/security-spire-agent/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t edge-hy/security-spire-agent:$(GIT_SHA) \
		-t edge-hy/security-spire-agent:$(DOCKER_TAG) \
		.

dspirec: docker_security_spire_config
docker_security_spire_config: docker_base
	docker build \
		--build-arg http_proxy \
		--build-arg https_proxy \
		--build-arg BUILDER_BASE=$(LOCAL_CACHE_IMAGE) \
		--build-arg ALPINE_MIRROR=$(ALPINE_MIRROR) \
		--build-arg SPIRE_SERVER_IMAGE=$(SPIRE_SERVER_IMAGE) \
		-f cmd/security-spire-config/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t edge-hy/security-spire-config:$(GIT_SHA) \
		-t edge-hy/security-spire-config:$(DOCKER_TAG) \
		.

dspiffetp: docker_security_spiffe_token_provider
docker_security_spiffe_token_provider: docker_base
	docker build \
		--build-arg http_proxy \
		--build-arg https_proxy \
		--build-arg BUILDER_BASE=$(LOCAL_CACHE_IMAGE) \
		--build-arg ALPINE_MIRROR=$(ALPINE_MIRROR) \
		-f cmd/security-spiffe-token-provider/Dockerfile \
		--label "git_sha=$(GIT_SHA)" \
		-t edge-hy/security-spiffe-token-provider:$(GIT_SHA) \
		-t edge-hy/security-spiffe-token-provider:$(DOCKER_TAG) \
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

fuzz-test-notifications:
# not joining the edgex-network due to swagger file url pointing to localhost for fuzz testing in the container
	docker run --net host --rm -v "$$PWD/fuzz_test/fuzz_results:/fuzz_results" fuzz-edgex-go:latest support-notifications /restler-fuzzer/openapi/support-notifications.yaml
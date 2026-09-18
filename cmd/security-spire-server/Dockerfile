#  ----------------------------------------------------------------------------------
#  Copyright 2022 Intel Corporation
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.
# 
#  ----------------------------------------------------------------------------------

# Build utility container
ARG BUILDER_BASE=golang:1.25-alpine3.22
ARG SPIRE_SERVER_IMAGE=ghcr.io/spiffe/spire-server:1.13.3
FROM ${BUILDER_BASE} AS builder

FROM ${SPIRE_SERVER_IMAGE} AS spire_server

# Deployment image
FROM alpine:3.22

ARG ALPINE_MIRROR=mirrors.aliyun.com
# Alpine dl-cdn/dl-4 mirrors are flaky (TLS resets / 403); use the mirror from ALPINE_MIRROR.
RUN [ -z "${ALPINE_MIRROR}" ] || sed -i "s#dl-cdn[.]alpinelinux[.]org#${ALPINE_MIRROR}#g; s#dl-4[.]alpinelinux[.]org#${ALPINE_MIRROR}#g" /etc/apk/repositories

LABEL license='SPDX-License-Identifier: Apache-2.0' \
      copyright='Copyright (c) 2022 Intel Corporation'

RUN apk update && apk --no-cache --update add dumb-init openssl gcompat
# Ensure using latest versions of all installed packages to avoid any recent CVEs
RUN apk --no-cache upgrade

COPY --from=spire_server /opt/spire/bin/spire-server /usr/local/bin

COPY Attribution.txt /
COPY security.txt /

COPY cmd/security-spire-server/docker-entrypoint.sh /usr/local/bin/
COPY cmd/security-spire-server/server.conf /usr/local/etc/spire/server.conf.tpl
COPY cmd/security-spire-server/openssl.conf /usr/local/etc/

ENTRYPOINT [ "/usr/bin/dumb-init" ]
CMD [ "--verbose", "docker-entrypoint.sh" ]

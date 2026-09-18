#  ----------------------------------------------------------------------------------
#  Copyright (C) 2024-2025 IOTech Ltd
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

# Docker image for Golang Core Keeper micro service
ARG BUILDER_BASE=golang:1.25-alpine3.22
FROM ${BUILDER_BASE} AS builder

ARG ALPINE_MIRROR=mirrors.aliyun.com
# Alpine dl-cdn/dl-4 mirrors are flaky (TLS resets / 403); use the mirror from ALPINE_MIRROR.
RUN [ -z "${ALPINE_MIRROR}" ] || sed -i "s#dl-cdn[.]alpinelinux[.]org#${ALPINE_MIRROR}#g; s#dl-4[.]alpinelinux[.]org#${ALPINE_MIRROR}#g" /etc/apk/repositories

ARG ADD_BUILD_TAGS=""

WORKDIR /edgex-go

RUN apk add --update --no-cache make git

COPY go.mod vendor* ./
RUN [ ! -d "vendor" ] && go mod download all || echo "skipping..."

COPY . .
RUN make -e ADD_BUILD_TAGS=$ADD_BUILD_TAGS cmd/core-keeper/core-keeper

#Next image - Copy built Go binary into new workspace
FROM alpine:3.22

ARG ALPINE_MIRROR=mirrors.aliyun.com
# Alpine dl-cdn/dl-4 mirrors are flaky (TLS resets / 403); use the mirror from ALPINE_MIRROR.
RUN [ -z "${ALPINE_MIRROR}" ] || sed -i "s#dl-cdn[.]alpinelinux[.]org#${ALPINE_MIRROR}#g; s#dl-4[.]alpinelinux[.]org#${ALPINE_MIRROR}#g" /etc/apk/repositories

LABEL license='SPDX-License-Identifier: Apache-2.0' \
      copyright='Copyright (c) 2024: IOTech'

ENV APP_PORT=59890
#expose core keeper port
EXPOSE $APP_PORT

RUN apk add --update --no-cache dumb-init
COPY --from=builder /edgex-go/Attribution.txt /
COPY --from=builder /edgex-go/cmd/core-keeper/core-keeper /
COPY --from=builder /edgex-go/cmd/core-keeper/res/configuration.yaml /res/configuration.yaml

ENTRYPOINT ["/core-keeper"]

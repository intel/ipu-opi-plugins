# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2024 Intel Corporation
FROM golang:1.22-alpine@sha256:cdc86d9f363e8786845bea2040312b4efa321b828acdeb26f393faa864d887b0 as builder
ARG TARGETOS
ARG TARGETARCH

ENV HTTP_PROXY $http_proxy
ENV HTTPS_PROXY $https_proxy

RUN apk add --no-cache --virtual build-dependencies build-base
COPY . /usr/src/ipu-opi-plugin
WORKDIR /usr/src/ipu-opi-plugin
RUN go mod download
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o bin/ipuplugin ipuplugin/main.go



# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2024 Intel Corporation

FROM registry.access.redhat.com/ubi9/ubi:9.3-1610

ENV HTTP_PROXY $http_proxy
ENV HTTPS_PROXY $https_proxy

COPY . /src
WORKDIR /src
RUN mkdir -pv /opt/ && tar -zxvf p4.tar.gz -C /opt/ && rm -f p4.tar.gz

COPY p4 /opt/p4/rh_mvp
COPY images/entrypoint.sh /
COPY images/es2k_skip_p4.conf.template /

RUN dnf -y install \
    kmod-28-9.el9.aarch64 \
    gettext-0.21-8.el9.aarch64 \
    python3-pip-21.2.3-7.el9_3.1.noarch \
    pciutils-3.7.0-5.el9.aarch64 \
    libconfig-1.7.2-9.el9.aarch64 \
    libnl3-3.7.0-1.el9.aarch64 \
    libedit-3.1-38.20210216cvs.el9.aarch64 \
    libatomic-11.4.1-2.1.el9.aarch64 \
    openssh-8.7p1-34.el9_3.3.aarch64 \
    && dnf clean all
RUN python3 -m pip install --no-cache-dir \
    grpcio==1.62.1 \
    protobuf==4.25.3 \
    netaddr==1.2.1 \
    /opt/p4/p4-cp-nws/share/stratum/p4runtime/p4runtime-2023.11.0-py3-none-any.whl

COPY --from=builder /usr/src/ipu-opi-plugin/bin/ipuplugin /usr/bin/
COPY p4-rh_mvp/rh_mvp.pkg /
WORKDIR /
LABEL io.k8s.display-name="IPU OPI Plugin"

ENTRYPOINT ["/entrypoint.sh"]
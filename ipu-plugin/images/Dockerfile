# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2024 Intel Corporation
FROM golang:1.23.6-alpine@sha256:f8113c4b13e2a8b3a168dceaee88ac27743cc84e959f43b9dbd2291e9c3f57a0 AS builder
RUN apk add --no-cache git
ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache --virtual build-dependencies build-base
COPY . /usr/src/ipu-opi-plugin
WORKDIR /usr/src/ipu-opi-plugin
RUN go mod download
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o bin/ipuplugin ipuplugin/main.go

FROM alpine:3@sha256:a8560b36e8b8210634f77d9f7f9efd7ffa463e380b75e2e74aff4511df3ef88c
ARG P4_NAME
ENV P4_NAME=$P4_NAME

COPY --from=builder /usr/src/ipu-opi-plugin/bin/ipuplugin /usr/bin/
COPY ${P4_NAME}/${P4_NAME}.pkg /
RUN mkdir -p /opt/p4/p4-cp-nws/bin/p4
COPY bin/p4rt-ctl /opt/p4/p4-cp-nws/bin/
#TODO: Update to newer p4runtime, for latest release.
COPY bin/p4runtime-2023.11.0/p4 /opt/p4/p4-cp-nws/bin/p4
COPY bin/p4runtime-2023.11.0/copy_p4rt_python_deps.sh /opt/p4/p4-cp-nws/bin/p4

WORKDIR /
LABEL io.k8s.display-name="IPU OPI Plugin"
ENV PYTHONUNBUFFERED=1
#Install deps for p4runtime(previously deps got installed per p4runtime wheel package's->p4runtime-2023.11.0.dist-info/METADATA)
RUN apk add -U --no-cache python3'>='3.8  py3-pip'=='23.3.1-r0 \
    py3-grpcio'=='1.59.3-r0 \
    py3-netaddr'=='0.9.0-r0

#Install deps for p4runtime(previously deps got installed per p4runtime wheel package's->p4runtime-2023.11.0.dist-info/METADATA)
RUN python3 -m pip install --no-cache-dir --break-system-packages -v "protobuf==4.25.0" "googleapis-common-protos==1.66.0"

#Note: Redhat's build system does not allow p4runtime wheel package. Manually installing python protobuf
#files(for p4runtime) under pip's installation path.
RUN chmod a+x /opt/p4/p4-cp-nws/bin/p4/copy_p4rt_python_deps.sh && /opt/p4/p4-cp-nws/bin/p4/copy_p4rt_python_deps.sh

RUN rm -rf /var/cache/apk/*
RUN apk add --update --no-cache openssh networkmanager-cli openvswitch iproute2

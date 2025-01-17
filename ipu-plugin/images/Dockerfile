# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2024 Intel Corporation
FROM golang:1.23.5-alpine@sha256:47d337594bd9e667d35514b241569f95fb6d95727c24b19468813d596d5ae596 AS builder
RUN apk add --no-cache git
ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache --virtual build-dependencies build-base
COPY . /usr/src/ipu-opi-plugin
WORKDIR /usr/src/ipu-opi-plugin
RUN go mod download
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o bin/ipuplugin ipuplugin/main.go

FROM alpine:3@sha256:c5b1261d6d3e43071626931fc004f70149baeba2c8ec672bd4f27761f8e1ad6b
ARG P4_NAME
ENV P4_NAME=$P4_NAME

COPY --from=builder /usr/src/ipu-opi-plugin/bin/ipuplugin /usr/bin/
COPY ${P4_NAME}/${P4_NAME}.pkg /
RUN mkdir -p /opt/p4/p4-cp-nws/bin/
COPY bin/p4rt-ctl /opt/p4/p4-cp-nws/bin/
#TODO: Update to newer package, according to release.
COPY bin/p4runtime-2023.11.0-py3-none-any.whl /opt/p4/p4-cp-nws/bin/

WORKDIR /
LABEL io.k8s.display-name="IPU OPI Plugin"
ENV PYTHONUNBUFFERED=1
RUN apk add -U --no-cache python3 py3-pip'=='23.3.1-r0 \
    py3-grpcio'=='1.59.3-r0 \
    py3-protobuf'=='4.24.4-r0 \
    py3-netaddr'=='0.9.0-r0
RUN python3 -m pip install --no-cache-dir --break-system-packages /opt/p4/p4-cp-nws/bin/p4runtime-2023.11.0-py3-none-any.whl

RUN rm -rf /var/cache/apk/*
RUN apk add --update --no-cache openssh networkmanager-cli openvswitch iproute2

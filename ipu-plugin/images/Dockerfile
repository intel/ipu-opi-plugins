# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2024 Intel Corporation
# Apply environment
FROM golang:1.22.5-alpine@sha256:0d3653dd6f35159ec6e3d10263a42372f6f194c3dea0b35235d72aabde86486e as builder
ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache --virtual build-dependencies build-base
RUN apk add --no-cache git
COPY . /usr/src/ipu-opi-plugin
WORKDIR /usr/src/ipu-opi-plugin
RUN go mod download
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o bin/ipuplugin ipuplugin/main.go

FROM python:3.11-alpine as builderpy
COPY bin/p4runtime-2024.8.0-py3-none-any.whl /opt/p4/p4-cp-nws/bin/
RUN apk add -U --no-cache py3-pip'=='24.0-r2 \
    g++ linux-headers python3-dev binutils
SHELL ["/bin/ash", "-o", "pipefail", "-c"]
RUN python3 -m pip install --no-cache-dir /opt/p4/p4-cp-nws/bin/p4runtime-2024.8.0-py3-none-any.whl
RUN find /usr/local/lib/python3.11/site-packages/ -name "cygrpc*" -print0 | xargs -0 strip --strip-debug

FROM alpine:3.19.4

ENV PYTHONUNBUFFERED=1
RUN apk add -U --no-cache python3 \
    py3-netaddr'=='0.9.0-r0

COPY --from=builder /usr/src/ipu-opi-plugin/bin/ipuplugin /usr/bin/
COPY --from=builderpy /usr/local/lib/python3.11/site-packages /usr/lib/python3.11/site-packages
COPY --from=builderpy /usr/local/bin /usr/local/bin
COPY p4-rh_mvp/rh_mvp.pkg /
RUN mkdir -p /opt/p4/p4-cp-nws/bin/
COPY bin/p4rt-ctl /opt/p4/p4-cp-nws/bin/

WORKDIR /
LABEL io.k8s.display-name="IPU OPI Plugin"

RUN rm -rf /var/cache/apk/*
RUN apk add --update --no-cache openssh binutils

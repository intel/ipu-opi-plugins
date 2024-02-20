# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2024 Intel Corporation
FROM golang:1.20-alpine as builder
ARG TARGETOS
ARG TARGETARCH

ENV HTTP_PROXY $http_proxy
ENV HTTPS_PROXY $https_proxy

RUN apk add --no-cache --virtual build-dependencies build-base
COPY . /usr/src/ipu-opi-plugin
WORKDIR /usr/src/ipu-opi-plugin
RUN go mod download
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o bin/ipuplugin ipuplugin/main.go


FROM alpine:3

COPY --from=builder /usr/src/ipu-opi-plugin/bin/ipuplugin /usr/bin/
WORKDIR /
LABEL io.k8s.display-name="IPU OPI Plugin"

RUN rm -rf /var/cache/apk/*

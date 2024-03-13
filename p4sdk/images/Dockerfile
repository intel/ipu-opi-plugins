# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2024 Intel Corporation
FROM --platform=linux/arm64 alpine:3@sha256:c5b1261d6d3e43071626931fc004f70149baeba2c8ec672bd4f27761f8e1ad6b as builder
ARG TARGETOS
ARG TARGETARCH

ENV HTTP_PROXY $http_proxy
ENV HTTPS_PROXY $https_proxy

RUN apk add --no-cache --virtual tar
COPY . /src
WORKDIR /src
RUN mkdir -pv /opt/p4 && tar -zxvf p4.tar.gz -C /opt/p4 && rm -f p4.tar.gz

FROM --platform=linux/arm64 alpine:3@sha256:c5b1261d6d3e43071626931fc004f70149baeba2c8ec672bd4f27761f8e1ad6b
RUN apk add --no-cache --virtual gawk gettext
COPY --from=builder /opt/p4 /opt/p4
COPY p4 /root/linux_networking
COPY images/entrypoint.sh /

LABEL io.k8s.display-name="IPU P4 SDK"

RUN rm -rf /var/cache/apk/*
WORKDIR /
ENTRYPOINT ["/entrypoint.sh"]

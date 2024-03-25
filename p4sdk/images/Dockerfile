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
ENV GLIBC_REPO=https://github.com/sgerrand/alpine-pkg-glibc
ENV GLIBC_VERSION=2.35-r1
RUN apk add --no-cache --virtual openssl libressl-dev ncurses-libs libstdc++ gawk gettext
RUN set -ex && \
    apk --update add libstdc++ curl ca-certificates openssl libressl-dev ncurses-libs gawk gettext gcompat && \
    for pkg in glibc-${GLIBC_VERSION} glibc-bin-${GLIBC_VERSION}; \
        do curl -sSL ${GLIBC_REPO}/releases/download/${GLIBC_VERSION}/${pkg}.apk -o /tmp/${pkg}.apk; done && \
    apk add --allow-untrusted /tmp/*.apk && \
    rm -v /tmp/*.apk && \
    /usr/glibc-compat/sbin/ldconfig /lib /usr/glibc-compat/lib
COPY --from=builder /opt/p4 /opt/
COPY p4 /root/linux_networking
COPY images/entrypoint.sh /
COPY images/es2k_skip_p4.conf.template /

LABEL io.k8s.display-name="IPU P4 SDK"

RUN rm -rf /var/cache/apk/*
WORKDIR /
# ENTRYPOINT ["/entrypoint.sh"]
ENTRYPOINT ["/bin/sh","-c","while true; do sleep 3000; done"]

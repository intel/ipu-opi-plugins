# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2024 Intel Corporation
FROM golang:1.22.5 as builder
ARG TARGETOS
ARG TARGETARCH

COPY . /usr/src/ipu-opi-plugin
WORKDIR /usr/src/ipu-opi-plugin
RUN go mod download
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o bin/ipuplugin ipuplugin/main.go

FROM registry.access.redhat.com/ubi9/ubi:9.4 as builder-ovs

RUN dnf -y install git \
                   autoconf \
                   automake \
                   libtool && dnf clean all
WORKDIR /
RUN git clone https://github.com/openvswitch/ovs.git
WORKDIR /ovs
RUN ./boot.sh && ./configure && make -j && make install


FROM registry.access.redhat.com/ubi9/ubi:9.4
ARG TARGETOS
ARG TARGETARCH
ARG P4_NAME
ENV P4_NAME $P4_NAME
ENV ARCHSUFFIX="aarch64"
ENV PKG_NAME=""

COPY --from=builder /usr/src/ipu-opi-plugin/bin/ipuplugin /usr/bin/
COPY --from=builder-ovs /usr/local/bin /usr/local/bin
COPY ${P4_NAME}/${P4_NAME}.pkg /
RUN mkdir -p /opt/p4/p4-cp-nws/bin/
COPY bin/p4rt-ctl /opt/p4/p4-cp-nws/bin/
#TODO: Update to newer package, according to release.
COPY bin/p4runtime-2023.11.0-py3-none-any.whl /opt/p4/p4-cp-nws/bin/

WORKDIR /
LABEL io.k8s.display-name="IPU OPI Plugin"
ENV PYTHONUNBUFFERED=1
RUN dnf -y install python3-pip-21.2.3-8.el9 openssh && dnf clean all
RUN python3 -m pip install --no-cache-dir /opt/p4/p4-cp-nws/bin/p4runtime-2023.11.0-py3-none-any.whl \
    netaddr==1.2.1


#NetworkManager needs NetowrkManager-libnm and libnd

RUN if [ "$TARGETARCH" = "amd64" ]; then export ARCHSUFFIX="x86_64"; \
    else export ARCHSUFFIX="aarch64"; fi; \
    export PKG_NAME="libndp-1.8-4.el9.$ARCHSUFFIX.rpm"; \
    curl -L -o $PKG_NAME \
      https://mirror.stream.centos.org/9-stream/BaseOS/$ARCHSUFFIX/os/Packages/$PKG_NAME \
    && dnf -y install $PKG_NAME && rm -f $PKG_NAME && dnf clean all

RUN if [ "$TARGETARCH" = "amd64" ]; then export ARCHSUFFIX="x86_64"; \
    else export ARCHSUFFIX="aarch64"; fi; \
    export PKG_NAME="NetworkManager-libnm-1.48.10-1.el9.$ARCHSUFFIX.rpm"; \
    curl -L -o $PKG_NAME \
      https://mirror.stream.centos.org/9-stream/BaseOS/$ARCHSUFFIX/os/Packages/$PKG_NAME \
    && dnf -y install $PKG_NAME && rm -f $PKG_NAME && dnf clean all


RUN if [ "$TARGETARCH" = "amd64" ]; then export ARCHSUFFIX="x86_64"; \
    else export ARCHSUFFIX="aarch64"; fi; \
    export PKG_NAME="NetworkManager-1.48.10-1.el9.$ARCHSUFFIX.rpm"; \
    curl -L -o $PKG_NAME \
      https://mirror.stream.centos.org/9-stream/BaseOS/$ARCHSUFFIX/os/Packages/$PKG_NAME \
    && dnf -y install $PKG_NAME && rm -f $PKG_NAME && dnf clean all

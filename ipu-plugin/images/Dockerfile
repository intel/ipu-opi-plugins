# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2024 Intel Corporation
FROM registry.ci.openshift.org/ocp/builder:rhel-9-golang-1.24-openshift-4.20 AS builder
ARG TARGETOS
ARG TARGETARCH

COPY . /usr/src/ipu-opi-plugin
WORKDIR /usr/src/ipu-opi-plugin
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o bin/ipuplugin ipuplugin/main.go

FROM registry.ci.openshift.org/ocp/4.20:base-rhel9
ARG TARGETARCH
ENV PYTHONUNBUFFERED=1
ARG P4_NAME
ENV P4_NAME=$P4_NAME

WORKDIR /
COPY redhat.aarch64.repo /etc/yum.repos.d/
COPY redhat.amd64.repo /etc/yum.repos.d/
COPY --from=builder /usr/src/ipu-opi-plugin/bin/ipuplugin /usr/bin/
COPY ${P4_NAME}/${P4_NAME}.pkg /
RUN mkdir -p /opt/p4/p4-cp-nws/bin/p4
COPY bin/p4rt-ctl /opt/p4/p4-cp-nws/bin/
#TODO: Update to newer p4runtime, for latest release.
COPY bin/p4runtime-2023.11.0/p4 /opt/p4/p4-cp-nws/bin/p4
COPY bin/p4runtime-2023.11.0/copy_p4rt_python_deps.sh /opt/p4/p4-cp-nws/bin/p4

RUN mkdir -p /opt/p4/p4-cp-nws/bin/p4
RUN rm /usr/bin/python3
RUN ln -s /usr/bin/python3.11 /usr/bin/python3

WORKDIR /
LABEL io.k8s.display-name="IPU OPI Plugin"
ENV PYTHONUNBUFFERED=1
#Install deps for p4runtime(previously deps got installed per p4runtime wheel package's->p4runtime-2023.11.0.dist-info/METADATA)
RUN ART_DNF_WRAPPER_POLICY="skip" dnf install -y \
	NetworkManager openvswitch3.4 iproute python3.11 python3.11-devel openssh-clients gcc gcc-c++ --excludepkgs=iputils  \
    && python3.11 -m ensurepip --upgrade
#RUN ART_DNF_WRAPPER_POLICY="skip" dnf install -y py3-grpcio'=='1.59.3-r0 \
#    py3-netaddr'=='0.9.0-r0

#Install deps for p4runtime(previously deps got installed per p4runtime wheel package's->p4runtime-2023.11.0.dist-info/METADATA)
RUN python3 -m pip install --no-cache-dir -v "protobuf==4.25.0" "googleapis-common-protos==1.66.0"

#Note: Redhat's build system does not allow p4runtime wheel package. Manually installing python protobuf
#files(for p4runtime) under pip's installation path.
RUN chmod a+x /opt/p4/p4-cp-nws/bin/p4/copy_p4rt_python_deps.sh && /opt/p4/p4-cp-nws/bin/p4/copy_p4rt_python_deps.sh


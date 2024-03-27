# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2024 Intel Corporation

FROM registry.access.redhat.com/ubi9/ubi:9.3-1610

ENV HTTP_PROXY $http_proxy
ENV HTTPS_PROXY $https_proxy

COPY . /src
WORKDIR /src
RUN mkdir -pv /opt/ && tar -zxvf p4.tar.gz -C /opt/ && rm -f p4.tar.gz
RUN mkdir -p /usr/share/target_sys/
RUN cp /opt/p4/p4sde/share/target_sys/zlog-cfg /usr/share/target_sys/zlog-cfg
RUN mkdir -p /usr/share/stratum/es2k/
RUN cp /opt/p4/p4-cp-nws/share/stratum/es2k/es2k_port_config.pb.txt /usr/share/stratum/es2k/es2k_port_config.pb.txt

COPY p4 /root/linux_networking
COPY images/entrypoint.sh /
COPY images/es2k_skip_p4.conf.template /
RUN dnf -y install kmod gettext python3-pip pciutils libconfig libnl3 libedit libatomic
RUN python3 -m pip install grpcio protobuf==3.19.4 netaddr
RUN python3 -m pip install /opt/p4/p4-cp-nws/share/stratum/p4runtime/p4runtime-2023.11.0-py3-none-any.whl

WORKDIR /

ENTRYPOINT ["/entrypoint.sh"]

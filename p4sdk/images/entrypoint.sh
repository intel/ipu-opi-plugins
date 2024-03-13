#!/bin/sh

# Copyright 2024 Intel Corp. All Rights Reserved.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#     http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


set -e

export SDE_INSTALL=/opt/p4/p4sde
export P4CP_INSTALL=/opt/p4/p4-cp-nws
export DEPEND_INSTALL=$P4CP_INSTALL
export PROTOCOL_BUFFERS_PYTHON_IMPLEMENTATION=python


CPF_INFO_FILE=cpf_info_file.txt
CONF_DIR=/usr/share/stratum/es2k
CONF_FILE=es2k_skip_p4.conf
$SDE_INSTALL/bin/vfio_bind.sh 8086:1453 > $CPF_INFO_FILE 2>&1
IOMMU_GROUP=$(awk '{print $5}' $CPF_INFO_FILE)
CPF_BDF=$(awk '{print $2}' $CPF_INFO_FILE)
export IOMMU_GROUP
export CPF_BDF

mkdir -p $CONF_DIR
envsubst < $CONF_FILE.template > $CONF_DIR/$CONF_FILE

$P4CP_INSTALL/bin/tdi_pipeline_builder \
    --p4c_conf_file=/usr/share/stratum/es2k/es2k_skip_p4.conf \
    --bf_pipeline_config_binary_file=/root/linux_networking/linux_networking.pb.bin

# Set hugepages
mkdir -p /dev/hugepages
mount -t hugetlbfs -o pagesize=2M none /dev/hugepages || true
echo 512 > /sys/devices/system/node/node0/hugepages/hugepages-2048kB/nr_hugepages

# Start Infrap4d 
/opt/p4/p4-cp-nws/sbin/infrap4d -grpc_open_insecure_mode=true -nodetach -disable_krnlmon=true  > /tmp/infrap4d.log 2>&1 &

# # Allow infrap4d to run
sleep 30

# Set the Forwarding pipeline
$P4CP_INSTALL/bin/p4rt-ctl set-pipe br0 /root/linux_networking/linux_networking.pb.bin /root/linux_networking/linux_networking.p4info.txt

# Monitor infrap4d logs
tail -f /tmp/infrap4d.log


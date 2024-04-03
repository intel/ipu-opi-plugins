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


set -ex

export SDE_INSTALL=/opt/p4/p4sde
export P4CP_INSTALL=/opt/p4/p4-cp-nws
export DEPEND_INSTALL=$P4CP_INSTALL
export PROTOCOL_BUFFERS_PYTHON_IMPLEMENTATION=python

# Allow set_br_pipe on a separate process so that we can sleep 30s for infrap4d to come up
func_set_br_pipe(){
    # Wait for 30s
    sleep 30
    # Set the Forwarding pipeline
    $P4CP_INSTALL/bin/p4rt-ctl set-pipe br0 /opt/p4/rh_mvp/rh_mvp.pb.bin /opt/p4/rh_mvp/rh_mvp.p4info.txt
}


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

touch /opt/p4/rh_mvp/tofino.bin
$P4CP_INSTALL/bin/tdi_pipeline_builder \
    --p4c_conf_file=/usr/share/stratum/es2k/es2k_skip_p4.conf \
    --bf_pipeline_config_binary_file=/opt/p4/rh_mvp/rh_mvp.pb.bin

# Set hugepages
mkdir -p /dev/hugepages
mount -t hugetlbfs -o pagesize=2M none /dev/hugepages || true
echo 512 > /sys/devices/system/node/node0/hugepages/hugepages-2048kB/nr_hugepages

# Copy files required by infrap4d
mkdir -p /usr/share/target_sys/
cp /opt/p4/p4sde/share/target_sys/zlog-cfg /usr/share/target_sys/zlog-cfg
mkdir -p /usr/share/stratum/es2k/
cp /opt/p4/p4-cp-nws/share/stratum/es2k/es2k_port_config.pb.txt /usr/share/stratum/es2k/es2k_port_config.pb.txt

# Start Infrap4d
func_set_br_pipe &
/opt/p4/p4-cp-nws/sbin/infrap4d -grpc_open_insecure_mode=true -nodetach -disable_krnlmon=true

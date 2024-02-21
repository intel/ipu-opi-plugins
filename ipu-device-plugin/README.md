# ipu-device-plugin

The ipu-device-plugin is a Kubernetes device plugin for discovering and advertising networking resources in the form of:
- Adaptive Physical Functions (APFs)

which are available on an IPU host

## Table of Contents

- [IPU Device Plugin](#ipu-device-plugin)
- [Table of Contents](#table-of-contents)
- [Prerequisites](#prerequisites)
- [Install NF-CNI](#install-nf-cni)
- [Build IPU Device Plugin image locally](#build-ipu-device-plugin-image-locally)
- [Install IPU Device Plugin](#install-ipu-device-plugin)
- [Deploy config map](#deploy-config-map)
- [Deploy daemonset](#deploy-daemonset)
- [Install Multus-CNI](#install-multus-cni)
- [Network Attachment Definition](#network-attachment-definition)
- [Example Pod Spec](#example-pod-spec)
# Quick Start

## Prerequisites

APF devices must be already created and present on the IPU environment

## Install NF-CNI

See the [NF CNI](https://github.com/intel/ipu-opi-plugins/ipu-nf-cni) repository for build and installation instructions.

### Build IPU Device Plugin image locally
```sh
 make image
```
> On a successful build, a docker image with tag `localhost:5000/github.com/intel/ipu-device-plugin:latest` will be created.

### Push image to local registry
```sh
docker push localhost:5000/github.com/intel/ipu-device-plugin:latest
```

### Install IPU Device Plugin

#### Deploy config map

Create a ConfigMap that defines APF resource exclusion list
```sh
export ROOT_DIR=$pwd

 kubectl create -f kubectl create -f  $ROOT_DIR/e2e/artefacts/k8s/ipu-device-plugin/configMap.yaml
```

#### Deploy daemonset
```sh
 kubectl create -f $ROOT_DIR/e2e/artefacts/k8s/ipu-device-plugin/apfdp-daemonset.yaml
```

### Install Multus-CNI
Please refer to Multus [Quickstart Installation Guide](https://github.com/k8snetworkplumbingwg/multus-cni#quickstart-installation-guide) to install Multus.

### Network Attachment Definition
Create the APF Network NAD
```sh
 kubectl create -f $ROOT_DIR/e2e/artefacts/k8s/ipu-device-plugin/nad.yaml
```

### Example pod spec
```sh
apiVersion: v1
kind: Pod
metadata:
  name: testpod1
  annotations:
    k8s.v1.cni.cncf.io/networks: sriov-net1,sriov-net1
spec:
  containers:
  - name: appcntr1 
    image: arm64v8/centos
    imagePullPolicy: IfNotPresent
    command: [ "/bin/bash", "-c", "--" ]
    args: [ "while true; do sleep 300000; done;" ]
    resources:
      requests:
        intel.com/ipu: '2'
      limits:
        intel.com/ipu: '2'
```
Note: we request 2 devices in the resources struct of the pod spec. This will allocate 2 devices and mount their device names as env variables into the container.

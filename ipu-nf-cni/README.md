# NF-CNI 

This component will be invoked by Multus to provision Network Functions with APFs and it will interface with the IPU Operator to call the Vendor plugin to program the entries into the FXP for the NFs.

**Note**: At the moment this CNI makes a no-op request to an IPU Manager. To integrate this with a running IPU Manager, the content of the gRPC protobuf file of the IPU Manager needs to be added to `proto/ipu_manager.proto` and the content of the gRPC requests in `nf-cni/nf-cni.go` and `nf-cni/nf-cni_test.go` adjusted to the new API.

## Requirements

- The following installed:
    - protobuf 3.15.8
    - gRPC 1.61.1
    - plugins to generate protobuf/gRPC code in Go:
        - protoc-gen-go v1.28.1
        - protoc-gen-go-grpc 1.3.0

- A running kubernetes environment.

- A running IPU Manager with a gRPC endpoint.

- This component depends on Device Plugin to identify the APFs on the IPU host. Follow [Procedure to deploy Device Plugin](#deploy-device-plugin-procedure).

- [static](https://github.com/containernetworking/cni.dev/blob/main/content/plugins/current/ipam/static.md) IPAM plugin needs to be in the path to successfully run the tests:
    - For example: `ln -s /opt/cni/bin/static /usr/local/bin/static`

### Procedure to deploy Device Plugin<a name="deploy-device-plugin-procedure"></a>


```bash
git clone https://github.com/intel-sandbox/ipu-device-plugin.git
make image 
docker push localhost:5000/github.com/intel/ipu-device-plugin
kubectl create -f deployments/configMap.yaml
kubectl create -f deployments/apfdp-daemonset.yaml
```

## Build

```bash
make
```

## Demo Deploy

This command assumes that a running Kubernetes environment is deployed.

Also, this command assumes that there are at least two available APFs on the IPU host.

Start two terminals.

In terminal one, start the fake IPU Manager:

```bash
cd e2e

go build dummy_ipu_manager.go

./dummy_ipu_manager
```

In terminal two, run the following commands:

```bash

# To copy the nf-cni binary into Multus' plugins path
make deploy

# To deploy the example pod with APFs
kubectl apply -f deployments/pod-tc1.yaml
```

## Usage

To use NF-CNI:
1. Create a `NetworkAttachmentDefinition` similar to the one below and set the `type` attribute in the config to `nf-cni` (see [example](#nad-example)). 
2. Annotate the pod definition `k8s.v1.cni.cncf.io/networks` with the `name` used in the `metadata` in the `NetworkAttachmentDefinition` (see [example](#pod-example)).
3. Specify the number of APFs to be attached to the pod by setting `intel.com/ipu` in the pod definition.
4. Set the `address` of the gRPC endpoint used by the IPU Manager in the `NetworkAttachmentDefinition`.

### NetworkAttachmentDefinition example<a name="nad-example"></a>

```yaml
apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  name: sriov-net1
  annotations:
    k8s.v1.cni.cncf.io/resourceName: intel.com/ipu
spec:
  config: '{
  "type": "nf-cni",  
  "cniVersion": "0.3.1",
  "name": "apf-network",
  "ipam": {
    "type": "host-local",
    "subnet": "10.56.217.0/24",
    "routes": [{
      "dst": "0.0.0.0/0"
    }],
    "gateway": "10.56.217.1"
  },
  "address": "127.0.0.1:50551"
}'

```

### Pod example <a name="pod-example"></a>

```yaml
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

## Testing

To test without an IPU Mananger available, compile and deploy `e2e/dummy_ipu_manager.go` before deploying the pod definition.
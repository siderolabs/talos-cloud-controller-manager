# Install

## Prepare nodes

We need to set the `--cloud-provider=external` kubelet flag for each node.

```yaml
# Talos machine config
machine:
  kubelet:
    extraArgs:
      cloud-provider: external
      # For security reasons, it is recommended to enable the rotation of server certificates.
      rotate-server-certificates: true
```

On the control-plane you need to allow [API access feature](https://docs.siderolabs.com/kubernetes-guides/advanced-guides/talos-api-access-from-k8s#talos-api-access-from-kubernetes):

```yaml
# Talos machine config
machine:
  kubelet:
    extraArgs:
      # For security reasons, it is recommended to enable the rotation of server certificates.
      rotate-server-certificates: true
  features:
    kubernetesTalosAPIAccess:
      enabled: true
      allowedRoles:
        - os:reader
      allowedKubernetesNamespaces:
        - kube-system
```

## Install Talos Cloud Controller Manager

### Method 1: talos machine config

```yaml
cluster:
  externalCloudProvider:
    enabled: true
    manifests:
      - https://raw.githubusercontent.com/siderolabs/talos-cloud-controller-manager/main/docs/deploy/cloud-controller-manager.yml
```

### Method 2: kubectl

Latest release:

```shell
kubectl apply -f https://raw.githubusercontent.com/siderolabs/talos-cloud-controller-manager/main/docs/deploy/cloud-controller-manager.yml
```

Latest stable version (edge):

```shell
kubectl apply -f https://raw.githubusercontent.com/siderolabs/talos-cloud-controller-manager/main/docs/deploy/cloud-controller-manager-edge.yml
```

### Method 3: helm chart

Helm chart documentation can be found [here](../charts/talos-cloud-controller-manager/README.md).
Values example can be found [here](../charts/talos-cloud-controller-manager/values-example.yaml)

```shell
helm upgrade -i -n kube-system talos-cloud-controller-manager oci://ghcr.io/siderolabs/charts/talos-cloud-controller-manager
```

## Example

This example deploys the Talos Cloud Controller Manager on a Talos cluster with __IPv4__ and __IPv6__ support.
IPv6 is globally routable, and the subnet is allocated to the node and used for podCIDRs.
If you don't need IPv6 on pods, please follow instructions above.

Talos Machine Config:

```yaml
machine:
  kubelet:
    extraArgs:
      cloud-provider: external
      rotate-server-certificates: true
  features:
    kubernetesTalosAPIAccess:
      enabled: true
      allowedRoles:
        - os:reader
      allowedKubernetesNamespaces:
        - kube-system
cluster:
  controllerManager:
    extraArgs:
      # Disable node IPAM controller
      controllers: "*,tokencleaner,-node-ipam-controller"
  network:
    # Example of IPv4 and IPv6 CIDR ranges, podSubnets-v6 will use as fallback for IPv6
    podSubnets: ["10.32.0.0/12","fd00:10:32::/64"]
    serviceSubnets: ["10.200.0.0/22","fd40:10:200::/108"]
```

We use the [values-example.yaml](../charts/talos-cloud-controller-manager/values-example.yaml) to deploy your Talos Cloud Controller Manager.

```shell
helm upgrade -i -n kube-system -f https://raw.githubusercontent.com/siderolabs/talos-cloud-controller-manager/main/charts/talos-cloud-controller-manager/values-example.yaml talos-cloud-controller-manager oci://ghcr.io/siderolabs/charts/talos-cloud-controller-manager
```

Check the result:

```shell
# kubectl get nodes -owide
NAME               STATUS   ROLES           AGE   VERSION   INTERNAL-IP    EXTERNAL-IP                 OS-IMAGE         KERNEL-VERSION   CONTAINER-RUNTIME
controlplane-01a   Ready    control-plane   61d   v1.30.2   172.16.0.142   2a01:4f8:0:3064:1::2d02   Talos (v1.7.4)   6.6.32-talos     containerd://1.7.16
web-01a            Ready    web             61d   v1.30.2   172.16.0.129   2a01:4f8:0:3064:2::2c0c   Talos (v1.7.4)   6.6.32-talos     containerd://1.7.16
web-02a            Ready    web             61d   v1.30.2   172.16.0.145   2a01:4f8:0:30ac:3::2ff4   Talos (v1.7.4)   6.6.32-talos     containerd://1.7.16

# kubectl get nodes web-01a -o jsonpath='{.metadata.labels}' | jq
{
  "beta.kubernetes.io/arch": "amd64",
  "beta.kubernetes.io/instance-type": "2VCPU-2GB",
  "beta.kubernetes.io/os": "linux",
  "failure-domain.beta.kubernetes.io/region": "region-1",
  "failure-domain.beta.kubernetes.io/zone": "region-1a",
  "kubernetes.io/arch": "amd64",
  "kubernetes.io/hostname": "web-01a",
  "kubernetes.io/os": "linux",
  "node-role.kubernetes.io/web": "",
  "node.cloudprovider.kubernetes.io/platform": "nocloud",
  "node.kubernetes.io/instance-type": "2VCPU-2GB",
  "topology.kubernetes.io/region": "region-1",
  "topology.kubernetes.io/zone": "region-1a"
}

# kubectl get nodes -o jsonpath='{.items[*].spec.podCIDRs}'; echo
["10.32.0.0/24","2a01:4f8:0:3064::/80"] ["10.32.3.0/24","2a01:4f8:0:3064:1::/80"] ["10.32.1.0/24","2a01:4f8:0:30ac::/80"]
```

Talos CCM did the following:
* adds the node-role label to the nodes by hostname
* define the EXTERNAL-IP
* podCIDRs allocation from IPv6 node subnet, they have two different IPv6/64 subnets (2a01:4f8:0:3064/64, 2a01:4f8:0:30ac::/64)

## Troubleshooting

How CCM works:

1. kubelet in mode `cloud-provider=external` join the cluster and send the `Node` object to the API server.
Node object has values:
    * `node.cloudprovider.kubernetes.io/uninitialized` taint.
    * `alpha.kubernetes.io/provided-node-ip` annotation with the node IP.
    * `nodeInfo` field with system information.
2. CCM detects the new node and sends a request to the Talos API to get the node configuration.
3. CCM updates the `Node` object with labels, taints and `providerID` field.
4. CCM removes the `node.cloudprovider.kubernetes.io/uninitialized` taint.
5. Node now is initialized and ready to use.

If `kubelet` does not have `cloud-provider=external` flag, kubelet will expect that no external CCM is running and will try to manage the node lifecycle by itself.
This can cause issues with Talos CCM.
So, CCM will skip the node and will not update the `Node` object.

### Steps to troubleshoot

1. Scale down the CCM deployment to 1 replica (in deployment case). In multiple replicas, only one pod is responsible for the node initialization all other pods are in the `standby` mode.
2. Set log level to `--v=5` in the deployment.
3. Check the logs
4. Check kubelet flag `--cloud-provider=external`, delete the node resource and restart the kubelet.
5. Check the logs
7. Check tains, labels, and providerID in the `Node` object.

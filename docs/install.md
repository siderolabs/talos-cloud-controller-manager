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

On the control-plane you need to allow [API access feature](https://www.talos.dev/v1.2/reference/configuration/#featuresconfig):

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

## Result example

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

Talos CCM:
* adds the node-role label to the nodes by hostname
* define the EXTERNAL-IP
* podCIDRs allocation from IPv6 node subnet, they have two different IPv6/64 subnets (2a01:4f8:0:3064/64, 2a01:4f8:0:30ac::/64)

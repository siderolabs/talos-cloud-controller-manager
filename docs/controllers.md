# Talos CCM Controllers

Talos CCM is a set of controllers that manage cloud resources in a Kubernetes cluster.
To check the default state of the controllers, run the following command:

```shell
docker run --rm -ti ghcr.io/siderolabs/talos-cloud-controller-manager:edge --help | grep -A 2 'controllers'
```

Output:

```shell
      --controllers strings                      A list of controllers to enable. '*' enables all on-by-default controllers, 'foo' enables the controller named 'foo', '-foo' disables the controller named 'foo'.
                                                 All controllers: certificatesigningrequest-approving-controller, cloud-node-controller, cloud-node-lifecycle-controller, node-ipam-controller, node-route-controller, service-lb-controller
                                                 Disabled-by-default controllers: certificatesigningrequest-approving-controller, node-ipam-controller, node-lifecycle-controller (default [*])
```

## Overview

* [cloud-node](#cloud-node)
* [cloud-node-lifecycle](#cloud-node-lifecycle)
* [route](#route)
* [service](#service)
* [nodeipam](#node-ipam)
* [node-csr-approval](#node-certificate-approval)

## Cloud node

Enabled by default.

CLI flags to enable the controller:
```shell
--controllers=cloud-node
```

Detects new instances launched in the cloud and registers them as nodes in the Kubernetes cluster.
Assigns labels and taints based on cloud metadata and configuration.
See [configuration options](config.md) for more details.

Well-Known [labels](https://kubernetes.io/docs/reference/labels-annotations-taints/):
* topology.kubernetes.io/region
* topology.kubernetes.io/zone
* node.kubernetes.io/instance-type

Talos specific labels:
* node.cloudprovider.kubernetes.io/clustername - talos cluster name
* node.cloudprovider.kubernetes.io/platform - name of platform
* node.cloudprovider.kubernetes.io/lifecycle - spot instance type

Node specs:
* providerID magic string
* InternalIP and ExternalIP addresses

## Cloud node lifecycle

Disabled by default.

CLI flags to enable the controller:
```shell
--controllers=node-lifecycle-controller
```

Currently, it make sense only for GCP cloud.
GCP spot instances change their IP address when they are evicted. CCM catches this event and remove the node resource from the cluster. After instance recreation, the node will initialize again and join the cluster.

## Route

Is not implemented yet.

## Service

Is not implemented yet.

## Node IPAM

Disabled by default.

CLI flags to enable the controller:
```shell
--controllers=node-ipam-controller
```

Node IPAM is responsible for managing the allocation and assignment of CIDR addresses to pods across the nodes in a Kubernetes cluster. It ensures that IP addresses are efficiently distributed without conflicts, supporting scalable and flexible networking within the cluster.

Pod CIDR allocation is based on the node CIDR range, which is defined by the `--node-cidr-mask-size-ipv4` and `--node-cidr-mask-size-ipv6` flags. The node CIDR range is divided into smaller subnets, which are then assigned to nodes in the cluster.
The CIDR allocator type can be set to `CloudAllocator` or `RangeAllocator`.

* RangeAllocator - is the default Kubernetes CIDR allocator.
* CloudAllocator - is a custom CIDR allocator that uses Talos metadata to allocate CIDR ranges to nodes.

This controller solves the IPv6 CIDR allocation problem in Kubernetes in hybrid environments.
Each node can have a different IPv6 CIDR range, which is not possible with the default Kubernetes CIDR allocator.

In hybrid environments, the Cloud Controller Manager (CCM) can utilize cloud provider-specific metadata to determine the IPv6 subnet assigned to each node.
By doing so, it allocates a unique pod CIDR range for each node based on its specific IPv6 subnet.
This ensures seamless integration of Kubernetes networking with the existing cloud infrastructure, enabling each node to have a distinct IPv6 CIDR range that suits its environment.

Recommended arguments for the controller:

```shell
# Talos CCM args
--controllers=node-ipam-controller \
--allocate-node-cidrs --node-cidr-mask-size-ipv4=24 --node-cidr-mask-size-ipv6=80 --cidr-allocator-type=CloudAllocator
```

Disable the default Kubernetes CIDR allocator and enable the Talos CloudAllocator.

```yaml
# Talos machine configuration
cluster:
  controllerManager:
    extraArgs:
      controllers: "*,tokencleaner,-node-ipam-controller"
  network:
    # Example of IPv4 and IPv6 CIDR ranges, podSubnets-v6 will use as fallback for IPv6
    podSubnets: ["10.32.0.0/12","fd00:10:32::/64"]
    serviceSubnets: ["10.200.0.0/22","fd40:10:200::/108"]
```

## Node certificate approval

Disabled by default.

CLI flags to enable the controller:
```shell
--controllers=certificatesigningrequest-approving-controller
```

Talos CCM is responsible for validating a node's certificate signing request (CSR) and approving it.
When a node wants to join a cluster, it generates a CSR, which includes its identity and other relevant information.
It checks if the CSR is properly formatted, contains all the required information, and matches the node's identity.

By validating and approving node CSRs, Talos CCM plays a crucial role in maintaining the security and integrity of the cluster by ensuring that only trusted and authorized nodes are allowed to have signed kubelet certificate.

The kubelet certificate is used to secure the communication between the kubelet and other components in the cluster, such as the Kubernetes control plane. It ensures that the communication is encrypted and authenticated and preventing a man-in-the-middle (MITM) attack.

Talos machine chenges for all nodes:
```yaml
machine:
  kubelet:
    extraArgs:
      rotate-server-certificates: true
```
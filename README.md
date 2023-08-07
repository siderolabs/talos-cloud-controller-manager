# Talos Cloud Controller Manager

Thank you for visiting the `Talos Cloud Controller Manager` repository!

One way to achieve a multi-cloud Kubernetes solution is to use a hybrid cloud approach, where you deploy one Kubernetes cluster on multiple cloud providers and use a tool such as [Omni](https://omni.siderolabs.com) to manage and orchestrate it.
This allows you to take advantage of the unique features and pricing models of different cloud providers and potentially reduce vendor lock-in.

It's also worth noting that Kubernetes itself is designed to be cloud-agnostic and can be deployed on a variety of infrastructure, so you have flexibility in terms of how you want to set up your multi-cloud solution and `Talos Cloud Controller Manager` (CCM) helps you with that.

Cloud controllers are responsible for integrating Kubernetes with the underlying cloud infrastructure, such as managing resources like persistent volumes, load balancers, and networking. Each cloud provider typically has its own cloud controller implementation, and these controllers may have different approaches to managing resources and interacting with the cloud API.

If you have multiple cloud controllers installed in a single cluster, it's possible that they could interfere with each other or cause conflicts when trying to manage the same resources. This could lead to unpredictable behavior and difficulties in troubleshooting and debugging issues.

Talos CCM tries to solve these issues and helps you to launch multiple CCMs in one cluster.

## Features

### Node initialize

Talos CCM receives the metadata from the Talos platform resource and applies labels to the nodes during the initialization process.

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

### Node certificate approval

Talos CCM is responsible for validating a node's certificate signing request (CSR) and approving it.
When a node wants to join a cluster, it generates a CSR, which includes its identity and other relevant information.
It checks if the CSR is properly formatted, contains all the required information, and matches the node's identity.

By validating and approving node CSRs, Talos CCM plays a crucial role in maintaining the security and integrity of the cluster by ensuring that only trusted and authorized nodes are allowed to have signed kubelet certificate.

The kubelet certificate is used to secure the communication between the kubelet and other components in the cluster, such as the Kubernetes control plane. It ensures that the communication is encrypted and authenticated and preventing a man-in-the-middle (MITM) attack.

## Example

```yaml
apiVersion: v1
kind: Node
metadata:
  labels:
    ...
    node.cloudprovider.kubernetes.io/platform: someprovider
    node.kubernetes.io/instance-type: type-of-instance
    topology.kubernetes.io/region: region-2
    topology.kubernetes.io/zone: zone
  name: controlplane-1
spec:
  ...
  providerID: someproviderID:///e8e8c388-5812-4db0-87e2-ad1fee51a1c1
status:
  addresses:
  - address: 172.16.0.11
    type: InternalIP
  - address: 1.2.3.4
    type: ExternalIP
  - address: 2001:123:123:123::1
    type: ExternalIP
  - address: controlplane-1
    type: Hostname
```

## Install

We need to set the `--cloud-provider=external` flag for each node.

To allow CCM approves/signs the [kubelet certificate signing request](https://kubernetes.io/docs/reference/access-authn-authz/certificate-signing-requests/#kubernetes-signers) set the flag `--rotate-server-certificates=true`.

### Prepare control-plane

On the control-plane you need to allow [API access feature](https://www.talos.dev/v1.2/reference/configuration/#featuresconfig):

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
```

### Prepare worker nodes

```yaml
machine:
  kubelet:
    extraArgs:
      cloud-provider: external
      rotate-server-certificates: true
```

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

```shell
helm upgrade -i -n kube-system talos-cloud-controller-manager oci://ghcr.io/siderolabs/charts/talos-cloud-controller-manager
```

## Community

- Slack: Join our [slack channel](https://slack.dev.talos-systems.io)
- Support: Questions, bugs, feature requests [GitHub Discussions](https://github.com/siderolabs/talos-cloud-controller-manager/discussions)
- Forum: [community](https://groups.google.com/a/SideroLabs.com/forum/#!forum/community)
- Twitter: [@SideroLabs](https://twitter.com/SideroLabs)
- Email: [info@SideroLabs.com](mailto:info@SideroLabs.com)

## Contributing

Contributions are welcomed and appreciated!
See [Contributing](CONTRIBUTING.md) for our guidelines.

## License

See [LICENSE](LICENSE) (MIT)

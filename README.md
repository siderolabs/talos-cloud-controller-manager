# Talos Cloud Controller Manager

Thank you for visiting the `Talos Cloud Controller Manager` repository!

One way to achieve a multi-cloud Kubernetes solution is to use a hybrid cloud approach, where you deploy one Kubernetes cluster on multiple cloud providers and use a tool such as [Omni](https://omni.siderolabs.com) to manage and orchestrate it.
This allows you to take advantage of the unique features and pricing models of different cloud providers and potentially reduce vendor lock-in.

It's also worth noting that Kubernetes itself is designed to be cloud-agnostic and can be deployed on a variety of infrastructure, so you have flexibility in terms of how you want to set up your multi-cloud solution and `Talos Cloud Controller Manager` (CCM) helps you with that.

Cloud controllers are responsible for integrating Kubernetes with the underlying cloud infrastructure, such as managing resources like persistent volumes, load balancers, and networking. Each cloud provider typically has its own cloud controller implementation, and these controllers may have different approaches to managing resources and interacting with the cloud API.

If you have multiple cloud controllers installed in a single cluster, it's possible that they could interfere with each other or cause conflicts when trying to manage the same resources. This could lead to unpredictable behavior and difficulties in troubleshooting and debugging issues.

Talos CCM tries to solve these issues and helps you to launch multiple CCMs in one cluster.

## Controllers

Support controllers:

* cloud-node
  * Updates node resource with cloud metadata
  * Assigns labels and taints based on cloud metadata and configuration
* cloud-node-lifecycle
  * Cleans up node resource when cloud instance is deleted.
* node-ipam-controller
  * Manages the allocation and assignment of CIDR addresses to pods across the nodes in a Kubernetes cluster.
* node-csr-approval
  * Automatically approves Certificate Signing Requests (CSRs) for kubelet server certificates.

Read more about cloud [controllers](docs/controllers.md).

## Example

Kubernetes node resource:

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

See [Install](docs/install.md) for installation instructions.

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

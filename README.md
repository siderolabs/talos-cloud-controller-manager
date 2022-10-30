# Talos Cloud Provider Manager

Thank you for visiting the `Talos Cloud Provider Manager` repository!

## Install

### Prepare control-plane

On control-plane you need to allow [API access feature](https://www.talos.dev/v1.2/reference/configuration/#featuresconfig):

```yaml
machine:
  features:
    kubernetesTalosAPIAccess:
      enabled: true
      allowedRoles:
        - os:reader
      allowedKubernetesNamespaces:
        - kube-system
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

```shell
kubectl apply -f https://raw.githubusercontent.com/siderolabs/talos-cloud-controller-manager/main/docs/deploy/cloud-controller-manager.yml
```

### Method 3: helm chart

```shell
helm upgrade -i -n kube-system talos-cloud-controller-manager charts/talos-cloud-controller-manager
```

## Features

Talos receives the metadata from a platform and labels the node according to the received data.

Well-Known [labels](https://kubernetes.io/docs/reference/labels-annotations-taints/):
* topology.kubernetes.io/region
* topology.kubernetes.io/zone
* node.kubernetes.io/instance-type
* providerID magic string
* InternalIP and ExternalIP addresses

Talos specific:
* node.cloudprovider.kubernetes.io/clustername - talos cluster name
* node.cloudprovider.kubernetes.io/platform - name of platform

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

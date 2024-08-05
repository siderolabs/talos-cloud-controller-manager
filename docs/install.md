# Install

## Prepare nodes

We need to set the `--cloud-provider=external` kubelet flag for each node.

```yaml
# Talos machine config
machine:
  kubelet:
    extraArgs:
      cloud-provider: external
```

On the control-plane you need to allow [API access feature](https://www.talos.dev/v1.2/reference/configuration/#featuresconfig):

```yaml
# Talos machine config
machine:
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

Helm chart documentation can be found [here](../charts/talos-cloud-controller-manager/README.md)

```shell
helm upgrade -i -n kube-system talos-cloud-controller-manager oci://ghcr.io/siderolabs/charts/talos-cloud-controller-manager
```

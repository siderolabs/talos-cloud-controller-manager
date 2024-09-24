# talos-cloud-controller-manager

![Version: 0.4.2](https://img.shields.io/badge/Version-0.4.2-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: v1.8.0](https://img.shields.io/badge/AppVersion-v1.8.0-informational?style=flat-square)

Talos Cloud Controller Manager Helm Chart

**Homepage:** <https://github.com/siderolabs/talos-cloud-controller-manager>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| sergelogvinov |  | <https://github.com/sergelogvinov> |

## Source Code

* <https://github.com/siderolabs/talos-cloud-controller-manager>

## Requirements

Kubernetes: `>= 1.24.0`

## Talos machine config

The control plane configuration should be set with the following settings:

```yaml
machine:
  kubelet:
    extraArgs:
      cloud-provider: external
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

The worker nodes configuration should include the following settings:

```yaml
machine:
  kubelet:
    extraArgs:
      cloud-provider: external
      # For security reasons, it is recommended to enable the rotation of server certificates.
      rotate-server-certificates: true
```

## Deploy example

```yaml
# talos-ccm.yaml

replicaCount: 2

enabledControllers:
  - cloud-node
  - node-csr-approval

# Deploy CCM only on control-plane nodes
nodeSelector:
  node-role.kubernetes.io/control-plane: ""
tolerations:
  - key: node-role.kubernetes.io/control-plane
    effect: NoSchedule
```

Deploy chart:

```shell
helm upgrade -i --namespace=kube-system -f talos-ccm.yaml \
  talos-cloud-controller-manager charts/talos-cloud-controller-manager
```

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Affinity for data pods assignment. ref: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity |
| enabledControllers | list | `["cloud-node","node-csr-approval"]` | List of controllers should be enabled. Use '*' to enable all controllers. Support only `cloud-node, cloud-node-lifecycle, node-csr-approval, node-ipam-controller` controllers. |
| extraArgs | list | `[]` | Any extra arguments for talos-cloud-controller-manager |
| fullnameOverride | string | `""` | String to fully override deployment name. |
| image.pullPolicy | string | `"IfNotPresent"` | Pull policy: IfNotPresent or Always. |
| image.repository | string | `"ghcr.io/siderolabs/talos-cloud-controller-manager"` | CCM image repository. |
| image.tag | string | `""` | Overrides the image tag whose default is the chart appVersion. |
| imagePullSecrets | list | `[]` | Optionally specify an array of imagePullSecrets. Secrets must be manually created in the namespace. ref: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/ |
| logVerbosityLevel | int | `2` | Log verbosity level. See https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/logging.md for description of individual verbosity levels. |
| nameOverride | string | `""` | String to partially override deployment name. |
| nodeSelector | object | `{"node-role.kubernetes.io/control-plane":""}` | Node labels for data pods assignment. ref: https://kubernetes.io/docs/user-guide/node-selection/ |
| podAnnotations | object | `{}` | Annotations for data pods. ref: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/ |
| podSecurityContext | object | `{"fsGroup":10258,"fsGroupChangePolicy":"OnRootMismatch","runAsGroup":10258,"runAsNonRoot":true,"runAsUser":10258}` | Pods Security Context. ref: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod |
| priorityClassName | string | `"system-cluster-critical"` | CCM pods' priorityClassName. |
| replicaCount | int | `1` | Number of CCM replicas to deploy. |
| resources | object | `{"requests":{"cpu":"10m","memory":"64Mi"}}` | Resource requests and limits. ref: http://kubernetes.io/docs/user-guide/compute-resources/ |
| securityContext | object | `{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]},"seccompProfile":{"type":"RuntimeDefault"}}` | Container Security Context. ref: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#set-the-security-context-for-a-pod |
| service.annotations | object | `{}` | Additional custom annotations for Service. |
| service.containerPort | int | `50258` | Container HTTPS port. |
| service.port | int | `50258` | Service HTTPS port to expose controller. |
| serviceAccount | object | `{"annotations":{},"create":true,"name":""}` | Pods Service Account. ref: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/ |
| serviceAccount.annotations | object | `{}` | Annotations to add to the service account. |
| serviceAccount.create | bool | `true` | Specifies whether a service account should be created. |
| serviceAccount.name | string | `""` | The name of the service account to use. If not set and create is true, a name is generated using the fullname template. |
| tolerations | list | `[{"effect":"NoSchedule","key":"node-role.kubernetes.io/control-plane","operator":"Exists"},{"effect":"NoSchedule","key":"node.cloudprovider.kubernetes.io/uninitialized","operator":"Exists"}]` | Tolerations for data pods assignment. ref: https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/ |
| transformations | list | `[]` | List of node transformations. Available matchExpressions key values: https://github.com/siderolabs/talos/blob/main/pkg/machinery/resources/runtime/platform_metadata.go#L28 |
| updateStrategy | object | `{"rollingUpdate":{"maxUnavailable":1},"type":"RollingUpdate"}` | Deployment update stategy type. ref: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#updating-a-deployment |
| useDaemonSet | bool | `false` | Deploy CCM  in Daemonset mode. CCM will use hostNetwork and current node to access kubernetes/talos API You can run it without CNI plugin. |

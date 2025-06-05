# Talos CCM Configuration

## Overview

The Talos CCM is a Kubernetes controller that manages the lifecycle of Talos nodes in a Kubernetes cluster.

In scope of the Talos CCM, the following features are implemented:
* Initialize nodes in a Kubernetes cluster, set [well-known labels and annotations](https://kubernetes.io/docs/reference/labels-annotations-taints/).
* Label and annotate nodes based on the transformation rules.
* Approve and sign the [kubelet certificate signing request](https://kubernetes.io/docs/reference/access-authn-authz/certificate-signing-requests/#kubernetes-signers) for Talos nodes.

Result of kubernetes node object after Talos CCM initialization:

```yaml
apiVersion: v1
kind: Node
metadata:
  annotations:
    ...
    # Annotations based on transformation rules, see the configuration section
    custom-annotation/instance-id: "id-e8e8c388-5812-4db0-87e2-ad1fee51a1c1"
    ...
  labels:
    ...
    # Set well-known labels, sets by Talos CCM
    node.cloudprovider.kubernetes.io/platform: metal
    node.kubernetes.io/instance-type: t2.micro
    topology.kubernetes.io/region: us-west-1-on-metal
    topology.kubernetes.io/zone: us-west-1f
    ...
    # Label based on transformation rules, see the configuration section
    node-role.kubernetes.io/web: ""
    ...
  name: web-1
spec:
  ...
  # Define the provider ID of the node, it depends on the cloud platform
  providerID: someproviderID:///e8e8c388-5812-4db0-87e2-ad1fee51a1c1
status:
  # Define the addresses of the node
  addresses:
  - address: 172.16.0.11
    type: InternalIP
  - address: 1.2.3.4
    type: ExternalIP
  - address: 2001:123:123:123::1
    type: ExternalIP
  - address: web-1
    type: Hostname
```

## Configuration

Talos CCM configuration file:

```yaml
# Global parameters
global:
  # PreferIPv6 uses to prefer IPv6 addresses over IPv4 addresses
  PreferIPv6: false

# Transformations rules for nodes
transformations:
  # All rules are applied in order, all matched rules are applied to the node

  - name: nocloud-nodes
    # Match nodes by nodeSelector
    nodeSelector:
      - matchExpressions:
          - key: platform           <- talos platform metadata variable case insensitive
            operator: In            <- In, NotIn, Exists, DoesNotExist, Gt, Lt, Regexp
            values:                 <- array of string values
              - nocloud
    # Set labels for matched nodes
    labels:
      pvc-storage-class/name: "my-storage-class"

  - name: web-nodes                 <- transformation name, optional
    nodeSelector:
      # Or condition for nodeSelector
      - matchExpressions:
          # And condition for matchExpressions
          - key: platform           <- talos platform metadata variable case insensitive
            operator: In            <- In, NotIn, Exists, DoesNotExist, Gt, Lt, Regexp
            values:                 <- array of string values
              - metal
          - key: hostname
            operator: Regexp
            values:
              - ^web-[\w]+$         <- go regexp pattern
      - matchExpressions:
          - key: hostname
            operator: Regexp
            values:
              - ^web-cloud-.+$

    # Add/replace annotations, labels and taints for nodes that match the transformation
    annotations:
      # You can use the Go template to get the value of the platform metadata variable
      custom-annotation/instance-id: "id-{{ .InstanceID }}"
      # You can use the functions to modify the values
      # If hostname is "web-<id>-<name>", then set the cloud-id annotation to "<id>"
      custom-annotation/cloud-id: "{{ regexFindString "^web-([a-z0-9]+)-(.*)$" .Hostname 1 }}"
    labels:
      # Add label to the node, in this case, we add well-known node role label
      node-role.kubernetes.io/web: ""
      # Set capacity-type spot/on-demand
      node-role.kubernetes.io/capacity-type: "{{ if .Spot }}spot{{ else }}on-demand{{ end }}"
    taints:
      # Add taint to the node
      node.cloudprovider.kubernetes.io/storage-type: "ceph:NoSchedule"

    # Replace platform metadata variables for nodes that match the transformation
    platformMetadata:
      Region: "{{ .Region }}-on-metal"
      Zone: "us-west-1f"
      # SKUNumber is a system information variable "t2.micro"
      InstanceType: "{{ .SKUNumber }}"
      # UUID is a system information variable "e8e8c388-5812-4db0-87e2-ad1fee51a1c1"
      ProviderID: "someproviderID:///{{ .UUID }}"

    # Features flags for nodes that match the transformation
    features:
      # Try to discover the public IP address of the node
      publicIPDiscovery: true
```

### Transformations parameters

* `nodeSelector` - a list of node selector requirements by platform metadata variable.
  * `matchExpressions` - a list of node selector requirements by platform metadata variable.
    * `key` - the key that the selector applies to, case `insensitive`.
    * `operator` - represents a key's relationship to a set of values. Supported operators are `In`, `NotIn`, `Exists`, `DoesNotExist`, `Gt`, `Lt`, `Regexp`.
    * `values` - an array of string values.

* `annotations` - a map of key-value pairs to add to each node that matches the transformation.
  * `key` - the key of the annotation.
  * `value` - the value of the annotation. You can use the [Go template](https://golang.org/pkg/text/template/) to get the value of the platform metadata variable. Variables are case `sensitive`.

* `labels` - a map of key-value pairs to add to each node that matches the transformation.
  * `key` - the key of the label.
  * `value` - the value of the label. You can use the [Go template](https://golang.org/pkg/text/template/) to get the value of the platform metadata variable. Variables are case `sensitive`.

* `taints` - a map of key-value pairs to add to each node that matches the transformation.
  * `key` - the key of the taint. Can not be well-known taints name, like `node.kubernetes.io/unreachable`.
  * `value` - the string in format '<value>:<effect>', '<effect>'. Effect can be `NoExecute`, `NoSchedule`, `PreferNoSchedule`.

* `platformMetadata` - a map of key-value pairs to add to each node that matches the transformation.
  * `key` - the key of the platform metadata variable to replace.
  * `value` - the value of the platform metadata variable. You can use the [Go template](https://golang.org/pkg/text/template/) to get the value of the platform metadata variable. Variables are case `sensitive`.

* `features` - enable or disable features for each node that matches the transformation.
  * `publicIPDiscovery` - try to discover the public IP address of the node. The feature is `disable` by default.

### Platform metadata variables

Go struct for platform metadata,
original code: [platform_metadata.go](https://github.com/siderolabs/talos/blob/main/pkg/machinery/resources/runtime/platform_metadata.go)

```go
type PlatformMetadataSpec struct {
	Platform     string `yaml:"platform,omitempty" protobuf:"1"`
	Hostname     string `yaml:"hostname,omitempty" protobuf:"2"`
	Region       string `yaml:"region,omitempty" protobuf:"3"`
	Zone         string `yaml:"zone,omitempty" protobuf:"4"`
	InstanceType string `yaml:"instanceType,omitempty" protobuf:"5"`
	InstanceID   string `yaml:"instanceId,omitempty" protobuf:"6"`
	ProviderID   string `yaml:"providerId,omitempty" protobuf:"7"`
	Spot         bool   `yaml:"spot,omitempty" protobuf:"8"`
	InternalDNS  string `yaml:"internalDNS,omitempty" protobuf:"9"`
	ExternalDNS  string `yaml:"externalDNS,omitempty" protobuf:"10"`
}
```

* `Platform` - the Talos platform, for example, `aws`, `gcp`, `azure`, `openstack`, `metal`. Supported platforms are defined in the [platform.go](https://github.com/siderolabs/talos/blob/main/internal/app/machined/pkg/runtime/v1alpha1/platform/platform.go)
* `Hostname` - the hostname of the node.
* `Region` - the region of the node, for example, `us-east-1`.
* `Zone` - the zone of the node, for example, `us-west-1f`.
* `InstanceType` - the instance type of the node, for example, `t2.micro`.
* `InstanceID` - the instance ID of the node, for example, `i-1234567890abcdef0`.
* `ProviderID` - the provider ID of the node, for example, `aws:///us-east-1f/i-1234567890abcdef0`.
* `Spot` - the spot instance, for example, `true` or `false`.
* `InternalDNS` - the internal DNS name of the node in the cloud.
* `ExternalDNS` - the external DNS name of the node in the cloud.

You can use the following command to get the platform metadata:

```bash
talosctl get PlatformMetadatas -oyaml
```

### System information variables

Additionally you can use the system information variables in the transformations rules.

Go struct for system information,
original code: [system_information.go](https://github.com/siderolabs/talos/blob/main/pkg/machinery/resources/hardware/system_information.go)

```go
type SystemInformationSpec struct {
	Manufacturer string `yaml:"manufacturer,omitempty" protobuf:"1"`
	ProductName  string `yaml:"productName,omitempty" protobuf:"2"`
	Version      string `yaml:"version,omitempty" protobuf:"3"`
	SerialNumber string `yaml:"serialnumber,omitempty" protobuf:"4"`
	UUID         string `yaml:"uuid,omitempty" protobuf:"5"`
	WakeUpType   string `yaml:"wakeUpType,omitempty" protobuf:"6"`
	SKUNumber    string `yaml:"skuNumber,omitempty" protobuf:"7"`
}
```

You can use the following command to get the system information:

```bash
talosctl get SystemInformation -oyaml
```

### Talos OS Version

You can use Talos OS version in the transformations rules.

Example of rule:

```yaml
- labels:
    talos.os/containerd: "`{{ if semverCompare "=> 1.8" .TalosVersion }}2{{else}}1{{ end }}`"
```

Follow command help you to get the Talos OS version:

```bash
talosctl get version -oyaml
```

### Transformations functions

You can use the following functions in the Go template:

#### String modification functions

* `upper` - the function to convert the string to uppercase.

  ```yaml
  {{ upper "hello" }} -> HELLO
  ```

* `lower` - the function to convert the string to lowercase.

  ```yaml
  {{ lower "HELLO" }} -> hello
  ```

* `trim` - the function to remove leading and trailing whitespace from the string.

  ```yaml
  {{ trim "  hello  " }} -> hello
  ```

* `trimSuffix` - the function to remove the suffix from the string.

  ```yaml
  {{ trimSuffix "hello" "lo" }} -> hel
  ```

* `trimPrefix` - the function to remove the prefix from the string.

  ```yaml
  {{ trimPrefix "hello" "he" }} -> llo
  ```

* `replace` - the function to replace all occurrences of the old string with the new string.

  ```yaml
  {{ replace "hello" "l" "L" }} -> heLLo
  ```

* `regexFind` - return the first (left most) match of the regular expression in the input string.

  ```yaml
  {{ regexFind "[a-zA-Z][1-9]" "abcd1234" }} -> d1
  ```

* `regexFindString` - the function to find the match of the regular expression pattern in the string and return the submatch at the specified index.

  ```yaml
  {{ regexFindString "^type-([a-z0-9]+)-(.*)$" "type-metal1-asz" 1 }} -> metal1
  ```

* `regexReplaceAll` - the function to replace all occurrences of the regular expression in the input string with the replacement string.

  ```yaml
  {{ regexReplaceAll "a(x*)b" "-ab-axxb-" "${1}W" }} -> -W-xxW-
  ```

#### Conditional functions

* `contains` - the function to return true if the string contains the substring.

  ```yaml
  {{ contains "hello" "lo" }} -> true
  ```

* `hasPrefix` - the function to return true if the string has the specified prefix.

  ```yaml
  {{ hasPrefix "hello" "he" }} -> true
  ```

* `hasSuffix` - the function to return true if the string has the specified suffix.

  ```yaml
  {{ hasSuffix "hello" "lo" }} -> true
  ```

#### SemVer functions

* `semver` - the function to return the version of Talos OS in the format `major.minor.patch`.

  ```yaml
  {{ (semver "1.9.0").Major }} -> 1
  {{ (semver "1.9.0").Minor }} -> 9
  {{ (semver "1.9.0").Patch }} -> 0
  ```

* `semverCompare` - the function to compare the version with the specified constraint.

  ```yaml
  {{ semverCompare ">= 1.8" "1.9.0" }} -> true
  ```

  Compare example:

  * `~1.2.3` is equivalent to >= 1.2.3, < 1.3.0
  * `~1` is equivalent to >= 1, < 2
  * `~2.3` is equivalent to >= 2.3, < 2.4
  * `~1.2.x` is equivalent to >= 1.2.0, < 1.3.0
  * `~1.x` is equivalent to >= 1, < 2
  * `1.2.x` is equivalent to >= 1.2.0, < 1.3.0
  * `>= 1.2.x` is equivalent to >= 1.2.0
  * `<= 2.x` is equivalent to < 3

#### Encoding functions

* `b64enc` - the function to return the base64-encoded string.

  ```yaml
  {{ b64enc "hello" }} -> aGVsbG8=
  ```
* `b64dec` - the function to return the base64-decoded string.

  ```yaml
  {{ b64dec "aGVsbG8=" }} -> hello
  ```

#### String slice functions

* `getValue` - the function to get the value from the map by key.

  ```yaml
  {{ getValue "ds=nocloud;i=1234" "i" }} -> 1234
  ```

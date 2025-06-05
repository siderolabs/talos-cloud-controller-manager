package talos

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/siderolabs/talos-cloud-controller-manager/pkg/metrics"
	"github.com/siderolabs/talos-cloud-controller-manager/pkg/transformer"
	"github.com/siderolabs/talos-cloud-controller-manager/pkg/utils/net"
	"github.com/siderolabs/talos-cloud-controller-manager/pkg/utils/platform"
	"github.com/siderolabs/talos/pkg/machinery/resources/hardware"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"

	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
	cloudproviderapi "k8s.io/cloud-provider/api"
	"k8s.io/klog/v2"
)

type instances struct {
	c *client
}

var (
	uninitializedTaint = &v1.Taint{
		Key:    cloudproviderapi.TaintExternalCloudProvider,
		Effect: v1.TaintEffectNoSchedule,
	}

	notReadyTaint = &v1.Taint{
		Key:    v1.TaintNodeNotReady,
		Effect: v1.TaintEffectNoExecute,
	}

	initializedNodeDelay = time.Second * 30
)

func newInstances(client *client) *instances {
	return &instances{
		c: client,
	}
}

// InstanceExists returns true if the instance for the given node exists according to the cloud provider.
// Use the node.name or node.spec.providerID field to find the node in the cloud provider.
func (i *instances) InstanceExists(_ context.Context, node *v1.Node) (bool, error) {
	klog.V(4).InfoS("instances.InstanceExists() called", "node", klog.KRef("", node.Name))

	if node.Spec.ProviderID == "" {
		return true, nil
	}

	notReady := false

	for _, taint := range node.Spec.Taints {
		if taint.MatchTaint(uninitializedTaint) {
			return true, nil
		}

		if taint.MatchTaint(notReadyTaint) {
			notReady = true
		}
	}

	delay := time.Since(node.ObjectMeta.CreationTimestamp.Time)
	if delay < initializedNodeDelay {
		klog.V(4).InfoS("instances.InstanceExists() wait initialized node delay", "node", klog.KRef("", node.Name), "delay", delay)

		return true, nil
	}

	if notReady {
		if node.Labels[ClusterNodePlatformLabel] == "gcp" &&
			node.Labels[ClusterNodeLifeCycleLabel] == "spot" {
			return false, nil
		}
	}

	return true, nil
}

// InstanceShutdown returns true if the instance is shutdown according to the cloud provider.
// Use the node.name or node.spec.providerID field to find the node in the cloud provider.
func (i *instances) InstanceShutdown(_ context.Context, node *v1.Node) (bool, error) {
	klog.V(4).InfoS("instances.InstanceShutdown() called", "node", klog.KRef("", node.Name))

	return false, nil
}

// InstanceMetadata returns the instance's metadata. The values returned in InstanceMetadata are
// translated into specific fields in the Node object on registration.
// Use the node.name or node.spec.providerID field to find the node in the cloud provider.
//
//nolint:gocyclo,cyclop
func (i *instances) InstanceMetadata(ctx context.Context, node *v1.Node) (*cloudprovider.InstanceMetadata, error) {
	klog.V(4).InfoS("instances.InstanceMetadata() called", "node", klog.KRef("", node.Name))

	if providedIP, ok := node.ObjectMeta.Annotations[cloudproviderapi.AnnotationAlphaProvidedIPAddr]; ok {
		nodeIPs := net.PreferredDualStackNodeIPs(i.c.config.Global.PreferIPv6, strings.Split(providedIP, ","))

		var (
			meta   *runtime.PlatformMetadataSpec
			err    error
			nodeIP string
		)

		mc := metrics.NewMetricContext(runtime.PlatformMetadataID)

		for _, ip := range nodeIPs {
			meta, err = i.c.talos.GetNodeMetadata(ctx, ip)
			if mc.ObserveRequest(err) == nil {
				nodeIP = ip

				break
			}

			klog.ErrorS(err, "error getting metadata from the node", "node", klog.KRef("", node.Name))
		}

		if meta == nil {
			return nil, fmt.Errorf("error getting metadata from the node %s", node.Name)
		}

		klog.V(5).InfoS("instances.InstanceMetadata()", "node", klog.KRef("", node.Name), "resource", meta)

		if meta.ProviderID == "" {
			meta.ProviderID = fmt.Sprintf("%s://%s/%s", ProviderName, meta.Platform, nodeIP)
		}

		// Fix for Azure, resource group name must be lower case.
		// Since Talos 1.8 fixed it, we can remove this code in the future.
		if meta.Platform == "azure" {
			meta.ProviderID, err = platform.AzureConvertResourceGroupNameToLower(meta.ProviderID)
			if err != nil {
				return nil, fmt.Errorf("error converting resource group name to lower case: %w", err)
			}
		}

		if meta.Hostname == "" {
			meta.Hostname = node.Name
		}

		var (
			sysInfo      *hardware.SystemInformationSpec
			talosVersion string
		)

		if len(i.c.config.Transformations) > 0 {
			talosVersion, err = i.c.talos.GetNodeVersion(ctx, nodeIP)
			if err != nil {
				return nil, fmt.Errorf("error getting node version from the node %s: %w", node.Name, err)
			}

			msys := metrics.NewMetricContext(hardware.SystemInformationID)

			sysInfo, err = i.c.talos.GetNodeSystemInfo(ctx, nodeIP)
			if msys.ObserveRequest(err) != nil {
				return nil, fmt.Errorf("error getting system info from the node %s: %w", node.Name, err)
			}
		}

		mct := metrics.NewMetricContext("transformer")

		nodeSpec, err := transformer.TransformNode(i.c.config.Transformations, meta, sysInfo, talosVersion)
		if mct.ObserveTransformer(err) != nil {
			return nil, fmt.Errorf("error transforming node: %w", err)
		}

		if nodeSpec == nil {
			nodeSpec = &transformer.NodeSpec{}
		}

		mc = metrics.NewMetricContext("addresses")

		ifaces, err := i.c.talos.GetNodeIfaces(ctx, nodeIP)
		if mc.ObserveRequest(err) != nil {
			return nil, fmt.Errorf("error getting interfaces list from the node %s: %w", node.Name, err)
		}

		addresses := getNodeAddresses(i.c.config, meta.Platform, &nodeSpec.Features, nodeIPs, ifaces)

		addresses = append(addresses, v1.NodeAddress{Type: v1.NodeHostName, Address: node.Name})

		if meta.Hostname != "" && strings.IndexByte(meta.Hostname, '.') > 0 {
			addresses = append(addresses, v1.NodeAddress{Type: v1.NodeInternalDNS, Address: meta.Hostname})
		}

		if len(nodeSpec.Annotations) > 0 {
			klog.V(4).InfoS("instances.InstanceMetadata() node has annotations", "node", klog.KRef("", node.Name), "annotations", nodeSpec.Annotations)

			if err := syncNodeAnnotations(ctx, i.c, node, nodeSpec.Annotations); err != nil {
				klog.ErrorS(err, "error updating annotations for the node", "node", klog.KRef("", node.Name))
			}
		}

		if len(nodeSpec.Taints) > 0 {
			klog.V(4).InfoS("instances.InstanceMetadata() node has taints", "node", klog.KRef("", node.Name), "taints", nodeSpec.Taints)

			if taintExists(node.Spec.Taints, uninitializedTaint) {
				if err := syncNodeTaints(ctx, i.c, node, nodeSpec.Taints); err != nil {
					klog.ErrorS(err, "error updating taints for the node", "node", klog.KRef("", node.Name))
				}
			}
		}

		nodeLabels := setTalosNodeLabels(i.c, meta)

		if len(nodeSpec.Labels) > 0 {
			klog.V(4).InfoS("instances.InstanceMetadata() node has labels", "node", klog.KRef("", node.Name), "labels", nodeSpec.Labels)

			maps.Copy(nodeLabels, nodeSpec.Labels)
		}

		if err := syncNodeLabels(i.c, node, nodeLabels); err != nil {
			klog.ErrorS(err, "error updating labels for the node", "node", klog.KRef("", node.Name))
		}

		return &cloudprovider.InstanceMetadata{
			ProviderID:    meta.ProviderID,
			InstanceType:  meta.InstanceType,
			NodeAddresses: addresses,
			Zone:          meta.Zone,
			Region:        meta.Region,
		}, nil
	}

	klog.InfoS("instances.InstanceMetadata() is kubelet has args: --cloud-provider=external on the node?", node, klog.KRef("", node.Name))

	return &cloudprovider.InstanceMetadata{}, nil
}

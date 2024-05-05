package talos

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"github.com/siderolabs/talos-cloud-controller-manager/pkg/metrics"
	"github.com/siderolabs/talos-cloud-controller-manager/pkg/transformer"
	"github.com/siderolabs/talos-cloud-controller-manager/pkg/utils/net"
	"github.com/siderolabs/talos-cloud-controller-manager/pkg/utils/platform"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"

	v1 "k8s.io/api/core/v1"
	cloudprovider "k8s.io/cloud-provider"
	cloudproviderapi "k8s.io/cloud-provider/api"
	"k8s.io/klog/v2"
)

type instances struct {
	c *client
}

func newInstances(client *client) *instances {
	return &instances{
		c: client,
	}
}

// InstanceExists returns true if the instance for the given node exists according to the cloud provider.
// Use the node.name or node.spec.providerID field to find the node in the cloud provider.
func (i *instances) InstanceExists(_ context.Context, node *v1.Node) (bool, error) {
	klog.V(4).Info("instances.InstanceExists() called node: ", node.Name)

	return true, nil
}

// InstanceShutdown returns true if the instance is shutdown according to the cloud provider.
// Use the node.name or node.spec.providerID field to find the node in the cloud provider.
func (i *instances) InstanceShutdown(_ context.Context, node *v1.Node) (bool, error) {
	klog.V(4).Info("instances.InstanceShutdown() called, node: ", node.Name)

	return false, nil
}

// InstanceMetadata returns the instance's metadata. The values returned in InstanceMetadata are
// translated into specific fields in the Node object on registration.
// Use the node.name or node.spec.providerID field to find the node in the cloud provider.
//
//nolint:gocyclo,cyclop
func (i *instances) InstanceMetadata(ctx context.Context, node *v1.Node) (*cloudprovider.InstanceMetadata, error) {
	klog.V(4).Info("instances.InstanceMetadata() called, node: ", node.Name)

	if providedIP, ok := node.ObjectMeta.Annotations[cloudproviderapi.AnnotationAlphaProvidedIPAddr]; ok {
		nodeIPs := net.PreferedDualStackNodeIPs(i.c.config.Global.PreferIPv6, strings.Split(providedIP, ","))

		var (
			meta   *runtime.PlatformMetadataSpec
			err    error
			nodeIP string
		)

		if err = i.c.refreshTalosClient(ctx); err != nil {
			return nil, fmt.Errorf("error refreshing client connection: %w", err)
		}

		mc := metrics.NewMetricContext(runtime.PlatformMetadataID)

		for _, ip := range nodeIPs {
			meta, err = i.c.getNodeMetadata(ctx, ip)
			if mc.ObserveRequest(err) == nil {
				nodeIP = ip

				break
			}

			klog.Errorf("error getting metadata from the node %s: %v", node.Name, err)
		}

		if meta == nil {
			return nil, fmt.Errorf("error getting metadata from the node %s", node.Name)
		}

		klog.V(5).Infof("instances.InstanceMetadata() resource: %+v", meta)

		providerID := meta.ProviderID
		if providerID == "" {
			providerID = fmt.Sprintf("%s://%s/%s", ProviderName, meta.Platform, nodeIP)
		}

		// Fix for Azure, resource group name must be lower case.
		if meta.Platform == "azure" {
			providerID, err = platform.AzureConvertResourceGroupNameToLower(providerID)
			if err != nil {
				return nil, fmt.Errorf("error converting resource group name to lower case: %w", err)
			}
		}

		nodeSpec, err := transformer.TransformNode(i.c.config.Transformations, meta)
		if err != nil {
			return nil, fmt.Errorf("error transforming node: %w", err)
		}

		mc = metrics.NewMetricContext("addresses")

		ifaces, err := i.c.getNodeIfaces(ctx, nodeIP)
		if mc.ObserveRequest(err) != nil {
			return nil, fmt.Errorf("error getting interfaces list from the node %s: %w", node.Name, err)
		}

		addresses := getNodeAddresses(i.c.config, meta.Platform, &nodeSpec.Features, nodeIPs, ifaces)

		addresses = append(addresses, v1.NodeAddress{Type: v1.NodeHostName, Address: node.Name})

		if meta.Hostname != "" && strings.IndexByte(meta.Hostname, '.') > 0 {
			addresses = append(addresses, v1.NodeAddress{Type: v1.NodeInternalDNS, Address: meta.Hostname})
		}

		// Foreign node, update network only.
		if i.c.config.Global.SkipForeignNode && !strings.HasPrefix(node.Spec.ProviderID, ProviderName) {
			klog.V(4).Infof("instances.InstanceMetadata() node %s has foreign providerID: %s, skipped", node.Name, node.Spec.ProviderID)

			return &cloudprovider.InstanceMetadata{
				NodeAddresses: addresses,
			}, nil
		}

		if nodeSpec != nil && nodeSpec.Annotations != nil {
			klog.V(4).Infof("instances.InstanceMetadata() node %s has annotations: %+v", node.Name, nodeSpec.Annotations)

			if err := syncNodeAnnotations(ctx, i.c, node, nodeSpec.Annotations); err != nil {
				klog.Errorf("failed update annotations for node %s, %v", node.Name, err)
			}
		}

		nodeLabels := setTalosNodeLabels(i.c, meta)

		if nodeSpec != nil && nodeSpec.Labels != nil {
			klog.V(4).Infof("instances.InstanceMetadata() node %s has labels: %+v", node.Name, nodeSpec.Labels)

			maps.Copy(nodeLabels, nodeSpec.Labels)
		}

		if err := syncNodeLabels(i.c, node, nodeLabels); err != nil {
			klog.Errorf("failed update labels for node %s, %v", node.Name, err)
		}

		return &cloudprovider.InstanceMetadata{
			ProviderID:    providerID,
			InstanceType:  meta.InstanceType,
			NodeAddresses: addresses,
			Zone:          meta.Zone,
			Region:        meta.Region,
		}, nil
	}

	klog.Warningf("instances.InstanceMetadata() is kubelet has --cloud-provider=external on the node %s?", node.Name)

	return &cloudprovider.InstanceMetadata{}, nil
}

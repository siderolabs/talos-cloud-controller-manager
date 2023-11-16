package talos

import (
	"context"
	"fmt"
	"strings"

	"github.com/siderolabs/talos-cloud-controller-manager/pkg/utils/platform"

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
func (i *instances) InstanceMetadata(ctx context.Context, node *v1.Node) (*cloudprovider.InstanceMetadata, error) {
	klog.V(4).Info("instances.InstanceMetadata() called, node: ", node.Name)

	if providedIP, ok := node.ObjectMeta.Annotations[cloudproviderapi.AnnotationAlphaProvidedIPAddr]; ok {
		meta, err := i.c.getNodeMetadata(ctx, providedIP)
		if err != nil {
			return nil, fmt.Errorf("error getting metadata from the node %s: %w", node.Name, err)
		}

		klog.V(5).Infof("instances.InstanceMetadata() resource: %+v", meta)

		providerID := meta.ProviderID
		if providerID == "" {
			providerID = fmt.Sprintf("%s://%s/%s", ProviderName, meta.Platform, providedIP)
		}

		// Fix for Azure, resource group name must be lower case.
		if meta.Platform == "azure" {
			providerID, err = platform.AzureConvertResourceGroupNameToLower(providerID)
			if err != nil {
				return nil, fmt.Errorf("error converting resource group name to lower case: %w", err)
			}
		}

		ifaces, err := i.c.getNodeIfaces(ctx, providedIP)
		if err != nil {
			return nil, fmt.Errorf("error getting interfaces list from the node %s: %w", node.Name, err)
		}

		addresses := getNodeAddresses(i.c.config, meta.Platform, providedIP, ifaces)

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

		if err := syncNodeLabels(i.c, node, meta); err != nil {
			klog.Errorf("failed update labels for node %s, %w", node.Name, err)
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

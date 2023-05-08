package talos

import (
	"context"
	"crypto/x509"
	"fmt"
	"strings"

	utilsnet "github.com/siderolabs/talos-cloud-controller-manager/pkg/utils/net"
	"github.com/siderolabs/talos/pkg/machinery/resources/network"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientkubernetes "k8s.io/client-go/kubernetes"
	cloudprovider "k8s.io/cloud-provider"
	cloudproviderapi "k8s.io/cloud-provider/api"
	cloudnodeutil "k8s.io/cloud-provider/node/helpers"
	"k8s.io/klog/v2"
	"k8s.io/utils/strings/slices"
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

func getNodeAddresses(config *cloudConfig, platform, nodeIP string, ifaces []network.AddressStatusSpec) []v1.NodeAddress {
	var publicIPv4s, publicIPv6s, publicIPs []string

	switch platform {
	case "nocloud", "metal":
		for _, iface := range ifaces {
			if iface.LinkName == "kubespan" {
				continue
			}

			ip := iface.Address.Addr()
			if ip.IsGlobalUnicast() && !ip.IsPrivate() {
				if ip.Is6() {
					publicIPv6s = append(publicIPv6s, ip.String())
				} else {
					publicIPv4s = append(publicIPv4s, ip.String())
				}
			}
		}
	default:
		for _, iface := range ifaces {
			if iface.LinkName == "external" {
				ip := iface.Address.Addr()

				if ip.Is6() {
					publicIPv6s = append(publicIPv6s, ip.String())
				} else {
					publicIPv4s = append(publicIPv4s, ip.String())
				}
			}
		}
	}

	addresses := []v1.NodeAddress{{Type: v1.NodeInternalIP, Address: nodeIP}}

	if config.Global.PreferIPv6 {
		publicIPs = utilsnet.SortedNodeIPs(nodeIP, publicIPv6s, publicIPv4s)
	} else {
		publicIPs = utilsnet.SortedNodeIPs(nodeIP, publicIPv4s, publicIPv6s)
	}

	for _, ip := range publicIPs {
		addresses = append(addresses, v1.NodeAddress{Type: v1.NodeExternalIP, Address: ip})
	}

	return addresses
}

func syncNodeLabels(c *client, node *v1.Node, meta *runtime.PlatformMetadataSpec) error {
	nodeLabels := node.ObjectMeta.Labels
	labelsToUpdate := map[string]string{}

	if nodeLabels == nil {
		nodeLabels = map[string]string{}
	}

	if meta.Platform != "" && nodeLabels[ClusterNodePlatformLabel] != meta.Platform {
		labelsToUpdate[ClusterNodePlatformLabel] = meta.Platform
	}

	if meta.Spot && nodeLabels[ClusterNodeLifeCycleLabel] != "spot" {
		labelsToUpdate[ClusterNodeLifeCycleLabel] = "spot"
	}

	if clusterName := c.talos.GetClusterName(); clusterName != "" && nodeLabels[ClusterNameNodeLabel] != clusterName {
		labelsToUpdate[ClusterNameNodeLabel] = clusterName
	}

	if len(labelsToUpdate) > 0 {
		if !cloudnodeutil.AddOrUpdateLabelsOnNode(c.kclient, labelsToUpdate, node) {
			return fmt.Errorf("failed update labels for node %s", node.Name)
		}
	}

	return nil
}

// TODO: add more checks, like domain name, worker nodes don't have controlplane IPs, etc...
func csrNodeChecks(ctx context.Context, kclient clientkubernetes.Interface, x509cr *x509.CertificateRequest) (bool, error) {
	node, err := kclient.CoreV1().Nodes().Get(ctx, x509cr.DNSNames[0], metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to get node %s: %w", x509cr.DNSNames[0], err)
	}

	var nodeAddrs []string

	if node != nil {
		if providedIP, ok := node.ObjectMeta.Annotations[cloudproviderapi.AnnotationAlphaProvidedIPAddr]; ok {
			nodeAddrs = append(nodeAddrs, providedIP)
		}

		for _, ip := range node.Status.Addresses {
			nodeAddrs = append(nodeAddrs, ip.Address)
		}

		for _, ip := range x509cr.IPAddresses {
			if !slices.Contains(nodeAddrs, ip.String()) {
				return false, fmt.Errorf("csrNodeChecks: CSR %s Node IP addresses don't match corresponding "+
					"Node IP addresses %q, got %q", x509cr.DNSNames[0], nodeAddrs, ip)
			}
		}

		return true, nil
	}

	return false, fmt.Errorf("failed to get node %s", x509cr.DNSNames[0])
}

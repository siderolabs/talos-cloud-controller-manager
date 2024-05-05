package talos

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/siderolabs/talos-cloud-controller-manager/pkg/metrics"
	"github.com/siderolabs/talos-cloud-controller-manager/pkg/transformer"
	utilsnet "github.com/siderolabs/talos-cloud-controller-manager/pkg/utils/net"
	"github.com/siderolabs/talos/pkg/machinery/resources/network"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	clientkubernetes "k8s.io/client-go/kubernetes"
	cloudproviderapi "k8s.io/cloud-provider/api"
	cloudnodeutil "k8s.io/cloud-provider/node/helpers"
	"k8s.io/utils/strings/slices"
)

func ipDescovery(nodeIPs []string, ifaces []network.AddressStatusSpec) (publicIPv4s, publicIPv6s []string) {
	for _, iface := range ifaces {
		if iface.LinkName == "kubespan" || iface.LinkName == "lo" {
			continue
		}

		ip := iface.Address.Addr()
		if ip.IsGlobalUnicast() && !ip.IsPrivate() {
			if slices.Contains(nodeIPs, ip.String()) {
				continue
			}

			if ip.Is6() {
				publicIPv6s = append(publicIPv6s, ip.String())
			} else {
				publicIPv4s = append(publicIPv4s, ip.String())
			}
		}
	}

	return publicIPv4s, publicIPv6s
}

func getNodeAddresses(config *cloudConfig, platform string, features *transformer.NodeFeaturesFlagSpec, nodeIPs []string, ifaces []network.AddressStatusSpec) []v1.NodeAddress {
	var publicIPv4s, publicIPv6s, publicIPs []string

	switch platform {
	// Those platforms don't expose public IPs information in metadata
	case "nocloud", "metal", "openstack", "oracle":
		publicIPv4s, publicIPv6s = ipDescovery(nodeIPs, ifaces)
	default:
		for _, iface := range ifaces {
			if iface.LinkName == "external" {
				ip := iface.Address.Addr()

				if slices.Contains(nodeIPs, ip.String()) {
					continue
				}

				if ip.Is6() {
					publicIPv6s = append(publicIPv6s, ip.String())
				} else {
					publicIPv4s = append(publicIPv4s, ip.String())
				}
			}
		}
	}

	if features != nil && features.PublicIPDiscovery {
		ipv4, ipv6 := ipDescovery(nodeIPs, ifaces)
		publicIPv4s = append(publicIPv4s, ipv4...)
		publicIPv6s = append(publicIPv6s, ipv6...)
	}

	addresses := []v1.NodeAddress{}
	for _, ip := range utilsnet.PreferedDualStackNodeIPs(config.Global.PreferIPv6, nodeIPs) {
		addresses = append(addresses, v1.NodeAddress{Type: v1.NodeInternalIP, Address: ip})
	}

	publicIPs = utilsnet.PreferedDualStackNodeIPs(config.Global.PreferIPv6, append(publicIPv4s, publicIPv6s...))
	for _, ip := range publicIPs {
		addresses = append(addresses, v1.NodeAddress{Type: v1.NodeExternalIP, Address: ip})
	}

	return addresses
}

func syncNodeAnnotations(ctx context.Context, c *client, node *v1.Node, nodeAnnotations map[string]string) error {
	nodeAnnotationsOrig := node.ObjectMeta.Labels
	annotationsToUpdate := map[string]string{}

	for k, v := range nodeAnnotations {
		if r, ok := nodeAnnotationsOrig[k]; !ok || r != v {
			annotationsToUpdate[k] = v
		}
	}

	if len(annotationsToUpdate) > 0 {
		oldData, err := json.Marshal(node)
		if err != nil {
			return fmt.Errorf("failed to marshal the existing node %#v: %w", node, err)
		}

		newNode := node.DeepCopy()
		if newNode.Annotations == nil {
			newNode.Annotations = make(map[string]string)
		}

		for k, v := range annotationsToUpdate {
			newNode.Annotations[k] = v
		}

		newData, err := json.Marshal(newNode)
		if err != nil {
			return fmt.Errorf("failed to marshal the new node %#v: %w", newNode, err)
		}

		patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, &v1.Node{})
		if err != nil {
			return fmt.Errorf("failed to create a two-way merge patch: %v", err)
		}

		if _, err := c.kclient.CoreV1().Nodes().Patch(ctx, node.Name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{}); err != nil {
			return fmt.Errorf("failed to patch the node: %v", err)
		}
	}

	return nil
}

func setTalosNodeLabels(c *client, meta *runtime.PlatformMetadataSpec) map[string]string {
	if meta == nil {
		return make(map[string]string)
	}

	labels := make(map[string]string, 3)

	if meta.Platform != "" {
		labels[ClusterNodePlatformLabel] = meta.Platform
	}

	if meta.Spot {
		labels[ClusterNodeLifeCycleLabel] = "spot"
	}

	if clusterName := c.talos.GetClusterName(); clusterName != "" {
		labels[ClusterNameNodeLabel] = clusterName
	}

	return labels
}

func syncNodeLabels(c *client, node *v1.Node, nodeLabels map[string]string) error {
	nodeLabelsOrig := node.ObjectMeta.Labels
	labelsToUpdate := map[string]string{}

	for k, v := range nodeLabels {
		if r, ok := nodeLabelsOrig[k]; !ok || r != v {
			labelsToUpdate[k] = v
		}
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
			nodeAddrs = append(nodeAddrs, strings.Split(providedIP, ",")...)
		}

		for _, ip := range node.Status.Addresses {
			nodeAddrs = append(nodeAddrs, ip.Address)
		}

		for _, ip := range x509cr.IPAddresses {
			if !slices.Contains(nodeAddrs, ip.String()) {
				metrics.CSRApprovedCount(metrics.ApprovalStatusDeny)

				return false, fmt.Errorf("csrNodeChecks: CSR %s Node IP addresses don't match corresponding "+
					"Node IP addresses %q, got %q", x509cr.DNSNames[0], nodeAddrs, ip)
			}
		}

		metrics.CSRApprovedCount(metrics.ApprovalStatusApprove)

		return true, nil
	}

	return false, fmt.Errorf("failed to get node %s", x509cr.DNSNames[0])
}

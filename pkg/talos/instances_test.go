package talos

import (
	"context"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/siderolabs/talos/pkg/machinery/resources/network"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cloudprovider "k8s.io/cloud-provider"
	cloudproviderapi "k8s.io/cloud-provider/api"
)

func TestGetNodeAddresses(t *testing.T) {
	cfg := cloudConfig{}

	for _, tt := range []struct {
		name       string
		platform   string
		providedIP string
		ifaces     []network.AddressStatusSpec
		expected   []v1.NodeAddress
	}{
		{
			name:       "nocloud has no PublicIPs",
			platform:   "nocloud",
			providedIP: "192.168.0.1",
			ifaces: []network.AddressStatusSpec{
				{Address: netip.MustParsePrefix("192.168.0.1/24")},
				{Address: netip.MustParsePrefix("fe80::e0b5:71ff:fe24:7e60/64")},
				{Address: netip.MustParsePrefix("fd15:1:2::192:168:0:1/64")},
				{Address: netip.MustParsePrefix("fd43:fe8a:be2:ab02:dc3c:38ff:fe51:5022/64"), LinkName: "kubespan"},
			},
			expected: []v1.NodeAddress{
				{Type: v1.NodeInternalIP, Address: "192.168.0.1"},
			},
		},
		{
			name:       "nocloud has many PublicIPs",
			platform:   "nocloud",
			providedIP: "192.168.0.1",
			ifaces: []network.AddressStatusSpec{
				{Address: netip.MustParsePrefix("192.168.0.1/24")},
				{Address: netip.MustParsePrefix("fe80::e0b5:71ff:fe24:7e60/64")},
				{Address: netip.MustParsePrefix("fd15:1:2::192:168:0:1/64")},
				{Address: netip.MustParsePrefix("1.2.3.4/24")},
				{Address: netip.MustParsePrefix("4.3.2.1/24")},
				{Address: netip.MustParsePrefix("2001:1234::1/64")},
				{Address: netip.MustParsePrefix("2001:1234:4321::32/64")},
			},
			expected: []v1.NodeAddress{
				{Type: v1.NodeInternalIP, Address: "192.168.0.1"},
				{Type: v1.NodeExternalIP, Address: "1.2.3.4"},
				{Type: v1.NodeExternalIP, Address: "2001:1234:4321::32"},
			},
		},
		{
			name:       "metal has PublicIPs",
			platform:   "metal",
			providedIP: "192.168.0.1",
			ifaces: []network.AddressStatusSpec{
				{Address: netip.MustParsePrefix("192.168.0.1/24")},
				{Address: netip.MustParsePrefix("fe80::e0b5:71ff:fe24:7e60/64")},
				{Address: netip.MustParsePrefix("fd15:1:2::192:168:0:1/64")},
				{Address: netip.MustParsePrefix("1.2.3.4/24")},
				{Address: netip.MustParsePrefix("2001:1234::1/128")},
			},
			expected: []v1.NodeAddress{
				{Type: v1.NodeInternalIP, Address: "192.168.0.1"},
				{Type: v1.NodeExternalIP, Address: "1.2.3.4"},
				{Type: v1.NodeExternalIP, Address: "2001:1234::1"},
			},
		},
		{
			name:       "gcp has provided PublicIPs",
			platform:   "gcp",
			providedIP: "192.168.0.1",
			ifaces: []network.AddressStatusSpec{
				{Address: netip.MustParsePrefix("192.168.0.1/24")},
				{Address: netip.MustParsePrefix("fe80::e0b5:71ff:fe24:7e60/64")},
				{Address: netip.MustParsePrefix("1.2.3.4/24"), LinkName: "external"},
				{Address: netip.MustParsePrefix("4.3.2.1/24")},
				{Address: netip.MustParsePrefix("2001:1234::1/128"), LinkName: "external"},
				{Address: netip.MustParsePrefix("2001:1234::123/64")},
			},
			expected: []v1.NodeAddress{
				{Type: v1.NodeInternalIP, Address: "192.168.0.1"},
				{Type: v1.NodeExternalIP, Address: "1.2.3.4"},
				{Type: v1.NodeExternalIP, Address: "2001:1234::1"},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			addresses := getNodeAddresses(&cfg, tt.platform, tt.providedIP, tt.ifaces)

			assert.Equal(t, tt.expected, addresses)
		})
	}
}

func TestInstanceMetadata(t *testing.T) {
	cfg := cloudConfig{}
	cfg.Global.SkipForeignNode = true

	ctx := context.Background()
	client, err := newClient(ctx, &cfg)
	assert.NoError(t, err)

	i := newInstances(client)

	for _, tt := range []struct {
		name     string
		node     *v1.Node
		expected *cloudprovider.InstanceMetadata
	}{
		{
			name: "node has providerID",
			node: &v1.Node{
				Spec: v1.NodeSpec{ProviderID: "provider:///id"},
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
					cloudproviderapi.AnnotationAlphaProvidedIPAddr: "192.168.1.1",
				}},
			},
			expected: &cloudprovider.InstanceMetadata{},
		},
		{
			name: "node does not have --cloud-provider=external",
			node: &v1.Node{
				Spec: v1.NodeSpec{},
			},
			expected: &cloudprovider.InstanceMetadata{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := i.InstanceMetadata(ctx, tt.node)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, metadata)
		})
	}
}

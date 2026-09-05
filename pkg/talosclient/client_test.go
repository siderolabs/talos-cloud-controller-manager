/*
Copyright 2024 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package talosclient_test

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/siderolabs/talos-cloud-controller-manager/pkg/talosclient"
	"github.com/siderolabs/talos/pkg/machinery/nethelpers"
	"github.com/siderolabs/talos/pkg/machinery/resources/network"
)

func TestNodeIPDiscovery(t *testing.T) {
	for _, tt := range []struct {
		name          string
		nodeIPs       []string
		ifaces        []network.AddressStatusSpec
		expectedIPv4s []string
		expectedIPv6s []string
	}{
		{
			name:    "No interfaces",
			nodeIPs: []string{},
			ifaces:  []network.AddressStatusSpec{},
		},
		{
			name:    "Private and link-local addresses are skipped",
			nodeIPs: []string{},
			ifaces: []network.AddressStatusSpec{
				{Address: netip.MustParsePrefix("192.168.0.1/24")},
				{Address: netip.MustParsePrefix("10.0.0.1/8")},
				{Address: netip.MustParsePrefix("172.16.0.1/12")},
				{Address: netip.MustParsePrefix("fe80::e0b5:71ff:fe24:7e60/64")},
				{Address: netip.MustParsePrefix("fd15:1:2::192:168:0:1/64")},
			},
		},
		{
			name:    "Public IPv4 addresses",
			nodeIPs: []string{},
			ifaces: []network.AddressStatusSpec{
				{Address: netip.MustParsePrefix("192.168.0.1/24")},
				{Address: netip.MustParsePrefix("1.2.3.4/24"), LinkName: "external"},
				{Address: netip.MustParsePrefix("4.3.2.1/24")},
			},
			expectedIPv4s: []string{"1.2.3.4", "4.3.2.1"},
		},
		{
			name:    "Public IPv6 addresses",
			nodeIPs: []string{},
			ifaces: []network.AddressStatusSpec{
				{Address: netip.MustParsePrefix("fd15:1:2::192:168:0:1/64")},
				{Address: netip.MustParsePrefix("2001:1234::1/64")},
				{Address: netip.MustParsePrefix("2001:1234:4321::32/64")},
			},
			expectedIPv6s: []string{"2001:1234::1", "2001:1234:4321::32"},
		},
		{
			name:    "Filtered link names are skipped",
			nodeIPs: []string{},
			ifaces: []network.AddressStatusSpec{
				{Address: netip.MustParsePrefix("1.2.3.4/24"), LinkName: "kubespan"},
				{Address: netip.MustParsePrefix("4.3.2.1/24"), LinkName: "siderolink"},
				{Address: netip.MustParsePrefix("5.6.7.8/24"), LinkName: "lo"},
				{Address: netip.MustParsePrefix("9.8.7.6/24"), LinkName: "cilium_host"},
				{Address: netip.MustParsePrefix("11.12.13.14/24"), LinkName: "dummy0"},
				{Address: netip.MustParsePrefix("15.16.17.18/24"), LinkName: "eth0"},
			},
			expectedIPv4s: []string{"15.16.17.18"},
		},
		{
			name:    "Node IPs are excluded",
			nodeIPs: []string{"1.2.3.4", "2001:1234::1"},
			ifaces: []network.AddressStatusSpec{
				{Address: netip.MustParsePrefix("1.2.3.4/24")},
				{Address: netip.MustParsePrefix("4.3.2.1/24")},
				{Address: netip.MustParsePrefix("2001:1234::1/64")},
				{Address: netip.MustParsePrefix("2001:1234:4321::32/64")},
			},
			expectedIPv4s: []string{"4.3.2.1"},
			expectedIPv6s: []string{"2001:1234:4321::32"},
		},
		{
			name:    "Permanent IPv6 addresses have priority",
			nodeIPs: []string{},
			ifaces: []network.AddressStatusSpec{
				{Address: netip.MustParsePrefix("2001:1234:1:2:3:4:5:6/64"), Flags: nethelpers.AddressFlags(nethelpers.AddressManagementTemp)},
				{Address: netip.MustParsePrefix("2001:1234::1/64"), Flags: nethelpers.AddressFlags(nethelpers.AddressPermanent)},
			},
			expectedIPv6s: []string{"2001:1234::1", "2001:1234:1:2:3:4:5:6"},
		},
		{
			name:    "Dualstack with mixed addresses",
			nodeIPs: []string{"192.168.0.1", "fd15:1:2::192:168:0:1"},
			ifaces: []network.AddressStatusSpec{
				{Address: netip.MustParsePrefix("192.168.0.1/24")},
				{Address: netip.MustParsePrefix("fe80::e0b5:71ff:fe24:7e60/64")},
				{Address: netip.MustParsePrefix("fd15:1:2::192:168:0:1/64")},
				{Address: netip.MustParsePrefix("1.2.3.4/24"), LinkName: "external"},
				{Address: netip.MustParsePrefix("2001:1234::1/64"), Flags: nethelpers.AddressFlags(nethelpers.AddressPermanent)},
			},
			expectedIPv4s: []string{"1.2.3.4"},
			expectedIPv6s: []string{"2001:1234::1"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ipv4s, ipv6s := talosclient.NodeIPDiscovery(tt.nodeIPs, tt.ifaces)

			assert.Equal(t, tt.expectedIPv4s, ipv4s)
			assert.Equal(t, tt.expectedIPv6s, ipv6s)
		})
	}
}

func TestNodeCIDRDiscovery(t *testing.T) {
	for _, tt := range []struct {
		name            string
		filterIPs       []netip.Addr
		ifaces          []network.AddressStatusSpec
		expectedCIDRv4s []string
		expectedCIDRv6s []string
	}{
		{
			name:      "No interfaces",
			filterIPs: []netip.Addr{},
			ifaces:    []network.AddressStatusSpec{},
		},
		{
			name:      "Private and link-local addresses are skipped",
			filterIPs: []netip.Addr{},
			ifaces: []network.AddressStatusSpec{
				{Address: netip.MustParsePrefix("192.168.0.1/24")},
				{Address: netip.MustParsePrefix("10.0.0.1/8")},
				{Address: netip.MustParsePrefix("fe80::e0b5:71ff:fe24:7e60/64")},
				{Address: netip.MustParsePrefix("fd15:1:2::192:168:0:1/64")},
			},
		},
		{
			name:      "Public IPv4 CIDRs",
			filterIPs: []netip.Addr{},
			ifaces: []network.AddressStatusSpec{
				{Address: netip.MustParsePrefix("192.168.0.1/24")},
				{Address: netip.MustParsePrefix("1.2.3.4/24"), LinkName: "external"},
				{Address: netip.MustParsePrefix("4.3.2.1/24")},
			},
			expectedCIDRv4s: []string{"1.2.3.4/24", "4.3.2.1/24"},
		},
		{
			name:      "Public IPv6 CIDRs",
			filterIPs: []netip.Addr{},
			ifaces: []network.AddressStatusSpec{
				{Address: netip.MustParsePrefix("fd15:1:2::192:168:0:1/64")},
				{Address: netip.MustParsePrefix("2001:1234::1/64")},
				{Address: netip.MustParsePrefix("2001:1234:4321::32/64")},
			},
			expectedCIDRv6s: []string{"2001:1234::1/64", "2001:1234:4321::32/64"},
		},
		{
			name:      "Filtered link names are skipped",
			filterIPs: []netip.Addr{},
			ifaces: []network.AddressStatusSpec{
				{Address: netip.MustParsePrefix("1.2.3.4/24"), LinkName: "kubespan"},
				{Address: netip.MustParsePrefix("4.3.2.1/24"), LinkName: "siderolink"},
				{Address: netip.MustParsePrefix("5.6.7.8/24"), LinkName: "lo"},
				{Address: netip.MustParsePrefix("9.8.7.6/24"), LinkName: "cilium_host"},
				{Address: netip.MustParsePrefix("11.12.13.14/24"), LinkName: "dummy0"},
				{Address: netip.MustParsePrefix("15.16.17.18/24"), LinkName: "eth0"},
			},
			expectedCIDRv4s: []string{"15.16.17.18/24"},
		},
		{
			name:      "Filter IPs",
			filterIPs: []netip.Addr{netip.MustParseAddr("1.2.3.4"), netip.MustParseAddr("2001:1234::1")},
			ifaces: []network.AddressStatusSpec{
				{Address: netip.MustParsePrefix("1.2.3.4/24")},
				{Address: netip.MustParsePrefix("4.3.2.1/24")},
				{Address: netip.MustParsePrefix("2001:1234::1/64")},
				{Address: netip.MustParsePrefix("2001:1234:4321::32/64")},
			},
			expectedCIDRv4s: []string{"1.2.3.4/24"},
			expectedCIDRv6s: []string{"2001:1234::1/64"},
		},
		{
			name:      "Nested IPv6 prefixes are deduplicated",
			filterIPs: []netip.Addr{},
			ifaces: []network.AddressStatusSpec{
				{Address: netip.MustParsePrefix("2001:1234:1:2::1/64")},
				{Address: netip.MustParsePrefix("2001:1234:1:2:3::/80")},
				{Address: netip.MustParsePrefix("2001:1234:1:2:3:4:5:6/128")},
			},
			expectedCIDRv6s: []string{"2001:1234:1:2::1/64"},
		},
		{
			name:      "IPv6 prefixes are sorted by mask",
			filterIPs: []netip.Addr{},
			ifaces: []network.AddressStatusSpec{
				{Address: netip.MustParsePrefix("2001:1234:1:2:3:4:5:6/128")},
				{Address: netip.MustParsePrefix("2001:1234:4321::32/64")},
				{Address: netip.MustParsePrefix("2001:1234::1/56")},
			},
			expectedCIDRv6s: []string{"2001:1234::1/56", "2001:1234:4321::32/64", "2001:1234:1:2:3:4:5:6/128"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cidrv4s, cidrv6s := talosclient.NodeCIDRDiscovery(tt.filterIPs, tt.ifaces)

			assert.Equal(t, tt.expectedCIDRv4s, cidrv4s)
			assert.Equal(t, tt.expectedCIDRv6s, cidrv6s)
		})
	}
}

// Package net includes common net functions.
package net_test

import (
	"fmt"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/v3/assert"

	utilnet "github.com/siderolabs/talos-cloud-controller-manager/pkg/utils/net"
)

func TestFilterLocalNetIPs(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct { //nolint:govet
		name     string
		ips      []string
		expected string
	}{
		{
			name: "v4 and v6 local",
			ips: []string{
				"10.3.4.6",
				"fd00:db8::1",
				"fe80::9b87:57a7:38bf:6c71",
			},
			expected: "[10.3.4.6 fd00:db8::1]",
		},
		{
			name: "not local",
			ips: []string{
				"8.8.8.8",
				"2001:db8:123:445:204::1",
				"169.254.169.254",
			},
			expected: "[]",
		},
	} {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ips := make([]netip.Addr, len(tt.ips))

			for i := range ips {
				ips[i] = netip.MustParseAddr(tt.ips[i])
			}

			result, err := utilnet.FilterLocalNetIPs(ips)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, fmt.Sprintf("%s", result))
		})
	}
}

func TestSortedNodeIPs(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct { //nolint:govet
		name     string
		nodeIP   string
		ipGr1    []string
		ipGr2    []string
		expected string
	}{
		{
			name:   "remove nodeIP",
			nodeIP: "192.168.0.1",
			ipGr1: []string{
				"192.168.0.1",
			},
			ipGr2: []string{
				"fe80::9b87:57a7:38bf:6c71",
			},
			expected: "[fe80::9b87:57a7:38bf:6c71]",
		},
		{
			name:   "sorted list",
			nodeIP: "192.168.0.1",
			ipGr1: []string{
				"192.168.0.1",
				"1.1.1.1",
				"8.8.8.8",
			},
			ipGr2: []string{
				"fe80::9b87:57a7:38bf:6c71",
				"2000:123:123::9b87:57a7:38bf:6c71",
				"2000:123:123::f",
			},
			expected: "[1.1.1.1 2000:123:123::9b87:57a7:38bf:6c71]",
		},
	} {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := utilnet.SortedNodeIPs(tt.nodeIP, tt.ipGr1, tt.ipGr2)

			assert.Equal(t, tt.expected, fmt.Sprintf("%s", result))
		})
	}
}

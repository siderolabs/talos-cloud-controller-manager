// Package net includes common net functions.
package net

import (
	"net/netip"
	"sort"

	siderolabsnet "github.com/siderolabs/net"
)

// FilterLocalNetIPs filters list of IPs with the local subnets (rfc1918, rfc4193).
func FilterLocalNetIPs(ips []netip.Addr) ([]netip.Addr, error) {
	localSubnets := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "fd00::/8"}

	return siderolabsnet.FilterIPs(ips, localSubnets)
}

// SortedNodeIPs gets fists IP (excluded nodeIP) from the two sorted lists.
func SortedNodeIPs(nodeIP string, first, second []string) (res []string) {
	sort.Strings(first)
	sort.Strings(second)

	for _, ip := range first {
		if ip != nodeIP {
			res = append(res, ip)

			break
		}
	}

	for _, ip := range second {
		if ip != nodeIP {
			res = append(res, ip)

			break
		}
	}

	return res
}

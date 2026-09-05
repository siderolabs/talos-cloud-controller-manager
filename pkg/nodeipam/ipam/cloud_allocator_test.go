/*
Copyright 2016 The Kubernetes Authors.

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

package ipam

import (
	"fmt"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/siderolabs/talos-cloud-controller-manager/pkg/nodeipam/ipam/cidrset"
)

func TestAddCIDRSet(t *testing.T) {
	for _, tt := range []struct {
		name                string
		cidr                string
		maxPods             int
		expectedError       error
		expectedSize        int
		expectedClusterCIDR netip.Prefix
	}{
		{
			name:                "CIDRv6 with mask size 56",
			cidr:                "2000::1111:aaaa:bbbb:cccc:123/56",
			maxPods:             110,
			expectedSize:        1,
			expectedClusterCIDR: netip.MustParsePrefix("2000:0:0:1111::/64"),
		},
		{
			name:                "CIDRv6 with mask size 64",
			cidr:                "2000::aaaa:bbbb:cccc:123/64",
			maxPods:             110,
			expectedSize:        1,
			expectedClusterCIDR: netip.MustParsePrefix("2000::/64"),
		},
		{
			name:                "CIDRv6 with mask size 80",
			cidr:                "2000::aaaa:bbbb:cccc:123/80",
			maxPods:             110,
			expectedSize:        1,
			expectedClusterCIDR: netip.MustParsePrefix("2000::aaaa:0:0:0/80"),
		},
		{
			name:                "CIDRv6 with mask size 96",
			cidr:                "2000::aaaa:bbbb:cccc:123/96",
			maxPods:             110,
			expectedSize:        1,
			expectedClusterCIDR: netip.MustParsePrefix("2000::aaaa:bbbb:0:0/96"),
		},
		{
			name:                "CIDRv6 with mask size 100",
			cidr:                "2000::aaaa:bbbb:cccc:123/100",
			maxPods:             110,
			expectedSize:        1,
			expectedClusterCIDR: netip.MustParsePrefix("2000::aaaa:bbbb:c000:0/100"),
		},
		{
			name:                "CIDRv6 with mask size 106",
			cidr:                "2000::aaaa:bbbb:cccc:123/106",
			maxPods:             110,
			expectedSize:        1,
			expectedClusterCIDR: netip.MustParsePrefix("2000::aaaa:bbbb:ccc0:0/106"),
		},
		{
			name:                "CIDRv6 with mask size 110",
			cidr:                "2000::aaaa:bbbb:cccc:123/110",
			maxPods:             110,
			expectedSize:        1,
			expectedClusterCIDR: netip.MustParsePrefix("2000::aaaa:bbbb:cccc:0/110"),
		},
		{
			name:                "CIDRv6 with mask size 112",
			cidr:                "2000::aaaa:bbbb:cccc:123/112",
			maxPods:             110,
			expectedSize:        1,
			expectedClusterCIDR: netip.MustParsePrefix("2000::aaaa:bbbb:cccc:0/112"),
		},
		{
			name:                "CIDRv6 with mask size 119",
			cidr:                "2000::aaaa:bbbb:cccc:123/119",
			maxPods:             110,
			expectedSize:        1,
			expectedClusterCIDR: netip.MustParsePrefix("2000::aaaa:bbbb:cccc:0/119"),
		},
		{
			name:                "CIDRv6 with mask size 120 (256 addresses), 110 pods",
			cidr:                "2000::aaaa:bbbb:cccc:123/120",
			maxPods:             110,
			expectedSize:        1,
			expectedClusterCIDR: netip.MustParsePrefix("2000::aaaa:bbbb:cccc:100/120"),
		},
		{
			name:          "CIDRv6 with mask size 120 (256 addresses), not enough addresses for pods",
			cidr:          "2000::aaaa:bbbb:cccc:123/120",
			maxPods:       129,
			expectedError: fmt.Errorf("CIDRv6 2000::aaaa:bbbb:cccc:123/120 is too small: pod subnet /121 provides only 126 addresses for 129 pods"),
		},
		{
			name:                "CIDRv6 with mask size 122 (64 addresses), 30 pods",
			cidr:                "2000::aaaa:bbbb:cccc:123/122",
			maxPods:             30,
			expectedSize:        1,
			expectedClusterCIDR: netip.MustParsePrefix("2000::aaaa:bbbb:cccc:100/122"),
		},
		{
			name:          "CIDRv6 with mask size 122 (64 addresses), not enough addresses for 33 pods",
			cidr:          "2000::aaaa:bbbb:cccc:123/122",
			maxPods:       33,
			expectedSize:  1,
			expectedError: fmt.Errorf("CIDRv6 2000::aaaa:bbbb:cccc:123/122 is too small: pod subnet /123 provides only 30 addresses for 33 pods"),
		},
		{
			name:                "CIDRv6 with mask size 123 (32 addresses), 10 pods",
			cidr:                "2000::aaaa:bbbb:cccc:123/123",
			maxPods:             10,
			expectedSize:        1,
			expectedClusterCIDR: netip.MustParsePrefix("2000::aaaa:bbbb:cccc:120/123"),
		},
		{
			name:          "CIDRv6 with mask size 124 (16 addresses)",
			cidr:          "2000::aaaa:bbbb:cccc:123/124",
			maxPods:       110,
			expectedSize:  1,
			expectedError: fmt.Errorf("CIDRv6 is too small: 2000::aaaa:bbbb:cccc:123/124"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cidrSets := make(map[netip.Prefix]*cidrset.CidrSet, 0)
			allocator := cloudAllocator{
				cidrSets: cidrSets,
			}
			err := allocator.addCIDRSet(tt.cidr, tt.maxPods)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, cidrSets, tt.expectedSize)
				assert.Contains(t, cidrSets, tt.expectedClusterCIDR, "CIDRSet not found")
				assert.NotNil(t, cidrSets[tt.expectedClusterCIDR], "CIDRSet not found")
			}
		})
	}
}

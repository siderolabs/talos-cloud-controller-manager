/*
Copyright 2021 The Kubernetes Authors.

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

package node

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"slices"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	cloudproviderapi "k8s.io/cloud-provider/api"
	"k8s.io/klog/v2"
)

type nodeForCIDRMergePatch struct {
	Spec nodeSpecForMergePatch `json:"spec"`
}

type nodeSpecForMergePatch struct {
	PodCIDR  string   `json:"podCIDR"`
	PodCIDRs []string `json:"podCIDRs,omitempty"`
}

type nodeForConditionPatch struct {
	Status nodeStatusForPatch `json:"status"`
}

type nodeStatusForPatch struct {
	Conditions []v1.NodeCondition `json:"conditions"`
}

// PatchNodeCIDR patches the specified node's CIDR to the given value.
// nolint:nlreturn,nilerr,wsl
func PatchNodeCIDR(c clientset.Interface, node types.NodeName, cidr string) error {
	patch := nodeForCIDRMergePatch{
		Spec: nodeSpecForMergePatch{
			PodCIDR: cidr,
		},
	}
	patchBytes, err := json.Marshal(&patch)
	if err != nil {
		return fmt.Errorf("failed to json.Marshal CIDR: %w", err)
	}

	if _, err := c.CoreV1().Nodes().Patch(context.TODO(), string(node), types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("failed to patch node CIDR: %w", err)
	}
	return nil
}

// PatchNodeCIDRs patches the specified node.CIDR=cidrs[0] and node.CIDRs to the given value.
// nolint:nlreturn,nilerr,wsl
func PatchNodeCIDRs(ctx context.Context, c clientset.Interface, node types.NodeName, cidrs []string) error {
	// set the pod cidrs list and set the old pod cidr field
	patch := nodeForCIDRMergePatch{
		Spec: nodeSpecForMergePatch{
			PodCIDR:  cidrs[0],
			PodCIDRs: cidrs,
		},
	}

	patchBytes, err := json.Marshal(&patch)
	if err != nil {
		return fmt.Errorf("failed to json.Marshal CIDR: %v", err)
	}
	klog.FromContext(ctx).V(4).Info("cidrs patch bytes", "patchBytes", string(patchBytes))
	if _, err := c.CoreV1().Nodes().Patch(ctx, string(node), types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("failed to patch node CIDR: %v", err)
	}
	return nil
}

// SetNodeCondition updates specific node condition with patch operation.
// nolint:nlreturn,nilerr,wsl
func SetNodeCondition(c clientset.Interface, node types.NodeName, condition v1.NodeCondition) error {
	generatePatch := func(condition v1.NodeCondition) ([]byte, error) {
		patch := nodeForConditionPatch{
			Status: nodeStatusForPatch{
				Conditions: []v1.NodeCondition{
					condition,
				},
			},
		}
		patchBytes, err := json.Marshal(&patch)
		if err != nil {
			return nil, err
		}
		return patchBytes, nil
	}
	condition.LastHeartbeatTime = metav1.NewTime(time.Now())
	patch, err := generatePatch(condition)
	if err != nil {
		return nil
	}
	_, err = c.CoreV1().Nodes().PatchStatus(context.TODO(), string(node), patch)
	return err
}

// GetNodeIPs return the list of node IPs.
func GetNodeIPs(node *v1.Node) ([]netip.Addr, error) {
	if node == nil {
		return nil, fmt.Errorf("node is nil")
	}

	providedIPs := []string{}

	if providedIP, ok := node.ObjectMeta.Annotations[cloudproviderapi.AnnotationAlphaProvidedIPAddr]; ok {
		providedIPs = strings.Split(providedIP, ",")
	}

	nodeIPs := []netip.Addr{}

	for _, v := range node.Status.Addresses {
		if v.Type != v1.NodeExternalIP && v.Type != v1.NodeInternalIP {
			continue
		}

		ip, err := netip.ParseAddr(v.Address)
		if err != nil {
			return nil, fmt.Errorf("failed to parse IP address: %v", err)
		}

		nodeIPs = append(nodeIPs, ip)
	}

	for _, nodeIP := range providedIPs {
		ip, err := netip.ParseAddr(nodeIP)
		if err != nil {
			return nodeIPs, fmt.Errorf("failed to parse IP address: %v", err)
		}

		if !slices.Contains(nodeIPs, ip) {
			nodeIPs = append(nodeIPs, ip)
		}
	}

	return nodeIPs, nil
}

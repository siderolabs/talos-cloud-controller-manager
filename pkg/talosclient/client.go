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

// Package talosclient impelent talos client.
package talosclient

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/cosi-project/runtime/pkg/resource"

	"github.com/siderolabs/go-retry/retry"
	talos "github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/constants"
	"github.com/siderolabs/talos/pkg/machinery/nethelpers"
	"github.com/siderolabs/talos/pkg/machinery/resources/k8s"
	"github.com/siderolabs/talos/pkg/machinery/resources/network"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
)

// Client is the interface for the Talos client.
type Client struct {
	talos *talos.Client
}

// New is the interface for the Talos client.
func New(ctx context.Context) (*Client, error) {
	clientOpts := []talos.OptionFunc{}
	clientOpts = append(clientOpts, talos.WithDefaultConfig())

	endpoints := os.Getenv("TALOS_ENDPOINTS")
	if endpoints != "" {
		clientOpts = append(clientOpts, talos.WithEndpoints(strings.Split(endpoints, ",")...))
	}

	talos, err := talos.New(ctx, clientOpts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		talos: talos,
	}, nil
}

func (c *Client) refreshTalosClient(ctx context.Context) error {
	if _, err := c.talos.Version(ctx); err != nil {
		talos, err := New(ctx)
		if err != nil {
			return fmt.Errorf("failed to reinitialized talos client: %v", err)
		}

		c.talos.Close() //nolint:errcheck
		c.talos = talos.talos
	}

	return nil
}

// GetPodCIDRs returns the pod CIDRs of the cluster.
func (c *Client) GetPodCIDRs(ctx context.Context) ([]string, error) {
	res, err := c.talos.COSI.Get(ctx, resource.NewMetadata(k8s.ControlPlaneNamespaceName, k8s.ControllerManagerConfigType, k8s.ControllerManagerID, resource.VersionUndefined))
	if err != nil {
		return nil, err
	}

	return res.Spec().(*k8s.ControllerManagerConfigSpec).PodCIDRs, nil
}

// GetServiceCIDRs returns the service CIDRs of the cluster.
func (c *Client) GetServiceCIDRs(ctx context.Context) ([]string, error) {
	res, err := c.talos.COSI.Get(ctx, resource.NewMetadata(k8s.ControlPlaneNamespaceName, k8s.ControllerManagerConfigType, k8s.ControllerManagerID, resource.VersionUndefined))
	if err != nil {
		return nil, err
	}

	return res.Spec().(*k8s.ControllerManagerConfigSpec).ServiceCIDRs, nil
}

// GetNodeIfaces returns the network interfaces of the node.
func (c *Client) GetNodeIfaces(ctx context.Context, nodeIP string) ([]network.AddressStatusSpec, error) {
	nodeCtx := talos.WithNode(ctx, nodeIP)

	var resources resource.List

	err := retry.Constant(10*time.Second, retry.WithUnits(100*time.Millisecond)).Retry(func() error {
		var listErr error

		resources, listErr = c.talos.COSI.List(nodeCtx, resource.NewMetadata(network.NamespaceName, network.AddressStatusType, "", resource.VersionUndefined))
		if listErr != nil {
			err := c.refreshTalosClient(ctx) //nolint:errcheck
			if err != nil {
				return retry.ExpectedError(err)
			}

			return listErr
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error get resources: %w", err)
	}

	iface := []network.AddressStatusSpec{}

	for _, res := range resources.Items {
		iface = append(iface, res.(*network.AddressStatus).TypedSpec().DeepCopy())
	}

	return iface, nil
}

// GetNodeMetadata returns the metadata of the node.
func (c *Client) GetNodeMetadata(ctx context.Context, nodeIP string) (*runtime.PlatformMetadataSpec, error) {
	nodeCtx := talos.WithNode(ctx, nodeIP)

	var resources resource.Resource

	err := retry.Constant(10*time.Second, retry.WithUnits(100*time.Millisecond)).Retry(func() error {
		var getErr error

		resources, getErr = c.talos.COSI.Get(nodeCtx, resource.NewMetadata(runtime.NamespaceName, runtime.PlatformMetadataType, runtime.PlatformMetadataID, resource.VersionUndefined))
		if getErr != nil {
			err := c.refreshTalosClient(ctx) //nolint:errcheck
			if err != nil {
				return retry.ExpectedError(err)
			}

			return getErr
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error get resources: %w", err)
	}

	meta := resources.Spec().(*runtime.PlatformMetadataSpec).DeepCopy()

	return &meta, nil
}

// GetClusterName returns cluster name.
func (c *Client) GetClusterName() string {
	return c.talos.GetClusterName()
}

// NodeIPDiscovery returns the public IPs of the node excluding the given IPs.
func NodeIPDiscovery(nodeIPs []string, ifaces []network.AddressStatusSpec) (publicIPv4s, publicIPv6s []string) {
	for _, iface := range ifaces {
		if iface.LinkName == constants.KubeSpanLinkName ||
			iface.LinkName == constants.SideroLinkName ||
			iface.LinkName == "lo" ||
			strings.HasPrefix(iface.LinkName, "dummy") {
			continue
		}

		ip := iface.Address.Addr()
		if ip.IsGlobalUnicast() && !ip.IsPrivate() {
			if slices.Contains(nodeIPs, ip.String()) {
				continue
			}

			if ip.Is6() {
				// Prioritize permanent IPv6 addresses
				if nethelpers.AddressFlag(iface.Flags)&nethelpers.AddressPermanent != 0 {
					publicIPv6s = append([]string{ip.String()}, publicIPv6s...)
				} else {
					publicIPv6s = append(publicIPv6s, ip.String())
				}
			} else {
				publicIPv4s = append(publicIPv4s, ip.String())
			}
		}
	}

	return publicIPv4s, publicIPv6s
}

// NodeCIDRDiscovery returns the public CIDRs of the node with the given filter IPs.
func NodeCIDRDiscovery(filterIPs []netip.Addr, ifaces []network.AddressStatusSpec) (publicCIDRv4s, publicCIDRv6s []string) {
	for _, iface := range ifaces {
		if iface.LinkName == constants.KubeSpanLinkName ||
			iface.LinkName == constants.SideroLinkName ||
			iface.LinkName == "lo" ||
			strings.HasPrefix(iface.LinkName, "dummy") {
			continue
		}

		ip := iface.Address.Addr()
		if ip.IsGlobalUnicast() && !ip.IsPrivate() {
			if len(filterIPs) == 0 || slices.Contains(filterIPs, ip) {
				cidr := iface.Address.Masked().String()

				if ip.Is6() {
					if slices.Contains(publicCIDRv6s, cidr) {
						continue
					}

					// Prioritize permanent IPv6 addresses
					if nethelpers.AddressFlag(iface.Flags)&nethelpers.AddressPermanent != 0 {
						publicCIDRv6s = append([]string{cidr}, publicCIDRv6s...)
					} else {
						publicCIDRv6s = append(publicCIDRv6s, cidr)
					}
				} else {
					publicCIDRv4s = append(publicCIDRv4s, cidr)
				}
			}
		}
	}

	return publicCIDRv4s, publicCIDRv6s
}

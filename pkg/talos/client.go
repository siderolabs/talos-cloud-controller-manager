package talos

import (
	"context"
	"fmt"

	"github.com/cosi-project/runtime/pkg/resource"

	clienttalos "github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/resources/network"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"

	clientkubernetes "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

type client struct {
	config  *cloudConfig
	talos   *clienttalos.Client
	kclient clientkubernetes.Interface
}

func newClient(ctx context.Context, config *cloudConfig) (*client, error) {
	clientOpts := []clienttalos.OptionFunc{}

	if len(config.Global.Endpoints) > 0 {
		clientOpts = append(clientOpts, clienttalos.WithEndpoints(config.Global.Endpoints...))
	} else {
		clientOpts = append(clientOpts, clienttalos.WithDefaultConfig())
	}

	talos, err := clienttalos.New(ctx, clientOpts...)
	if err != nil {
		return nil, err
	}

	//nolint:revive
	if ver, err := talos.Version(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialized talos client: %w", err)
	} else {
		klog.V(4).Infof("talos api version: %s", ver.String())
	}

	return &client{
		config: config,
		talos:  talos,
	}, nil
}

func (c *client) getNodeMetadata(ctx context.Context, nodeIP string) (*runtime.PlatformMetadataSpec, error) {
	nodeCtx := clienttalos.WithNode(ctx, nodeIP)

	res, err := c.talos.COSI.Get(nodeCtx, resource.NewMetadata(runtime.NamespaceName, runtime.PlatformMetadataType, runtime.PlatformMetadataID, resource.VersionUndefined))
	if err != nil {
		return nil, err
	}

	meta := res.Spec().(*runtime.PlatformMetadataSpec).DeepCopy()

	return &meta, nil
}

func (c *client) getNodeIfaces(ctx context.Context, nodeIP string) ([]network.AddressStatusSpec, error) {
	nodeCtx := clienttalos.WithNode(ctx, nodeIP)

	resources, err := c.talos.COSI.List(nodeCtx, resource.NewMetadata(network.NamespaceName, network.AddressStatusType, "", resource.VersionUndefined))
	if err != nil {
		return nil, fmt.Errorf("error get resources: %w", err)
	}

	iface := []network.AddressStatusSpec{}

	for _, res := range resources.Items {
		iface = append(iface, res.(*network.AddressStatus).TypedSpec().DeepCopy())
	}

	return iface, nil
}

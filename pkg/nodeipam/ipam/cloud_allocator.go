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
	"context"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/siderolabs/talos-cloud-controller-manager/pkg/nodeipam/ipam/cidrset"
	"github.com/siderolabs/talos-cloud-controller-manager/pkg/talosclient"
	controllerutil "github.com/siderolabs/talos-cloud-controller-manager/pkg/utils/controller/node"
	utilnode "github.com/siderolabs/talos-cloud-controller-manager/pkg/utils/node"
	"github.com/siderolabs/talos/pkg/machinery/resources/network"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	informers "k8s.io/client-go/informers/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	cloudproviderapi "k8s.io/cloud-provider/api"
	"k8s.io/klog/v2"
	netutils "k8s.io/utils/net"
)

type cloudAllocator struct {
	client clientset.Interface

	// cluster cidrs as passed in during controller creation
	clusterCIDRs []*net.IPNet

	// for each entry in cidrSets we maintain a list of what is used and what is not
	lock     sync.Mutex
	cidrSets map[netip.Prefix]*cidrset.CidrSet

	// nodeLister is able to list/get nodes and is populated by the shared informer passed to controller
	nodeLister corelisters.NodeLister
	// nodesSynced returns true if the node shared informer has been synced at least once.
	nodesSynced cache.InformerSynced
	broadcaster record.EventBroadcaster
	recorder    record.EventRecorder

	// queues are where incoming work is placed to de-dup and to allow "easy"
	// rate limited requeues on errors
	queue workqueue.TypedRateLimitingInterface[string]
}

var (
	_ CIDRAllocator = &cloudAllocator{}

	// UninitializedTaint is the taint that is added to nodes that are not yet initialized.
	UninitializedTaint = &v1.Taint{
		Key:    cloudproviderapi.TaintExternalCloudProvider,
		Effect: v1.TaintEffectNoSchedule,
	}
)

// NewCIDRCloudAllocator returns a CIDRAllocator to allocate CIDRs for node (one from each of clusterCIDRs)
// Caller must ensure subNetMaskSize is not less than cluster CIDR mask size.
// Caller must always pass in a list of existing nodes so the new allocator.
// Caller must ensure that ClusterCIDRs are semantically correct e.g (1 for non DualStack, 2 for DualStack etc..)
// can initialize its CIDR map. NodeList is only nil in testing.
//
//nolint:wsl,predeclared,revive,errcheck
func NewCIDRCloudAllocator(
	ctx context.Context,
	client clientset.Interface,
	nodeInformer informers.NodeInformer,
	allocatorParams CIDRAllocatorParams,
	nodeList *v1.NodeList,
) (CIDRAllocator, error) {
	logger := klog.FromContext(ctx)
	if client == nil {
		logger.Error(nil, "kubeClient is nil when starting CIDRRangeAllocator")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	eventBroadcaster := record.NewBroadcaster(record.WithContext(ctx))
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "cidrAllocator"})

	// create a cidrSet for each cidr we operate on
	// cidrSet are mapped to clusterCIDR by CIDR
	cidrSets := make(map[netip.Prefix]*cidrset.CidrSet, len(allocatorParams.ClusterCIDRs))
	for idx, cidr := range allocatorParams.ClusterCIDRs {
		cidrSet, err := cidrset.NewCIDRSet(cidr, allocatorParams.NodeCIDRMaskSizes[idx])
		if err != nil {
			return nil, err
		}

		cidrSets[netip.MustParsePrefix(cidr.String())] = cidrSet
	}

	ra := &cloudAllocator{
		client:       client,
		clusterCIDRs: allocatorParams.ClusterCIDRs,
		cidrSets:     cidrSets,
		nodeLister:   nodeInformer.Lister(),
		nodesSynced:  nodeInformer.Informer().HasSynced,
		broadcaster:  eventBroadcaster,
		recorder:     recorder,
		queue: workqueue.NewTypedRateLimitingQueueWithConfig(
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{Name: "cidrallocator_node"},
		),
	}

	if allocatorParams.ServiceCIDR != nil {
		ra.filterOutServiceRange(logger, allocatorParams.ServiceCIDR)
	} else {
		logger.Info("No Service CIDR provided. Skipping filtering out service addresses")
	}

	if allocatorParams.SecondaryServiceCIDR != nil {
		ra.filterOutServiceRange(logger, allocatorParams.SecondaryServiceCIDR)
	} else {
		logger.Info("No Secondary Service CIDR provided. Skipping filtering out secondary service addresses")
	}

	if nodeList != nil {
		for _, node := range nodeList.Items {
			if len(node.Spec.PodCIDRs) == 0 {
				logger.V(4).Info("Node has no CIDR, ignoring", "node", klog.KObj(&node))

				continue
			}
			logger.V(4).Info("Node has CIDR, occupying it in CIDR map", "node", klog.KObj(&node), "podCIDRs", node.Spec.PodCIDRs)
			if err := ra.occupyPodCIDRs(ctx, &node); err != nil {
				// This will happen if:
				// 1. We find garbage in the podCIDRs field. Retrying is useless.
				// 2. CIDR out of range: This means a node CIDR has changed.
				// This error will keep crashing controller-manager.
				return nil, err
			}
		}
	}

	nodeInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				ra.queue.Add(key)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				ra.queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// The informer cache no longer has the object, and since Node doesn't have a finalizer,
			// we don't see the Update with DeletionTimestamp != 0.
			// TODO: instead of executing the operation directly in the handler, build a small cache with key node.Name
			// and value PodCIDRs use ReleaseCIDR on the reconcile loop so we can retry on `ReleaseCIDR` failures.
			if err := ra.ReleaseCIDR(logger, obj.(*v1.Node)); err != nil {
				utilruntime.HandleError(fmt.Errorf("error while processing CIDR Release: %w", err))
			}
			// IndexerInformer uses a delta nodeQueue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				ra.queue.Add(key)
			}
		},
	})

	return ra, nil
}

// Run starts the CIDRAllocator.
//
//nolint:dupl
func (r *cloudAllocator) Run(ctx context.Context) {
	defer utilruntime.HandleCrash()

	// Start event processing pipeline.
	r.broadcaster.StartStructuredLogging(3)

	logger := klog.FromContext(ctx)
	logger.Info("Sending events to api server")
	r.broadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: r.client.CoreV1().Events("")})

	defer r.broadcaster.Shutdown()

	defer r.queue.ShutDown()

	logger.Info("Starting range CIDR allocator")
	defer logger.Info("Shutting down range CIDR allocator")

	if !cache.WaitForNamedCacheSync("cidrallocator", ctx.Done(), r.nodesSynced) {
		return
	}

	for i := 0; i < cidrUpdateWorkers; i++ {
		go wait.UntilWithContext(ctx, r.runWorker, time.Second)
	}

	<-ctx.Done()
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// queue.
func (r *cloudAllocator) runWorker(ctx context.Context) {
	for r.processNextNodeWorkItem(ctx) {
	}
}

// processNextWorkItem will read a single work item off the queue and
// attempt to process it, by calling the syncHandler.
func (r *cloudAllocator) processNextNodeWorkItem(ctx context.Context) bool {
	obj, shutdown := r.queue.Get()
	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer r.queue.Done.
	err := func(logger klog.Logger, obj string) error {
		// We call Done here so the workNodeQueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the queue and attempted again after a back-off
		// period.
		defer r.queue.Done(obj)

		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced.
		if err := r.syncNode(ctx, obj); err != nil {
			// Put the item back on the queue to handle any transient errors.
			r.queue.AddRateLimited(obj)

			return fmt.Errorf("error syncing '%s': %s, requeuing", obj, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queue again until another change happens.
		r.queue.Forget(obj)
		logger.Info("Successfully synced", "node", obj)

		for k, cidrSet := range r.cidrSets {
			logger.V(5).Info("IPAM status", "node", obj, "subnet", k.String(), "size", cidrSet.String())
		}

		return nil
	}(klog.FromContext(ctx), obj)
	if err != nil {
		utilruntime.HandleError(err)

		return true
	}

	return true
}

func (r *cloudAllocator) syncNode(ctx context.Context, key string) error {
	logger := klog.FromContext(ctx)
	startTime := time.Now()

	defer func() {
		logger.V(4).Info("Finished syncing Node request", "node", key, "elapsed", time.Since(startTime))
	}()

	node, err := r.nodeLister.Get(key)
	if apierrors.IsNotFound(err) {
		logger.V(3).Info("node has been deleted", "node", key)
		// TODO: obtain the node object information to call ReleaseCIDR from here
		// and retry if there is an error.
		return nil
	}

	if err != nil {
		return err
	}
	// Check the DeletionTimestamp to determine if object is under deletion.
	if !node.DeletionTimestamp.IsZero() {
		logger.V(3).Info("node is being deleted", "node", key)

		return r.ReleaseCIDR(logger, node)
	}

	for _, taint := range node.Spec.Taints {
		if taint.MatchTaint(UninitializedTaint) {
			logger.V(4).Info("Node has uninitialized taint, skipping CIDR allocation", "node", klog.KObj(node))

			return nil
		}
	}

	return r.AllocateOrOccupyCIDR(ctx, node)
}

// marks node.PodCIDRs[...] as used in allocator's tracked cidrSet.
func (r *cloudAllocator) occupyPodCIDRs(ctx context.Context, node *v1.Node) error {
	if len(node.Spec.PodCIDRs) == 0 || len(node.Spec.PodCIDRs) > 2 {
		return nil
	}

	for _, cidr := range node.Spec.PodCIDRs {
		_, podCIDR, err := netutils.ParseCIDRSloppy(cidr)
		if err != nil {
			return fmt.Errorf("failed to parse node %s, CIDR %s", node.Name, node.Spec.PodCIDR)
		}

		ok, err := r.occupyCIDR(podCIDR)
		if err != nil {
			return fmt.Errorf("failed to mark cidr[%v] as occupied for node: %v: %v", podCIDR, node.Name, err)
		}

		if !ok {
			_, err := r.defineNodeGlobalCIDRs(ctx, node)
			if err != nil {
				return fmt.Errorf("failed to find a CIDRSet for node %s, CIDR %s: %v", node.Name, cidr, err)
			}
		}
	}

	return nil
}

func (r *cloudAllocator) occupyCIDR(cidr *net.IPNet) (bool, error) {
	ip, _ := netip.AddrFromSlice(cidr.IP)
	for k := range r.cidrSets {
		if k.Contains(ip) {
			if err := r.cidrSets[k].Occupy(cidr); err != nil {
				return false, fmt.Errorf("failed to mark cidr %v as occupied in subnet %s: %v", cidr, k.String(), err)
			}

			return true, nil
		}
	}

	return false, nil
}

// WARNING: If you're adding any return calls or defer any more work from this
// function you have to make sure to update nodesInProcessing properly with the
// disposition of the node when the work is done.
func (r *cloudAllocator) AllocateOrOccupyCIDR(ctx context.Context, node *v1.Node) error {
	if node == nil {
		return nil
	}

	if len(node.Spec.PodCIDRs) > 0 {
		return r.occupyPodCIDRs(ctx, node)
	}

	logger := klog.FromContext(ctx)
	allocatedCIDRs := make([]*net.IPNet, len(r.clusterCIDRs))

	globalCIDR, err := r.defineNodeGlobalCIDRs(ctx, node)
	if err != nil {
		return err
	}

	for idx, cidr := range r.clusterCIDRs {
		ip := netip.MustParseAddr(cidr.IP.String())
		if cidr.IP.To4() == nil && globalCIDR != "" {
			ip = netip.MustParsePrefix(globalCIDR).Addr()
		}

		for k := range r.cidrSets {
			if k.Contains(ip) {
				podCIDR, err := r.cidrSets[k].AllocateNext()
				if err != nil {
					controllerutil.RecordNodeStatusChange(logger, r.recorder, node, "CIDRNotAvailable")

					return fmt.Errorf("failed to allocate cidr from cluster cidr at idx:%v: %v", idx, err)
				}

				allocatedCIDRs[idx] = podCIDR

				break
			}
		}
	}

	// queue the assignment
	logger.V(4).Info("Putting node with CIDR into the work queue", "node", klog.KObj(node), "CIDRs", ipnetToStringList(allocatedCIDRs))

	return r.updateCIDRsAllocation(ctx, node.Name, allocatedCIDRs)
}

// ReleaseCIDR marks node.podCIDRs[...] as unused in our tracked cidrSets.
func (r *cloudAllocator) ReleaseCIDR(logger klog.Logger, node *v1.Node) error {
	if node == nil || len(node.Spec.PodCIDRs) == 0 {
		return nil
	}

	logger.V(4).Info("Release CIDR for node", "CIDR", node.Spec.PodCIDRs, "node", klog.KObj(node))

	return r.releaseCIDRs(node.Spec.PodCIDRs)
}

func (r *cloudAllocator) releaseCIDRs(addresses []string) error {
	for _, cidr := range addresses {
		_, podCIDR, err := netutils.ParseCIDRSloppy(cidr)
		if err != nil {
			return fmt.Errorf("failed to parse CIDR %s: %v", cidr, err)
		}

		for k := range r.cidrSets {
			ip := netip.MustParseAddr(podCIDR.IP.String())
			if k.Contains(ip) {
				if err := r.cidrSets[k].Release(podCIDR); err != nil {
					return fmt.Errorf("error when releasing CIDR %v: %v", cidr, err)
				}

				break
			}
		}
	}

	return nil
}

// Marks all CIDRs with subNetMaskSize that belongs to serviceCIDR as used across all cidrs
// so that they won't be assignable.
func (r *cloudAllocator) filterOutServiceRange(logger klog.Logger, serviceCIDR *net.IPNet) {
	// Checks if service CIDR has a nonempty intersection with cluster
	// CIDR. It is the case if either clusterCIDR contains serviceCIDR with
	// clusterCIDR's Mask applied (this means that clusterCIDR contains
	// serviceCIDR) or vice versa (which means that serviceCIDR contains
	// clusterCIDR).
	_, err := r.occupyCIDR(serviceCIDR)
	if err != nil {
		logger.Error(err, "Error filtering out service cidr out cluster cidr", "CIDR", serviceCIDR)
	}
}

// updateCIDRsAllocation assigns CIDR to Node and sends an update to the API server.
func (r *cloudAllocator) updateCIDRsAllocation(ctx context.Context, nodeName string, allocatedCIDRs []*net.IPNet) error {
	logger := klog.FromContext(ctx)
	cidrsString := ipnetToStringList(allocatedCIDRs)

	node, err := r.nodeLister.Get(nodeName)
	if err != nil {
		logger.Error(err, "Failed while getting node for updating Node.Spec.PodCIDRs", "node", klog.KRef("", nodeName))

		return err
	}

	// if cidr list matches the proposed.
	// then we possibly updated this node
	// and just failed to ack the success.
	if len(node.Spec.PodCIDRs) == len(allocatedCIDRs) {
		match := true

		for idx, cidr := range cidrsString {
			if node.Spec.PodCIDRs[idx] != cidr {
				match = false

				break
			}
		}

		if match {
			logger.V(4).Info("Node already has allocated CIDR. It matches the proposed one", "node", klog.KObj(node), "podCIDRs", cidrsString)

			return nil
		}
	}

	// node has cidrs, release the reserved
	if len(node.Spec.PodCIDRs) != 0 {
		logger.Error(nil, "Node already has a CIDR allocated. Releasing the new one", "node", klog.KObj(node), "podCIDRs", node.Spec.PodCIDRs)

		for _, cidr := range allocatedCIDRs {
			addrs := []string{cidr.String()}
			if releaseErr := r.releaseCIDRs(addrs); releaseErr != nil {
				logger.Error(releaseErr, "Error when releasing CIDR", "CIDR", cidr)
			}
		}

		return nil
	}

	// If we reached here, it means that the node has no CIDR currently assigned. So we set it.
	for i := 0; i < cidrUpdateRetries; i++ {
		if err = utilnode.PatchNodeCIDRs(ctx, r.client, types.NodeName(node.Name), cidrsString); err == nil {
			logger.Info("Set node PodCIDR", "node", klog.KObj(node), "podCIDRs", cidrsString)

			return nil
		}
	}
	// failed release back to the pool
	logger.Error(err, "Failed to update node PodCIDR after multiple attempts", "node", klog.KObj(node), "podCIDRs", cidrsString)
	controllerutil.RecordNodeStatusChange(logger, r.recorder, node, "CIDRAssignmentFailed")
	// We accept the fact that we may leak CIDRs here. This is safer than releasing
	// them in case when we don't know if request went through.
	// NodeController restart will return all falsely allocated CIDRs to the pool.
	if !apierrors.IsServerTimeout(err) {
		logger.Error(err, "CIDR assignment for node failed. Releasing allocated CIDR", "node", klog.KObj(node))

		for _, cidr := range allocatedCIDRs {
			addrs := []string{cidr.String()}
			if releaseErr := r.releaseCIDRs(addrs); releaseErr != nil {
				logger.Error(releaseErr, "Error when releasing CIDR", "CIDR", cidr)
			}
		}
	}

	return err
}

// defineNodeGlobalCIDRs returns the global CIDR for the node.
func (r *cloudAllocator) defineNodeGlobalCIDRs(ctx context.Context, node *v1.Node) (string, error) {
	if node == nil {
		return "", fmt.Errorf("node is nil")
	}

	nodeIPs, err := utilnode.GetNodeIPs(node)
	if err != nil {
		return "", fmt.Errorf("failed to get IPs for node %s: %v", node.Name, err)
	}

	if len(nodeIPs) == 0 {
		return "", fmt.Errorf("node has no addresses")
	}

	logger := klog.FromContext(ctx)
	logger.V(5).Info("Node has addresses", "node", klog.KObj(node), "addresses", nodeIPs)

	talos, err := talosclient.New(ctx)
	if err != nil {
		return "", err
	}

	var ifaces []network.AddressStatusSpec
	for _, ip := range nodeIPs {
		ifaces, err = talos.GetNodeIfaces(ctx, ip.String())

		if err == nil {
			break
		}
	}

	ipv6 := []netip.Addr{}

	for _, nodeIP := range nodeIPs {
		if nodeIP.Is6() && !nodeIP.IsPrivate() {
			ipv6 = append(ipv6, nodeIP)
		}
	}

	_, cidrs := talosclient.NodeCIDRDiscovery(ipv6, ifaces)
	logger.V(4).Info("Node has IPv6 CIDRs", "node", klog.KObj(node), "CIDRs", cidrs)

	if len(cidrs) > 0 {
		r.lock.Lock()
		defer r.lock.Unlock()

		subnets, err := netutils.ParseCIDRs(cidrs)
		if err != nil {
			return "", err
		}

		for k := range r.cidrSets {
			if k.Addr().Is4() || k.Addr().IsPrivate() {
				continue
			}

			for _, subnet := range subnets {
				if ip, ok := netip.AddrFromSlice(subnet.IP); ok && k.Contains(ip) {
					return k.String(), nil
				}
			}
		}

		for _, cidr := range cidrs {
			mask := netip.MustParsePrefix(cidr).Bits()
			if mask == 128 {
				continue
			}

			logger.V(4).Info("Add IPv6 to CIDRSet", "node", klog.KObj(node), "CIDR", cidr)

			err := r.addCIDRSet(cidr)
			if err != nil {
				return "", fmt.Errorf("error to add CIDRv6 to CIDRSet: %v", err)
			}
		}

		return cidrs[0], nil
	}

	return "", nil
}

// addCIDRSet adds a new CIDRSet-v6 to the allocator's tracked CIDR sets.
func (r *cloudAllocator) addCIDRSet(cidr string) error {
	subnet, err := netip.ParsePrefix(cidr)
	if err != nil {
		return err
	}

	mask := subnet.Bits()

	switch {
	case mask < 64:
		subnet, err = subnet.Addr().Prefix(64)
		if err != nil {
			return err
		}

		mask = 80
	case mask > 123:
		return fmt.Errorf("CIDRv6 is too small: %v", subnet.String())
	case mask > 119:
		break
	default:
		mask += 16
	}

	ip := subnet.Masked().Addr()
	net := &net.IPNet{IP: net.ParseIP(ip.String()), Mask: net.CIDRMask(subnet.Bits(), 128)}

	cidrSet, err := cidrset.NewCIDRSet(net, mask)
	if err != nil {
		return err
	}

	k := subnet.Masked()
	if _, ok := r.cidrSets[k]; !ok {
		r.cidrSets[k] = cidrSet
	}

	return nil
}

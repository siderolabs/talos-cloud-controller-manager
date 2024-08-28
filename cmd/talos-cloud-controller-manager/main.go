/*
Copyright 2020 The Kubernetes Authors.

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

// Package main provides the CCM implementation.
package main

import (
	"os"

	"github.com/spf13/pflag"

	kcmnames "github.com/siderolabs/talos-cloud-controller-manager/pkg/names"
	"github.com/siderolabs/talos-cloud-controller-manager/pkg/talos"

	"k8s.io/apimachinery/pkg/util/wait"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/cloud-provider/app"
	"k8s.io/cloud-provider/app/config"
	"k8s.io/cloud-provider/names"
	"k8s.io/cloud-provider/options"
	"k8s.io/component-base/cli"
	cliflag "k8s.io/component-base/cli/flag"
	_ "k8s.io/component-base/metrics/prometheus/clientgo" // load all the prometheus client-go plugins
	_ "k8s.io/component-base/metrics/prometheus/version"  // for version metric registration
	"k8s.io/klog/v2"
)

func main() {
	ccmOptions, err := options.NewCloudControllerManagerOptions()
	if err != nil {
		klog.ErrorS(err, "unable to initialize command options")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	controllerInitializers := app.DefaultInitFuncConstructors
	controllerAliases := names.CCMControllerAliases()

	nodeIpamController := nodeIPAMController{}
	nodeIpamController.nodeIPAMControllerOptions.NodeIPAMControllerConfiguration = &nodeIpamController.nodeIPAMControllerConfiguration
	fss := cliflag.NamedFlagSets{}
	nodeIpamController.nodeIPAMControllerOptions.AddFlags(fss.FlagSet(kcmnames.NodeIpamController))

	controllerInitializers[kcmnames.NodeIpamController] = app.ControllerInitFuncConstructor{
		// "node-controller" is the shared identity of all node controllers, including node, node lifecycle, and node ipam.
		// See https://github.com/kubernetes/kubernetes/pull/72764#issuecomment-453300990 for more context.
		InitContext: app.ControllerInitContext{
			ClientName: "node-controller",
		},
		Constructor: nodeIpamController.startNodeIpamControllerWrapper,
	}

	nodeCSRApproval := nodeCSRApprovalController{}
	controllerInitializers[kcmnames.CertificateSigningRequestApprovingController] = app.ControllerInitFuncConstructor{
		InitContext: app.ControllerInitContext{
			ClientName: talos.ServiceAccountName,
		},
		Constructor: nodeCSRApproval.startNodeCSRApprovalControllerWrapper,
	}

	app.ControllersDisabledByDefault.Insert(kcmnames.NodeLifecycleController)
	app.ControllersDisabledByDefault.Insert(kcmnames.NodeIpamController)
	app.ControllersDisabledByDefault.Insert(kcmnames.CertificateSigningRequestApprovingController)
	controllerAliases["nodeipam"] = kcmnames.NodeIpamController
	controllerAliases["node-csr-approval"] = kcmnames.CertificateSigningRequestApprovingController

	command := app.NewCloudControllerManagerCommand(ccmOptions, cloudInitializer, controllerInitializers, controllerAliases, fss, wait.NeverStop)
	command.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Name == "cloud-provider" {
			if err := flag.Value.Set(talos.ProviderName); err != nil {
				klog.ErrorS(err, "unable to set cloud-provider flag value")
				klog.FlushAndExit(klog.ExitFlushTimeout, 1)
			}
		}
	})

	code := cli.Run(command)
	os.Exit(code)
}

func cloudInitializer(config *config.CompletedConfig) cloudprovider.Interface {
	cloudConfig := config.ComponentConfig.KubeCloudShared.CloudProvider

	// initialize cloud provider with the cloud provider name and config file provided
	cloud, err := cloudprovider.InitCloudProvider(cloudConfig.Name, cloudConfig.CloudConfigFile)
	if err != nil {
		klog.ErrorS(err, "Cloud provider could not be initialized")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	if cloud == nil {
		klog.InfoS("Cloud provider is nil")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	if !cloud.HasClusterID() {
		if config.ComponentConfig.KubeCloudShared.AllowUntaggedCloud {
			klog.InfoS("detected a cluster without a ClusterID. A ClusterID will be required in the future. Please tag your cluster to avoid any future issues")
		} else {
			klog.InfoS("no ClusterID found. A ClusterID is required for the cloud provider to function properly. This check can be bypassed by setting the allow-untagged-cloud option")
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
	}

	return cloud
}

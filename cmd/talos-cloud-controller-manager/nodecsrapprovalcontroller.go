/*
Copyright 2018 The Kubernetes Authors.

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

package main

import (
	"context"

	"github.com/siderolabs/talos-cloud-controller-manager/pkg/certificatesigningrequest"
	"github.com/siderolabs/talos-cloud-controller-manager/pkg/talos"

	cloudprovider "k8s.io/cloud-provider"
	app "k8s.io/cloud-provider/app"
	cloudcontrollerconfig "k8s.io/cloud-provider/app/config"
	genericcontrollermanager "k8s.io/controller-manager/app"
	"k8s.io/controller-manager/controller"
	"k8s.io/klog/v2"
)

type nodeCSRApprovalController struct{}

func (approvalController *nodeCSRApprovalController) startNodeCSRApprovalControllerWrapper(
	initContext app.ControllerInitContext,
	_ *cloudcontrollerconfig.CompletedConfig,
	cloud cloudprovider.Interface,
) app.InitFunc {
	klog.V(4).InfoS("nodeCSRApprovalController.startNodeCSRApprovalControllerWrapper() called")

	return func(ctx context.Context, controllerContext genericcontrollermanager.ControllerContext) (controller.Interface, bool, error) {
		return startNodeCSRApprovalController(ctx, initContext, controllerContext, cloud)
	}
}

func startNodeCSRApprovalController(
	ctx context.Context,
	initContext app.ControllerInitContext,
	controllerContext genericcontrollermanager.ControllerContext,
	_ cloudprovider.Interface,
) (controller.Interface, bool, error) {
	csrController := certificatesigningrequest.NewCsrController(
		controllerContext.ClientBuilder.ClientOrDie(initContext.ClientName),
		talos.CSRNodeChecks,
	)

	go csrController.Run(ctx)

	return nil, true, nil
}

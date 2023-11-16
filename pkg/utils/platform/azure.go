/*
Copyright 2017 The Kubernetes Authors.

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

// Package platform includes platform specific functions.
package platform

import (
	"fmt"
	"regexp"
	"strings"
)

// https://github.com/kubernetes-sigs/cloud-provider-azure/blob/4192b264611aebef8070505dd56680a862acfbbf/pkg/provider/azure_wrap.go#L37
var (
	azureResourceGroupNameRE = regexp.MustCompile(`.*/subscriptions/(?:.*)/resourceGroups/(.+)/providers/(?:.*)`)
)

// AzureConvertResourceGroupNameToLower converts the resource group name in the resource ID to be lowered.
// https://github.com/kubernetes-sigs/cloud-provider-azure/blob/4192b264611aebef8070505dd56680a862acfbbf/pkg/provider/azure_wrap.go#L91
func AzureConvertResourceGroupNameToLower(resourceID string) (string, error) {
	matches := azureResourceGroupNameRE.FindStringSubmatch(resourceID)
	if len(matches) != 2 {
		return "", fmt.Errorf("%q isn't in Azure resource ID format %q", resourceID, azureResourceGroupNameRE.String())
	}

	resourceGroup := matches[1]

	return strings.Replace(resourceID, resourceGroup, strings.ToLower(resourceGroup), 1), nil
}

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

package certificatesigningrequest

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"reflect"
	"slices"
	"strings"

	certificatesv1 "k8s.io/api/certificates/v1"
)

// Source(08/2022): https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/certificates/helpers.go 160f015

func parseCSR(pemBytes []byte) (*x509.CertificateRequest, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		return nil, errNotCertificateRequest
	}

	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, err
	}

	return csr, nil
}

var (
	errNotCertificateRequest      = fmt.Errorf("PEM block type must be CERTIFICATE REQUEST")
	errOrganizationNotSystemNodes = fmt.Errorf("subject organization is not system:nodes")
	errCommonNameNotSystemNode    = fmt.Errorf("subject common name does not begin with system:node: ")
	errDNSOrIPSANRequired         = fmt.Errorf("DNS or IP subjectAltName is required")
	errDNSNameNotAllowed          = fmt.Errorf("DNS subjectAltNames are not allowed")
	errEmailSANNotAllowed         = fmt.Errorf("email subjectAltNames are not allowed")
	errURISANNotAllowed           = fmt.Errorf("URI subjectAltNames are not allowed")
	errKeyUsageMismatch           = fmt.Errorf("key usage does not match")
)

var kubeletServingRequiredUsages = []certificatesv1.KeyUsage{
	certificatesv1.UsageKeyEncipherment,
	certificatesv1.UsageDigitalSignature,
	certificatesv1.UsageServerAuth,
}

func validateKubeletServingCSR(req *x509.CertificateRequest, keyUsages []certificatesv1.KeyUsage) error {
	if len(req.DNSNames) == 0 && len(req.IPAddresses) == 0 {
		return errDNSOrIPSANRequired
	}

	if slices.ContainsFunc(req.DNSNames, func(name string) bool {
		return name == "kubernetes" || strings.HasPrefix(name, "kubernetes.")
	}) {
		return errDNSNameNotAllowed
	}

	if len(req.EmailAddresses) > 0 {
		return errEmailSANNotAllowed
	}

	if len(req.URIs) > 0 {
		return errURISANNotAllowed
	}

	if !reflect.DeepEqual([]string{"system:nodes"}, req.Subject.Organization) {
		return errOrganizationNotSystemNodes
	}

	if !strings.HasPrefix(req.Subject.CommonName, "system:node:") {
		return errCommonNameNotSystemNode
	}

	usageMap := map[certificatesv1.KeyUsage]bool{}
	for _, u := range kubeletServingRequiredUsages {
		usageMap[u] = false
	}

	for _, ku := range keyUsages {
		if _, u := usageMap[ku]; !u {
			return errKeyUsageMismatch
		}

		usageMap[ku] = true
	}

	if usageMap[certificatesv1.UsageServerAuth] && usageMap[certificatesv1.UsageDigitalSignature] {
		return nil
	}

	return errKeyUsageMismatch
}

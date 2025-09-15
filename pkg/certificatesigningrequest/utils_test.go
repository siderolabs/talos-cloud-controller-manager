//nolint:testpackage // Need to reach functions.
package certificatesigningrequest

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	certificatesv1 "k8s.io/api/certificates/v1"
)

func TestParseCSRValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		msg           string
		pemCSR        []byte
		csr           certificatesv1.CertificateSigningRequest
		expectedError error
	}{
		{
			msg:           "empty PEM CSR",
			pemCSR:        []byte(""),
			expectedError: errNotCertificateRequest,
		},
		{
			msg:           "empty PEM data",
			pemCSR:        []byte("-----BEGIN CERTIFICATE REQUEST-----\n-----END CERTIFICATE REQUEST-----\n"),
			expectedError: asn1.SyntaxError{Msg: "sequence truncated"},
		},
		{
			msg:           "wrong PEM data",
			pemCSR:        []byte("-----BEGIN CERTIFICATE REQUEST-----\n1234567890\n-----END CERTIFICATE REQUEST-----\n"),
			expectedError: errNotCertificateRequest,
		},
	}

	for _, testCase := range tests {
		t.Run(fmt.Sprint(testCase.msg), func(t *testing.T) {
			t.Parallel()

			csr, err := parseCSR(testCase.pemCSR)
			assert.NotNil(t, err)
			assert.Nil(t, csr)
			assert.Contains(t, err.Error(), testCase.expectedError.Error())
		})
	}
}

func TestValidateKubeletServingCSRValid(t *testing.T) {
	t.Parallel()

	org := "system:nodes"
	cname := "system:node:valid"
	usages := []certificatesv1.KeyUsage{
		certificatesv1.UsageKeyEncipherment,
		certificatesv1.UsageDigitalSignature,
		certificatesv1.UsageServerAuth,
	}

	tests := []struct {
		msg       string
		x509cr    x509.CertificateRequest
		keyUsages []certificatesv1.KeyUsage
	}{
		{
			msg: "Only DNSNames",
			x509cr: x509.CertificateRequest{
				Subject: pkix.Name{
					CommonName:   cname,
					Organization: []string{org},
				},
				DNSNames: []string{"valid"},
			},
			keyUsages: usages,
		},
		{
			msg: "Only IPAddresses",
			x509cr: x509.CertificateRequest{
				Subject: pkix.Name{
					CommonName:   cname,
					Organization: []string{org},
				},
				IPAddresses: []net.IP{net.ParseIP("1.2.3.4")},
			},
			keyUsages: usages,
		},
		{
			msg: "Key usages RSA",
			x509cr: x509.CertificateRequest{
				Subject: pkix.Name{
					CommonName:   cname,
					Organization: []string{org},
				},
				DNSNames:    []string{"valid"},
				IPAddresses: []net.IP{net.ParseIP("1.2.3.4")},
			},
			keyUsages: []certificatesv1.KeyUsage{
				certificatesv1.UsageKeyEncipherment,
				certificatesv1.UsageDigitalSignature,
				certificatesv1.UsageServerAuth,
			},
		},
		{
			msg: "Key usages ECDSA",
			x509cr: x509.CertificateRequest{
				Subject: pkix.Name{
					CommonName:   cname,
					Organization: []string{org},
				},
				DNSNames:    []string{"valid"},
				IPAddresses: []net.IP{net.ParseIP("1.2.3.4")},
			},
			keyUsages: []certificatesv1.KeyUsage{
				certificatesv1.UsageDigitalSignature,
				certificatesv1.UsageServerAuth,
			},
		},
	}

	for _, testCase := range tests {
		t.Run(fmt.Sprint(testCase.msg), func(t *testing.T) {
			t.Parallel()

			err := validateKubeletServingCSR(&testCase.x509cr, usages)
			assert.NoError(t, err)
		})
	}
}

func TestValidateKubeletServingCSRInvalid(t *testing.T) {
	t.Parallel()

	org := "system:nodes"
	cname := "system:node:invalid"
	dnsNames := []string{"valid"}
	ipAddresses := []net.IP{net.ParseIP("1.2.3.4")}

	usages := []certificatesv1.KeyUsage{
		certificatesv1.UsageDigitalSignature,
		certificatesv1.UsageServerAuth,
	}

	tests := []struct {
		msg           string
		x509cr        x509.CertificateRequest
		keyUsages     []certificatesv1.KeyUsage
		expectedError error
	}{
		{
			msg: "DNSNames or IPAddresses required",
			x509cr: x509.CertificateRequest{
				Subject: pkix.Name{
					CommonName:   cname,
					Organization: []string{org},
				},
			},
			keyUsages:     usages,
			expectedError: errDNSOrIPSANRequired,
		},
		{
			msg: "Invalid DNSNames",
			x509cr: x509.CertificateRequest{
				Subject: pkix.Name{
					CommonName:   cname,
					Organization: []string{org},
				},
				DNSNames: []string{"kubernetes"},
			},
			keyUsages:     usages,
			expectedError: errDNSNameNotAllowed,
		},
		{
			msg: "Invalid DNSNames long form",
			x509cr: x509.CertificateRequest{
				Subject: pkix.Name{
					CommonName:   cname,
					Organization: []string{org},
				},
				DNSNames: []string{"kubernetes.default.svc"},
			},
			keyUsages:     usages,
			expectedError: errDNSNameNotAllowed,
		},
		{
			msg: "Invalid Organization",
			x509cr: x509.CertificateRequest{
				Subject: pkix.Name{
					CommonName:   cname,
					Organization: []string{"invalid"},
				},
				DNSNames:    dnsNames,
				IPAddresses: ipAddresses,
			},
			keyUsages:     usages,
			expectedError: errOrganizationNotSystemNodes,
		},
		{
			msg: "Invalid CommonName",
			x509cr: x509.CertificateRequest{
				Subject: pkix.Name{
					CommonName:   "invalid",
					Organization: []string{org},
				},
				DNSNames:    dnsNames,
				IPAddresses: ipAddresses,
			},
			keyUsages:     usages,
			expectedError: errCommonNameNotSystemNode,
		},
		{
			msg: "Has email addresses",
			x509cr: x509.CertificateRequest{
				Subject: pkix.Name{
					CommonName:   cname,
					Organization: []string{org},
				},
				EmailAddresses: []string{"invalid"},
				DNSNames:       dnsNames,
				IPAddresses:    ipAddresses,
			},
			keyUsages:     usages,
			expectedError: errEmailSANNotAllowed,
		},
		{
			msg: "Has URI addresses",
			x509cr: x509.CertificateRequest{
				Subject: pkix.Name{
					CommonName:   cname,
					Organization: []string{org},
				},
				URIs:        []*url.URL{{Scheme: "https", Host: "invalid"}},
				DNSNames:    dnsNames,
				IPAddresses: ipAddresses,
			},
			keyUsages:     usages,
			expectedError: errURISANNotAllowed,
		},
		{
			msg: "Invalid key usages",
			x509cr: x509.CertificateRequest{
				Subject: pkix.Name{
					CommonName:   cname,
					Organization: []string{org},
				},
				DNSNames:    dnsNames,
				IPAddresses: ipAddresses,
			},
			keyUsages: []certificatesv1.KeyUsage{
				certificatesv1.UsageDigitalSignature,
				certificatesv1.UsageServerAuth,
				certificatesv1.UsageClientAuth,
			},
			expectedError: errKeyUsageMismatch,
		},
		{
			msg: "Invalid key usages, ServerAuth missing",
			x509cr: x509.CertificateRequest{
				Subject: pkix.Name{
					CommonName:   cname,
					Organization: []string{org},
				},
				DNSNames:    dnsNames,
				IPAddresses: ipAddresses,
			},
			keyUsages: []certificatesv1.KeyUsage{
				certificatesv1.UsageDigitalSignature,
				certificatesv1.UsageDigitalSignature,
			},
			expectedError: errKeyUsageMismatch,
		},
	}

	for _, testCase := range tests {
		t.Run(fmt.Sprint(testCase.msg), func(t *testing.T) {
			t.Parallel()

			err := validateKubeletServingCSR(&testCase.x509cr, testCase.keyUsages)
			assert.NotNil(t, err)
			assert.Contains(t, err.Error(), testCase.expectedError.Error())
		})
	}
}

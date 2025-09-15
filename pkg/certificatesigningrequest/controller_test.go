package certificatesigningrequest_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"net"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/siderolabs/talos-cloud-controller-manager/pkg/certificatesigningrequest"

	certificatesv1 "k8s.io/api/certificates/v1"
	clientkubernetes "k8s.io/client-go/kubernetes"
)

const (
	hostname     = "talos-1"
	organization = "system:nodes"
	username     = "system:node:" + hostname
)

var rsaKey *rsa.PrivateKey

func init() {
	res, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	rsaKey = res
}

func generateCSR(t *testing.T, csrTemplate *x509.CertificateRequest) []byte {
	t.Helper()

	csrCertificate, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, rsaKey)
	if err != nil {
		t.Fatalf("Can not create Certificate Request %v", err)
	}

	csr := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrCertificate,
	})

	return csr
}

func TestNewCsrController(t *testing.T) {
	t.Parallel()

	kclient := &clientkubernetes.Clientset{}

	controller := certificatesigningrequest.NewCsrController(kclient,
		func(context.Context, clientkubernetes.Interface, *x509.CertificateRequest) (bool, error) {
			return true, nil
		})

	assert.NotNil(t, controller)
}

func TestControllerReconcileCSR(t *testing.T) {
	t.Parallel()

	controller := certificatesigningrequest.NewCsrController(&clientkubernetes.Clientset{},
		func(_ context.Context, _ clientkubernetes.Interface, x509cr *x509.CertificateRequest) (bool, error) {
			if reflect.DeepEqual(x509cr.DNSNames, []string{"error"}) {
				return false, fmt.Errorf("someting went wrong")
			}

			if !reflect.DeepEqual(x509cr.DNSNames, []string{hostname}) {
				return false, nil
			}

			return true, nil
		})

	assert.NotNil(t, controller)

	tests := []struct {
		msg             string
		csr             certificatesv1.CertificateSigningRequest
		x509cr          x509.CertificateRequest
		expectedValid   bool
		expectedError   error
		expectedMessage string
	}{
		{
			msg: "Not Kubelet CSR",
			csr: certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: certificatesv1.KubeletServingSignerName,
				},
				Status: certificatesv1.CertificateSigningRequestStatus{
					Conditions: []certificatesv1.CertificateSigningRequestCondition{
						{},
					},
				},
			},
			expectedError: fmt.Errorf("already been approved or denied, signer kubernetes.io/kubelet-serving"),
		},
		{
			msg: "Not Kubelet CSR",
			csr: certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: "someothername",
				},
			},
			expectedError: fmt.Errorf("is not Kubelet serving certificate, signer someothername"),
		},
		{
			msg: "Not Kubelet CSR, wrong username",
			csr: certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: certificatesv1.KubeletServingSignerName,
					Username:   "invalid",
				},
				Status: certificatesv1.CertificateSigningRequestStatus{
					Certificate: []byte("somecert"),
				},
			},
			expectedError: fmt.Errorf("ignoring, subject common name does not begin with system:node: , signer kubernetes.io/kubelet-serving"),
		},
		{
			msg: "Already signed CSR",
			csr: certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: certificatesv1.KubeletServingSignerName,
					Username:   username,
				},
				Status: certificatesv1.CertificateSigningRequestStatus{
					Certificate: []byte("somecert"),
				},
			},
			expectedError: fmt.Errorf("ignoring, already signed, username %s", username),
		},
		{
			msg: "Wrong CSR body",
			csr: certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: certificatesv1.KubeletServingSignerName,
					Username:   username,
					Request:    []byte("somecert"),
				},
			},
			expectedError: fmt.Errorf("PEM block type must be CERTIFICATE REQUEST"),
		},
		{
			msg: "Wrong CSR DNS-IP",
			csr: certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: certificatesv1.KubeletServingSignerName,
					Username:   username,
					Request: generateCSR(t, &x509.CertificateRequest{
						Subject: pkix.Name{
							Organization: []string{organization},
							CommonName:   username,
						},
						SignatureAlgorithm: x509.SHA256WithRSA,
					}),
				},
			},
			expectedValid:   false,
			expectedMessage: "This CSR was denied by Talos Cloud Controller Manager, Reason: DNS or IP subjectAltName is required",
		},
		{
			msg: "Approved CSR",
			csr: certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: certificatesv1.KubeletServingSignerName,
					Username:   username,
					Request: generateCSR(t, &x509.CertificateRequest{
						Subject: pkix.Name{
							Organization: []string{organization},
							CommonName:   username,
						},
						DNSNames:           []string{hostname},
						IPAddresses:        []net.IP{net.ParseIP("1.2.3.4")},
						SignatureAlgorithm: x509.SHA256WithRSA,
					}),
					Usages: []certificatesv1.KeyUsage{
						certificatesv1.UsageDigitalSignature,
						certificatesv1.UsageServerAuth,
					},
				},
			},
			expectedValid:   true,
			expectedMessage: "This CSR was approved by Talos Cloud Controller Manager",
		},
		{
			msg: "Denied CSR with invalid DNS",
			csr: certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: certificatesv1.KubeletServingSignerName,
					Username:   username,
					Request: generateCSR(t, &x509.CertificateRequest{
						Subject: pkix.Name{
							Organization: []string{organization},
							CommonName:   username,
						},
						DNSNames:           []string{"invalid"},
						IPAddresses:        []net.IP{net.ParseIP("1.2.3.4")},
						SignatureAlgorithm: x509.SHA256WithRSA,
					}),
					Usages: []certificatesv1.KeyUsage{
						certificatesv1.UsageDigitalSignature,
						certificatesv1.UsageServerAuth,
					},
				},
			},
			expectedValid:   false,
			expectedMessage: "This CSR was denied by Talos Cloud Controller Manager, Reason: providerChecks failed",
		},
		{
			msg: "ProviderChecks has an error",
			csr: certificatesv1.CertificateSigningRequest{
				Spec: certificatesv1.CertificateSigningRequestSpec{
					SignerName: certificatesv1.KubeletServingSignerName,
					Username:   username,
					Request: generateCSR(t, &x509.CertificateRequest{
						Subject: pkix.Name{
							Organization: []string{organization},
							CommonName:   username,
						},
						DNSNames:           []string{"error"},
						IPAddresses:        []net.IP{net.ParseIP("1.2.3.4")},
						SignatureAlgorithm: x509.SHA256WithRSA,
					}),
					Usages: []certificatesv1.KeyUsage{
						certificatesv1.UsageDigitalSignature,
						certificatesv1.UsageServerAuth,
					},
				},
			},
			expectedError: fmt.Errorf("providerChecks has an error: someting went wrong"),
		},
	}

	for _, testCase := range tests {
		t.Run(fmt.Sprint(testCase.msg), func(t *testing.T) {
			t.Parallel()

			valid, err := controller.Reconcile(t.Context(), &testCase.csr)

			if testCase.expectedError != nil {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), testCase.expectedError.Error())
			} else {
				assert.Nil(t, err)
				assert.Equal(t, testCase.expectedValid, valid)
				assert.Len(t, testCase.csr.Status.Conditions, 1)
				assert.Contains(t, testCase.expectedMessage, testCase.csr.Status.Conditions[0].Message)
			}
		})
	}
}

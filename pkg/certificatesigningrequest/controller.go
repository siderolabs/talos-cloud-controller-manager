// Package certificatesigningrequest implements the controller for Node Certificate Signing Request.
package certificatesigningrequest

import (
	"context"
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8swatch "k8s.io/apimachinery/pkg/watch"
	clientkubernetes "k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// ProviderChecks is a function that checks if the CertificateSigningRequest is valid in the provider.
type ProviderChecks func(context.Context, clientkubernetes.Interface, *x509.CertificateRequest) (bool, error)

// Reconciler is the controller for CertificateSigningRequest.
type Reconciler struct {
	kclient        clientkubernetes.Interface
	providerChecks ProviderChecks
}

// NewCsrController returns a new CertificateSigningRequest controller.
func NewCsrController(kclient clientkubernetes.Interface, fn ProviderChecks) *Reconciler {
	return &Reconciler{
		kclient:        kclient,
		providerChecks: fn,
	}
}

// Run the CertificateSigningRequest controller.
//
//nolint:gocyclo
func (r *Reconciler) Run(ctx context.Context) {
	watchTimeoutSeconds := int64(time.Minute * 5)

	for {
		watcher, err := r.kclient.
			CertificatesV1().
			CertificateSigningRequests().
			Watch(ctx, metav1.ListOptions{
				Watch:          true,
				TimeoutSeconds: &watchTimeoutSeconds, // Default timeout: 20 minutes.
			})
		if err != nil {
			klog.Errorf("CertificateSigningRequestReconciler: failed to list CSR resources: %v", err)
			time.Sleep(10 * time.Second) // Pause for a while before retrying, otherwise we'll spam error logs.

			continue
		}

		csrWatcher := k8swatch.Filter(watcher, func(in k8swatch.Event) (out k8swatch.Event, keep bool) {
			if in.Type != k8swatch.Added {
				return in, false
			}

			return in, true
		})

	watch:
		for {
			select {
			case <-ctx.Done():
				klog.V(4).Infof("CertificateSigningRequestReconciler: context canceled, terminating")

				return

			case event, ok := <-csrWatcher.ResultChan():
				if !ok {
					// Server timeout closed the watcher channel, loop again to re-create a new one.
					klog.V(5).Infof("CertificateSigningRequestReconciler: API server closed watcher channel")

					break watch
				}

				csr, ok := event.Object.DeepCopyObject().(*certificatesv1.CertificateSigningRequest)
				if !ok {
					klog.Errorf("CertificateSigningRequestReconciler: expected event of type *CertificateSigningRequest, got %v",
						event.Object.GetObjectKind())

					continue
				}

				valid, err := r.Reconcile(ctx, csr)
				if err != nil {
					klog.Errorf("CertificateSigningRequestReconciler: failed to reconcile CSR %s: %v", csr.Name, err)

					continue
				}

				if _, err := r.kclient.CertificatesV1().CertificateSigningRequests().UpdateApproval(ctx, csr.Name, csr, metav1.UpdateOptions{}); err != nil {
					klog.Errorf("CertificateSigningRequestReconciler: failed to approve/deny CSR %s: %v", csr.Name, err)
				}

				if !valid {
					klog.Warningf("CertificateSigningRequestReconciler: has been denied: %s", csr.Name)
				} else {
					klog.V(3).Infof("CertificateSigningRequestReconciler: has been approved: %s", csr.Name)
				}
			}
		}
	}
}

// Reconcile the CertificateSigningRequest.
func (r *Reconciler) Reconcile(ctx context.Context, csr *certificatesv1.CertificateSigningRequest) (bool, error) {
	switch {
	case len(csr.Status.Conditions) > 0:
		return false, fmt.Errorf("already been approved or denied, signer %s", csr.Spec.SignerName)
	case csr.Spec.SignerName != certificatesv1.KubeletServingSignerName:
		return false, fmt.Errorf("is not Kubelet serving certificate, signer %s", csr.Spec.SignerName)
	case !strings.HasPrefix(csr.Spec.Username, "system:node:"):
		return false, fmt.Errorf("ignoring, %s, signer %s", errCommonNameNotSystemNode, csr.Spec.SignerName)
	case csr.Status.Certificate != nil:
		return false, fmt.Errorf("ignoring, already signed, username %s", csr.Spec.Username)
	default:
		x509cr, err := parseCSR(csr.Spec.Request)
		if err != nil {
			return false, err
		}

		err = validateKubeletServingCSR(x509cr, csr.Spec.Usages)
		if err != nil {
			r.updateApproval(csr, false, err.Error())

			return false, nil
		}

		valid, err := r.providerChecks(ctx, r.kclient, x509cr)
		if err != nil {
			return valid, fmt.Errorf("providerChecks has an error: %v", err)
		}

		if valid {
			r.updateApproval(csr, valid, "all checks passed")
		} else {
			r.updateApproval(csr, valid, "providerChecks failed")
		}

		return valid, nil
	}
}

func (r *Reconciler) updateApproval(csr *certificatesv1.CertificateSigningRequest, approved bool, reason string) {
	if approved {
		csr.Status.Conditions = append(csr.Status.Conditions, certificatesv1.CertificateSigningRequestCondition{
			Type:           certificatesv1.CertificateApproved,
			Status:         corev1.ConditionTrue,
			Reason:         "Approved by TalosCloudControllerManager",
			Message:        "This CSR was approved by Talos Cloud Controller Manager",
			LastUpdateTime: metav1.Time{Time: time.Now().UTC()},
		})
	} else {
		csr.Status.Conditions = append(csr.Status.Conditions, certificatesv1.CertificateSigningRequestCondition{
			Type:           certificatesv1.CertificateDenied,
			Status:         corev1.ConditionTrue,
			Reason:         "Denied by TalosCloudControllerManager",
			Message:        "This CSR was denied by Talos Cloud Controller Manager, Reason: " + reason,
			LastUpdateTime: metav1.Time{Time: time.Now().UTC()},
		})
	}
}

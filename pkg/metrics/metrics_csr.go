package metrics

import (
	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
)

// CSRMetrics contains the metrics for certificate signing requests.
type CSRMetrics struct {
	approvalCount *metrics.CounterVec
}

// CSRApprovalStatus is the status of a CSR.
type CSRApprovalStatus string

const (
	// ApprovalStatusDeny is used when a CSR is denied.
	ApprovalStatusDeny CSRApprovalStatus = "deny"
	// ApprovalStatusApprove is used when a CSR is approved.
	ApprovalStatusApprove CSRApprovalStatus = "approve"
)

var csrMetrics = registerCSRMetrics()

// CSRApprovedCount counts the number of approved, denied and ignored CSRs.
func CSRApprovedCount(status CSRApprovalStatus) {
	csrMetrics.approvalCount.WithLabelValues(string(status)).Inc()
}

func registerCSRMetrics() *CSRMetrics {
	metrics := &CSRMetrics{
		approvalCount: metrics.NewCounterVec(
			&metrics.CounterOpts{
				Name: "talosccm_csr_approval_count",
				Help: "Count of approved, denied and ignored node CSRs",
			}, []string{"status"}),
	}

	legacyregistry.MustRegister(
		metrics.approvalCount,
	)

	return metrics
}

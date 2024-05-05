package metrics

import (
	"time"

	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
)

// TalosMetrics contains the metrics for Talos API calls.
type TalosMetrics struct {
	Duration *metrics.HistogramVec
	Errors   *metrics.CounterVec
}

var apiMetrics = registerAPIMetrics()

// ObserveRequest records the request latency and counts the errors.
func (mc *MetricContext) ObserveRequest(err error) error {
	apiMetrics.Duration.WithLabelValues(mc.attributes...).Observe(
		time.Since(mc.start).Seconds())

	if err != nil {
		apiMetrics.Errors.WithLabelValues(mc.attributes...).Inc()
	}

	return err
}

func registerAPIMetrics() *TalosMetrics {
	metrics := &TalosMetrics{
		Duration: metrics.NewHistogramVec(
			&metrics.HistogramOpts{
				Name:    "talosccm_api_request_duration_seconds",
				Help:    "Latency of an Talos API call",
				Buckets: []float64{.1, .25, .5, 1, 2.5, 5, 10, 30},
			}, []string{"request"}),
		Errors: metrics.NewCounterVec(
			&metrics.CounterOpts{
				Name: "talosccm_api_request_errors_total",
				Help: "Total number of errors for an Talos API call",
			}, []string{"request"}),
	}

	legacyregistry.MustRegister(
		metrics.Duration,
		metrics.Errors,
	)

	return metrics
}

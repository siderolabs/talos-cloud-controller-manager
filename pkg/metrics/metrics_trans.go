package metrics

import (
	"time"

	"k8s.io/component-base/metrics"
	"k8s.io/component-base/metrics/legacyregistry"
)

// TransformerMetrics contains the metrics for transformer.
type TransformerMetrics struct {
	Duration *metrics.HistogramVec
	Errors   *metrics.CounterVec
}

var transformerMetrics = registerTransformerMetrics()

// ObserveTransformer records the transformer latency and counts the errors.
func (mc *MetricContext) ObserveTransformer(err error) error {
	transformerMetrics.Duration.WithLabelValues(mc.attributes...).Observe(
		time.Since(mc.start).Seconds())

	if err != nil {
		transformerMetrics.Errors.WithLabelValues(mc.attributes...).Inc()
	}

	return err
}

func registerTransformerMetrics() *TransformerMetrics {
	metrics := &TransformerMetrics{
		Duration: metrics.NewHistogramVec(
			&metrics.HistogramOpts{
				Name:    "talosccm_transformer_duration_seconds",
				Help:    "Latency of an Transformer call",
				Buckets: []float64{.001, .01, .05, .1},
			}, []string{"type"}),
		Errors: metrics.NewCounterVec(
			&metrics.CounterOpts{
				Name: "talosccm_transformer_errors_total",
				Help: "Total number of errors for an Transformer call",
			}, []string{"type"}),
	}

	legacyregistry.MustRegister(
		metrics.Duration,
		metrics.Errors,
	)

	return metrics
}

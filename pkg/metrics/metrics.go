// Package metrics collects metrics.
package metrics

import (
	"time"
)

// MetricContext indicates the context for Talos client metrics.
type MetricContext struct {
	start      time.Time
	attributes []string
}

// NewMetricContext creates a new MetricContext.
func NewMetricContext(resource string) *MetricContext {
	return &MetricContext{
		start:      time.Now(),
		attributes: []string{resource},
	}
}

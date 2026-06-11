package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ServiceMetrics holds all Prometheus metrics exposed by this service.
// Scraped by Grafana Alloy at /metrics (port APP_METRICS_PORT).
type ServiceMetrics struct {
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestsTotal   *prometheus.CounterVec
	BusinessErrorsTotal *prometheus.CounterVec
}

// NewServiceMetrics registers and returns all service-level metrics.
// Safe to call once at startup — panics if called more than once (promauto behavior).
func NewServiceMetrics(namespace string) *ServiceMetrics {
	return &ServiceMetrics{
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request latency in seconds, by method, route and status code.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "route", "status_code"},
		),
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total HTTP requests received.",
			},
			[]string{"method", "route", "status_code"},
		),
		BusinessErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "business_errors_total",
				Help:      "Business rule violations by domain and rule ID.",
			},
			[]string{"domain", "rule_id"},
		),
	}
}

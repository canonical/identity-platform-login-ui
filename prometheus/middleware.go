package prometheus

import (
	"net/http"
)

type MetricsManager struct {
	prometheusMetrics *Metrics
	routes            []string
}

func NewMetricsManager(app, version, hash, buildTime string) *MetricsManager {
	return NewMetricsManagerWithPrefix(app, "", version, hash, buildTime)
}

// NewMetricsManagerWithPrefix creates MetricsManager that uses metricsPrefix parameters as a prefix
// for all metrics registered within this middleware. Setting empty string in metricsPrefix will be equivalent to calling NewMetricsManager.
func NewMetricsManagerWithPrefix(app, metricsPrefix, version, hash, buildTime string) *MetricsManager {
	return &MetricsManager{
		prometheusMetrics: NewMetrics(app, metricsPrefix, version, hash, buildTime),
	}
}

// Middleware Implementation method to collect metrics for Prometheus.
func (pmm *MetricsManager) Middleware(next http.HandlerFunc) func(rw http.ResponseWriter, r *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		pmm.prometheusMetrics.Instrument(rw, next, pmm.getLabelForPath(r))(rw, r)
	}
}

func (pmm *MetricsManager) RegisterRoutes(routes ...string) {
	for _, route := range routes {
		pmm.RegisterRoute(route)
	}
}

func (pmm *MetricsManager) RegisterRoute(route string) {
	pmm.routes = append(pmm.routes, route)
}

// The URLs we Proxy for Ory APIs do not use path parameters
func (pmm *MetricsManager) getLabelForPath(r *http.Request) string {
	if !pmm.lookupRoutes(r.URL.Path) {
		return "{unmatched}"
	}
	return r.URL.Path
}

// lookupRoutes returns true if url is registered with Middleware Manager
func (pmm *MetricsManager) lookupRoutes(url string) bool {
	for _, v := range pmm.routes {
		if v == url {
			return true
		}
	}

	return false
}

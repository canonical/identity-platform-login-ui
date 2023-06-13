package prometheus

import "github.com/prometheus/client_golang/prometheus"

func cleanup(mm *MetricsManager) func() {
	return func() {
		prometheus.DefaultRegisterer.Unregister(mm.prometheusMetrics)
	}
}

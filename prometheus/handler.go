package prometheus

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	PrometheusPath = "/metrics/prometheus"
)

func PrometheusMetrics(rw http.ResponseWriter, r *http.Request) {
	promhttp.Handler().ServeHTTP(rw, r)
}

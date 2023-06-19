package prometheus

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"regexp"
)

const PrometheusPath = "/metrics/prometheus"

type MetricsManager struct {
	prometheusMetrics *Metrics
	routes            map[string]bool
}

// NewMetricsManagerWithPrefix creates MetricsManager that uses metricsPrefix parameters as a prefix
// for all metrics registered within this middleware. Setting empty string in metricsPrefix will be equivalent to calling NewMetricsManagerWithPrefix.
func NewMetricsManagerWithPrefix(app, metricsPrefix, version, hash, buildTime string) *MetricsManager {
	return &MetricsManager{
		prometheusMetrics: NewMetrics(app, metricsPrefix, version, hash, buildTime),
		routes:            make(map[string]bool),
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
		pmm.routes[route] = true
	}
}

// This method fetches the path for a call for labeling. The URLs we Proxy for Ory APIs do not use path parameters.
func (pmm *MetricsManager) getLabelForPath(r *http.Request) string {
	if !pmm.lookupRoutes(r.URL.Path) {
		return "{unmatched}"
	}
	return r.URL.Path
}

// lookupRoutes returns true if url is registered with Middleware Manager
func (pmm *MetricsManager) lookupRoutes(url string) bool {
	_, ok := pmm.routes[url]

	return ok
}

func setUpPrometheus(jsFS embed.FS) *MetricsManager {
	mm := NewMetricsManagerWithPrefix("identity-platform-login-ui-operator", "http", "", "", "")
	mm.RegisterRoutes(
		"/api/kratos/self-service/login/browser",
		"/api/kratos/self-service/login/flows",
		"/api/kratos/self-service/login",
		"/api/kratos/self-service/errors",
		"/api/consent",
		"/health/alive",
		PrometheusPath,
	)

	pages, err := jsFS.ReadDir("ui/dist")
	if err != nil {
		log.Printf("Error when calling `setUpPrometheus`: %v\n", err)
	}
	mm.RegisterRoutes(registerHelper(pages...)...)
	return mm
}

func registerHelper(dirs ...fs.DirEntry) []string {
	r, _ := regexp.Compile("html")
	ret := make([]string, 0)
	for _, d := range dirs {
		name := d.Name()
		if r.MatchString(name) {
			ret = append(ret, name)
		}
	}
	ret = append(ret, "/")

	return ret
}

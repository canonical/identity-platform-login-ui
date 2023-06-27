package tracing

import (
	"net/http"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Middleware is the monitoring middleware object implementing Prometheus monitoring
type Middleware struct {
	service string

	monitor monitoring.MonitorInterface
	logger  logging.LoggerInterface
}

func (mdw *Middleware) OpenTelemetry(handler http.Handler) http.Handler {
	return otelhttp.NewHandler(handler, "server")
}

// NewMiddleware returns a Middleware based on the type of monitor
func NewMiddleware(monitor monitoring.MonitorInterface, logger logging.LoggerInterface) *Middleware {
	mdw := new(Middleware)

	mdw.monitor = monitor

	mdw.logger = logger

	return mdw
}

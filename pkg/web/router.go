package web

import (
	"io/fs"
	"net/http"

	"github.com/canonical/identity-platform-login-ui/internal/http_meta"
	ih "github.com/canonical/identity-platform-login-ui/internal/hydra"
	ik "github.com/canonical/identity-platform-login-ui/internal/kratos"
	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
	chi "github.com/go-chi/chi/v5"
	middleware "github.com/go-chi/chi/v5/middleware"
	trace "go.opentelemetry.io/otel/trace"

	"github.com/canonical/identity-platform-login-ui/pkg/extra"
	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
	"github.com/canonical/identity-platform-login-ui/pkg/metrics"
	"github.com/canonical/identity-platform-login-ui/pkg/status"
	"github.com/canonical/identity-platform-login-ui/pkg/ui"
)

func NewRouter(kratosClient *ik.Client, hydraClient *ih.Client, distFS fs.FS, baseURL string, tracer trace.Tracer, monitor monitoring.MonitorInterface, logger logging.LoggerInterface) http.Handler {
	router := chi.NewMux()

	middlewares := make(chi.Middlewares, 0)
	middlewares = append(
		middlewares,
		middleware.RequestID,
		monitoring.NewMiddleware(monitor, logger).ResponseTime(),
		middlewareCORS([]string{"*"}),
	)

	// TODO @shipperizer add a proper configuration to enable http logger middleware as it's expensive
	if true {
		middlewares = append(
			middlewares,
			middleware.RequestLogger(logging.NewLogFormatter(logger)), // LogFormatter will only work if logger is set to DEBUG level
		)
	}

	router.Use(middlewares...)

	routerWithMonitoring := http_meta.NewRegisterMux(router, monitor)

	kratos.NewAPI(
		kratos.NewService(kratosClient, hydraClient, tracer, monitor, logger),
		baseURL,
		logger,
	).RegisterEndpoints(routerWithMonitoring)
	extra.NewAPI(
		extra.NewService(kratosClient, hydraClient, tracer, monitor, logger),
		logger,
	).RegisterEndpoints(routerWithMonitoring)
	status.NewAPI(
		status.NewService(kratosClient.MetadataApi(), hydraClient.MetadataApi(), tracer, monitor, logger),
		tracer,
		monitor,
		logger,
	).RegisterEndpoints(routerWithMonitoring)
	ui.NewAPI(distFS, monitor, logger).RegisterEndpoints(routerWithMonitoring)
	metrics.NewAPI(logger).RegisterEndpoints(routerWithMonitoring)

	return tracing.NewMiddleware(monitor, logger).OpenTelemetry(routerWithMonitoring.GetMux())
}

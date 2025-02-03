package web

import (
	"io/fs"
	"net/http"

	chi "github.com/go-chi/chi/v5"
	middleware "github.com/go-chi/chi/v5/middleware"

	authz "github.com/canonical/identity-platform-login-ui/internal/authorization"
	ih "github.com/canonical/identity-platform-login-ui/internal/hydra"
	ik "github.com/canonical/identity-platform-login-ui/internal/kratos"
	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"

	"github.com/canonical/identity-platform-login-ui/pkg/device"
	"github.com/canonical/identity-platform-login-ui/pkg/extra"
	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
	"github.com/canonical/identity-platform-login-ui/pkg/metrics"
	"github.com/canonical/identity-platform-login-ui/pkg/status"
	"github.com/canonical/identity-platform-login-ui/pkg/ui"
)

func NewRouter(
	kratosClient *ik.Client,
	kratosAdminClient *ik.Client,
	hydraClient *ih.Client,
	authzClient authz.AuthorizerInterface,
	cookieManager *kratos.AuthCookieManager,
	distFS fs.FS,
	mfaEnabled bool,
	oidcWebAuthnSequencingEnabled bool,
	baseURL string,
	tracer tracing.TracingInterface,
	monitor monitoring.MonitorInterface,
	logger logging.LoggerInterface,
) http.Handler {
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

	device.NewAPI(
		device.NewService(hydraClient, tracer, monitor, logger),
		logger,
	).RegisterEndpoints(router)
	kratosService := kratos.NewService(kratosClient, kratosAdminClient, hydraClient, authzClient, oidcWebAuthnSequencingEnabled, tracer, monitor, logger)
	kratos.NewAPI(
		kratosService,
		mfaEnabled,
		oidcWebAuthnSequencingEnabled,
		baseURL,
		cookieManager,
		logger,
	).RegisterEndpoints(router)
	extra.NewAPI(extra.NewService(hydraClient, tracer, monitor, logger), kratosService, baseURL, mfaEnabled, oidcWebAuthnSequencingEnabled, logger).RegisterEndpoints(router)
	status.NewAPI(
		baseURL,
		oidcWebAuthnSequencingEnabled,
		status.NewService(kratosClient.MetadataApi(), hydraClient.MetadataApi(), tracer, monitor, logger),
		tracer,
		monitor,
		logger,
	).RegisterEndpoints(router)
	ui.NewAPI(distFS, logger).RegisterEndpoints(router)
	metrics.NewAPI(logger).RegisterEndpoints(router)

	return tracing.NewMiddleware(monitor, logger).OpenTelemetry(router)
}

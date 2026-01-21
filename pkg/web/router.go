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

type Option func(config *routerConfig)

func WithKratosClients(public, admin *ik.Client) Option {
	return func(r *routerConfig) {
		r.kratosClient = public
		r.kratosAdminClient = admin
	}
}

func WithHydraClient(c *ih.Client) Option {
	return func(r *routerConfig) {
		r.hydraClient = c
	}
}

func WithAuthzClient(a authz.AuthorizerInterface) Option {
	return func(r *routerConfig) {
		r.authzClient = a
	}
}

func WithCookieManager(cm *kratos.AuthCookieManager) Option {
	return func(r *routerConfig) {
		r.cookieManager = cm
	}
}

func WithFS(fsys fs.FS) Option {
	return func(r *routerConfig) {
		r.distFS = fsys
	}
}

func WithFlags(mfa, oidcSeq, identifierFirst bool) Option {
	return func(r *routerConfig) {
		r.mfaEnabled = mfa
		r.oidcWebAuthnSequencingEnabled = oidcSeq
		r.identifierFirstEnabled = identifierFirst
	}
}

func WithBaseURL(url string) Option {
	return func(r *routerConfig) {
		r.baseURL = url
	}
}

func WithSupportEmail(email string) Option {
	return func(r *routerConfig) {
		r.supportEmail = email
	}
}

func WithFeatureFlags(flags []string) Option {
	return func(r *routerConfig) {
		r.featureFlags = flags
	}
}

func WithKratosPublicURL(url string) Option {
	return func(r *routerConfig) {
		r.kratosPublicURL = url
	}
}

func WithTracing(t tracing.TracingInterface) Option {
	return func(r *routerConfig) {
		r.tracer = t
	}
}

func WithMonitoring(m monitoring.MonitorInterface) Option {
	return func(r *routerConfig) {
		r.monitor = m
	}
}

func WithLogger(l logging.LoggerInterface) Option {
	return func(r *routerConfig) {
		r.logger = l
	}
}

type routerConfig struct {
	kratosClient                  *ik.Client
	kratosAdminClient             *ik.Client
	hydraClient                   *ih.Client
	authzClient                   authz.AuthorizerInterface
	cookieManager                 *kratos.AuthCookieManager
	distFS                        fs.FS
	mfaEnabled                    bool
	oidcWebAuthnSequencingEnabled bool
	identifierFirstEnabled        bool
	baseURL                       string
	supportEmail                  string
	featureFlags                  []string
	kratosPublicURL               string
	tracer                        tracing.TracingInterface
	monitor                       monitoring.MonitorInterface
	logger                        logging.LoggerInterface
}

func NewRouter(opts ...Option) http.Handler {

	config := &routerConfig{}
	for _, opt := range opts {
		opt(config)
	}

	router := chi.NewMux()
	router.Use(buildMiddlewares(config)...)

	registerAPIs(config, router)

	wrappedRouter := tracing.NewMiddleware(config.monitor, config.logger).OpenTelemetry(router)
	return wrappedRouter
}

func buildMiddlewares(config *routerConfig) chi.Middlewares {
	middlewares := make(chi.Middlewares, 0)
	middlewares = append(
		middlewares,
		middleware.RequestID,
		monitoring.NewMiddleware(config.monitor, config.logger).ResponseTime(),
		middlewareCORS([]string{"*"}),
	)

	if config.logger != nil {
		middlewares = append(
			middlewares,
			middleware.RequestLogger(logging.NewLogFormatter(config.logger)), // LogFormatter will only work if logger is set to DEBUG level
		)
	}

	return middlewares
}

func registerAPIs(config *routerConfig, router *chi.Mux) {
	device.NewAPI(
		device.NewService(config.hydraClient, config.tracer, config.monitor, config.logger),
		config.tracer,
		config.logger,
	).RegisterEndpoints(router)

	kratosService := kratos.NewService(config.kratosClient, config.kratosAdminClient, config.hydraClient, config.authzClient, config.oidcWebAuthnSequencingEnabled, config.tracer, config.monitor, config.logger)
	kratos.NewAPI(
		kratosService,
		config.mfaEnabled,
		config.oidcWebAuthnSequencingEnabled,
		config.baseURL,
		config.cookieManager,
		config.tracer,
		config.logger,
	).RegisterEndpoints(router)

	extra.NewAPI(
		extra.NewService(config.hydraClient, config.tracer, config.monitor, config.logger),
		kratosService,
		config.baseURL,
		config.mfaEnabled,
		config.oidcWebAuthnSequencingEnabled,
		config.tracer,
		config.logger,
	).RegisterEndpoints(router)

	status.NewAPI(
		config.baseURL,
		config.supportEmail,
		config.oidcWebAuthnSequencingEnabled,
		config.identifierFirstEnabled,
		config.featureFlags,
		status.NewService(config.kratosClient.MetadataApi(), config.hydraClient.MetadataAPI(), config.tracer, config.monitor, config.logger),
		config.tracer,
		config.monitor,
		config.logger,
	).RegisterEndpoints(router)

	ui.NewAPI(
		config.distFS,
		config.baseURL,
		config.kratosPublicURL,
		config.logger,
	).RegisterEndpoints(router)

	metrics.NewAPI(config.logger).RegisterEndpoints(router)
}

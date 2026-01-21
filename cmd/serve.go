package cmd

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/go-playground/validator/v10"
	"github.com/kelseyhightower/envconfig"

	authz "github.com/canonical/identity-platform-login-ui/internal/authorization"
	"github.com/canonical/identity-platform-login-ui/internal/config"
	ih "github.com/canonical/identity-platform-login-ui/internal/hydra"
	ik "github.com/canonical/identity-platform-login-ui/internal/kratos"
	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring/prometheus"
	fga "github.com/canonical/identity-platform-login-ui/internal/openfga"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
	"github.com/canonical/identity-platform-login-ui/pkg/web"
)

//go:embed ui/dist
//go:embed ui/dist/_next
//go:embed ui/dist/_next/static/chunks/pages/*.js
//go:embed ui/dist/_next/static/*/*.js
//go:embed ui/dist/_next/static/*/*.css
var jsFS embed.FS

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "serve starts the web server",
	Long:  `Launch the web application, list of environment variables is available in the readme`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return serve()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func serve() error {

	specs := new(config.EnvSpec)
	if err := envconfig.Process("", specs); err != nil {
		panic(fmt.Errorf("issues with environment sourcing: %s", err))
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(specs); err != nil {
		return fmt.Errorf("issues with environment variables validation: %w", err)
	}

	logger := logging.NewLogger(specs.LogLevel)
	defer logger.Sync()

	logger.Debugf("env vars: %v", specs)

	distFS, err := fs.Sub(jsFS, "ui/dist")
	if err != nil {
		return fmt.Errorf("issue with js distribution files: %w", err)
	}

	router, err := buildRouter(specs, distFS, logger)
	if err != nil {
		return err
	}

	logger.Infof("Starting server on port %v", specs.Port)
	srv := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%v", specs.Port),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      router,
	}

	return handleServeAndShutdown(srv, logger.Security())
}

func buildRouter(specs *config.EnvSpec, distFS fs.FS, logger *logging.Logger) (http.Handler, error) {
	monitor := prometheus.NewMonitor("identity-login-ui", logger)
	tracer := tracing.NewTracer(tracing.NewConfig(specs.TracingEnabled, specs.OtelGRPCEndpoint, specs.OtelHTTPEndpoint, logger))

	kClient := ik.NewClient(specs.KratosPublicURL, specs.Debug)
	kAdminClient := ik.NewClient(specs.KratosAdminURL, specs.Debug)
	hClient := ih.NewClient(specs.HydraAdminURL, specs.Debug)

	encrypt := kratos.NewEncrypt([]byte(specs.CookiesEncryptionKey), logger, tracer)
	cookieManager := kratos.NewAuthCookieManager(
		specs.CookieTTL,
		encrypt,
		logger,
	)

	var authzClient authz.AuthzClientInterface
	if specs.AuthorizationEnabled {
		logger.Info("Authorization is enabled")
		cfg := fga.NewConfig(specs.ApiScheme, specs.ApiHost, specs.StoreId, specs.ApiToken, specs.AuthorizationModelId, specs.Debug, tracer, monitor, logger)
		authzClient = fga.NewClient(cfg)
	} else {
		logger.Info("Authorization is disabled, using noop authorizer")
		authzClient = fga.NewNoopClient(tracer, monitor, logger)
	}

	authorizer := authz.NewAuthorizer(authzClient, tracer, monitor, logger)
	if err := authorizer.ValidateModel(context.Background()); err != nil {
		return nil, fmt.Errorf("invalid authorization model provided: %w", err)
	}

	router := web.NewRouter(
		web.WithKratosClients(kClient, kAdminClient),
		web.WithHydraClient(hClient),
		web.WithAuthzClient(authorizer),
		web.WithCookieManager(cookieManager),
		web.WithFS(distFS),
		web.WithFlags(specs.MFAEnabled, specs.OIDCWebAuthnSequencingEnabled, specs.IdentifierFirstEnabled),
		web.WithBaseURL(specs.BaseURL),
		web.WithSupportEmail(specs.SupportEmail),
		web.WithFeatureFlags(specs.FeatureFlags),
		web.WithKratosPublicURL(specs.KratosPublicURL),
		web.WithTracing(tracer),
		web.WithMonitoring(monitor),
		web.WithLogger(logger),
	)
	return router, nil
}

func handleServeAndShutdown(srv *http.Server, securityLogger logging.SecurityLoggerInterface) error {
	var listenAndServeError, shutdownError error
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		securityLogger.SystemStartup()
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			listenAndServeError = fmt.Errorf("server error: %w", err)
			c <- os.Interrupt
		}
	}()

	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	securityLogger.SystemShutdown()
	if err := srv.Shutdown(ctx); err != nil {
		shutdownError = fmt.Errorf("server shutdown error: %w", err)
	}

	return errors.Join(listenAndServeError, shutdownError)
}

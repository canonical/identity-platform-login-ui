package cmd

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

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
	Run: func(cmd *cobra.Command, args []string) {
		serve()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func serve() {

	specs := new(config.EnvSpec)

	if err := envconfig.Process("", specs); err != nil {
		panic(fmt.Errorf("issues with environment sourcing: %s", err))
	}

	logger := logging.NewLogger(specs.LogLevel)

	logger.Debugf("env vars: %v", specs)

	monitor := prometheus.NewMonitor("identity-login-ui", logger)
	tracer := tracing.NewTracer(tracing.NewConfig(specs.TracingEnabled, specs.OtelGRPCEndpoint, specs.OtelHTTPEndpoint, logger))

	distFS, err := fs.Sub(jsFS, "ui/dist")

	if err != nil {
		logger.Fatalf("issue with js distribution files %s", err)
	}

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
	if authorizer.ValidateModel(context.Background()) != nil {
		panic("Invalid authorization model provided")
	}

	router := web.NewRouter(kClient, kAdminClient, hClient, authorizer, cookieManager, distFS, specs.MFAEnabled, specs.OIDCWebAuthnSequencingEnabled, specs.BaseURL, specs.SupportEmail, specs.KratosPublicURL, tracer, monitor, logger)

	logger.Infof("Starting server on port %v", specs.Port)

	srv := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%v", specs.Port),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			logger.Fatal(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)

	logger.Desugar().Sync()

	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	logger.Info("Shutting down")
	os.Exit(0)

}

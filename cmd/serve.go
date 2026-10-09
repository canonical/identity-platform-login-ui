// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

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
	"github.com/canonical/identity-platform-login-ui/internal/cookies"
	ig "github.com/canonical/identity-platform-login-ui/internal/grpc"
	ih "github.com/canonical/identity-platform-login-ui/internal/hydra"
	ik "github.com/canonical/identity-platform-login-ui/internal/kratos"
	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring/prometheus"
	fga "github.com/canonical/identity-platform-login-ui/internal/openfga"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
	"github.com/canonical/identity-platform-login-ui/pkg/byosso"
	"github.com/canonical/identity-platform-login-ui/pkg/tenants"
	"github.com/canonical/identity-platform-login-ui/pkg/web"

	sso "github.com/canonical/identity-platform-api/v0/sso"
	tenant "github.com/canonical/identity-platform-api/v0/tenant"
	"google.golang.org/grpc"
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

// validateTenantSettings refuses the multi-tenancy and BYO-SSO settings the
// service cannot start with.
func validateTenantSettings(specs *config.EnvSpec) error {
	switch {
	case specs.MultiTenancyEnabled && specs.TenantServiceGRPCAddress == "":
		return fmt.Errorf("cannot enable multi-tenancy without TENANT_SERVICE_GRPC_ADDRESS")
	case specs.BYOSSOEnabled && !specs.MultiTenancyEnabled:
		return fmt.Errorf("cannot enable BYO-SSO without MULTI_TENANCY_ENABLED")
	case specs.BYOSSOEnabled && (specs.SSOServiceGRPCAddress == "" || specs.ServiceTokenURL == "" || specs.ServiceClientID == "" || specs.ServiceClientSecret == ""):
		return fmt.Errorf("cannot enable BYO-SSO without SSO_SERVICE_GRPC_ADDRESS, SERVICE_TOKEN_URL, SERVICE_CLIENT_ID and SERVICE_CLIENT_SECRET")
	case specs.BYOSSOEnabled && specs.OIDCWebAuthnSequencingEnabled:
		// OIDC_WEBAUTHN_SEQUENCING_ENABLED asks a WebAuthn key of every
		// sign-in through an external provider, and with BYO-SSO each tenant
		// decides MFA: with both, two rules would answer the same question.
		return fmt.Errorf("cannot enable BYO-SSO with OIDC_WEBAUTHN_SEQUENCING_ENABLED")
	}

	return nil
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

	if err := validateTenantSettings(specs); err != nil {
		return err
	}

	logger := logging.NewLogger(specs.LogLevel)
	defer logger.Sync()

	logger.Debugf("env vars: %v", specs)

	distFS, err := fs.Sub(jsFS, "ui/dist")
	if err != nil {
		return fmt.Errorf("issue with js distribution files: %w", err)
	}

	var tenantServiceDialOpts, ssoServiceDialOpts []grpc.DialOption
	if specs.BYOSSOEnabled {
		tokens := byosso.NewServiceTokenSource(specs.ServiceTokenURL, specs.ServiceClientID, specs.ServiceClientSecret, specs.ServiceTokenScopes)
		tenantServiceDialOpts = byosso.TenantServiceDialOptions(tokens)
		ssoServiceDialOpts = byosso.SSOServiceDialOptions(tokens)
	}

	var grpcConn *grpc.ClientConn
	if specs.MultiTenancyEnabled {
		conn, err := ig.NewConn("tenant-service", specs.TenantServiceGRPCAddress, specs.TenantServiceTLSEnabled, tenantServiceDialOpts...)
		if err != nil {
			return err
		}
		grpcConn = conn
		defer grpcConn.Close()
		logger.Infof("Tenant validation enabled (tenant-service: %s, tls: %v, timeout: %s)", specs.TenantServiceGRPCAddress, specs.TenantServiceTLSEnabled, specs.TenantServiceGRPCTimeout)
	}

	var ssoGRPCConn *grpc.ClientConn
	if specs.BYOSSOEnabled {
		conn, err := ig.NewConn("sso-service", specs.SSOServiceGRPCAddress, specs.SSOServiceTLSEnabled, ssoServiceDialOpts...)
		if err != nil {
			return err
		}
		ssoGRPCConn = conn
		defer ssoGRPCConn.Close()
		logger.Infof("BYO-SSO enabled (sso-service: %s, tls: %v, timeout: %s)", specs.SSOServiceGRPCAddress, specs.SSOServiceTLSEnabled, specs.SSOServiceGRPCTimeout)
	}

	router, err := buildRouter(specs, distFS, logger, grpcConn, ssoGRPCConn)
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

func buildRouter(specs *config.EnvSpec, distFS fs.FS, logger *logging.Logger, grpcConn, ssoGRPCConn *grpc.ClientConn) (http.Handler, error) {
	monitor := prometheus.NewMonitor("identity-login-ui", logger)
	tracer := tracing.NewTracer(tracing.NewConfig(specs.TracingEnabled, specs.OtelGRPCEndpoint, specs.OtelHTTPEndpoint, logger))

	kClient := ik.NewClient(specs.KratosPublicURL, specs.Debug)
	kAdminClient := ik.NewClient(specs.KratosAdminURL, specs.Debug)
	hClient := ih.NewClient(specs.HydraAdminURL, specs.Debug)

	encrypt := cookies.NewEncrypt([]byte(specs.CookiesEncryptionKey), logger, tracer)
	cookieManager := cookies.NewAuthCookieManager(
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

	var tenantsServiceClient tenants.TenantServiceClientInterface
	var tenantSignInClient tenant.TenantSignInServiceClient
	if grpcConn != nil {
		tenantsServiceClient = tenant.NewTenantServiceClient(grpcConn)
		tenantSignInClient = tenant.NewTenantSignInServiceClient(grpcConn)
	}

	var ssoServiceClient sso.SSOSignInServiceClient
	if ssoGRPCConn != nil {
		ssoServiceClient = sso.NewSSOSignInServiceClient(ssoGRPCConn)
	}

	return web.NewRouter(
		web.WithBYOSSOEnabled(specs.BYOSSOEnabled),
		web.WithBYOSSOClients(tenantSignInClient, ssoServiceClient),
		web.WithSSOGRPCTimeout(specs.SSOServiceGRPCTimeout),
		web.WithCookieEncryption(encrypt),
		web.WithKratosPrivilegedSessionMaxAge(specs.KratosPrivilegedSessionMaxAge),
		web.WithKratosClients(kClient, kAdminClient),
		web.WithTenantsServiceClient(tenantsServiceClient),
		web.WithTenantsGRPCTimeout(specs.TenantServiceGRPCTimeout),
		web.WithHydraClient(hClient),
		web.WithAuthzClient(authorizer),
		web.WithCookieManager(cookieManager),
		web.WithFS(distFS),
		web.WithFlags(specs.VerificationEnabled, specs.MFAEnabled, specs.OIDCWebAuthnSequencingEnabled, specs.IdentifierFirstEnabled, specs.MultiTenancyEnabled),
		web.WithBaseURL(specs.BaseURL),
		web.WithSupportEmail(specs.SupportEmail),
		web.WithFeatureFlags(specs.FeatureFlags),
		web.WithKratosPublicURL(specs.KratosPublicURL),
		web.WithTracing(tracer),
		web.WithMonitoring(monitor),
		web.WithLogger(logger),
	)
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

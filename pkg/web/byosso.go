// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package web

import (
	"fmt"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
	"github.com/canonical/identity-platform-login-ui/pkg/byosso"
	"github.com/canonical/identity-platform-login-ui/pkg/extra"
	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
	"github.com/canonical/identity-platform-login-ui/pkg/tenants"
)

func WithBYOSSOEnabled(enabled bool) Option {
	return func(r *routerConfig) {
		r.byosso.enabled = enabled
	}
}

func WithBYOSSOClients(tenantSignIn byosso.TenantSignInServiceClientInterface, ssoService byosso.SSOServiceClientInterface) Option {
	return func(r *routerConfig) {
		r.byosso.tenantSignInClient = tenantSignIn
		r.byosso.ssoServiceClient = ssoService
	}
}

func WithSSOGRPCTimeout(d time.Duration) Option {
	return func(r *routerConfig) {
		r.byosso.ssoGRPCTimeout = d
	}
}

func WithCookieEncryption(e cookies.EncryptInterface) Option {
	return func(r *routerConfig) {
		r.byosso.encrypt = e
	}
}

func WithKratosPrivilegedSessionMaxAge(d time.Duration) Option {
	return func(r *routerConfig) {
		r.byosso.privilegedSessionMaxAge = d
	}
}

type byossoConfig struct {
	enabled                 bool
	tenantSignInClient      byosso.TenantSignInServiceClientInterface
	ssoServiceClient        byosso.SSOServiceClientInterface
	ssoGRPCTimeout          time.Duration
	encrypt                 cookies.EncryptInterface
	privilegedSessionMaxAge time.Duration
}

// tenantsServiceOptions makes the tenant lookups by email include the
// tenants the address may join.
func (c byossoConfig) tenantsServiceOptions() []tenants.ServiceOption {
	if !c.enabled {
		return nil
	}
	return []tenants.ServiceOption{tenants.WithSignInTenants(c.tenantSignInClient)}
}

func (c byossoConfig) tenantsOptions() []tenants.Option {
	if !c.enabled {
		return nil
	}
	return []tenants.Option{tenants.WithSessionLookupByEmail()}
}

// byossoAPIs is what BYO-SSO changes in the kratos and extra APIs.
type byossoAPIs struct {
	kratosService kratos.ServiceInterface
	mfaEnabled    bool
	kratosOpts    []kratos.Option
	extraOpts     []extra.Option
}

// registerBYOSSO registers the BYO-SSO API and returns what the kratos and
// extra APIs are built with. When BYO-SSO is disabled that is kratosService
// and the MFA flag, unchanged. When it is enabled without what it depends on
// it returns an error: a tenant that requires company sign-in must never be
// served without its checks.
func registerBYOSSO(config *routerConfig, router *chi.Mux, kratosService kratos.ServiceInterface, resolver kratos.TenantResolverInterface) (*byossoAPIs, error) {
	c := config.byosso
	if !c.enabled {
		return &byossoAPIs{kratosService: kratosService, mfaEnabled: config.mfaEnabled}, nil
	}
	if !resolver.Enabled() {
		return nil, fmt.Errorf("cannot enable BYO-SSO without multi-tenancy")
	}
	if c.tenantSignInClient == nil || c.ssoServiceClient == nil {
		return nil, fmt.Errorf("cannot enable BYO-SSO without the tenant-service and sso-service gRPC clients")
	}

	decorated := byosso.NewKratosServiceDecorator(kratosService)
	api, err := byosso.NewAPI(
		decorated,
		byosso.NewSSO(c.ssoServiceClient, c.ssoGRPCTimeout, config.tracer, config.logger),
		byosso.NewDirectory(c.tenantSignInClient, config.tenantsGRPCTimeout, config.tracer, config.logger),
		byosso.NewHydra(config.hydraClient, config.tracer, config.logger),
		byosso.NewVerification(config.kratosClient, config.tracer, config.logger),
		byosso.NewIdentityFinder(config.kratosAdminClient, config.tracer, config.logger),
		config.cookieManager,
		resolver,
		byosso.NewCookieStore(c.encrypt),
		config.baseURL,
		c.privilegedSessionMaxAge,
		config.tracer,
		config.logger,
	)
	if err != nil {
		return nil, err
	}
	api.RegisterEndpoints(router)

	return &byossoAPIs{
		kratosService: decorated,
		// The MFA each tenant asks for replaces the one MFA_ENABLED asks of
		// everyone.
		mfaEnabled: false,
		// A backup code meets the MFA of a tenant as it meets MFA_ENABLED's.
		kratosOpts: []kratos.Option{kratos.WithExtension(api), kratos.WithBackupCodesRegeneration()},
		extraOpts:  []extra.Option{extra.WithExtension(api)},
	}, nil
}

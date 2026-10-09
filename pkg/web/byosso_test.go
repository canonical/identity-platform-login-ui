// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package web

import (
	"testing"
	"testing/fstest"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/mock/gomock"

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
	ih "github.com/canonical/identity-platform-login-ui/internal/hydra"
	ik "github.com/canonical/identity-platform-login-ui/internal/kratos"
	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
	"github.com/canonical/identity-platform-login-ui/pkg/byosso"
	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
	"github.com/canonical/identity-platform-login-ui/pkg/tenants"
)

// routerOptions returns the options of a router with BYO-SSO, built on
// clients that are never called.
func routerOptions(ctrl *gomock.Controller, byossoEnabled, multiTenancy bool) []Option {
	logger := logging.NewNoopLogger()
	tracer := tracing.NewNoopTracer()
	encrypt := cookies.NewEncrypt([]byte("0123456789abcdef0123456789abcdef"), logger, tracer)

	return []Option{
		WithKratosClients(ik.NewClient("http://kratos.invalid", false), ik.NewClient("http://kratos-admin.invalid", false)),
		WithHydraClient(ih.NewClient("http://hydra.invalid", false)),
		WithCookieManager(cookies.NewAuthCookieManager(300, encrypt, logger)),
		WithCookieEncryption(encrypt),
		WithFS(fstest.MapFS{}),
		WithFlags(false, true, false, true, multiTenancy),
		WithBaseURL("https://example.com"),
		WithTenantsServiceClient(tenants.NewMockTenantServiceClientInterface(ctrl)),
		WithTenantsGRPCTimeout(time.Second),
		WithBYOSSOEnabled(byossoEnabled),
		WithBYOSSOClients(byosso.NewMockTenantSignInServiceClientInterface(ctrl), byosso.NewMockSSOServiceClientInterface(ctrl)),
		WithSSOGRPCTimeout(time.Second),
		WithKratosPrivilegedSessionMaxAge(time.Hour),
		WithTracing(tracer),
		WithMonitoring(monitoring.NewNoopMonitor("test", logger)),
		WithLogger(logger),
	}
}

func TestNewRouterWithBYOSSO(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	router, err := NewRouter(routerOptions(ctrl, true, true)...)

	if router == nil {
		t.Fatalf("expected router not nil")
	}
	if err != nil {
		t.Fatalf("expected error to be nil, got %v", err)
	}
}

func TestNewRouterFailOnBYOSSO(t *testing.T) {
	tests := []struct {
		name         string
		multiTenancy bool
		opts         []Option
	}{
		{
			name:         "multi-tenancy disabled",
			multiTenancy: false,
		},
		{
			name:         "no tenant-service client",
			multiTenancy: true,
			opts:         []Option{WithTenantsServiceClient(nil)},
		},
		{
			name:         "no tenant-service client for BYO-SSO",
			multiTenancy: true,
			opts: []Option{func(r *routerConfig) {
				r.byosso.tenantSignInClient = nil
			}},
		},
		{
			name:         "no sso-service client",
			multiTenancy: true,
			opts: []Option{func(r *routerConfig) {
				r.byosso.ssoServiceClient = nil
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// a router that cannot check the sign-ins of a tenant is not built
			router, err := NewRouter(append(routerOptions(ctrl, true, tt.multiTenancy), tt.opts...)...)

			if router != nil {
				t.Fatalf("expected router to be nil, got %v", router)
			}
			if err == nil {
				t.Fatalf("expected error not nil")
			}

			// the same options build a router with BYO-SSO disabled
			router, err = NewRouter(append(routerOptions(ctrl, false, tt.multiTenancy), tt.opts...)...)

			if router == nil {
				t.Fatalf("expected router not nil")
			}
			if err != nil {
				t.Fatalf("expected error to be nil, got %v", err)
			}
		})
	}
}

func TestNewRouterFailOnBaseURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	router, err := NewRouter(append(routerOptions(ctrl, true, true), WithBaseURL("https://example.com/%zz"))...)

	if router != nil {
		t.Fatalf("expected router to be nil, got %v", router)
	}
	if err == nil {
		t.Fatalf("expected error not nil")
	}
}

func TestRegisterBYOSSO(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	config := &routerConfig{}
	for _, opt := range routerOptions(ctrl, true, true) {
		opt(config)
	}
	mockResolver := kratos.NewMockTenantResolverInterface(ctrl)
	mockResolver.EXPECT().Enabled().Times(1).Return(true)

	apis, err := registerBYOSSO(config, chi.NewMux(), kratos.NewMockServiceInterface(ctrl), mockResolver)

	if err != nil {
		t.Fatalf("expected error to be nil, got %v", err)
	}
	// the MFA of the tenants replaces MFA_ENABLED, and the prompt to make new
	// backup codes is kept by an option of its own
	if apis.mfaEnabled {
		t.Fatalf("expected mfa to be disabled")
	}
	if len(apis.kratosOpts) != 2 {
		t.Fatalf("expected 2 kratos options, got %d", len(apis.kratosOpts))
	}
}

func TestRegisterBYOSSODisabled(t *testing.T) {
	config := &routerConfig{mfaEnabled: true}

	apis, err := registerBYOSSO(config, nil, nil, nil)

	if err != nil {
		t.Fatalf("expected error to be nil, got %v", err)
	}
	if apis.kratosService != nil {
		t.Fatalf("expected the kratos service to be unchanged, got %v", apis.kratosService)
	}
	if !apis.mfaEnabled {
		t.Fatalf("expected mfa to stay enabled")
	}
	if len(apis.kratosOpts) != 0 {
		t.Fatalf("expected no kratos options, got %d", len(apis.kratosOpts))
	}
	if len(apis.extraOpts) != 0 {
		t.Fatalf("expected no extra options, got %d", len(apis.extraOpts))
	}
	if len(config.byosso.tenantsOptions()) != 0 {
		t.Fatalf("expected no tenants options, got %d", len(config.byosso.tenantsOptions()))
	}
	if len(config.byosso.tenantsServiceOptions()) != 0 {
		t.Fatalf("expected no tenants service options, got %d", len(config.byosso.tenantsServiceOptions()))
	}
}

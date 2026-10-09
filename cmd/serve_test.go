// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package cmd

import (
	"testing"

	"github.com/canonical/identity-platform-login-ui/internal/config"
)

func TestValidateTenantSettings(t *testing.T) {
	byosso := func(change func(*config.EnvSpec)) *config.EnvSpec {
		specs := &config.EnvSpec{
			MultiTenancyEnabled:      true,
			TenantServiceGRPCAddress: "tenant-service:50051",
			BYOSSOEnabled:            true,
			SSOServiceGRPCAddress:    "sso-service:50051",
			ServiceTokenURL:          "https://hydra.example/oauth2/token",
			ServiceClientID:          "login-ui",
			ServiceClientSecret:      "secret",
		}
		if change != nil {
			change(specs)
		}
		return specs
	}

	tests := []struct {
		name          string
		specs         *config.EnvSpec
		expectedError string
	}{
		{name: "nothing enabled", specs: &config.EnvSpec{}},
		{name: "multi-tenancy", specs: &config.EnvSpec{MultiTenancyEnabled: true, TenantServiceGRPCAddress: "tenant-service:50051"}},
		{
			name:          "multi-tenancy without the tenant service",
			specs:         &config.EnvSpec{MultiTenancyEnabled: true},
			expectedError: "cannot enable multi-tenancy without TENANT_SERVICE_GRPC_ADDRESS",
		},
		// the rule against sequencing belongs to BYO-SSO alone
		{name: "OIDC WebAuthn sequencing without BYO-SSO", specs: &config.EnvSpec{OIDCWebAuthnSequencingEnabled: true}},
		{name: "BYO-SSO", specs: byosso(nil)},
		{
			name:          "BYO-SSO without multi-tenancy",
			specs:         byosso(func(s *config.EnvSpec) { s.MultiTenancyEnabled = false }),
			expectedError: "cannot enable BYO-SSO without MULTI_TENANCY_ENABLED",
		},
		{
			name:          "BYO-SSO without the SSO service",
			specs:         byosso(func(s *config.EnvSpec) { s.SSOServiceGRPCAddress = "" }),
			expectedError: "cannot enable BYO-SSO without SSO_SERVICE_GRPC_ADDRESS, SERVICE_TOKEN_URL, SERVICE_CLIENT_ID and SERVICE_CLIENT_SECRET",
		},
		{
			name:          "BYO-SSO without a service client secret",
			specs:         byosso(func(s *config.EnvSpec) { s.ServiceClientSecret = "" }),
			expectedError: "cannot enable BYO-SSO without SSO_SERVICE_GRPC_ADDRESS, SERVICE_TOKEN_URL, SERVICE_CLIENT_ID and SERVICE_CLIENT_SECRET",
		},
		{
			name:          "BYO-SSO with OIDC WebAuthn sequencing",
			specs:         byosso(func(s *config.EnvSpec) { s.OIDCWebAuthnSequencingEnabled = true }),
			expectedError: "cannot enable BYO-SSO with OIDC_WEBAUTHN_SEQUENCING_ENABLED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTenantSettings(tt.specs)

			if tt.expectedError == "" {
				if err != nil {
					t.Fatalf("expected error to be nil, got %v", err)
				}
				return
			}
			if err == nil || err.Error() != tt.expectedError {
				t.Fatalf("expected error %q, got %v", tt.expectedError, err)
			}
		})
	}
}

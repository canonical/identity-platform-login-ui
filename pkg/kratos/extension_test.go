// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package kratos

import (
	"context"
	"net/http/httptest"
	"testing"

	kClient "github.com/ory/kratos-client-go/v25"
	"go.opentelemetry.io/otel/trace"
	gomock "go.uber.org/mock/gomock"

	"github.com/canonical/identity-platform-login-ui/pkg/tenants"
)

func TestWithExtension(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockExtension := NewMockExtensionInterface(ctrl)

	api := NewAPI(mockService, false, false, false, tenants.NewNoOpTenantResolver(), BASE_URL, mockCookieManager, mockTracer, mockLogger, WithExtension(mockExtension))

	if api.ext != mockExtension {
		t.Fatalf("expected the extension to be set")
	}
}

func TestWithBackupCodesRegeneration(t *testing.T) {
	lookupSecret := "lookup_secret"
	password := "password"
	session := &kClient.Session{
		Identity: &kClient.Identity{Id: "test"},
		AuthenticationMethods: []kClient.SessionAuthenticationMethod{
			{Method: &password},
			{Method: &lookupSecret},
		},
	}

	tests := []struct {
		name       string
		mfaEnabled bool
		opts       []Option
		// expected is whether the account is asked how many codes it has left
		expected bool
	}{
		{name: "mfa off", expected: false},
		{name: "mfa on", mfaEnabled: true, expected: true},
		{name: "mfa off with the option", opts: []Option{WithBackupCodesRegeneration()}, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockService := NewMockServiceInterface(ctrl)
			mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockTracer.EXPECT().Start(gomock.Any(), "kratos.API.shouldRegenerateBackupCodesWithSession").Times(1).Return(context.Background(), trace.SpanFromContext(context.Background()))
			if tt.expected {
				mockService.EXPECT().HasNotEnoughLookupSecretsLeft(gomock.Any(), "test").Times(1).Return(true, nil)
			}

			api := NewAPI(mockService, false, tt.mfaEnabled, false, tenants.NewNoOpTenantResolver(), BASE_URL, mockCookieManager, mockTracer, mockLogger, tt.opts...)

			regenerate, err := api.shouldRegenerateBackupCodesWithSession(context.Background(), session)

			if err != nil {
				t.Fatalf("expected error to be nil, got %v", err)
			}
			if regenerate != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, regenerate)
			}
		})
	}
}

func TestUnsetSessionCookie(t *testing.T) {
	w := httptest.NewRecorder()
	UnsetSessionCookie(w)

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].Name != KRATOS_SESSION_COOKIE_NAME {
		t.Fatalf("expected cookie %s, got %s", KRATOS_SESSION_COOKIE_NAME, cookies[0].Name)
	}
	if cookies[0].Value != "" {
		t.Fatalf("expected an empty value, got %s", cookies[0].Value)
	}
}

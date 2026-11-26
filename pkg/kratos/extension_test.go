// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package kratos

import (
	"net/http/httptest"
	"testing"

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

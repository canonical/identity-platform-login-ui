// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

package tenants

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	kClient "github.com/ory/kratos-client-go/v25"
	"go.uber.org/mock/gomock"

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
)

// mockTenantLookup is a hand-built stub of tenantLookupService for tests.
type mockTenantLookup struct {
	tenants []Tenant
	err     error
}

func (m *mockTenantLookup) LookupTenantsByEmail(_ context.Context, _ string) ([]Tenant, error) {
	return m.tenants, m.err
}

func TestTenantHashDeterministic(t *testing.T) {
	h1 := cookies.ChallengeHash("challenge-abc")
	h2 := cookies.ChallengeHash("challenge-abc")
	if h1 != h2 {
		t.Fatalf("cookies.ChallengeHash is not deterministic: %q != %q", h1, h2)
	}
}

func TestTenantHashDistinct(t *testing.T) {
	h1 := cookies.ChallengeHash("challenge-a")
	h2 := cookies.ChallengeHash("challenge-b")
	if h1 == h2 {
		t.Fatal("cookies.ChallengeHash produced identical outputs for different inputs")
	}
}

func TestNoOpTenantResolverEnabled(t *testing.T) {
	r := NewNoOpTenantResolver()
	if r.Enabled() {
		t.Fatal("expected Enabled() to return false")
	}
}

func TestNoOpTenantResolverTenantID(t *testing.T) {
	r := NewNoOpTenantResolver()
	got := r.TenantID(cookies.FlowStateCookie{TenantID: "t1"}, "challenge")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestNoOpTenantResolverStoreTenant(t *testing.T) {
	r := NewNoOpTenantResolver()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	if err := r.StoreTenant(w, req, "t1", "challenge"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCookieTenantResolverEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCM := NewMockCookieManagerInterface(ctrl)
	r := NewCookieTenantResolver(mockCM, &mockTenantLookup{})
	if !r.Enabled() {
		t.Fatal("expected Enabled() to return true")
	}
}

func TestCookieTenantResolverTenantIDMatchingChallenge(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCM := NewMockCookieManagerInterface(ctrl)
	r := NewCookieTenantResolver(mockCM, &mockTenantLookup{})

	challenge := "test-challenge-xyz"
	cookie := cookies.FlowStateCookie{
		TenantID:           "t1",
		LoginChallengeHash: cookies.ChallengeHash(challenge),
	}

	got := r.TenantID(cookie, challenge)
	if got != "t1" {
		t.Fatalf("expected tenant ID %q, got %q", "t1", got)
	}
}

func TestCookieTenantResolverTenantIDWrongChallenge(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCM := NewMockCookieManagerInterface(ctrl)
	r := NewCookieTenantResolver(mockCM, &mockTenantLookup{})

	cookie := cookies.FlowStateCookie{
		TenantID:           "t1",
		LoginChallengeHash: cookies.ChallengeHash("original-challenge"),
	}

	got := r.TenantID(cookie, "different-challenge")
	if got != "" {
		t.Fatalf("expected empty string for mismatched challenge, got %q", got)
	}
}

func TestCookieTenantResolverTenantIDEmptyTenantID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCM := NewMockCookieManagerInterface(ctrl)
	r := NewCookieTenantResolver(mockCM, &mockTenantLookup{})

	challenge := "test-challenge"
	cookie := cookies.FlowStateCookie{
		TenantID:           "",
		LoginChallengeHash: cookies.ChallengeHash(challenge),
	}

	got := r.TenantID(cookie, challenge)
	if got != "" {
		t.Fatalf("expected empty string when TenantID is empty, got %q", got)
	}
}

func TestCookieTenantResolverStoreTenant(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCM := NewMockCookieManagerInterface(ctrl)
	r := NewCookieTenantResolver(mockCM, &mockTenantLookup{})

	challenge := "store-challenge"
	tenantID := "tenant-42"
	existingCookie := cookies.FlowStateCookie{}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	mockCM.EXPECT().GetStateCookie(req).Return(existingCookie, nil)
	mockCM.EXPECT().SetStateCookie(w, cookies.FlowStateCookie{
		TenantID:           tenantID,
		LoginChallengeHash: cookies.ChallengeHash(challenge),
	}).Return(nil)

	if err := r.StoreTenant(w, req, tenantID, challenge); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNoOpTenantResolverHasTenants(t *testing.T) {
	r := NewNoOpTenantResolver()
	got, err := r.HasTenants(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatal("expected HasTenants to return false")
	}
}

func TestCookieTenantResolverHasTenantsTrue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCM := NewMockCookieManagerInterface(ctrl)
	svc := &mockTenantLookup{tenants: []Tenant{{ID: "t1", Name: "Acme"}}}
	r := NewCookieTenantResolver(mockCM, svc)

	session := kClient.NewSessionWithDefaults()
	session.Identity = kClient.NewIdentityWithDefaults()
	session.Identity.Traits = map[string]interface{}{"email": "user@example.com"}

	got, err := r.HasTenants(context.Background(), session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatal("expected HasTenants to return true")
	}
}

func TestCookieTenantResolverHasTenantsFalse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCM := NewMockCookieManagerInterface(ctrl)
	svc := &mockTenantLookup{tenants: []Tenant{}}
	r := NewCookieTenantResolver(mockCM, svc)

	session := kClient.NewSessionWithDefaults()
	session.Identity = kClient.NewIdentityWithDefaults()
	session.Identity.Traits = map[string]interface{}{"email": "user@example.com"}

	got, err := r.HasTenants(context.Background(), session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatal("expected HasTenants to return false")
	}
}

func TestCookieTenantResolverHasTenantsServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCM := NewMockCookieManagerInterface(ctrl)
	svc := &mockTenantLookup{err: fmt.Errorf("network error")}
	r := NewCookieTenantResolver(mockCM, svc)

	session := kClient.NewSessionWithDefaults()
	session.Identity = kClient.NewIdentityWithDefaults()
	session.Identity.Traits = map[string]interface{}{"email": "user@example.com"}

	_, err := r.HasTenants(context.Background(), session)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCookieTenantResolverHasTenantsNilSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCM := NewMockCookieManagerInterface(ctrl)
	svc := &mockTenantLookup{tenants: []Tenant{{ID: "t1", Name: "Acme"}}}
	r := NewCookieTenantResolver(mockCM, svc)

	got, err := r.HasTenants(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatal("expected HasTenants to return false for nil session")
	}
}

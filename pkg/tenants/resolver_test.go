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

// sessionWithEmail builds a Kratos session whose identity traits contain the given email.
func sessionWithEmail(email string) *kClient.Session {
	s := kClient.NewSession("test")
	s.Identity = kClient.NewIdentity("test-identity", "default", "https://example.com/schema", map[string]interface{}{"email": email})
	return s
}

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
	got, err := r.HasTenants(context.Background(), sessionWithEmail("user@example.com"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatal("expected HasTenants to return false")
	}
}

func TestCookieTenantResolverHasTenantsByEmailTrue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCM := NewMockCookieManagerInterface(ctrl)
	svc := &mockTenantLookup{tenants: []Tenant{{ID: "t1", Name: "Acme"}}}
	r := NewCookieTenantResolver(mockCM, svc)

	got, err := r.HasTenants(context.Background(), sessionWithEmail("user@example.com"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatal("expected HasTenants to return true")
	}
}

func TestCookieTenantResolverHasTenantsByEmailFalse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCM := NewMockCookieManagerInterface(ctrl)
	svc := &mockTenantLookup{tenants: []Tenant{}}
	r := NewCookieTenantResolver(mockCM, svc)

	got, err := r.HasTenants(context.Background(), sessionWithEmail("user@example.com"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatal("expected HasTenants to return false")
	}
}

func TestCookieTenantResolverHasTenantsByEmailServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCM := NewMockCookieManagerInterface(ctrl)
	svc := &mockTenantLookup{err: fmt.Errorf("network error")}
	r := NewCookieTenantResolver(mockCM, svc)

	_, err := r.HasTenants(context.Background(), sessionWithEmail("user@example.com"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCookieTenantResolverHasTenantsByEmailEmpty(t *testing.T) {
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

func TestNoOpIsAuthenticatedForChallenge(t *testing.T) {
	r := NewNoOpTenantResolver()
	if !r.IsAuthenticatedForChallenge(cookies.FlowStateCookie{}, "any") {
		t.Fatal("NoOp should always return true")
	}
}

func TestNoOpNeedsTenantSelection(t *testing.T) {
	r := NewNoOpTenantResolver()
	need, c, err := r.NeedsTenantSelection(context.Background(), nil, cookies.FlowStateCookie{TenantID: "x"}, "ch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if need {
		t.Fatal("NoOp should never need selection")
	}
	if c.TenantID != "x" {
		t.Fatal("NoOp should return cookie unchanged")
	}
}

func TestCookieTenantResolverIsAuthenticatedMatching(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := NewCookieTenantResolver(NewMockCookieManagerInterface(ctrl), &mockTenantLookup{})
	challenge := "ch-1"
	c := cookies.FlowStateCookie{LoginChallengeHash: cookies.ChallengeHash(challenge)}
	if !r.IsAuthenticatedForChallenge(c, challenge) {
		t.Fatal("expected true for matching challenge")
	}
}

func TestCookieTenantResolverIsAuthenticatedMismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := NewCookieTenantResolver(NewMockCookieManagerInterface(ctrl), &mockTenantLookup{})
	c := cookies.FlowStateCookie{LoginChallengeHash: cookies.ChallengeHash("ch-1")}
	if r.IsAuthenticatedForChallenge(c, "ch-2") {
		t.Fatal("expected false for mismatched challenge")
	}
}

func TestCookieTenantResolverIsAuthenticatedEmptyChallenge(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	r := NewCookieTenantResolver(NewMockCookieManagerInterface(ctrl), &mockTenantLookup{})
	if !r.IsAuthenticatedForChallenge(cookies.FlowStateCookie{}, "") {
		t.Fatal("expected true for empty challenge")
	}
}

func TestNeedsTenantSelectionAlreadySelected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	challenge := "ch-1"
	c := cookies.FlowStateCookie{
		TenantID:           "t1",
		LoginChallengeHash: cookies.ChallengeHash(challenge),
	}
	r := NewCookieTenantResolver(NewMockCookieManagerInterface(ctrl), &mockTenantLookup{})
	need, _, err := r.NeedsTenantSelection(context.Background(), sessionWithEmail("u@e.com"), c, challenge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if need {
		t.Fatal("should not need selection when tenant already selected")
	}
}

func TestNeedsTenantSelectionHasTenants(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	challenge := "ch-1"
	c := cookies.FlowStateCookie{LoginChallengeHash: cookies.ChallengeHash(challenge)}
	svc := &mockTenantLookup{tenants: []Tenant{{ID: "t1", Name: "Acme"}}}
	r := NewCookieTenantResolver(NewMockCookieManagerInterface(ctrl), svc)

	need, _, err := r.NeedsTenantSelection(context.Background(), sessionWithEmail("u@e.com"), c, challenge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !need {
		t.Fatal("expected selection needed when user has tenants")
	}
}

func TestNeedsTenantSelectionNoTenants(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	challenge := "ch-1"
	c := cookies.FlowStateCookie{LoginChallengeHash: cookies.ChallengeHash(challenge)}
	svc := &mockTenantLookup{tenants: []Tenant{}}
	r := NewCookieTenantResolver(NewMockCookieManagerInterface(ctrl), svc)

	need, updated, err := r.NeedsTenantSelection(context.Background(), sessionWithEmail("u@e.com"), c, challenge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if need {
		t.Fatal("should not need selection when user has no tenants")
	}
	if updated.TenantID != cookies.NoTenantAvailable {
		t.Fatalf("expected sentinel %q, got %q", cookies.NoTenantAvailable, updated.TenantID)
	}
}

func TestNeedsTenantSelectionLookupError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	challenge := "ch-1"
	c := cookies.FlowStateCookie{LoginChallengeHash: cookies.ChallengeHash(challenge)}
	svc := &mockTenantLookup{err: fmt.Errorf("network error")}
	r := NewCookieTenantResolver(NewMockCookieManagerInterface(ctrl), svc)

	_, _, err := r.NeedsTenantSelection(context.Background(), sessionWithEmail("u@e.com"), c, challenge)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- InterceptLogin tests ---

func TestNoOpInterceptLogin(t *testing.T) {
	r := NewNoOpTenantResolver()
	result, err := r.InterceptLogin(context.Background(), nil, cookies.FlowStateCookie{TenantID: "x"}, "ch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DeferMFAChecks || result.SelectTenant || result.AcceptLogin {
		t.Fatal("NoOp should return zero-value fields (no intervention)")
	}
	if result.Cookie.TenantID != "x" {
		t.Fatal("NoOp should return cookie unchanged")
	}
}

func TestInterceptLoginDefersMFAWhenNotAuthenticated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	challenge := "ch-1"
	c := cookies.FlowStateCookie{} // no LoginChallengeHash — not authenticated
	r := NewCookieTenantResolver(NewMockCookieManagerInterface(ctrl), &mockTenantLookup{})

	// No session → identifier-first in progress, defer everything.
	result, err := r.InterceptLogin(context.Background(), nil, c, challenge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.DeferMFAChecks {
		t.Fatal("expected DeferMFAChecks=true when not authenticated for challenge")
	}
	if result.SelectTenant || result.AcceptLogin {
		t.Fatal("expected SelectTenant=false and AcceptLogin=false")
	}
}

func TestInterceptLoginSessionReuseSelectsTenantWhenMultiTenant(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	challenge := "ch-new" // new challenge, cookie has no matching hash
	c := cookies.FlowStateCookie{}
	svc := &mockTenantLookup{tenants: []Tenant{{ID: "t1", Name: "Acme"}}}
	r := NewCookieTenantResolver(NewMockCookieManagerInterface(ctrl), svc)

	// Existing session + multi-tenant → defer MFA + select tenant.
	result, err := r.InterceptLogin(context.Background(), sessionWithEmail("u@e.com"), c, challenge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.DeferMFAChecks {
		t.Fatal("expected DeferMFAChecks=true on session reuse")
	}
	if !result.SelectTenant {
		t.Fatal("expected SelectTenant=true when session-reuse user has tenants")
	}
	if result.AcceptLogin {
		t.Fatal("expected AcceptLogin=false")
	}
}

func TestInterceptLoginSessionReuseAcceptsWhenNoTenants(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	challenge := "ch-new"
	c := cookies.FlowStateCookie{}
	svc := &mockTenantLookup{tenants: []Tenant{}}
	r := NewCookieTenantResolver(NewMockCookieManagerInterface(ctrl), svc)

	// Existing session + zero tenants → defer MFA + accept immediately.
	result, err := r.InterceptLogin(context.Background(), sessionWithEmail("u@e.com"), c, challenge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.DeferMFAChecks {
		t.Fatal("expected DeferMFAChecks=true on session reuse")
	}
	if !result.AcceptLogin {
		t.Fatal("expected AcceptLogin=true when session-reuse user has no tenants")
	}
	if result.SelectTenant {
		t.Fatal("expected SelectTenant=false")
	}
	if result.Cookie.TenantID != cookies.NoTenantAvailable {
		t.Fatalf("expected sentinel %q, got %q", cookies.NoTenantAvailable, result.Cookie.TenantID)
	}
	if result.Cookie.LoginChallengeHash != cookies.ChallengeHash(challenge) {
		t.Fatal("expected cookie to be bound to the new challenge")
	}
}

func TestInterceptLoginSelectsTenantWhenHasTenants(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	challenge := "ch-1"
	c := cookies.FlowStateCookie{LoginChallengeHash: cookies.ChallengeHash(challenge)}
	svc := &mockTenantLookup{tenants: []Tenant{{ID: "t1", Name: "Acme"}}}
	r := NewCookieTenantResolver(NewMockCookieManagerInterface(ctrl), svc)

	result, err := r.InterceptLogin(context.Background(), sessionWithEmail("u@e.com"), c, challenge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.SelectTenant {
		t.Fatal("expected SelectTenant=true when user has tenants")
	}
	if result.DeferMFAChecks || result.AcceptLogin {
		t.Fatal("expected DeferMFAChecks=false and AcceptLogin=false")
	}
}

func TestInterceptLoginAcceptsWhenTenantAlreadySelected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	challenge := "ch-1"
	c := cookies.FlowStateCookie{
		LoginChallengeHash: cookies.ChallengeHash(challenge),
		TenantID:           "t1",
	}
	r := NewCookieTenantResolver(NewMockCookieManagerInterface(ctrl), &mockTenantLookup{})

	result, err := r.InterceptLogin(context.Background(), sessionWithEmail("u@e.com"), c, challenge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.AcceptLogin {
		t.Fatal("expected AcceptLogin=true when tenant already selected")
	}
	if result.DeferMFAChecks || result.SelectTenant {
		t.Fatal("expected DeferMFAChecks=false and SelectTenant=false")
	}
}

func TestInterceptLoginAcceptsWithSentinelWhenNoTenants(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	challenge := "ch-1"
	c := cookies.FlowStateCookie{LoginChallengeHash: cookies.ChallengeHash(challenge)}
	svc := &mockTenantLookup{tenants: []Tenant{}}
	r := NewCookieTenantResolver(NewMockCookieManagerInterface(ctrl), svc)

	result, err := r.InterceptLogin(context.Background(), sessionWithEmail("u@e.com"), c, challenge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.AcceptLogin {
		t.Fatal("expected AcceptLogin=true when user has no tenants")
	}
	if result.Cookie.TenantID != cookies.NoTenantAvailable {
		t.Fatalf("expected sentinel %q, got %q", cookies.NoTenantAvailable, result.Cookie.TenantID)
	}
}

func TestInterceptLoginPropagatesLookupError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	challenge := "ch-1"
	c := cookies.FlowStateCookie{LoginChallengeHash: cookies.ChallengeHash(challenge)}
	svc := &mockTenantLookup{err: fmt.Errorf("network error")}
	r := NewCookieTenantResolver(NewMockCookieManagerInterface(ctrl), svc)

	_, err := r.InterceptLogin(context.Background(), sessionWithEmail("u@e.com"), c, challenge)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- NeedsTenantSelectionByEmail tests ---

func TestNoOpNeedsTenantSelectionByEmail(t *testing.T) {
	r := NewNoOpTenantResolver()
	need, c, err := r.NeedsTenantSelectionByEmail(context.Background(), "u@e.com", cookies.FlowStateCookie{TenantID: "x"}, "ch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if need {
		t.Fatal("NoOp should never need selection")
	}
	if c.TenantID != "x" {
		t.Fatal("NoOp should return cookie unchanged")
	}
}

func TestNeedsTenantSelectionByEmailAlreadySelected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	challenge := "ch-1"
	c := cookies.FlowStateCookie{
		TenantID:           "t1",
		LoginChallengeHash: cookies.ChallengeHash(challenge),
	}
	r := NewCookieTenantResolver(NewMockCookieManagerInterface(ctrl), &mockTenantLookup{})
	need, _, err := r.NeedsTenantSelectionByEmail(context.Background(), "u@e.com", c, challenge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if need {
		t.Fatal("should not need selection when tenant already selected")
	}
}

func TestNeedsTenantSelectionByEmailHasTenants(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	challenge := "ch-1"
	c := cookies.FlowStateCookie{LoginChallengeHash: cookies.ChallengeHash(challenge)}
	svc := &mockTenantLookup{tenants: []Tenant{{ID: "t1", Name: "Acme"}}}
	r := NewCookieTenantResolver(NewMockCookieManagerInterface(ctrl), svc)

	need, _, err := r.NeedsTenantSelectionByEmail(context.Background(), "u@e.com", c, challenge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !need {
		t.Fatal("expected selection needed when user has tenants")
	}
}

func TestNeedsTenantSelectionByEmailNoTenants(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	challenge := "ch-1"
	c := cookies.FlowStateCookie{LoginChallengeHash: cookies.ChallengeHash(challenge)}
	svc := &mockTenantLookup{tenants: []Tenant{}}
	r := NewCookieTenantResolver(NewMockCookieManagerInterface(ctrl), svc)

	need, updated, err := r.NeedsTenantSelectionByEmail(context.Background(), "u@e.com", c, challenge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if need {
		t.Fatal("should not need selection when user has no tenants")
	}
	if updated.TenantID != cookies.NoTenantAvailable {
		t.Fatalf("expected sentinel %q, got %q", cookies.NoTenantAvailable, updated.TenantID)
	}
}

func TestNeedsTenantSelectionByEmailEmptyEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	challenge := "ch-1"
	c := cookies.FlowStateCookie{LoginChallengeHash: cookies.ChallengeHash(challenge)}
	r := NewCookieTenantResolver(NewMockCookieManagerInterface(ctrl), &mockTenantLookup{tenants: []Tenant{{ID: "t1"}}})

	need, _, err := r.NeedsTenantSelectionByEmail(context.Background(), "", c, challenge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if need {
		t.Fatal("should not need selection when email is empty")
	}
}

func TestNeedsTenantSelectionByEmailLookupError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	challenge := "ch-1"
	c := cookies.FlowStateCookie{LoginChallengeHash: cookies.ChallengeHash(challenge)}
	svc := &mockTenantLookup{err: fmt.Errorf("network error")}
	r := NewCookieTenantResolver(NewMockCookieManagerInterface(ctrl), svc)

	_, _, err := r.NeedsTenantSelectionByEmail(context.Background(), "u@e.com", c, challenge)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

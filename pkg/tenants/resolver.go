// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

package tenants

import (
	"context"
	"fmt"
	"net/http"

	kClient "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
)

// tenantLookupService is the subset of ServiceInterface needed to check
// whether a user has any tenants. Defined locally to keep dependencies
// explicit and to avoid circular imports.
type tenantLookupService interface {
	LookupTenantsByEmail(ctx context.Context, email string) ([]Tenant, error)
}

// NoOpTenantResolver is used when tenant selection is disabled.
// All methods are no-ops and Enabled always returns false.
type NoOpTenantResolver struct{}

func NewNoOpTenantResolver() *NoOpTenantResolver {
	return &NoOpTenantResolver{}
}

func (n *NoOpTenantResolver) Enabled() bool { return false }

func (n *NoOpTenantResolver) TenantID(_ cookies.FlowStateCookie, _ string) string {
	return ""
}

func (n *NoOpTenantResolver) StoreTenant(_ http.ResponseWriter, _ *http.Request, _, _ string) error {
	return nil
}

func (n *NoOpTenantResolver) HasTenants(_ context.Context, _ *kClient.Session) (bool, error) {
	return false, nil
}

// CookieTenantResolver stores and retrieves the selected tenant from the
// encrypted FlowStateCookie. The selection is bound to a login challenge via
// LoginChallengeHash so it cannot be replayed across different challenges.
type CookieTenantResolver struct {
	cookieManager CookieManagerInterface
	service       tenantLookupService
}

func NewCookieTenantResolver(cm CookieManagerInterface, svc tenantLookupService) *CookieTenantResolver {
	return &CookieTenantResolver{cookieManager: cm, service: svc}
}

func (c *CookieTenantResolver) Enabled() bool { return true }

func (c *CookieTenantResolver) TenantID(cookie cookies.FlowStateCookie, loginChallenge string) string {
	if cookie.TenantID != "" && cookie.LoginChallengeHash == cookies.ChallengeHash(loginChallenge) {
		return cookie.TenantID
	}
	return ""
}

func (c *CookieTenantResolver) StoreTenant(w http.ResponseWriter, r *http.Request, tenantID, loginChallenge string) error {
	stateCookie, err := c.cookieManager.GetStateCookie(r)
	if err != nil {
		return fmt.Errorf("cannot read state cookie: %w", err)
	}
	stateCookie.TenantID = tenantID
	stateCookie.LoginChallengeHash = cookies.ChallengeHash(loginChallenge)
	return c.cookieManager.SetStateCookie(w, stateCookie)
}

func (c *CookieTenantResolver) HasTenants(ctx context.Context, session *kClient.Session) (bool, error) {
	email := emailFromSession(session)
	if email == "" {
		return false, nil
	}
	tenants, err := c.service.LookupTenantsByEmail(ctx, email)
	if err != nil {
		return false, fmt.Errorf("cannot look up tenants: %w", err)
	}
	return len(tenants) > 0, nil
}

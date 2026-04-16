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
	LookupTenantsByIdentityID(ctx context.Context, identityID string) ([]Tenant, error)
}

// LoginInterception tells handleCreateFlow what to do with the login flow.
// The resolver acts as a plugin: the handler calls InterceptLogin once and
// the returned value drives all subsequent branching—no tenant-specific
// logic leaks into the handler.
type LoginInterception struct {
	// DeferMFAChecks is true when MFA/WebAuthn enforcement should be
	// skipped for now (e.g. the user hasn't completed first-factor auth
	// for this challenge yet).
	DeferMFAChecks bool
	// SelectTenant is true when the user should be redirected to the
	// tenant selection page.
	SelectTenant bool
	// AcceptLogin is true when the login can be accepted immediately
	// (the tenant has been resolved or isn't required).
	AcceptLogin bool
	// Cookie is the (possibly updated) state cookie the handler should
	// use for subsequent operations.
	Cookie cookies.FlowStateCookie
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

func (n *NoOpTenantResolver) IsAuthenticatedForChallenge(_ cookies.FlowStateCookie, _ string) bool {
	return true
}

func (n *NoOpTenantResolver) NeedsTenantSelection(_ context.Context, _ *kClient.Session, c cookies.FlowStateCookie, _ string) (bool, cookies.FlowStateCookie, error) {
	return false, c, nil
}

func (n *NoOpTenantResolver) NeedsTenantSelectionByEmail(_ context.Context, _ string, c cookies.FlowStateCookie, _ string) (bool, cookies.FlowStateCookie, error) {
	return false, c, nil
}

func (n *NoOpTenantResolver) InterceptLogin(_ context.Context, _ *kClient.Session, cookie cookies.FlowStateCookie, _ string) (LoginInterception, error) {
	return LoginInterception{Cookie: cookie}, nil
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
	identityID := identityIDFromSession(session)
	if identityID == "" {
		return false, nil
	}
	tenants, err := c.service.LookupTenantsByIdentityID(ctx, identityID)
	if err != nil {
		return false, fmt.Errorf("cannot look up tenants: %w", err)
	}
	return len(tenants) > 0, nil
}

func (c *CookieTenantResolver) IsAuthenticatedForChallenge(cookie cookies.FlowStateCookie, loginChallenge string) bool {
	return loginChallenge == "" || cookie.LoginChallengeHash == cookies.ChallengeHash(loginChallenge)
}

func (c *CookieTenantResolver) NeedsTenantSelection(ctx context.Context, session *kClient.Session, cookie cookies.FlowStateCookie, loginChallenge string) (bool, cookies.FlowStateCookie, error) {
	identityID := identityIDFromSession(session)
	if identityID != "" {
		return c.needsTenantSelectionByIdentityID(ctx, identityID, cookie, loginChallenge)
	}
	return c.NeedsTenantSelectionByEmail(ctx, emailFromSession(session), cookie, loginChallenge)
}

func (c *CookieTenantResolver) NeedsTenantSelectionByEmail(ctx context.Context, email string, cookie cookies.FlowStateCookie, loginChallenge string) (bool, cookies.FlowStateCookie, error) {
	if c.TenantID(cookie, loginChallenge) != "" {
		return false, cookie, nil
	}
	if email == "" {
		return false, cookie, nil
	}
	tenants, err := c.service.LookupTenantsByEmail(ctx, email)
	if err != nil {
		return false, cookie, fmt.Errorf("cannot look up tenants: %w", err)
	}
	if len(tenants) == 1 {
		cookie.TenantID = tenants[0].ID
		return false, cookie, nil
	}
	if len(tenants) > 1 {
		return true, cookie, nil
	}
	cookie.TenantID = cookies.NoTenantAvailable
	return false, cookie, nil
}

// needsTenantSelectionByIdentityID is the identity_id-based equivalent of
// NeedsTenantSelectionByEmail. It skips the Kratos email resolution on the
// tenant-service side, resulting in faster lookups.
func (c *CookieTenantResolver) needsTenantSelectionByIdentityID(ctx context.Context, identityID string, cookie cookies.FlowStateCookie, loginChallenge string) (bool, cookies.FlowStateCookie, error) {
	if c.TenantID(cookie, loginChallenge) != "" {
		return false, cookie, nil
	}
	tenants, err := c.service.LookupTenantsByIdentityID(ctx, identityID)
	if err != nil {
		return false, cookie, fmt.Errorf("cannot look up tenants: %w", err)
	}
	if len(tenants) == 1 {
		cookie.TenantID = tenants[0].ID
		return false, cookie, nil
	}
	if len(tenants) > 1 {
		return true, cookie, nil
	}
	cookie.TenantID = cookies.NoTenantAvailable
	return false, cookie, nil
}

func (c *CookieTenantResolver) InterceptLogin(ctx context.Context, session *kClient.Session, cookie cookies.FlowStateCookie, loginChallenge string) (LoginInterception, error) {
	if !c.IsAuthenticatedForChallenge(cookie, loginChallenge) {
		// The cookie's challenge hash doesn't match the current challenge.
		// If there is no existing session the user is still authenticating
		// (identifier-first in progress) — defer all checks.
		if session == nil {
			return LoginInterception{DeferMFAChecks: true, Cookie: cookie}, nil
		}

		// Session reuse: the user has a valid Kratos session from a
		// previous flow. Authentication is skipped, but multi-tenant
		// users must still select a tenant for this challenge.
		// Bind the cookie to the new challenge and clear the stale
		// TenantID so tenant selection is re-evaluated for this flow.
		cookie.LoginChallengeHash = cookies.ChallengeHash(loginChallenge)
		cookie.TenantID = ""
		needsSelection, updatedCookie, err := c.NeedsTenantSelection(ctx, session, cookie, loginChallenge)
		if err != nil {
			return LoginInterception{}, err
		}
		if needsSelection {
			return LoginInterception{DeferMFAChecks: true, SelectTenant: true, Cookie: cookie}, nil
		}
		return LoginInterception{DeferMFAChecks: true, AcceptLogin: true, Cookie: updatedCookie}, nil
	}

	needsSelection, updatedCookie, err := c.NeedsTenantSelection(ctx, session, cookie, loginChallenge)
	if err != nil {
		return LoginInterception{}, err
	}
	if needsSelection {
		return LoginInterception{SelectTenant: true, Cookie: cookie}, nil
	}

	return LoginInterception{AcceptLogin: true, Cookie: updatedCookie}, nil
}

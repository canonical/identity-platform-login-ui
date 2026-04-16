// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

package tenants

import (
	"context"
	"net/http"

	kClient "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
)

// CookieManagerInterface is the subset of the cookie manager this package needs.
// Re-defined locally to avoid a hard dependency on internal/cookies interfaces
// and to keep this package's dependencies explicit and testable.
type CookieManagerInterface interface {
	SetStateCookie(http.ResponseWriter, cookies.FlowStateCookie) error
	GetStateCookie(*http.Request) (cookies.FlowStateCookie, error)
}

// TenantStorerInterface is the subset of TenantResolverInterface needed by the
// tenant-redirect handler to persist the user's tenant selection.
type TenantStorerInterface interface {
	StoreTenant(w http.ResponseWriter, r *http.Request, tenantID, loginChallenge string) error
}

type ServiceInterface interface {
	LookupTenantsByFlow(ctx context.Context, flowID string, cookies []*http.Cookie) ([]Tenant, error)
	LookupTenantsByEmail(ctx context.Context, email string) ([]Tenant, error)
	LookupTenantsByIdentityID(ctx context.Context, identityID string) ([]Tenant, error)
}

// FlowFetcherInterface is the subset of kratos.ServiceInterface needed by the
// service to retrieve a login flow and extract the email.
type FlowFetcherInterface interface {
	GetLoginFlow(ctx context.Context, id string, cookies []*http.Cookie) (*kClient.LoginFlow, []*http.Cookie, error)
}

// SessionCheckerInterface is the subset of kratos.ServiceInterface needed to
// verify the caller's Kratos session.
type SessionCheckerInterface interface {
	CheckSession(ctx context.Context, cookies []*http.Cookie) (*kClient.Session, []*http.Cookie, error)
}

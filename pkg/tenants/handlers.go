// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

package tenants

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	kClient "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
)

// API exposes the tenant selection endpoint.
type API struct {
	service        ServiceInterface
	sessionChecker SessionCheckerInterface
	storer         TenantStorerInterface
	baseURL        string

	tracer tracing.TracingInterface
	logger logging.LoggerInterface
}

func (a *API) RegisterEndpoints(mux *chi.Mux) {
	mux.Get("/api/v0/tenants", a.handleLookupTenants)
	mux.Post("/api/v0/auth/tenant", a.handleTenantSelection)
}

// handleLookupTenants accepts an optional ?flow= query parameter. When flow is
// provided it fetches the Kratos flow to extract the email; otherwise the
// identity ID is read from the active Kratos session. Email is never accepted
// as a URL parameter to prevent unauthenticated tenant enumeration.
func (a *API) handleLookupTenants(w http.ResponseWriter, r *http.Request) {
	flowID := r.URL.Query().Get("flow")

	var (
		tenants []Tenant
		err     error
	)
	if flowID != "" {
		tenants, err = a.service.LookupTenantsByFlow(r.Context(), flowID, r.Cookies())
	} else {
		var session *kClient.Session
		session, _, err = a.sessionChecker.CheckSession(r.Context(), r.Cookies())
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		identityID := identityIDFromSession(session)
		if identityID == "" {
			http.Error(w, "could not determine identity from session", http.StatusUnauthorized)
			return
		}
		tenants, err = a.service.LookupTenantsByIdentityID(r.Context(), identityID)
	}
	if err != nil {
		a.logger.Errorf("failed to look up tenants: %v", err)
		http.Error(w, "failed to look up tenants", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"tenants": tenants})
}

// tenantSelectionRequest is the JSON body for POST /api/v0/auth/tenant.
type tenantSelectionRequest struct {
	LoginChallenge string `json:"login_challenge"`
	TenantID       string `json:"tenant_id"`
	Flow           string `json:"flow"`
}

// handleTenantSelection receives a JSON body with login_challenge, tenant_id,
// and flow ID. It persists the tenant selection into the encrypted state cookie
// and returns a JSON response redirecting to /ui/login?flow=<flow> so the user
// lands on the already-advanced Kratos flow (choose_method step).
//
// If tenant_id is empty the handler performs a server-side tenant lookup to
// verify the user genuinely has no tenants available, then stores the
// no-tenant sentinel on their behalf. The sentinel value itself is never
// accepted directly from the client to prevent bypassing tenant selection.
func (a *API) handleTenantSelection(w http.ResponseWriter, r *http.Request) {
	var body tenantSelectionRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if body.LoginChallenge == "" {
		http.Error(w, "login_challenge is required", http.StatusBadRequest)
		return
	}

	// The backend determines when to store the no-tenant sentinel; rejecting it
	// from the client prevents a user from bypassing tenant selection.
	if body.TenantID == cookies.NoTenantAvailable {
		http.Error(w, "invalid tenant_id", http.StatusBadRequest)
		return
	}

	// Determine the tenant ID to persist. An empty submission is allowed only
	// when server-side verification confirms the user has no tenants.
	// A non-empty tenant_id is stored without membership validation here.
	// Defense-in-depth is provided by two server-side gates in the Tenant
	// Service: the Kratos login hook and the Hydra token hook both call
	// GetActiveMemberByTenantAndUserID and reject non-members with 403,
	// preventing any forged tenant_id from reaching the final OAuth2 tokens.
	tenantToStore := body.TenantID
	if tenantToStore == "" {
		tenants, err := a.lookupTenants(r.Context(), body.Flow, r.Cookies())
		if err != nil {
			a.logger.Errorf("failed to look up tenants for empty-selection check: %v", err)
			http.Error(w, "failed to verify tenant list", http.StatusInternalServerError)
			return
		}
		if len(tenants) != 0 {
			http.Error(w, "tenant_id is required", http.StatusBadRequest)
			return
		}
		tenantToStore = cookies.NoTenantAvailable
	}

	if err := a.storer.StoreTenant(w, r, tenantToStore, body.LoginChallenge); err != nil {
		a.logger.Errorf("failed to persist tenant selection state: %v", err)
		http.Error(w, "failed to persist tenant selection", http.StatusInternalServerError)
		return
	}

	// When flow is absent (active-session path) redirect back to the login page
	// with only the login_challenge so the backend can accept the Hydra challenge
	// using the existing Kratos session, without starting a new credentials flow.
	var redirectTo string
	if body.Flow != "" {
		redirectTo = a.loginURL(body.Flow)
	} else {
		redirectTo = a.loginChallengeURL(body.LoginChallenge)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"redirect_to": redirectTo})
}

// lookupTenants returns the caller's tenant list using the Kratos flow when
// provided, falling back to the active Kratos session's identity ID.
func (a *API) lookupTenants(ctx context.Context, flowID string, httpCookies []*http.Cookie) ([]Tenant, error) {
	if flowID != "" {
		return a.service.LookupTenantsByFlow(ctx, flowID, httpCookies)
	}
	session, _, err := a.sessionChecker.CheckSession(ctx, httpCookies)
	if err != nil {
		return nil, fmt.Errorf("cannot check session: %v", err)
	}
	identityID := identityIDFromSession(session)
	if identityID == "" {
		return nil, fmt.Errorf("cannot determine identity from session")
	}
	return a.service.LookupTenantsByIdentityID(ctx, identityID)
}

// loginChallengeURL builds the /ui/login?login_challenge=<challenge> URL used
// in the active-session path where no Kratos flow exists yet. Redirecting here
// lets the backend accept the Hydra challenge via the existing session.
func (a *API) loginChallengeURL(loginChallenge string) string {
	loginPath, _ := url.JoinPath(a.baseURL, "/ui/login")
	u, _ := url.Parse(loginPath)
	uq := u.Query()
	uq.Set("login_challenge", loginChallenge)
	u.RawQuery = uq.Encode()
	return u.String()
}

// loginURL builds the /ui/login?flow=<flowID> URL for redirecting back to
// the Kratos 1 FA after tenant selection.
func (a *API) loginURL(flowID string) string {
	loginPath, _ := url.JoinPath(a.baseURL, "/ui/login")
	u, _ := url.Parse(loginPath)
	uq := u.Query()
	uq.Set("flow", flowID)
	u.RawQuery = uq.Encode()
	return u.String()
}

func NewAPI(
	service ServiceInterface,
	storer TenantStorerInterface,
	sessionChecker SessionCheckerInterface,
	baseURL string,
	tracer tracing.TracingInterface,
	logger logging.LoggerInterface,
) *API {
	return &API{
		service:        service,
		sessionChecker: sessionChecker,
		storer:         storer,
		baseURL:        baseURL,
		tracer:         tracer,
		logger:         logger,
	}
}

// emailFromSession extracts the email trait from a Kratos session identity.
func emailFromSession(session *kClient.Session) string {
	if session == nil || session.Identity == nil {
		return ""
	}
	traits, ok := session.Identity.Traits.(map[string]interface{})
	if !ok {
		return ""
	}
	email, _ := traits["email"].(string)
	return email
}

// identityIDFromSession extracts the Kratos identity ID from a session.
func identityIDFromSession(session *kClient.Session) string {
	if session == nil || session.Identity == nil {
		return ""
	}
	return session.Identity.Id
}

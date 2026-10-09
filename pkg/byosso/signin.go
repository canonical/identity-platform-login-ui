// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	kClient "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

// tenantServiceFailed answers a failed call to tenant-service and logs it.
// A tenant-service that could not be reached, or was too slow, has refused
// nothing: the user is asked to try again in a moment.
func (a *API) tenantServiceFailed(w http.ResponseWriter, err error, format string, args ...any) {
	a.logger.Errorf("%s: %v", fmt.Sprintf(format, args...), err)
	if isUnavailable(err) {
		tenantsUnavailable(w)
		return
	}
	ssoUnavailable(w)
}

// identityExists reports whether an account holds the address. It is asked
// only about an address Kratos refused at the identifier step that no tenant
// is offered to: an unknown address keeps the answer of Kratos, while an
// account with no sign-in method is offered a way in.
func (a *API) identityExists(ctx context.Context, email string) bool {
	exists, err := a.identities.IdentityExists(ctx, email)
	if err != nil {
		a.logger.Errorf("failed to look up the identity of an address: %v", err)
		return false
	}
	return exists
}

// decideSignIn stores what the sign-in screen starts from and tells the
// browser where to go next. With one tenant it is bound to the login
// challenge, and the browser goes to its company sign-in when that is the
// only choice. With several nothing is bound, and the login page lists them.
// With none the user signs in with what the account has.
func (a *API) decideSignIn(w http.ResponseWriter, r *http.Request, flow *kClient.LoginFlow, tenants []SignInTenant) {
	loginChallenge := flow.GetOauth2LoginChallenge()

	var err error
	switch len(tenants) {
	case 0:
		err = a.bindTenant(w, r, loginChallenge, cookies.NoTenantAvailable, false)
	case 1:
		if a.routeToCompanySignIn(w, r, flow, tenants[0].ID, false) {
			return
		}
		err = a.bindTenant(w, r, loginChallenge, tenants[0].ID, false)
	default:
		err = a.clearTenant(w, true)
	}
	if err != nil {
		a.logger.Errorf("failed to set state cookie: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	redirectResponse(w, r, newRedirect(a.loginFlowURL(flow.Id)))
}

// routeToCompanySignIn sends the browser straight to the company sign-in of
// the tenant when that is the only choice the tenant gives the address of the
// flow. chosen is set when the user picked the tenant from several. It
// returns false when the login page is to show what the tenant offers.
func (a *API) routeToCompanySignIn(w http.ResponseWriter, r *http.Request, flow *kClient.LoginFlow, tenantID string, chosen bool) bool {
	ctx := r.Context()
	email := flowInputValue(flow.Ui.Nodes, "identifier")

	sc, err := a.directory.SignInContext(ctx, tenantID, email, "")
	if err != nil {
		a.tenantServiceFailed(w, err, "failed to get the sign-in context of tenant %s", tenantID)
		return true
	}
	if !sc.Offered() {
		a.logger.Debugf("tenant %s is not offered to the address of login flow %s", tenantID, flow.Id)
		http.Error(w, "tenant not allowed", http.StatusForbidden)
		return true
	}
	if !sc.CompanyOnly() {
		return false
	}

	options, err := a.sso.Options(ctx, sc.ConnectionIDs)
	switch {
	case err != nil:
		a.logger.Errorf("failed to list the company sign-ins of tenant %s: %v", tenantID, err)
		ssoUnavailable(w)
		return true
	case len(options) == 0:
		ssoNotApplicable(w)
		return true
	case len(options) > 1:
		return false
	}

	lr, err := a.loginRequestOf(ctx, flow)
	if err != nil {
		a.logger.Errorf("failed to read the login request: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return true
	}
	current := a.currentSession(r)
	to, ok := a.startAttempt(w, r, flow, nil, attempt{
		loginChallenge: flow.GetOauth2LoginChallenge(),
		tenantID:       tenantID,
		email:          email,
		connectionID:   options[0].ConnectionID,
		reauthenticate: lr.Reauthenticate(current, a.now()),
		tenantChosen:   chosen,
		prior:          current.GetId(),
	})
	if ok {
		redirectResponse(w, r, newRedirect(to).withLabel(options[0].Label))
	}
	return true
}

// hydrateSignInScreen lays out a flow with no tenant chosen yet: the tenant
// list and nothing else or, for an account with no tenant, its sign-ins or
// the recovery prompt.
func (a *API) hydrateSignInScreen(r *http.Request, flow *kClient.LoginFlow, email string) *kClient.LoginFlow {
	if email == "" {
		return flow
	}

	if accountNotFound(flow) {
		tenants, err := a.directory.SignInTenants(r.Context(), email)
		if err == nil && len(tenants) > 0 {
			return withTenants(withoutFirstFactors(withoutIdentifierStep(flow)), tenants)
		}
		if !a.identityExists(r.Context(), email) {
			return flow
		}
		flow = withoutIdentifierStep(flow)
		if err != nil {
			a.logger.Errorf("failed to look up tenants on the sign-in screen: %v", err)
			return withMessage(flow, lookupFailedID, lookupFailedText, "error")
		}
		return a.withRecoverPrompt(flow)
	}
	if hasIdentifierFirst(flow) {
		return flow
	}

	otherFirstFactor := hasOtherFirstFactor(flow)
	tenants, err := a.directory.SignInTenants(r.Context(), email)
	if err != nil {
		a.logger.Errorf("failed to look up tenants on the sign-in screen: %v", err)
		if otherFirstFactor {
			return flow
		}
		return withMessage(flow, lookupFailedID, lookupFailedText, "error")
	}
	switch {
	case len(tenants) == 0 && !otherFirstFactor:
		return a.withRecoverPrompt(flow)
	case len(tenants) == 0:
		return flow
	}
	// The sign-ins of the account belong to the tenants that accept them, and
	// are shown once one is picked.
	return withTenants(withoutFirstFactors(flow), tenants)
}

// hydrateNoTenant lays out the flow of an account with no tenant: the user
// signs in with the portal password or a public sign-in, and the personal
// tenant is created when the login is accepted. An account Kratos refused at
// the identifier step has no sign-in method: the recovery prompt. An unknown
// address keeps the answer of Kratos.
func (a *API) hydrateNoTenant(r *http.Request, flow *kClient.LoginFlow) *kClient.LoginFlow {
	if accountNotFound(flow) {
		if !a.identityExists(r.Context(), flowInputValue(flow.Ui.Nodes, "identifier")) {
			return flow
		}
		return a.withRecoverPrompt(withoutIdentifierStep(flow))
	}
	if hasOtherFirstFactor(flow) || hasIdentifierFirst(flow) {
		return flow
	}
	return a.withRecoverPrompt(flow)
}

// hydrateChosenTenant lays out what the chosen tenant offers the address:
// its company sign-ins and, unless it requires them, the portal password and
// the public sign-ins. The flow of a registration offers company sign-ins
// only. chosen is set when the user picked the tenant from several, and may
// go back to choose another. It returns false when it answered the request.
func (a *API) hydrateChosenTenant(w http.ResponseWriter, r *http.Request, flow *kClient.LoginFlow, tenantID, email string, registration, chosen bool) (*kClient.LoginFlow, bool) {
	ctx := r.Context()

	sc, err := a.directory.SignInContext(ctx, tenantID, email, "")
	if err != nil {
		a.tenantServiceFailed(w, err, "failed to get the sign-in context of tenant %s", tenantID)
		return nil, false
	}
	if !sc.Offered() {
		if registration {
			ssoNotApplicable(w)
			return nil, false
		}
		// The tenant is bound to the login challenge for another address:
		// nothing is chosen for this one.
		return a.hydrateSignInScreen(r, flow, email), true
	}
	if !registration && accountNotFound(flow) {
		// The tenant offers itself to an address Kratos refused at the
		// identifier step: an account with no sign-in method yet, or an
		// address with no account that a pending invitation or auto-join
		// admits, whose company sign-in creates the account.
		flow = withoutIdentifierStep(flow)
	}

	companyOnly := registration || sc.CompanyOnly()
	otherFirstFactor := hasOtherFirstFactor(flow)
	if companyOnly {
		flow = withoutFirstFactors(flow)
		otherFirstFactor = false
	}

	options, err := a.sso.Options(ctx, sc.ConnectionIDs)
	switch {
	case err != nil && companyOnly:
		a.logger.Errorf("failed to list the company sign-ins of tenant %s: %v", tenantID, err)
		ssoUnavailable(w)
		return nil, false
	case err != nil:
		a.logger.Errorf("failed to list the company sign-ins of tenant %s: %v", tenantID, err)
		flow = withMessage(flow, companyUnavailableID, companyUnavailableText, "info")
		options = nil
	case companyOnly && len(options) == 0:
		// No company sign-in applies, e.g. the address is outside the domains
		// of the tenant. The password is never a fallback.
		ssoNotApplicable(w)
		return nil, false
	case len(options) == 0 && !otherFirstFactor:
		flow = a.withRecoverPrompt(flow)
	}

	for _, o := range options {
		flow.Ui.Nodes = append(flow.Ui.Nodes, companySignInNode(ssoConnectionField, o.ConnectionID, o.Label))
	}
	if chosen {
		flow.Ui.Nodes = append(flow.Ui.Nodes, submitNode(tenantNodeGroup, tenantResetField, "1", "Choose another tenant", signInWith))
	}
	return flow, true
}

// pick starts the chosen company sign-in. When the state cookie has expired,
// and the tenant with it, the page of the flow lists the tenants again.
func (a *API) pick(w http.ResponseWriter, r *http.Request, flow *kClient.LoginFlow, connectionID string) {
	tenantID, email, registration := a.flowTenant(r, flow)
	if tenantID == "" && email != "" && registration == nil {
		redirectResponse(w, r, newRedirect(a.loginFlowURL(flow.Id)))
		return
	}
	if tenantID == "" || tenantID == cookies.NoTenantAvailable || email == "" {
		http.Error(w, "company sign-in requires a chosen tenant", http.StatusBadRequest)
		return
	}
	a.pickFor(w, r, flow, tenantID, email, registration, connectionID)
}

// pickFor starts the company sign-in connectionID of tenantID for email on
// flow. The list the browser picked from is a hint: the tenant must still
// offer the connection to the address.
func (a *API) pickFor(w http.ResponseWriter, r *http.Request, flow *kClient.LoginFlow, tenantID, email string, registration *RegistrationCookie, connectionID string) {
	ctx := r.Context()

	sc, err := a.directory.SignInContext(ctx, tenantID, email, "")
	if err != nil {
		a.tenantServiceFailed(w, err, "failed to get the sign-in context of tenant %s", tenantID)
		return
	}
	if !sc.Offered() || !sc.Applies(connectionID) {
		a.logger.Debugf("connection %s is not offered by tenant %s to the address of login flow %s", connectionID, tenantID, flow.Id)
		http.Error(w, "Provider not allowed", http.StatusForbidden)
		return
	}

	loginChallenge := flow.GetOauth2LoginChallenge()
	current := a.currentSession(r)
	p := attempt{
		loginChallenge: loginChallenge,
		tenantID:       tenantID,
		email:          email,
		connectionID:   connectionID,
		prior:          current.GetId(),
	}
	if registration != nil {
		p.join = true
		p.returnTo = registration.ReturnTo
	}
	if loginChallenge != "" {
		lr, err := a.loginRequestOf(ctx, flow)
		if err != nil {
			a.logger.Errorf("failed to read the login request: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		p.reauthenticate = lr.Reauthenticate(current, a.now())
	}

	to, ok := a.startAttempt(w, r, flow, nil, p)
	if !ok {
		return
	}
	if registration != nil {
		a.store.ClearRegistration(w)
	}
	redirectResponse(w, r, newRedirect(to))
}

// chooseTenant takes a pick from the tenant list. The tenant must offer
// itself to the address of the flow. The browser goes straight to its company
// sign-in when that is the only choice, otherwise to the login page showing
// what the tenant offers.
func (a *API) chooseTenant(w http.ResponseWriter, r *http.Request, flow *kClient.LoginFlow, tenantID string) {
	loginChallenge := flow.GetOauth2LoginChallenge()
	if loginChallenge == "" || tenantID == cookies.NoTenantAvailable {
		http.Error(w, "choosing a tenant requires an oauth2 flow", http.StatusBadRequest)
		return
	}
	chosen := a.tenantChoice(r)
	if a.routeToCompanySignIn(w, r, flow, tenantID, chosen) {
		return
	}
	if err := a.bindTenant(w, r, loginChallenge, tenantID, chosen); err != nil {
		a.logger.Errorf("failed to set state cookie: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	redirectResponse(w, r, newRedirect(a.loginFlowURL(flow.Id)))
}

// resetTenant goes back to the tenant list with nothing chosen.
func (a *API) resetTenant(w http.ResponseWriter, r *http.Request, flow *kClient.LoginFlow) {
	if flow.GetOauth2LoginChallenge() == "" {
		http.Error(w, "choosing a tenant requires an oauth2 flow", http.StatusBadRequest)
		return
	}
	if err := a.clearTenant(w, a.tenantChoice(r)); err != nil {
		a.logger.Errorf("failed to set state cookie: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	redirectResponse(w, r, newRedirect(a.loginFlowURL(flow.Id)))
}

// tenantChoice reports whether the state cookie says that the user has
// several tenants to choose from.
func (a *API) tenantChoice(r *http.Request) bool {
	stateCookie, err := a.state.GetStateCookie(r)
	if err != nil {
		a.logger.Errorf("failed to read state cookie: %v", err)
	}
	return stateCookie.TenantChoice
}

// bindTenant binds tenantID to the login challenge in the state cookie.
// chosen is set when the user has just picked the tenant from several. A
// tenant bound as such already stays one.
func (a *API) bindTenant(w http.ResponseWriter, r *http.Request, loginChallenge, tenantID string, chosen bool) error {
	stateCookie, err := a.state.GetStateCookie(r)
	if err != nil {
		a.logger.Errorf("failed to read state cookie: %v", err)
	}
	if stateCookie.TenantChoice && a.tenants.TenantID(stateCookie, loginChallenge) == tenantID {
		chosen = true
	}
	stateCookie = stateCookie.RenewForChallenge(loginChallenge)
	stateCookie.TenantID = tenantID
	stateCookie.TenantChoice = chosen
	return a.state.SetStateCookie(w, stateCookie)
}

// clearTenant leaves nothing chosen. choice is set when the user has several
// tenants to choose from. The cookie names no login challenge: bound to the
// challenge with no tenant, it would be read as a tenant selection still
// owed after the first factor.
func (a *API) clearTenant(w http.ResponseWriter, choice bool) error {
	return a.state.SetStateCookie(w, cookies.FlowStateCookie{TenantChoice: choice})
}

// flowTenant returns the tenant and the address a login flow signs in for:
// those of a registration, or the tenant bound to the login challenge.
func (a *API) flowTenant(r *http.Request, flow *kClient.LoginFlow) (string, string, *RegistrationCookie) {
	if registration, ok := a.registrationFor(r, flow.Id); ok {
		return registration.TenantID, registration.Email, &registration
	}
	loginChallenge := flow.GetOauth2LoginChallenge()
	if loginChallenge == "" {
		return "", "", nil
	}
	stateCookie, err := a.state.GetStateCookie(r)
	if err != nil {
		a.logger.Errorf("failed to read state cookie: %v", err)
		return "", "", nil
	}
	return a.tenants.TenantID(stateCookie, loginChallenge), flowInputValue(flow.Ui.Nodes, "identifier"), nil
}

// loginRequestOf reads prompt and max_age from the Hydra login request of
// the flow, asking Hydra only when the flow does not carry it.
func (a *API) loginRequestOf(ctx context.Context, flow *kClient.LoginFlow) (*LoginRequest, error) {
	if lr := flow.Oauth2LoginRequest; lr != nil && lr.RequestUrl != nil {
		return parseLoginRequestURL(*lr.RequestUrl), nil
	}
	return a.hydra.LoginRequest(ctx, flow.GetOauth2LoginChallenge())
}

// currentSession returns the Kratos session of the browser, nil when it has
// none.
func (a *API) currentSession(r *http.Request) *kClient.Session {
	if _, err := r.Cookie(kratos.KRATOS_SESSION_COOKIE_NAME); err != nil {
		return nil
	}
	session, _, err := a.kratos.CheckSession(r.Context(), r.Cookies())
	if err != nil || session == nil {
		return nil
	}
	return session
}

// withRecoverPrompt adds the prompt to recover the account, and a link to
// the recovery page that returns to this login.
func (a *API) withRecoverPrompt(flow *kClient.LoginFlow) *kClient.LoginFlow {
	flow = withMessage(flow, recoverPromptID, recoverPromptText, "info")

	returnTo := a.loginURL(flow.GetOauth2LoginChallenge())
	anchor := kClient.NewUiNodeAnchorAttributes(
		a.pageURL("/ui/reset_email", url.Values{"return_to": {returnTo}}),
		recoverAnchorID, "a", *kClient.NewUiText(recoverPromptID, "Recover your account", "info"))

	node := kClient.NewUiNodeWithDefaults()
	node.Type = "a"
	node.Group = tenantNodeGroup
	node.Attributes = kClient.UiNodeAttributes{UiNodeAnchorAttributes: anchor}
	node.Meta = *kClient.NewUiNodeMeta()
	flow.Ui.Nodes = append(flow.Ui.Nodes, *node)
	return flow
}

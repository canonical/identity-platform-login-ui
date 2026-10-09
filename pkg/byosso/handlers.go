// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

// Package byosso implements "Bring Your Own SSO": the employees of a tenant
// sign in with the identity provider of their company. It takes part in the
// login, registration, settings and consent flows through the extension hooks
// of the kratos and extra packages.
//
// tenant-service tells what a tenant offers an address or an account, and
// sso-service lists the company sign-ins and runs one. login-ui submits the
// one Kratos provider behind every company sign-in with a ticket of
// sso-service, records which company sign-in a session came from in its own
// cookies, and checks every Hydra login against the chosen tenant before
// accepting it. An account the tenant admits joins it when its login is
// accepted.
package byosso

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	kClient "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
)

type API struct {
	baseURL                 string
	contextPath             string
	origin                  string
	privilegedSessionMaxAge time.Duration

	kratos       KratosServiceInterface
	sso          SSOInterface
	directory    DirectoryInterface
	hydra        HydraInterface
	verification VerificationInterface
	identities   IdentityFinderInterface
	state        StateCookieInterface
	tenants      TenantResolverInterface
	store        CookieStoreInterface
	now          func() time.Time

	tracer tracing.TracingInterface
	logger logging.LoggerInterface
}

func (a *API) RegisterEndpoints(mux *chi.Mux) {
	mux.Get(completePath, a.handleComplete)
}

// HydrateLoginFlow lays out the flow the frontend renders: the tenant list
// while no tenant is chosen, then what the chosen tenant offers.
func (a *API) HydrateLoginFlow(w http.ResponseWriter, r *http.Request, flow *kClient.LoginFlow) (*kClient.LoginFlow, bool) {
	if flow == nil || flow.GetRequestedAal() == kClient.AUTHENTICATORASSURANCELEVEL_AAL2 {
		return flow, true
	}
	r, cancel := budget(r)
	defer cancel()

	if isAccountLinking(flow) {
		return a.hydrateAccountLinking(r, flow), true
	}
	if isRefresh(flow) {
		return a.hydrateRefresh(r, flow), true
	}
	if registration, ok := a.registrationFor(r, flow.Id); ok {
		if registration.TenantID == "" {
			return a.hydrateRegistrationChoice(r, flow, registration), true
		}
		return a.hydrateChosenTenant(w, r, flow, registration.TenantID, registration.Email, true, false)
	}

	loginChallenge := flow.GetOauth2LoginChallenge()
	if loginChallenge == "" {
		return flow, true
	}
	stateCookie, err := a.state.GetStateCookie(r)
	if err != nil {
		a.logger.Errorf("failed to read state cookie: %v", err)
		return flow, true
	}
	email := flowInputValue(flow.Ui.Nodes, "identifier")
	switch tenantID := a.tenants.TenantID(stateCookie, loginChallenge); tenantID {
	case "":
		return a.hydrateSignInScreen(r, flow, email), true
	case cookies.NoTenantAvailable:
		return a.hydrateNoTenant(r, flow), true
	default:
		return a.hydrateChosenTenant(w, r, flow, tenantID, email, false, stateCookie.TenantChoice)
	}
}

// BeforeTenantSelection decides what follows the email for an OAuth2 login:
// one tenant goes straight to its sign-in, several are listed, and with none
// the user signs in with what the account has. When tenant-service cannot be
// reached the user is asked to try again in a moment. Any other failure of
// the lookup is left to the handler.
func (a *API) BeforeTenantSelection(w http.ResponseWriter, r *http.Request, flow *kClient.LoginFlow, email string) bool {
	if email == "" || flow.GetOauth2LoginChallenge() == "" {
		return false
	}
	r, cancel := budget(r)
	defer cancel()

	tenants, err := a.directory.SignInTenants(r.Context(), email)
	switch {
	case isUnavailable(err):
		a.tenantServiceFailed(w, err, "failed to look up tenants after the identifier step")
		return true
	case err != nil && accountNotFound(flow) && !a.identityExists(r.Context(), email):
		// tenant-service refuses an identifier Kratos refused too (e.g. one
		// that is not an address) and no account holds it: the answer of
		// Kratos stands.
		tenants = nil
	case err != nil:
		a.logger.Errorf("failed to look up tenants after the identifier step: %v", err)
		return false
	}

	a.decideSignIn(w, r, flow, tenants)
	return true
}

// InterceptLoginSubmission refuses the company sign-in provider submitted by
// the browser, answers the pick of a tenant or of a company sign-in, and
// marks a first factor started for a login challenge.
func (a *API) InterceptLoginSubmission(w http.ResponseWriter, r *http.Request, flow *kClient.LoginFlow) (*http.Request, bool) {
	if flow == nil {
		return r, false
	}
	fields, ok := a.submissionFields(w, r)
	if !ok {
		return r, true
	}
	if stringField(fields, "provider") == providerID {
		a.logger.Warnf("refused a submission of provider %s on login flow %s", providerID, flow.Id)
		http.Error(w, "Provider not allowed", http.StatusForbidden)
		return r, true
	}
	if flow.GetRequestedAal() == kClient.AUTHENTICATORASSURANCELEVEL_AAL2 {
		return r, false
	}
	if a.answerPick(w, r, flow, fields) {
		return r, true
	}
	if flow.GetOauth2LoginChallenge() != "" && isFirstFactorSubmission(fields) {
		return a.markFresh(w, r, flow), false
	}
	return r, false
}

// submissionFields returns the fields of a submission. It returns false when
// it refused one that names a field twice.
func (a *API) submissionFields(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	fields, err := requestFields(r)
	if err != nil {
		a.logger.Warnf("refused a submission to %s: %v", r.URL.Path, err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return nil, false
	}
	return fields, true
}

// answerPick answers a submission that picks a tenant or a company sign-in.
// It returns false when the submission is neither.
func (a *API) answerPick(w http.ResponseWriter, r *http.Request, flow *kClient.LoginFlow, fields map[string]any) bool {
	r, cancel := budget(r)
	defer cancel()

	if connectionID := stringField(fields, ssoLinkField); connectionID != "" {
		a.linkingPick(w, r, flow, connectionID)
		return true
	}
	if connectionID := stringField(fields, ssoReauthField); connectionID != "" {
		a.refreshPick(w, r, flow, connectionID)
		return true
	}
	if connectionID := stringField(fields, ssoConnectionField); connectionID != "" {
		a.pick(w, r, flow, connectionID)
		return true
	}
	if tenantID := stringField(fields, tenantField); tenantID != "" {
		if registration, ok := a.registrationFor(r, flow.Id); ok && registration.TenantID == "" {
			a.chooseRegistrationTenant(w, r, flow, registration, tenantID)
			return true
		}
		a.chooseTenant(w, r, flow, tenantID)
		return true
	}
	if stringField(fields, tenantResetField) != "" {
		a.resetTenant(w, r, flow)
		return true
	}
	return false
}

// HandlesSessionLogin reports that every Hydra login made with a session is
// decided here.
func (a *API) HandlesSessionLogin() bool { return true }

// HandleSessionLogin answers the login page of a Hydra login request with a
// Kratos session.
func (a *API) HandleSessionLogin(w http.ResponseWriter, r *http.Request, session *kClient.Session, loginChallenge string) {
	r, cancel := budget(r)
	defer cancel()

	stateCookie, err := a.state.GetStateCookie(r)
	if err != nil {
		a.logger.Errorf("failed to read state cookie: %v", err)
	}
	a.checkAndAccept(w, r, session, loginChallenge, stateCookie)
}

// BeforeAcceptLogin checks the session a login submission produced before
// the Hydra login is accepted.
func (a *API) BeforeAcceptLogin(w http.ResponseWriter, r *http.Request, session *kClient.Session, loginChallenge string, stateCookie cookies.FlowStateCookie) bool {
	if loginChallenge == "" || session == nil || session.Identity == nil {
		return false
	}
	r, cancel := budget(r)
	defer cancel()

	a.checkAndAccept(w, r, session, loginChallenge, stateCookie)
	return true
}

// InterceptRegistrationSubmission refuses a registration with the company
// sign-in provider or with two addresses, and sends an address tenants admit
// to their company sign-in instead of a password: by auto-join, or by a
// pending invitation of
// an address with no account to a tenant that requires company sign-in. That
// sign-in registers the account, so that nobody else can register the invited
// address first. With one such tenant the browser goes straight to its
// company sign-in; with several the user picks one.
func (a *API) InterceptRegistrationSubmission(w http.ResponseWriter, r *http.Request, flowID string) bool {
	fields, ok := a.submissionFields(w, r)
	if !ok {
		return true
	}
	if stringField(fields, "provider") == providerID {
		a.logger.Warnf("refused a registration with provider %s on flow %s", providerID, flowID)
		http.Error(w, "Provider not allowed", http.StatusForbidden)
		return true
	}
	email, ok := traitsEmail(fields)
	if !ok {
		a.logger.Warnf("refused a registration with two addresses on flow %s", flowID)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return true
	}
	if email == "" {
		return false
	}
	r, cancel := budget(r)
	defer cancel()

	// Without the answer the address may be one that tenants admit, which
	// must not get a password account: the registration waits.
	admitted, err := a.admittingTenants(r.Context(), email)
	if err != nil {
		a.logger.Errorf("failed to look up the tenants admitting a registration: %v", err)
		http.Error(w, registrationUnavailableMessage, http.StatusServiceUnavailable)
		return true
	}
	switch len(admitted) {
	case 0:
		return false
	case 1:
		a.registrationSignIn(w, r, flowID, email, admitted[0])
	default:
		a.registrationChoice(w, r, flowID, email)
	}
	return true
}

// HydrateSettingsFlow lists the company sign-ins of the account. When
// sso-service cannot be reached the settings are rendered without them.
func (a *API) HydrateSettingsFlow(ctx context.Context, flow *kClient.SettingsFlow, httpCookies []*http.Cookie) *kClient.SettingsFlow {
	if flow == nil {
		return flow
	}
	ctx, cancel := context.WithTimeout(ctx, requestBudget)
	defer cancel()

	identityID := flow.Identity.GetId()
	if identityID == "" {
		session, _, err := a.kratos.CheckSession(ctx, httpCookies)
		if err != nil || session == nil || session.Identity == nil {
			a.logger.Errorf("failed to check the session of a settings flow: %v", err)
			return flow
		}
		identityID = session.Identity.GetId()
	}
	return a.withLinks(ctx, flow, identityID)
}

// InterceptSettingsSubmission removes a company sign-in through sso-service,
// and refuses to link or unlink the company sign-in provider in Kratos.
func (a *API) InterceptSettingsSubmission(w http.ResponseWriter, r *http.Request, flowID string) bool {
	fields, ok := a.submissionFields(w, r)
	if !ok {
		return true
	}
	if connectionID := stringField(fields, ssoUnlinkField); connectionID != "" {
		// Kratos checks the CSRF token of a settings submission, but this
		// one never reaches Kratos: only the pages of login-ui may send it.
		if !sameOrigin(r, a.origin) {
			a.logger.Warnf("refused a cross-site unlink on settings flow %s", flowID)
			http.Error(w, "forbidden", http.StatusForbidden)
			return true
		}
		r, cancel := budget(r)
		defer cancel()

		a.unlink(w, r, flowID, connectionID)
		return true
	}
	if stringField(fields, "link") == providerID || stringField(fields, "unlink") == providerID {
		a.logger.Warnf("refused to link or unlink provider %s on settings flow %s", providerID, flowID)
		http.Error(w, "Provider not allowed", http.StatusForbidden)
		return true
	}
	return false
}

// handleComplete records which company sign-in the session came from when it
// was made with no Hydra login request, then goes to return_to: only the one
// the sign-in was started with, which came from a Kratos flow and so is one
// of the return URLs Kratos allows. The account a registration was made for
// joins its tenant here, as no Hydra login is accepted for it, once its
// address is verified; until then the join waits for a sign-in to the tenant.
func (a *API) handleComplete(w http.ResponseWriter, r *http.Request) {
	r, cancel := budget(r)
	defer cancel()

	session, _, err := a.kratos.CheckSession(r.Context(), r.Cookies())
	if err != nil || session == nil || session.Identity == nil {
		http.Redirect(w, r, a.absoluteURL("/ui/login"), http.StatusSeeOther)
		return
	}

	_, completed, err := a.completePending(w, r, session)
	if err != nil {
		a.logger.Errorf("failed to complete the company sign-in: %v", err)
		http.Error(w, ssoUnavailableMessage, http.StatusServiceUnavailable)
		return
	}

	if completed != nil && completed.join && !addressUnverified(session) {
		err := a.directory.JoinTenant(r.Context(), completed.tenantID, session.Identity.GetId())
		switch {
		case errors.Is(err, errNotAdmitted):
			a.logger.Debugf("tenant %s does not admit identity %s any more: %v", completed.tenantID, session.Identity.GetId(), err)
			http.Error(w, strings.ToUpper(notAMemberMessage[:1])+notAMemberMessage[1:], http.StatusForbidden)
			return
		case err != nil:
			a.logger.Errorf("failed to join identity %s to tenant %s: %v", session.Identity.GetId(), completed.tenantID, err)
			http.Error(w, ssoUnavailableMessage, http.StatusServiceUnavailable)
			return
		}
	}

	target := a.absoluteURL("/ui/manage_details")
	if completed != nil && completed.returnTo != "" && r.URL.Query().Get("return_to") == completed.returnTo {
		target = completed.returnTo
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func NewAPI(
	kratos KratosServiceInterface,
	sso SSOInterface,
	directory DirectoryInterface,
	hydra HydraInterface,
	verification VerificationInterface,
	identities IdentityFinderInterface,
	cookieManager StateCookieInterface,
	tenantMgr TenantResolverInterface,
	store CookieStoreInterface,
	baseURL string,
	privilegedSessionMaxAge time.Duration,
	tracer tracing.TracingInterface,
	logger logging.LoggerInterface,
) (*API, error) {
	fullBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to construct API base URL: %w", err)
	}

	a := new(API)

	a.baseURL = baseURL
	a.contextPath = fullBaseURL.Path
	a.origin = fullBaseURL.Scheme + "://" + fullBaseURL.Host
	a.privilegedSessionMaxAge = privilegedSessionMaxAge

	a.kratos = kratos
	a.sso = sso
	a.directory = directory
	a.hydra = hydra
	a.verification = verification
	a.identities = identities
	a.state = cookieManager
	a.tenants = tenantMgr
	a.store = store
	a.now = time.Now

	a.tracer = tracer
	a.logger = logger

	return a, nil
}

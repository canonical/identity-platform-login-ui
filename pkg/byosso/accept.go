// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"errors"
	"net/http"
	"net/url"

	kClient "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

// checkAndAccept checks the session against the tenant chosen for the login
// challenge, and accepts the Hydra login or asks for what is missing. Every
// Hydra login goes through it: the login page with a session (the return from
// a company or a public sign-in, or the reuse of a session), and a first
// factor or MFA just submitted.
func (a *API) checkAndAccept(w http.ResponseWriter, r *http.Request, session *kClient.Session, loginChallenge string, stateCookie cookies.FlowStateCookie) {
	ctx := r.Context()

	if session == nil || session.Identity == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	identityID := session.Identity.GetId()
	email := emailFromSession(session)

	// Tenants admit users by their address. Kratos registers an account
	// through a company sign-in with the address unverified unless the IdP
	// confirmed it, so it is verified before the sign-in completes.
	if includesCompanySignIn(session) && addressUnverified(session) {
		a.verifyAddress(w, r, session, loginChallenge)
		return
	}

	provenance, completed, err := a.completePending(w, r, session)
	if err != nil {
		a.logger.Errorf("failed to complete the company sign-in of identity %s: %v", identityID, err)
		ssoUnavailable(w)
		return
	}
	if completed != nil && completed.loginChallengeHash == cookies.ChallengeHash(loginChallenge) && completed.tenantID != "" && a.tenants.TenantID(stateCookie, loginChallenge) == "" {
		// The attempt was for this login request, and the state cookie has
		// lost its tenant: the tenant of the attempt is the one chosen.
		stateCookie = a.setTenant(w, stateCookie, loginChallenge, completed.tenantID)
	}

	lr, err := a.hydra.LoginRequest(ctx, loginChallenge)
	if err != nil {
		a.logger.Errorf("failed to read the login request: %v", err)
		http.Error(w, "failed to read the login request", http.StatusInternalServerError)
		return
	}

	// Hydra remembers another subject in this browser, who signed in to an
	// app before this one. Accepting this one would send the request of the
	// app back with prompt=login, and a company sign-in asked to
	// re-authenticate can fail at an IdP with a live session. The Kratos
	// session tells who is here: the login session of the other subject ends,
	// and the request of the app starts again.
	if lr.RemembersAnother(identityID) {
		if err := a.hydra.RevokeLoginSession(ctx, lr.SessionID); err != nil {
			a.logger.Errorf("failed to revoke the login session: %v", err)
			http.Error(w, "failed to end the previous sign-in", http.StatusInternalServerError)
			return
		}
		a.logger.Debugf("revoked the login session of another subject, the request of the app starts again for identity %s", identityID)
		redirectResponse(w, r, newRedirect(lr.RequestURL))
		return
	}

	tenantID := a.tenants.TenantID(stateCookie, loginChallenge)
	if tenantID == "" {
		tenants, err := a.directory.SignInTenants(ctx, email)
		if err != nil {
			a.tenantServiceFailed(w, err, "failed to look up the tenants of identity %s", identityID)
			return
		}
		switch len(tenants) {
		case 0:
			tenantID = cookies.NoTenantAvailable
		case 1:
			tenantID = tenants[0].ID
		default:
			a.selectTenant(w, r, loginChallenge)
			return
		}
		stateCookie = a.setTenant(w, stateCookie, loginChallenge, tenantID)
	}

	if tenantID == cookies.NoTenantAvailable {
		if includesCompanySignIn(session) {
			// A company sign-in is never accepted at a personal tenant.
			a.askFor(w, r, needFreshFirstFactor, session, loginChallenge, lr, tenantID, nil)
			return
		}
		personalTenantID, err := a.directory.CreatePersonalTenant(ctx, identityID)
		switch {
		case errors.Is(err, errHasTenant):
			// Pending invitations wait for a sign-in to their tenant, or the
			// account got a tenant since the lookup: its tenants are looked
			// up again.
			tenants, err := a.directory.SignInTenants(ctx, email)
			if err == nil && len(tenants) == 0 {
				err = errors.New("the account belongs to a tenant, and the lookup lists none")
			}
			if err != nil {
				a.tenantServiceFailed(w, err, "failed to look up the tenants of identity %s again", identityID)
				return
			}
			if len(tenants) > 1 {
				a.selectTenant(w, r, loginChallenge)
				return
			}
			tenantID = tenants[0].ID
			a.setTenant(w, stateCookie, loginChallenge, tenantID)
		case err != nil:
			a.tenantServiceFailed(w, err, "failed to create the personal tenant of identity %s", identityID)
			return
		default:
			tenantID = personalTenantID
		}
	}

	sc, err := a.directory.SignInContext(ctx, tenantID, "", identityID)
	if err != nil {
		a.tenantServiceFailed(w, err, "failed to get the sign-in context of identity %s at tenant %s", identityID, tenantID)
		return
	}
	needed := decide(checkInput{
		context:        sc,
		session:        session,
		provenance:     provenance,
		reauthenticate: lr.NeedsFreshFirstFactor(session, a.now()) && !a.isFresh(r, loginChallenge, session),
	})
	a.logger.Debugf("the session of identity %s at tenant %s needs: %s", identityID, tenantID, needed)
	if needed != needNothing {
		a.askFor(w, r, needed, session, loginChallenge, lr, tenantID, sc)
		return
	}

	if !sc.Member {
		// A pending invitation or auto-join admits the address, and its
		// session passes the checks of the tenant: the account joins it.
		err := a.directory.JoinTenant(ctx, tenantID, identityID)
		switch {
		case errors.Is(err, errNotAdmitted):
			// The tenant stopped admitting the address since the check: a
			// refusal, not an outage.
			a.logger.Debugf("tenant %s does not admit identity %s any more: %v", tenantID, identityID, err)
			writeError(w, http.StatusForbidden, notAMemberError, notAMemberMessage)
			return
		case err != nil:
			a.tenantServiceFailed(w, err, "failed to join identity %s to tenant %s", identityID, tenantID)
			return
		}
	}
	a.accept(w, r, session, loginChallenge, tenantID)
}

// setTenant binds a tenant the user did not pick to the login challenge in
// the state cookie, and returns the cookie it wrote.
func (a *API) setTenant(w http.ResponseWriter, stateCookie cookies.FlowStateCookie, loginChallenge, tenantID string) cookies.FlowStateCookie {
	stateCookie = stateCookie.RenewForChallenge(loginChallenge)
	stateCookie.TenantID = tenantID
	if err := a.state.SetStateCookie(w, stateCookie); err != nil {
		a.logger.Errorf("failed to set state cookie: %v", err)
	}
	return stateCookie
}

func (a *API) accept(w http.ResponseWriter, r *http.Request, session *kClient.Session, loginChallenge, tenantID string) {
	redirectTo, acceptCookies, err := a.kratos.AcceptLoginRequest(r.Context(), session, loginChallenge, tenantID)
	if err != nil {
		a.logger.Errorf("failed to accept login request: %v", err)
		http.Error(w, "failed to accept login request", http.StatusInternalServerError)
		return
	}
	a.state.ClearStateCookie(w)
	a.store.ClearFresh(w)
	setCookies(w, acceptCookies)
	redirectResponse(w, r, newRedirect(redirectTo.GetRedirectTo()))
}

// askFor asks for what the session lacks.
func (a *API) askFor(w http.ResponseWriter, r *http.Request, needed need, session *kClient.Session, loginChallenge string, lr *LoginRequest, tenantID string, sc *SignInContext) {
	switch needed {
	case needRefused:
		writeError(w, http.StatusForbidden, notAMemberError, notAMemberMessage)
	case needCompanySignIn:
		a.askCompanySignIn(w, r, session, loginChallenge, lr, tenantID, sc)
	case needFreshFirstFactor:
		if sc.CompanyOnly() {
			a.askCompanySignIn(w, r, session, loginChallenge, lr, tenantID, sc)
			return
		}
		a.freshFlow(w, r, loginChallenge, tenantID, emailFromSession(session))
	case needMFA:
		a.askMFA(w, r, session, loginChallenge)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// askCompanySignIn restarts the company sign-in of the tenant: straight to
// it when there is one, otherwise a new flow offering them.
func (a *API) askCompanySignIn(w http.ResponseWriter, r *http.Request, session *kClient.Session, loginChallenge string, lr *LoginRequest, tenantID string, sc *SignInContext) {
	email := emailFromSession(session)

	options, err := a.sso.Options(r.Context(), sc.ConnectionIDs)
	if err != nil {
		a.logger.Errorf("failed to list the company sign-ins of tenant %s: %v", tenantID, err)
	}
	if err != nil || len(options) == 0 {
		switch {
		case !sc.CompanyOnly():
			a.freshFlow(w, r, loginChallenge, tenantID, email)
		case err != nil:
			ssoUnavailable(w)
		default:
			ssoNotApplicable(w)
		}
		return
	}

	flow, flowCookies, ok := a.newLoginFlow(w, r, loginChallenge)
	if !ok {
		return
	}
	if len(options) > 1 {
		a.advanceToChoices(w, r, flow, flowCookies, loginChallenge, tenantID, email)
		return
	}

	// The flow carries the address, as when the user enters it: the page the
	// company sign-in comes back to, a refusal included, reads it.
	kratosCookies, _ := a.identify(r, flow, flowCookies, email)
	flowCookies = mergeCookies(flowCookies, kratosCookies)
	to, ok := a.startAttempt(w, r, flow, flowCookies, attempt{
		loginChallenge: loginChallenge,
		tenantID:       tenantID,
		email:          email,
		connectionID:   options[0].ConnectionID,
		reauthenticate: lr.Reauthenticate(session, a.now()),
		prior:          session.GetId(),
	})
	if ok {
		redirectResponse(w, r, newRedirect(to).withLabel(options[0].Label))
	}
}

// freshFlow asks for another first factor on a flow created without the
// session cookie, for the address of the session and the tenant, where what
// the tenant offers is shown.
func (a *API) freshFlow(w http.ResponseWriter, r *http.Request, loginChallenge, tenantID, email string) {
	flow, flowCookies, ok := a.newLoginFlow(w, r, loginChallenge)
	if !ok {
		return
	}
	a.advanceToChoices(w, r, flow, flowCookies, loginChallenge, tenantID, email)
}

func (a *API) newLoginFlow(w http.ResponseWriter, r *http.Request, loginChallenge string) (*kClient.LoginFlow, []*http.Cookie, bool) {
	flow, flowCookies, err := a.kratos.CreateBrowserLoginFlow(r.Context(), "", a.loginURL(loginChallenge), loginChallenge, false, withoutSession(r.Cookies()))
	if err != nil {
		kratos.WriteGetFlowError(w, a.logger, "login", err, "failed to start sign-in")
		return nil, nil, false
	}
	return flow, flowCookies, true
}

// advanceToChoices binds the tenant and submits the identifier step for the
// address of the session, so that the flow the browser lands on shows what
// the tenant offers without asking the email again.
func (a *API) advanceToChoices(w http.ResponseWriter, r *http.Request, flow *kClient.LoginFlow, flowCookies []*http.Cookie, loginChallenge, tenantID, email string) {
	if tenantID != "" {
		if err := a.bindTenant(w, r, loginChallenge, tenantID, false); err != nil {
			a.logger.Errorf("failed to set state cookie: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	to := a.loginFlowURL(flow.Id)
	kratosCookies, next := a.identify(r, flow, flowCookies, email)
	if next != "" {
		to = next
	}
	setCookies(w, kratosCookies)
	setCookies(w, flowCookies)
	redirectResponse(w, r, newRedirect(to))
}

// identify submits the address at the identifier step, when the flow has
// one, so that the flow carries it: the login page a company sign-in comes
// back to reads the address from the flow. It returns the cookies of Kratos
// and where Kratos would go next. An account Kratos refuses at that step
// keeps the flow as it is, which still shows the company sign-ins of the
// tenant.
func (a *API) identify(r *http.Request, flow *kClient.LoginFlow, flowCookies []*http.Cookie, email string) ([]*http.Cookie, string) {
	if !hasIdentifierFirst(flow) || email == "" {
		return nil, ""
	}
	body := kClient.NewUpdateLoginFlowWithIdentifierFirstMethod(email, methodIdentifierFirst)
	body.SetCsrfToken(flowInputValue(flow.Ui.Nodes, "csrf_token"))

	redirectTo, kratosCookies, err := a.kratos.UpdateIdentifierFirstLoginFlow(r.Context(), flow.Id, *body, mergeCookies(withoutSession(r.Cookies()), flowCookies))
	if err != nil {
		a.logger.Errorf("failed to submit the identifier step of login flow %s: %v", flow.Id, err)
	}
	if redirectTo == nil {
		return kratosCookies, ""
	}
	return kratosCookies, redirectTo.GetRedirectTo()
}

// selectTenant sends a signed-in user with several tenants to the tenant
// page: the email is not asked again.
func (a *API) selectTenant(w http.ResponseWriter, r *http.Request, loginChallenge string) {
	if err := a.clearTenant(w, true); err != nil {
		a.logger.Errorf("failed to set state cookie: %v", err)
	}
	to := a.pageURL("/ui/select_tenant", url.Values{"login_challenge": {loginChallenge}})
	redirectResponse(w, r, newRedirect(to).withError(tenantSelectionRequired))
}

// verifyAddress starts the verification of the address of the account, and
// sends the browser to enter the code. Kratos returns to the login request
// once the address is verified.
func (a *API) verifyAddress(w http.ResponseWriter, r *http.Request, session *kClient.Session, loginChallenge string) {
	flowID, kratosCookies, err := a.verification.Start(r.Context(), a.loginURL(loginChallenge), emailFromSession(session), r.Cookies())
	if err != nil {
		a.logger.Errorf("failed to start the verification of the address: %v", err)
		http.Error(w, "failed to start verification", http.StatusInternalServerError)
		return
	}
	setCookies(w, kratosCookies)
	to := a.pageURL("/ui/verification", url.Values{"flow": {flowID}})
	redirectResponse(w, r, newRedirect(to).withError(kratos.VERIFICATION_REQUIRED))
}

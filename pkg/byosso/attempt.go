// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"errors"
	"net/http"

	kClient "github.com/ory/kratos-client-go/v25"
	"go.opentelemetry.io/otel/codes"

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

// attempt is one company sign-in to start.
type attempt struct {
	loginChallenge string
	tenantID       string
	email          string
	connectionID   string
	reauthenticate bool
	// tenantChosen is set when the user has just picked the tenant from
	// several.
	tenantChosen bool
	// join is set for the company sign-in of a registration: with no Hydra
	// login request, completePath makes the account a member of the tenant.
	join bool
	// prior is the ID of the Kratos session of the browser, "" when it has
	// none.
	prior string
	// returnTo is where completePath goes on to, for an attempt with no
	// Hydra login request.
	returnTo string
	// linkTicket is the attempt Kratos is adding to the account, when this
	// attempt is a company sign-in the account already has, started from the
	// account-linking page.
	linkTicket string
}

// completion is a sign-in attempt sso-service has just confirmed.
type completion struct {
	loginChallengeHash string
	tenantID           string
	returnTo           string
	join               bool
}

// freshKey is the context key of a fresh mark set in this very request.
type freshKey struct{}

// startAttempt gets a ticket from sso-service and submits the company
// sign-in provider on flow with the ticket as login_hint. It writes the
// sign-in cookie and expires the Kratos session cookie: the IdP returns to
// Kratos directly, and must not carry a session there. flowCookies are the
// cookies of Kratos when flow was created in this request. It returns where
// the browser goes, or false when it answered the request with an error.
func (a *API) startAttempt(w http.ResponseWriter, r *http.Request, flow *kClient.LoginFlow, flowCookies []*http.Cookie, p attempt) (string, bool) {
	ctx := r.Context()

	ticket, err := a.sso.StartAttempt(ctx, AttemptRequest{
		TenantID:       p.tenantID,
		Email:          p.email,
		ConnectionID:   p.connectionID,
		Reauthenticate: p.reauthenticate,
	})
	switch {
	case errors.Is(err, errNotApplicable):
		a.logger.Debugf("connection %s is not applicable: %v", p.connectionID, err)
		ssoNotApplicable(w)
		return "", false
	case err != nil:
		a.logger.Errorf("failed to start a company sign-in with connection %s: %v", p.connectionID, err)
		ssoUnavailable(w)
		return "", false
	}

	body := kClient.NewUpdateLoginFlowWithOidcMethod(methodOIDC, providerID)
	if csrfToken := flowInputValue(flow.Ui.Nodes, "csrf_token"); csrfToken != "" {
		body.SetCsrfToken(csrfToken)
	}
	body.SetUpstreamParameters(map[string]interface{}{"login_hint": ticket})

	redirectTo, _, kratosCookies, err := a.kratos.UpdateLoginFlow(
		ctx, flow.Id, kClient.UpdateLoginFlowWithOidcMethodAsUpdateLoginFlowBody(body), mergeCookies(withoutSession(r.Cookies()), flowCookies),
	)
	if err != nil {
		kratos.WriteUpdateFlowError(w, a.logger, "login", err)
		return "", false
	}
	if redirectTo == nil || redirectTo.GetRedirectTo() == "" {
		a.logger.Errorf("no redirect to the company sign-in on login flow %s", flow.Id)
		http.Error(w, "failed to start company sign-in", http.StatusInternalServerError)
		return "", false
	}

	signIn := SignInCookie{
		Ticket:         ticket,
		PriorSessionID: p.prior,
		ReturnTo:       p.returnTo,
		TenantID:       p.tenantID,
		ConnectionID:   p.connectionID,
		LinkTicket:     p.linkTicket,
		Join:           p.join,
	}
	if p.loginChallenge != "" {
		signIn.LoginChallengeHash = cookies.ChallengeHash(p.loginChallenge)
		if err := a.bindTenant(w, r, p.loginChallenge, p.tenantID, p.tenantChosen); err != nil {
			a.logger.Errorf("failed to set state cookie: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return "", false
		}
		if err := a.store.SetFresh(w, FreshCookie{LoginChallengeHash: signIn.LoginChallengeHash, PriorSessionID: p.prior, StartedAt: flow.GetIssuedAt()}); err != nil {
			a.logger.Errorf("failed to set fresh cookie: %v", err)
		}
	}
	if err := a.store.SetSignIn(w, signIn); err != nil {
		a.logger.Errorf("failed to set sign-in cookie: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return "", false
	}

	a.logger.Debugf("started connection %s of tenant %s on login flow %s", p.connectionID, p.tenantID, flow.Id)
	setCookies(w, flowCookies)
	// The session cookie is expired once, here, whatever the service returns.
	setCookies(w, withoutSession(kratosCookies))
	kratos.UnsetSessionCookie(w)
	return redirectTo.GetRedirectTo(), true
}

// completePending records which company sign-in the session came from, when
// one has just come back: the session is not the one the browser had before,
// it includes a company sign-in, and sso-service confirms that the attempt
// ended at this account and in this browser, from the receipt it left there.
// When the account-linking page sent the user through a company sign-in the
// account already had, the attempt being linked is confirmed first: its
// connection is the one signed in for. It returns the connection behind the
// session, "" when it is not known, and the attempt it has just completed,
// if any.
func (a *API) completePending(w http.ResponseWriter, r *http.Request, session *kClient.Session) (string, *completion, error) {
	ctx, span := a.tracer.Start(r.Context(), "byosso.API.completePending")
	defer span.End()

	signIn, err := a.store.GetSignIn(r)
	if err != nil {
		a.logger.Errorf("failed to read sign-in cookie: %v", err)
		a.store.ClearSignIn(w)
		signIn = SignInCookie{}
	}

	if signIn.Ticket != "" && session.GetId() != signIn.PriorSessionID {
		var tickets []string
		switch {
		case !includesCompanySignIn(session):
			a.logger.Debugf("session %s includes no company sign-in", session.GetId())
		case signIn.LinkTicket != "":
			tickets = []string{signIn.LinkTicket, signIn.Ticket}
		default:
			tickets = []string{signIn.Ticket}
		}

		for _, ticket := range tickets {
			// With no receipt sso-service is still asked: it refuses, and
			// counts the refusal.
			completed, err := a.sso.CompleteAttempt(ctx, ticket, session.Identity.GetId(), receiptFor(r, ticket))
			if errors.Is(err, errNotApplicable) {
				continue
			}
			if err == nil {
				err = a.store.SetProvenance(w, ProvenanceCookie{SessionID: session.GetId(), ConnectionID: completed.ConnectionID}, session)
			}
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return "", nil, err
			}

			a.clearSignIn(w, signIn)
			span.SetStatus(codes.Ok, "")
			return completed.ConnectionID, &completion{
				loginChallengeHash: signIn.LoginChallengeHash,
				tenantID:           completed.TenantID,
				returnTo:           signIn.ReturnTo,
				join:               signIn.Join,
			}, nil
		}

		// No attempt ended at this account.
		a.clearSignIn(w, signIn)
	}

	span.SetStatus(codes.Ok, "")
	provenance, err := a.store.GetProvenance(r)
	if err != nil {
		a.logger.Errorf("failed to read provenance cookie: %v", err)
		return "", nil, nil
	}
	return provenanceFor(provenance, session), nil, nil
}

// clearSignIn clears the sign-in cookie and the receipts of its tickets.
func (a *API) clearSignIn(w http.ResponseWriter, signIn SignInCookie) {
	a.store.ClearSignIn(w)
	for _, ticket := range []string{signIn.Ticket, signIn.LinkTicket} {
		if ticket != "" {
			clearReceipt(w, ticket)
		}
	}
}

// markFresh records that a first factor starts on the login flow for its
// login challenge: the session it produces is not the one the browser has,
// and its first factor completes after the flow was issued. It returns the
// request carrying the mark, as the same request may go on to accept the
// login (a password completes the first factor at once).
func (a *API) markFresh(w http.ResponseWriter, r *http.Request, flow *kClient.LoginFlow) *http.Request {
	mark := FreshCookie{LoginChallengeHash: cookies.ChallengeHash(flow.GetOauth2LoginChallenge()), PriorSessionID: a.currentSession(r).GetId(), StartedAt: flow.GetIssuedAt()}
	if err := a.store.SetFresh(w, mark); err != nil {
		a.logger.Errorf("failed to set fresh cookie: %v", err)
	}
	return r.WithContext(context.WithValue(r.Context(), freshKey{}, mark))
}

// isFresh reports whether the first factor of the session was started for
// the login challenge.
func (a *API) isFresh(r *http.Request, loginChallenge string, session *kClient.Session) bool {
	hash := cookies.ChallengeHash(loginChallenge)
	if mark, ok := r.Context().Value(freshKey{}).(FreshCookie); ok && fresh(mark, hash, session) {
		return true
	}
	mark, err := a.store.GetFresh(r)
	if err != nil {
		return false
	}
	return fresh(mark, hash, session)
}

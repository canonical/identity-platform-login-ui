// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"net/http"

	kClient "github.com/ory/kratos-client-go/v25"
	"go.opentelemetry.io/otel/codes"
	grpccodes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

// admittedTenant is a tenant whose company sign-in registers an address
// that creates an account.
type admittedTenant struct {
	tenant  SignInTenant
	context *SignInContext
}

// admittingTenants returns the tenants whose company sign-in an address
// creating an account is sent to: auto-join admits it, or a pending
// invitation of an address with no account at a tenant that requires company
// sign-in.
func (a *API) admittingTenants(ctx context.Context, email string) ([]admittedTenant, error) {
	ctx, span := a.tracer.Start(ctx, "byosso.API.admittingTenants")
	defer span.End()

	tenants, err := a.directory.SignInTenants(ctx, email)
	if status.Code(err) == grpccodes.InvalidArgument {
		// Not an address: no tenant admits it, and Kratos says what is wrong
		// with it.
		span.SetStatus(codes.Ok, "")
		return nil, nil
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	var admitted []admittedTenant
	for _, t := range tenants {
		if !t.AutoJoinCandidate && !t.Invited {
			continue
		}
		sc, err := a.directory.SignInContext(ctx, t.ID, email, "")
		if err != nil {
			// A failure says nothing about the tenant: it is not left out as
			// one that does not admit the address.
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		if sc.Member {
			continue
		}
		invited := sc.InvitationAdmits && !sc.AccountExists && sc.Enforcement == EnforcementRequired
		if sc.AutoJoinAdmits || invited {
			admitted = append(admitted, admittedTenant{tenant: t, context: sc})
		}
	}

	span.SetStatus(codes.Ok, "")
	return admitted, nil
}

// registrationFor returns the registration cookie of the login flow, if any.
func (a *API) registrationFor(r *http.Request, flowID string) (RegistrationCookie, bool) {
	registration, err := a.store.GetRegistration(r)
	if err != nil || registration.FlowID == "" || registration.FlowID != flowID {
		return RegistrationCookie{}, false
	}
	return registration, true
}

// registrationSignIn runs the company sign-in of the tenant from the
// registration page: straight to it when there is one, otherwise on a login
// flow offering them.
func (a *API) registrationSignIn(w http.ResponseWriter, r *http.Request, flowID, email string, admitted admittedTenant) {
	tenantID := admitted.tenant.ID

	options, err := a.sso.Options(r.Context(), admitted.context.ConnectionIDs)
	if err != nil {
		a.logger.Errorf("failed to list the company sign-ins of tenant %s: %v", tenantID, err)
		ssoUnavailable(w)
		return
	}
	if len(options) == 0 {
		ssoNotApplicable(w)
		return
	}

	flow, flowCookies, loginChallenge, returnTo, ok := a.registrationLoginFlow(w, r, flowID)
	if !ok {
		return
	}

	if len(options) == 1 {
		to, ok := a.startAttempt(w, r, flow, flowCookies, attempt{
			loginChallenge: loginChallenge,
			tenantID:       tenantID,
			email:          email,
			connectionID:   options[0].ConnectionID,
			join:           true,
			prior:          a.currentSession(r).GetId(),
			returnTo:       returnTo,
		})
		if !ok {
			return
		}
		writeJSON(w, http.StatusOK, newRedirect(to).withLabel(options[0].Label).withContinue())
		return
	}

	if err := a.store.SetRegistration(w, RegistrationCookie{FlowID: flow.Id, TenantID: tenantID, Email: email, ReturnTo: returnTo}); err != nil {
		a.logger.Errorf("failed to set registration cookie: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if loginChallenge != "" {
		if err := a.bindTenant(w, r, loginChallenge, tenantID, false); err != nil {
			a.logger.Errorf("failed to set state cookie: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}
	setCookies(w, flowCookies)
	writeJSON(w, http.StatusOK, newRedirect(a.loginFlowURL(flow.Id)).withContinue())
}

// registrationChoice sends a registration several tenants admit to the list
// of those tenants: a login flow whose registration cookie has the address
// and no tenant yet.
func (a *API) registrationChoice(w http.ResponseWriter, r *http.Request, flowID, email string) {
	flow, flowCookies, _, returnTo, ok := a.registrationLoginFlow(w, r, flowID)
	if !ok {
		return
	}
	if err := a.store.SetRegistration(w, RegistrationCookie{FlowID: flow.Id, Email: email, ReturnTo: returnTo}); err != nil {
		a.logger.Errorf("failed to set registration cookie: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	setCookies(w, flowCookies)
	writeJSON(w, http.StatusOK, newRedirect(a.loginFlowURL(flow.Id)).withContinue())
}

// registrationLoginFlow creates the login flow the company sign-in of a
// registration runs on. It returns to the login request of the registration
// when it has one, otherwise to completePath. It also returns the login
// challenge and the return_to of the registration, and false when it answered
// the request.
func (a *API) registrationLoginFlow(w http.ResponseWriter, r *http.Request, flowID string) (*kClient.LoginFlow, []*http.Cookie, string, string, bool) {
	loginChallenge, registrationReturnTo := a.registrationTarget(r, flowID)
	returnTo := a.completeURL(registrationReturnTo)
	if loginChallenge != "" {
		returnTo = a.loginURL(loginChallenge)
	}

	flow, flowCookies, err := a.kratos.CreateBrowserLoginFlow(r.Context(), "", returnTo, "", false, withoutSession(r.Cookies()))
	if err != nil {
		kratos.WriteGetFlowError(w, a.logger, "login", err, "failed to start company sign-in")
		return nil, nil, "", "", false
	}
	return flow, flowCookies, loginChallenge, registrationReturnTo, true
}

// registrationTarget returns the login challenge of the registration flow,
// read from its return_to when the flow does not carry it, and its return_to.
func (a *API) registrationTarget(r *http.Request, flowID string) (string, string) {
	flow, _, err := a.kratos.GetRegistrationFlow(r.Context(), flowID, r.Cookies())
	if err != nil {
		a.logger.Errorf("failed to get registration flow %s: %v", flowID, err)
		return "", ""
	}
	returnTo := flow.GetReturnTo()
	loginChallenge := flow.GetOauth2LoginChallenge()
	if loginChallenge == "" {
		loginChallenge = challengeFromReturnTo(returnTo)
	}
	return loginChallenge, returnTo
}

// hydrateRegistrationChoice lays out the tenants the address of a
// registration may join, when none is picked yet.
func (a *API) hydrateRegistrationChoice(r *http.Request, flow *kClient.LoginFlow, registration RegistrationCookie) *kClient.LoginFlow {
	flow = withoutIdentifierStep(withoutFirstFactors(flow))

	admitted, err := a.admittingTenants(r.Context(), registration.Email)
	if err != nil {
		a.logger.Errorf("failed to look up the tenants admitting a registration: %v", err)
		return withMessage(flow, lookupFailedID, lookupFailedText, "error")
	}
	if len(admitted) == 0 {
		return withMessage(flow, noTenantAdmitsID, noTenantAdmitsText, "error")
	}
	tenants := make([]SignInTenant, 0, len(admitted))
	for _, t := range admitted {
		tenants = append(tenants, t.tenant)
	}
	return withTenants(flow, tenants)
}

// chooseRegistrationTenant takes the pick of a registration from that list:
// one of the tenants admitting its address. The browser goes straight to the
// company sign-in of the tenant when it is the only one, otherwise the same
// flow offers them.
func (a *API) chooseRegistrationTenant(w http.ResponseWriter, r *http.Request, flow *kClient.LoginFlow, registration RegistrationCookie, tenantID string) {
	admitted, err := a.admittingTenants(r.Context(), registration.Email)
	if err != nil {
		a.tenantServiceFailed(w, err, "failed to look up the tenants admitting a registration")
		return
	}
	var sc *SignInContext
	for _, t := range admitted {
		if t.tenant.ID == tenantID {
			sc = t.context
		}
	}
	if sc == nil {
		a.logger.Debugf("tenant %s does not admit the address of the registration", tenantID)
		http.Error(w, "tenant not allowed", http.StatusForbidden)
		return
	}

	options, err := a.sso.Options(r.Context(), sc.ConnectionIDs)
	if err != nil {
		a.logger.Errorf("failed to list the company sign-ins of tenant %s: %v", tenantID, err)
		ssoUnavailable(w)
		return
	}
	if len(options) == 0 {
		ssoNotApplicable(w)
		return
	}

	registration.TenantID = tenantID
	if len(options) == 1 {
		a.pickFor(w, r, flow, tenantID, registration.Email, &registration, options[0].ConnectionID)
		return
	}
	if err := a.store.SetRegistration(w, registration); err != nil {
		a.logger.Errorf("failed to set registration cookie: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	redirectResponse(w, r, newRedirect(a.loginFlowURL(flow.Id)))
}

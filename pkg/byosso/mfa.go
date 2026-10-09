// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"net/http"
	"net/url"

	kClient "github.com/ory/kratos-client-go/v25"
	"go.opentelemetry.io/otel/codes"

	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

// askMFA sends the session to the MFA of the tenant, which must be done in
// the session before its Hydra login is accepted. A user with no second
// factor is sent to set one up, trusting the first factor.
func (a *API) askMFA(w http.ResponseWriter, r *http.Request, session *kClient.Session, loginChallenge string) {
	identityID := session.Identity.GetId()
	returnTo := a.loginURL(loginChallenge)

	enrolled, err := a.hasSecondFactor(r.Context(), identityID)
	if err != nil {
		a.logger.Errorf("failed to check MFA status: %v", err)
		http.Error(w, "failed to check MFA status", http.StatusInternalServerError)
		return
	}
	if !enrolled {
		to := a.pageURL("/ui/setup_secure", url.Values{"return_to": {returnTo}})
		redirectResponse(w, r, newRedirect(to).withError(kratos.TOTP_REGISTRATION_REQUIRED))
		return
	}

	// A session at aal2 by a method that does not count as MFA here: Kratos
	// needs refresh, or it answers that the session is already there.
	q := url.Values{"aal": {"aal2"}, "return_to": {returnTo}}
	if session.GetAuthenticatorAssuranceLevel() == kClient.AUTHENTICATORASSURANCELEVEL_AAL2 {
		q.Set("refresh", "true")
	}
	redirectResponse(w, r, newRedirect(a.pageURL("/ui/login", q)).withError(aal2RequiredError))
}

// hasSecondFactor reports whether the account holds a second factor: an
// authenticator app, a security key, or recovery codes, which meet MFA as
// well. The user then steps up with it rather than setting up a new
// authenticator on the first factor alone.
func (a *API) hasSecondFactor(ctx context.Context, identityID string) (bool, error) {
	ctx, span := a.tracer.Start(ctx, "byosso.API.hasSecondFactor")
	defer span.End()

	checks := []func(context.Context, string) (bool, error){
		a.kratos.HasWebAuthnAvailable,
		a.kratos.HasTOTPAvailable,
		a.identities.HasRecoveryCodes,
	}
	for _, has := range checks {
		ok, err := has(ctx, identityID)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return false, err
		}
		if ok {
			span.SetStatus(codes.Ok, "")
			return true, nil
		}
	}

	span.SetStatus(codes.Ok, "")
	return false, nil
}

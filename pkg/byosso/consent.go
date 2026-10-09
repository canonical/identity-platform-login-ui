// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"

	hClient "github.com/ory/hydra-client-go/v26"
	kClient "github.com/ory/kratos-client-go/v25"
)

// GateConsent refuses the consent of a login login-ui did not accept, or
// whose subject is not the account of the session, through the rejection of
// Hydra (login_required). It calls neither tenant-service nor sso-service.
// It returns where the browser goes when refused, "" to go on.
func (a *API) GateConsent(ctx context.Context, session *kClient.Session, consent *hClient.OAuth2ConsentRequest) (string, error) {
	reason := consentRefusal(session, consent)
	if reason == "" {
		return "", nil
	}
	ctx, cancel := context.WithTimeout(ctx, requestBudget)
	defer cancel()

	a.logger.Warnf("refused the consent of subject %s: %s", consent.GetSubject(), reason)
	return a.hydra.RejectConsent(ctx, consent.GetChallenge(), reason)
}

// consentRefusal returns why the consent is refused, "" when it is not. A
// login login-ui accepted has the tenant it was checked for in its context.
func consentRefusal(session *kClient.Session, consent *hClient.OAuth2ConsentRequest) string {
	if session == nil || session.Identity == nil || consent.GetSubject() != session.Identity.GetId() {
		return "this sign-in belongs to another account"
	}
	loginContext, _ := consent.GetContext().(map[string]interface{})
	if tenantID, _ := loginContext["tenant_id"].(string); tenantID == "" {
		return "this sign-in did not go through the portal's sign-in checks"
	}
	return ""
}

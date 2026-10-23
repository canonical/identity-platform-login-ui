// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	kClient "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

// withLinks appends the company sign-ins of the account to the settings
// flow, each with its unlink button.
func (a *API) withLinks(ctx context.Context, flow *kClient.SettingsFlow, identityID string) *kClient.SettingsFlow {
	links, err := a.sso.Links(ctx, identityID)
	if err != nil {
		a.logger.Errorf("failed to list the company sign-ins of identity %s: %v", identityID, err)
		return flow
	}
	for _, l := range links {
		flow.Ui.Nodes = append(flow.Ui.Nodes, linkNode(l), submitNode(ssoNodeGroup, ssoUnlinkField, l.ConnectionID, fmt.Sprintf("Unlink %s", l.Label), unlinkProvider))
	}
	return flow
}

// unlink removes a company sign-in of the account and answers with the
// settings flow; a refusal is an error message on it. As Kratos does for its
// own unlink, it requires that the settings flow loads and that the session
// is within the privileged session age.
func (a *API) unlink(w http.ResponseWriter, r *http.Request, flowID, connectionID string) {
	ctx := r.Context()

	session, _, err := a.kratos.CheckSession(ctx, r.Cookies())
	if err != nil || session == nil || session.Identity == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	identityID := session.Identity.GetId()

	_, redirectTo, err := a.kratos.GetSettingsFlow(ctx, flowID, r.Cookies())
	if err != nil {
		kratos.WriteGetFlowError(w, a.logger, "settings", err, "failed to load settings flow")
		return
	}
	if redirectTo != nil && redirectTo.HasRedirectTo() {
		redirectResponse(w, r, &redirect{BrowserLocationChangeRequired: *redirectTo})
		return
	}
	if !privileged(session, a.privilegedSessionMaxAge, a.now()) {
		id := kratos.SESSION_REFRESH_REQUIRED
		returnTo := a.absoluteURL("/ui/manage_connected_accounts")
		resp := newRedirect(a.pageURL("/ui/login", url.Values{"refresh": {"true"}, "return_to": {returnTo}}))
		resp.Error = &kClient.GenericError{Id: &id, Message: "sign in again to remove a company sign-in"}
		redirectResponse(w, r, resp)
		return
	}

	var refusal string
	switch err := a.sso.DeleteLink(ctx, connectionID, identityID); {
	case err == nil:
		a.logger.Debugf("identity %s removed connection %s", identityID, connectionID)
	case errors.Is(err, errLastCredential):
		refusal = "You cannot remove your only way to sign in. Set a password first."
	case errors.Is(err, errNoLink), errors.Is(err, errNotApplicable):
		refusal = "This account is not linked to that sign-in method."
	default:
		a.logger.Errorf("failed to remove connection %s of identity %s: %v", connectionID, identityID, err)
		http.Error(w, "failed to unlink", http.StatusInternalServerError)
		return
	}

	flow, _, err := a.kratos.GetSettingsFlow(ctx, flowID, r.Cookies())
	if err != nil {
		kratos.WriteGetFlowError(w, a.logger, "settings", err, "failed to load settings flow")
		return
	}
	if flow == nil {
		a.logger.Errorf("no settings flow %s after removing a company sign-in", flowID)
		http.Error(w, "failed to load settings flow", http.StatusInternalServerError)
		return
	}
	flow = a.withLinks(ctx, flow, identityID)
	if refusal != "" {
		flow.Ui.Messages = append(flow.Ui.Messages, *kClient.NewUiText(validationFailure, refusal, "error"))
	}
	writeJSON(w, http.StatusOK, flow)
}

// hydrateRefresh adds the company sign-ins of the account to the refresh
// login of Kratos. Kratos offers the company sign-in provider there to an
// account linked through it, but that node is hidden: only login-ui submits
// the provider, with a ticket. Without these, a user whose only sign-ins are
// company sign-ins could not sign in again for a privileged settings change.
func (a *API) hydrateRefresh(r *http.Request, flow *kClient.LoginFlow) *kClient.LoginFlow {
	session := a.currentSession(r)
	if session == nil || session.Identity == nil {
		return flow
	}
	links, err := a.sso.Links(r.Context(), session.Identity.GetId())
	if err != nil {
		a.logger.Errorf("failed to list the company sign-ins of identity %s: %v", session.Identity.GetId(), err)
		return withMessage(flow, companyUnavailableID, companyUnavailableText, "info")
	}
	for _, l := range links {
		flow.Ui.Nodes = append(flow.Ui.Nodes, companySignInNode(ssoReauthField, l.ConnectionID, l.Label))
	}
	return flow
}

// refreshPick signs the user of the session in again through a company
// sign-in the account already has, picked on the refresh login, asking the
// IdP for a fresh login. It runs on a new login flow that returns through
// completePath: the refresh flow returns straight to its return_to, which
// would skip CompleteAttempt. The OIDC login of Kratos makes a new session,
// authenticated now, and sso-service lets the sign-in through only to the
// account with the address of the session. Only a connection the account is
// linked to is started.
func (a *API) refreshPick(w http.ResponseWriter, r *http.Request, flow *kClient.LoginFlow, connectionID string) {
	ctx := r.Context()

	if !isRefresh(flow) {
		http.Error(w, "not a refresh flow", http.StatusBadRequest)
		return
	}
	session := a.currentSession(r)
	email := emailFromSession(session)
	if email == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	links, err := a.sso.Links(ctx, session.Identity.GetId())
	if err != nil {
		a.logger.Errorf("failed to list the company sign-ins of identity %s: %v", session.Identity.GetId(), err)
		ssoUnavailable(w)
		return
	}
	link := findLink(links, connectionID)
	if link == nil {
		a.logger.Debugf("connection %s is not a company sign-in of identity %s", connectionID, session.Identity.GetId())
		http.Error(w, "Provider not allowed", http.StatusForbidden)
		return
	}

	returnTo := flow.GetReturnTo()
	newFlow, flowCookies, err := a.kratos.CreateBrowserLoginFlow(ctx, "", a.completeURL(returnTo), "", false, withoutSession(r.Cookies()))
	if err != nil {
		kratos.WriteGetFlowError(w, a.logger, "login", err, "failed to start company sign-in")
		return
	}
	to, ok := a.startAttempt(w, r, newFlow, flowCookies, attempt{
		tenantID:       link.TenantID,
		email:          email,
		connectionID:   connectionID,
		reauthenticate: true,
		prior:          session.GetId(),
		returnTo:       returnTo,
	})
	if !ok {
		return
	}
	redirectResponse(w, r, newRedirect(to).withLabel(link.Label))
}

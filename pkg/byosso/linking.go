// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"net/http"

	kClient "github.com/ory/kratos-client-go/v25"
	"go.opentelemetry.io/otel/codes"
)

// hydrateAccountLinking lays out the account-linking page of Kratos: the
// sign-ins the account has, which Kratos lists (portal password, public
// sign-ins, passkey), plus the other company sign-ins of the account, which
// Kratos hides there because they share the provider used to get there. All
// of them are offered, even when the tenant requires company sign-in: this
// sign-in only proves the account before Kratos links the company sign-in,
// which then counts as the first factor of the session. With none of them,
// the recovery prompt: the recovery of Kratos gives the account a way in
// first.
func (a *API) hydrateAccountLinking(r *http.Request, flow *kClient.LoginFlow) *kClient.LoginFlow {
	links, err := a.accountLinks(r.Context(), flowInputValue(flow.Ui.Nodes, "identifier"))
	if err != nil {
		a.logger.Errorf("failed to list the company sign-ins of the account being linked: %v", err)
		return withMessage(flow, companyUnavailableID, companyUnavailableText, "info")
	}

	// The connection being linked is left out: its IdP answers with the
	// subject that has no link (e.g. a recreated IdP account), so it cannot
	// prove the account.
	linking := ""
	if pending, err := a.store.GetSignIn(r); err == nil {
		linking = pending.ConnectionID
	}
	offered := 0
	for _, l := range links {
		if l.ConnectionID == linking {
			continue
		}
		flow.Ui.Nodes = append(flow.Ui.Nodes, companySignInNode(ssoLinkField, l.ConnectionID, l.Label))
		offered++
	}
	if offered == 0 && !hasOtherFirstFactor(flow) {
		flow = a.withRecoverPrompt(flow)
	}
	return flow
}

// accountLinks returns the company sign-ins of the account holding email.
func (a *API) accountLinks(ctx context.Context, email string) ([]Link, error) {
	ctx, span := a.tracer.Start(ctx, "byosso.API.accountLinks")
	defer span.End()

	identityID, err := a.identities.IdentityID(ctx, email)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if identityID == "" {
		span.SetStatus(codes.Ok, "")
		return nil, nil
	}

	links, err := a.sso.Links(ctx, identityID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return links, nil
}

// linkingPick starts a company sign-in the account already has from the
// account-linking page, on that flow: Kratos logs the user in through the
// existing link and then adds the pending one. The attempt being linked is
// kept in the sign-in cookie, as its connection is the one signed in for.
// Only a connection the account is linked to is started.
func (a *API) linkingPick(w http.ResponseWriter, r *http.Request, flow *kClient.LoginFlow, connectionID string) {
	if !isAccountLinking(flow) {
		http.Error(w, "not an account-linking flow", http.StatusBadRequest)
		return
	}

	email := flowInputValue(flow.Ui.Nodes, "identifier")
	links, err := a.accountLinks(r.Context(), email)
	if err != nil {
		a.logger.Errorf("failed to list the company sign-ins of the account being linked: %v", err)
		ssoUnavailable(w)
		return
	}
	link := findLink(links, connectionID)
	if link == nil {
		a.logger.Debugf("connection %s is not a company sign-in of the account being linked", connectionID)
		http.Error(w, "Provider not allowed", http.StatusForbidden)
		return
	}

	// The company sign-in being linked must be one this browser started, and
	// not this very connection.
	pending, err := a.store.GetSignIn(r)
	if err != nil || pending.Ticket == "" || pending.TenantID == "" || pending.ConnectionID == connectionID {
		http.Error(w, "company sign-in to link not found", http.StatusBadRequest)
		return
	}
	linkTicket := pending.LinkTicket
	if linkTicket == "" {
		linkTicket = pending.Ticket
	}

	to, ok := a.startAttempt(w, r, flow, nil, attempt{
		loginChallenge: flow.GetOauth2LoginChallenge(),
		tenantID:       pending.TenantID,
		email:          email,
		connectionID:   connectionID,
		join:           pending.Join,
		prior:          a.currentSession(r).GetId(),
		returnTo:       pending.ReturnTo,
		linkTicket:     linkTicket,
	})
	if !ok {
		return
	}
	redirectResponse(w, r, newRedirect(to).withLabel(link.Label))
}

// findLink returns the link to connectionID, or nil.
func findLink(links []Link, connectionID string) *Link {
	for i := range links {
		if links[i].ConnectionID == connectionID {
			return &links[i]
		}
	}
	return nil
}

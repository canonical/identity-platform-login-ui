// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"time"

	kClient "github.com/ory/kratos-client-go/v25"
)

// need is what a Kratos session lacks for a tenant.
type need int

const (
	// needNothing means the Hydra login can be accepted.
	needNothing need = iota
	// needRefused means the account is not a member, and nothing admits its
	// address.
	needRefused
	// needFreshFirstFactor asks for another first factor, from what the
	// tenant offers.
	needFreshFirstFactor
	// needCompanySignIn asks for one of the company sign-ins of the tenant.
	needCompanySignIn
	// needMFA asks for the MFA of the tenant, in this session.
	needMFA
)

func (n need) String() string {
	switch n {
	case needNothing:
		return "accept"
	case needRefused:
		return "refused"
	case needFreshFirstFactor:
		return "fresh_first_factor"
	case needCompanySignIn:
		return "company_sign_in"
	case needMFA:
		return "mfa"
	}
	return "unknown"
}

// checkInput is what the accept checks decide on.
type checkInput struct {
	context *SignInContext
	session *kClient.Session
	// provenance is the connection behind the first factor of the session,
	// "" when it is not known.
	provenance string
	// reauthenticate is set when the app asked for a fresh first factor and
	// that of the session is not.
	reauthenticate bool
}

// decide applies the checks of a tenant to a session in order: membership,
// re-authentication, the company sign-in, MFA. An address the tenant admits
// by a pending invitation or auto-join is checked as a member. A session that
// includes a company sign-in, as its first factor or as one Kratos added by
// account linking, is checked as one.
func decide(in checkInput) need {
	sc := in.context
	if !sc.Offered() {
		return needRefused
	}
	if in.reauthenticate {
		return needFreshFirstFactor
	}
	if includesCompanySignIn(in.session) {
		switch {
		case in.provenance == "" && len(sc.ConnectionIDs) > 0:
			return needCompanySignIn
		case !sc.Applies(in.provenance) && sc.Enforcement == EnforcementRequired:
			return needCompanySignIn
		case !sc.Applies(in.provenance):
			// The tenant has no active binding, is a personal tenant, or does
			// not use this connection.
			return needFreshFirstFactor
		}
	} else if sc.Enforcement == EnforcementRequired {
		return needCompanySignIn
	}
	if mfaNeeded(in.session, sc.MFARequirement) && !mfaDone(in.session) {
		return needMFA
	}
	return needNothing
}

// firstMethod returns the first authentication method of the session, or
// nil.
func firstMethod(session *kClient.Session) *kClient.SessionAuthenticationMethod {
	if session == nil || len(session.AuthenticationMethods) == 0 {
		return nil
	}
	return &session.AuthenticationMethods[0]
}

// includesCompanySignIn reports whether the session includes a company
// sign-in: its first factor, or one Kratos added when it linked the company
// sign-in to the account. Account linking appends it after the factor the
// user signed in with.
func includesCompanySignIn(session *kClient.Session) bool {
	if session == nil {
		return false
	}
	for _, m := range session.AuthenticationMethods {
		if m.GetMethod() == methodOIDC && m.GetProvider() == providerID {
			return true
		}
	}
	return false
}

// mfaNeeded reports whether the session needs MFA at a tenant with the given
// policy: company and public sign-ins need it only when the tenant requires
// it, every other first factor always does. A company sign-in added by
// account linking counts as the first factor.
func mfaNeeded(session *kClient.Session, policy MFARequirement) bool {
	f := firstMethod(session)
	if (f != nil && f.GetMethod() == methodOIDC) || includesCompanySignIn(session) {
		return policy == MFARequirementRequired
	}
	return true
}

// isSecondFactor reports whether method is a second factor Kratos offers:
// an authenticator app, a recovery code, or a security key where the platform
// enables WebAuthn as one.
func isSecondFactor(method string) bool {
	switch method {
	case methodTOTP, methodWebAuthn, methodLookup:
		return true
	}
	return false
}

// mfaDone reports whether the session performed MFA: an entry after the
// first factor, at aal2.
func mfaDone(session *kClient.Session) bool {
	if session == nil {
		return false
	}
	for i, m := range session.AuthenticationMethods {
		if i == 0 || m.GetAal() != kClient.AUTHENTICATORASSURANCELEVEL_AAL2 {
			continue
		}
		if isSecondFactor(m.GetMethod()) {
			return true
		}
	}
	return false
}

// provenanceFor returns the connection the cookie records for the session,
// or "".
func provenanceFor(p ProvenanceCookie, session *kClient.Session) string {
	if session == nil || p.SessionID == "" || p.SessionID != session.GetId() {
		return ""
	}
	return p.ConnectionID
}

// fresh reports whether the first factor of the session was started for the
// login challenge with the given hash: the session is not the one the browser
// had when that first factor began, and its first factor completed after the
// login flow it began on was issued. Kratos timed both.
func fresh(mark FreshCookie, loginChallengeHash string, session *kClient.Session) bool {
	if session == nil || mark.LoginChallengeHash == "" || mark.LoginChallengeHash != loginChallengeHash || session.GetId() == mark.PriorSessionID {
		return false
	}
	completedAt := firstFactorTime(session)
	return !mark.StartedAt.IsZero() && completedAt != nil && !completedAt.Before(mark.StartedAt)
}

// firstFactorTime returns when the first factor of the session completed:
// the time of its first authentication method or, when the method carries
// none, the time the session authenticated.
func firstFactorTime(session *kClient.Session) *time.Time {
	if f := firstMethod(session); f != nil && f.CompletedAt != nil {
		return f.CompletedAt
	}
	return session.AuthenticatedAt
}

// privileged reports whether the session authenticated within maxAge, as
// Kratos requires before changing credentials.
func privileged(session *kClient.Session, maxAge time.Duration, now time.Time) bool {
	if session == nil || session.AuthenticatedAt == nil {
		return false
	}
	return !session.AuthenticatedAt.Add(maxAge).Before(now)
}

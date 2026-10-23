// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"testing"
	"time"

	kClient "github.com/ory/kratos-client-go/v25"
)

const (
	connA = "0190a000-0000-7000-8000-00000000000a"
	connB = "0190a000-0000-7000-8000-00000000000b"
)

var t0 = time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)

func authenticationMethod(method, provider string, aal kClient.AuthenticatorAssuranceLevel) kClient.SessionAuthenticationMethod {
	m := kClient.SessionAuthenticationMethod{}
	m.SetMethod(method)
	if provider != "" {
		m.SetProvider(provider)
	}
	m.SetAal(aal)
	m.SetCompletedAt(t0)
	return m
}

// sessionWith builds a session of identity "iid", whose email is testEmail,
// with the given authentication methods completed at t0.
func sessionWith(id string, methods ...kClient.SessionAuthenticationMethod) *kClient.Session {
	s := kClient.NewSessionWithDefaults()
	s.Id = id
	s.AuthenticationMethods = methods
	s.Identity = kClient.NewIdentity("iid", "default", "", map[string]interface{}{"email": testEmail})
	aal := kClient.AUTHENTICATORASSURANCELEVEL_AAL1
	for i, m := range methods {
		if i > 0 && m.GetAal() == kClient.AUTHENTICATORASSURANCELEVEL_AAL2 {
			aal = kClient.AUTHENTICATORASSURANCELEVEL_AAL2
		}
	}
	s.SetAuthenticatorAssuranceLevel(aal)
	return s
}

func companySession(id string, more ...kClient.SessionAuthenticationMethod) *kClient.Session {
	first := authenticationMethod("oidc", providerID, kClient.AUTHENTICATORASSURANCELEVEL_AAL1)
	return sessionWith(id, append([]kClient.SessionAuthenticationMethod{first}, more...)...)
}

func passwordSession(id string, more ...kClient.SessionAuthenticationMethod) *kClient.Session {
	first := authenticationMethod("password", "", kClient.AUTHENTICATORASSURANCELEVEL_AAL1)
	return sessionWith(id, append([]kClient.SessionAuthenticationMethod{first}, more...)...)
}

func googleSession(id string) *kClient.Session {
	return sessionWith(id, authenticationMethod("oidc", "google", kClient.AUTHENTICATORASSURANCELEVEL_AAL1))
}

// linkedSession builds the session account linking produces: a password, then
// the company sign-in Kratos added.
func linkedSession(id string, more ...kClient.SessionAuthenticationMethod) *kClient.Session {
	linked := authenticationMethod("oidc", providerID, kClient.AUTHENTICATORASSURANCELEVEL_AAL1)
	return passwordSession(id, append([]kClient.SessionAuthenticationMethod{linked}, more...)...)
}

func totp() kClient.SessionAuthenticationMethod {
	return authenticationMethod("totp", "", kClient.AUTHENTICATORASSURANCELEVEL_AAL2)
}

func member(enforcement Enforcement, requirement MFARequirement, connectionIDs ...string) *SignInContext {
	return &SignInContext{Member: true, Enforcement: enforcement, MFARequirement: requirement, ConnectionIDs: connectionIDs}
}

func invited(enforcement Enforcement, connectionIDs ...string) *SignInContext {
	return &SignInContext{InvitationAdmits: true, Enforcement: enforcement, MFARequirement: MFARequirementNone, ConnectionIDs: connectionIDs}
}

func autoJoinAdmitted(requirement MFARequirement, connectionIDs ...string) *SignInContext {
	return &SignInContext{AutoJoinAdmits: true, Enforcement: EnforcementRequired, MFARequirement: requirement, ConnectionIDs: connectionIDs}
}

func TestDecide(t *testing.T) {
	tests := []struct {
		name           string
		context        *SignInContext
		session        *kClient.Session
		provenance     string
		reauthenticate bool
		expected       need
	}{
		{
			name:     "not a member and not admitted",
			context:  &SignInContext{Enforcement: EnforcementOff},
			session:  passwordSession("s"),
			expected: needRefused,
		},
		{
			name:       "off with company sign-in",
			context:    member(EnforcementOff, MFARequirementNone),
			session:    companySession("s"),
			provenance: connA,
			expected:   needFreshFirstFactor,
		},
		{
			name:     "optional with password",
			context:  member(EnforcementOptional, MFARequirementNone, connA),
			session:  passwordSession("s", totp()),
			expected: needNothing,
		},
		{
			name:       "optional with company sign-in",
			context:    member(EnforcementOptional, MFARequirementNone, connA),
			session:    companySession("s"),
			provenance: connA,
			expected:   needNothing,
		},
		{
			name:     "optional with company sign-in without provenance",
			context:  member(EnforcementOptional, MFARequirementNone, connA),
			session:  companySession("s"),
			expected: needCompanySignIn,
		},
		{
			name:       "optional with company sign-in of another connection",
			context:    member(EnforcementOptional, MFARequirementNone, connA),
			session:    companySession("s"),
			provenance: connB,
			expected:   needFreshFirstFactor,
		},
		{
			name:     "required with password",
			context:  member(EnforcementRequired, MFARequirementNone, connA),
			session:  passwordSession("s"),
			expected: needCompanySignIn,
		},
		{
			name:     "required with public sign-in",
			context:  member(EnforcementRequired, MFARequirementNone, connA),
			session:  googleSession("s"),
			expected: needCompanySignIn,
		},
		{
			name:       "required with company sign-in of another connection",
			context:    member(EnforcementRequired, MFARequirementNone, connA),
			session:    companySession("s"),
			provenance: connB,
			expected:   needCompanySignIn,
		},
		{
			name:       "required with address outside the domains",
			context:    member(EnforcementRequired, MFARequirementNone),
			session:    companySession("s"),
			provenance: connA,
			expected:   needCompanySignIn,
		},
		{
			name:           "re-authentication comes first",
			context:        member(EnforcementOptional, MFARequirementNone, connA),
			session:        companySession("s"),
			provenance:     connA,
			reauthenticate: true,
			expected:       needFreshFirstFactor,
		},
		{
			name:     "password needs MFA",
			context:  member(EnforcementOff, MFARequirementNone),
			session:  passwordSession("s"),
			expected: needMFA,
		},
		{
			name:     "password with totp",
			context:  member(EnforcementOff, MFARequirementNone),
			session:  passwordSession("s", totp()),
			expected: needNothing,
		},
		{
			name:     "public sign-in at a tenant without MFA",
			context:  member(EnforcementOff, MFARequirementNone),
			session:  googleSession("s"),
			expected: needNothing,
		},
		{
			name:     "public sign-in at a tenant with MFA",
			context:  member(EnforcementOff, MFARequirementRequired),
			session:  googleSession("s"),
			expected: needMFA,
		},
		{
			name:       "company sign-in at a tenant without MFA",
			context:    member(EnforcementRequired, MFARequirementNone, connA),
			session:    companySession("s"),
			provenance: connA,
			expected:   needNothing,
		},
		{
			name:       "company sign-in at a tenant with MFA",
			context:    member(EnforcementRequired, MFARequirementRequired, connA),
			session:    companySession("s"),
			provenance: connA,
			expected:   needMFA,
		},
		{
			name:       "company sign-in with totp at a tenant with MFA",
			context:    member(EnforcementRequired, MFARequirementRequired, connA),
			session:    companySession("s", totp()),
			provenance: connA,
			expected:   needNothing,
		},
		{
			name:       "linked session at required",
			context:    member(EnforcementRequired, MFARequirementNone, connA),
			session:    linkedSession("s"),
			provenance: connA,
			expected:   needNothing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needed := decide(checkInput{
				context:        tt.context,
				session:        tt.session,
				provenance:     tt.provenance,
				reauthenticate: tt.reauthenticate,
			})

			if needed != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, needed)
			}
		})
	}
}

func TestIncludesCompanySignIn(t *testing.T) {
	publicThenCompany := sessionWith("s",
		authenticationMethod("oidc", "google", kClient.AUTHENTICATORASSURANCELEVEL_AAL1),
		authenticationMethod("oidc", providerID, kClient.AUTHENTICATORASSURANCELEVEL_AAL1),
	)

	tests := []struct {
		name     string
		session  *kClient.Session
		expected bool
	}{
		{name: "company sign-in as first factor", session: companySession("s"), expected: true},
		{name: "company sign-in added by account linking", session: linkedSession("s"), expected: true},
		{name: "company sign-in added after a public sign-in", session: publicThenCompany, expected: true},
		{name: "password", session: passwordSession("s"), expected: false},
		{name: "public sign-in", session: googleSession("s"), expected: false},
		{name: "no session", session: nil, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if included := includesCompanySignIn(tt.session); included != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, included)
			}
		})
	}
}

func TestIsSecondFactor(t *testing.T) {
	tests := []struct {
		method   string
		expected bool
	}{
		{method: "webauthn", expected: true},
		{method: "lookup_secret", expected: true},
		{method: "password", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			if second := isSecondFactor(tt.method); second != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, second)
			}
		})
	}
}

func TestMFADone(t *testing.T) {
	webAuthnFirst := sessionWith("s", authenticationMethod("webauthn", "", kClient.AUTHENTICATORASSURANCELEVEL_AAL1))

	tests := []struct {
		name     string
		session  *kClient.Session
		expected bool
	}{
		{name: "password with totp", session: passwordSession("s", totp()), expected: true},
		{name: "password", session: passwordSession("s"), expected: false},
		{name: "security key as first factor", session: webAuthnFirst, expected: false},
		{name: "no session", session: nil, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if done := mfaDone(tt.session); done != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, done)
			}
		})
	}
}

func TestProvenanceFor(t *testing.T) {
	session := companySession("new")

	tests := []struct {
		name     string
		cookie   ProvenanceCookie
		session  *kClient.Session
		expected string
	}{
		{name: "cookie of the session", cookie: ProvenanceCookie{SessionID: "new", ConnectionID: connA}, session: session, expected: connA},
		{name: "cookie of another session", cookie: ProvenanceCookie{SessionID: "old", ConnectionID: connA}, session: session, expected: ""},
		{name: "no cookie", cookie: ProvenanceCookie{}, session: session, expected: ""},
		{name: "no session", cookie: ProvenanceCookie{SessionID: "new", ConnectionID: connA}, session: nil, expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if provenance := provenanceFor(tt.cookie, tt.session); provenance != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, provenance)
			}
		})
	}
}

func TestFresh(t *testing.T) {
	// the first factor of the session completed at t0
	session := companySession("new")
	authenticatedOnly := companySession("new")
	authenticatedOnly.AuthenticationMethods[0].CompletedAt = nil
	authenticatedOnly.AuthenticatedAt = &t0

	tests := []struct {
		name     string
		mark     FreshCookie
		session  *kClient.Session
		expected bool
	}{
		{name: "new session after the flow was issued", mark: FreshCookie{LoginChallengeHash: "h", PriorSessionID: "old", StartedAt: t0}, session: session, expected: true},
		{name: "session the browser already had", mark: FreshCookie{LoginChallengeHash: "h", PriorSessionID: "new", StartedAt: t0}, session: session, expected: false},
		{name: "mark of another challenge", mark: FreshCookie{LoginChallengeHash: "other", PriorSessionID: "old", StartedAt: t0}, session: session, expected: false},
		{name: "no mark", mark: FreshCookie{}, session: session, expected: false},
		{name: "no session", mark: FreshCookie{LoginChallengeHash: "h", PriorSessionID: "old", StartedAt: t0}, session: nil, expected: false},
		// a session from before the flow, shown by a browser that had none when the first factor began
		{name: "first factor from before the flow was issued", mark: FreshCookie{LoginChallengeHash: "h", StartedAt: t0.Add(time.Minute)}, session: session, expected: false},
		{name: "mark without a time", mark: FreshCookie{LoginChallengeHash: "h", PriorSessionID: "old"}, session: session, expected: false},
		{name: "first factor without a time", mark: FreshCookie{LoginChallengeHash: "h", StartedAt: t0}, session: authenticatedOnly, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if isFresh := fresh(tt.mark, "h", tt.session); isFresh != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, isFresh)
			}
		})
	}
}

func TestPrivileged(t *testing.T) {
	authenticated := passwordSession("s")
	authenticated.AuthenticatedAt = &t0

	tests := []struct {
		name     string
		session  *kClient.Session
		now      time.Time
		expected bool
	}{
		{name: "within the privileged session age", session: authenticated, now: t0.Add(30 * time.Minute), expected: true},
		{name: "past the privileged session age", session: authenticated, now: t0.Add(61 * time.Minute), expected: false},
		{name: "session without authentication time", session: passwordSession("s"), now: t0, expected: false},
		{name: "no session", session: nil, now: t0, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if isPrivileged := privileged(tt.session, time.Hour, tt.now); isPrivileged != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, isPrivileged)
			}
		})
	}
}

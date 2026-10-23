// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package kratos

import (
	"slices"

	kClient "github.com/ory/kratos-client-go/v25"
)

// SignIn is how a user got in: the authentication methods the session
// completed, in order.
type SignIn struct {
	Methods []string
}

// Requirement is what a sign-in still needs before it may complete.
type Requirement struct {
	// SecondFactor says that the session must have completed a second factor
	// (AAL2) before consent is given.
	SecondFactor bool
	// SetUp is the second factor method the user must have set up, "totp" or
	// "webauthn". It is empty when there is none.
	SetUp string
	// RegenerateBackupCodes says that a user who signs in with a backup code
	// and is left with too few of them regenerates them before the sign-in
	// continues.
	RegenerateBackupCodes bool
}

// PlatformSecondFactorPolicy is the platform's second factor policy: what the
// MFA_ENABLED and OIDC_WEBAUTHN_SEQUENCING_ENABLED settings ask of a sign-in.
type PlatformSecondFactorPolicy struct {
	mfaEnabled                    bool
	oidcWebAuthnSequencingEnabled bool
}

// For returns what the platform asks of the sign-in.
func (p *PlatformSecondFactorPolicy) For(signIn SignIn) Requirement {
	var requirement Requirement

	// the first method decides whether a second factor is asked for
	if len(signIn.Methods) > 0 {
		switch signIn.Methods[0] {
		case "oidc":
			requirement.SecondFactor = p.oidcWebAuthnSequencingEnabled
		case "password", "webauthn":
			requirement.SecondFactor = p.mfaEnabled
		}
	}

	// a user who came through an external provider sets up a WebAuthn key,
	// any other user an authenticator app
	usedOIDC := slices.Contains(signIn.Methods, "oidc")

	switch {
	case usedOIDC && p.oidcWebAuthnSequencingEnabled:
		requirement.SetUp = "webauthn"
	case !usedOIDC && p.mfaEnabled:
		requirement.SetUp = "totp"
	}

	requirement.RegenerateBackupCodes = p.mfaEnabled

	return requirement
}

// NewSignIn returns the sign-in behind a Kratos session.
func NewSignIn(session *kClient.Session) SignIn {
	signIn := SignIn{}

	for _, method := range session.GetAuthenticationMethods() {
		signIn.Methods = append(signIn.Methods, method.GetMethod())
	}

	return signIn
}

func NewPlatformSecondFactorPolicy(mfaEnabled, oidcWebAuthnSequencingEnabled bool) *PlatformSecondFactorPolicy {
	p := new(PlatformSecondFactorPolicy)

	p.mfaEnabled = mfaEnabled
	p.oidcWebAuthnSequencingEnabled = oidcWebAuthnSequencingEnabled

	return p
}

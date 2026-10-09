// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import "time"

// providerID is the Kratos OIDC provider behind every company sign-in. It
// is fixed in sso-service.
const providerID = "byo-sso"

// completePath is the endpoint a company sign-in with no Hydra login request
// returns to.
const completePath = "/api/v0/sso/complete"

const (
	// requestBudget is the time the calls behind one browser request may
	// take together. It is less than the write timeout of the server, so
	// that the browser gets an answer rather than a dropped connection.
	requestBudget = 12 * time.Second
	// tokenTimeout is the longest one service token request may take.
	tokenTimeout = 5 * time.Second
	// defaultGRPCTimeout is the longest one call to a backend may take when
	// no timeout is configured.
	defaultGRPCTimeout = 5 * time.Second
	// attemptTTL is the lifetime of a sign-in attempt in sso-service.
	attemptTTL = 30 * time.Minute
	// defaultSessionTTL bounds a cookie tied to a session without an expiry.
	defaultSessionTTL = time.Hour

	reconnectBaseDelay = 200 * time.Millisecond
	reconnectMaxDelay  = 5 * time.Second
	connectTimeout     = 5 * time.Second
)

// Node groups of the nodes login-ui adds to a flow. The company sign-ins are
// not in the oidc group: the browser never submits one to Kratos.
const (
	ssoNodeGroup    = "sso"
	tenantNodeGroup = "tenant"
)

const (
	ssoConnectionField = ssoNodeGroup + "_connection"
	ssoUnlinkField     = ssoNodeGroup + "_unlink"
	// ssoLinkField is a company sign-in the account already has, offered on
	// the account-linking page to sign in to the account.
	ssoLinkField = ssoNodeGroup + "_account_link"
	// ssoReauthField is a company sign-in the account already has, offered
	// on the refresh login to sign in again.
	ssoReauthField = ssoNodeGroup + "_reauthenticate"
	// ssoLinkNodePrefix, followed by the connection ID, names the text node
	// of one company sign-in of the account on the settings page.
	ssoLinkNodePrefix = ssoNodeGroup + "_link_"

	tenantField      = ssoNodeGroup + "_tenant"
	tenantResetField = ssoNodeGroup + "_tenant_reset"
	recoverAnchorID  = ssoNodeGroup + "_recover"
)

// Kratos methods and node groups.
const (
	methodPassword        = "password"
	methodOIDC            = "oidc"
	methodPasskey         = "passkey"
	methodWebAuthn        = "webauthn"
	methodCode            = "code"
	methodTOTP            = "totp"
	methodLookup          = "lookup_secret"
	methodIdentifierFirst = "identifier_first"
)

// Kratos message IDs.
const (
	signInWith        = 1010002
	signInAndLink     = 1010016
	unlinkProvider    = 1050003
	nodeLabel         = 1070000
	validationFailure = 4000000
)

// Messages login-ui adds to a login flow.
const (
	recoverPromptID        = 1990001
	recoverPromptText      = "You have no way to sign in with this account any more. Recover it to set a portal password."
	lookupFailedID         = 1990002
	lookupFailedText       = "Your tenants could not be loaded. Try again in a moment."
	companyUnavailableID   = 1990003
	companyUnavailableText = "Company sign-in is unavailable right now. You can sign in another way."
	noTenantAdmitsID       = 1990004
	noTenantAdmitsText     = "No tenant accepts this address any more. Start creating your account again."
)

// invitationLabel marks a tenant the address is invited to.
const invitationLabel = " — invitation"

// Error IDs the frontend knows.
const (
	ssoUnavailableError     = "sso_unavailable"
	ssoNotApplicableError   = "sso_not_applicable"
	notAMemberError         = "tenant_not_a_member"
	aal2RequiredError       = "session_aal2_required"
	tenantSelectionRequired = "tenant_selection_required"
)

const (
	ssoUnavailableMessage          = "single sign-on is temporarily unavailable"
	tenantsUnavailableMessage      = "signing in is temporarily unavailable; try again in a moment"
	registrationUnavailableMessage = "creating an account is temporarily unavailable; try again in a moment"
	ssoNotApplicableMessage        = "company sign-in is not available for this address; contact your tenant's administrator"
	notAMemberMessage              = "this account is not a member of that tenant"
)

// Cookies of login-ui, encrypted with its cookie cipher.
const (
	// signInCookieName is the cookie of a company sign-in in progress. It
	// lives as long as the attempt.
	signInCookieName = "login_ui_sso_signin"
	// provenanceCookieName is the cookie telling which connection produced
	// the first factor of the session. It lives until the session expires.
	provenanceCookieName = "login_ui_sso_provenance"
	// freshCookieName is the cookie marking a first factor started for a
	// login challenge.
	freshCookieName = "login_ui_sso_fresh"
	// registrationCookieName is the cookie of a registration sent to the
	// company sign-in of a tenant.
	registrationCookieName = "login_ui_sso_registration"
)

// receiptCookiePrefix, followed by the first 16 hex digits of the SHA-256 of
// a ticket, names the cookie sso-service sets in the browser that completes
// the sign-in of the ticket. Only sso-service can make or check its value.
const receiptCookiePrefix = "__Host-sso_receipt_"

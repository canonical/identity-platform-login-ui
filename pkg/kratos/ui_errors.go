// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package kratos

import (
	"errors"
	"fmt"
	"strings"

	kClient "github.com/ory/kratos-client-go/v25"
)

// Kratos UI message IDs, see https://github.com/ory/kratos/blob/master/text/id.go
const (
	ValidationGeneric            = 4000001
	PropertyMissing              = 4000002
	NotEnoughCharacters          = 4000003
	PasswordPolicyViolation      = 4000005
	IncorrectCredentials         = 4000006
	DuplicateIdentifier          = 4000007
	InvalidAuthCode              = 4000008
	AddressNotVerified           = 4000010
	MissingTOTPSetup             = 4000011
	BackupCodeAlreadyUsed        = 4000012
	MissingSecurityKey           = 4000013
	MissingBackupCodesSetup      = 4000014
	MissingSecurityKeySetup      = 4000015
	InvalidBackupCode            = 4000016
	TooManyCharacters            = 4000017
	DuplicateIdentifierOIDCLink  = 4000027
	DuplicateIdentifierWithHints = 4000028
	PasswordIdentifierSimilarity = 4000031
	PasswordTooShort             = 4000032
	PasswordTooLong              = 4000033
	PasswordBreached             = 4000034
	IncorrectAccountIdentifier   = 4000037
	PasswordSameAsOld            = 4000039
	IdentityDisabled             = 4010011
	InvalidRecoveryCode          = 4060006
)

// Kratos does not use message IDs in this range. A registration webhook returns
// one to reject the registration outright, so that the user is sent to the error
// page instead of back to the registration form.
const (
	RegistrationRejectedFirst = 4200000
	RegistrationRejectedLast  = 4299999
)

// alreadyLoggedInReason is how the reason of Kratos's login.ErrAlreadyLoggedIn
// starts; registration and recovery have errors of their own with another
// wording. Kratos refuses a submission to a login flow that the session
// already satisfies with that error
// (https://github.com/ory/kratos/blob/v25.4.0/selfservice/flow/login/handler.go#L840-L858),
// and answers with the flow, the reason in a ValidationGeneric message
// (https://github.com/ory/kratos/blob/v25.4.0/ui/container/container.go#L168-L171),
// not with the session_already_available error it answers when such a flow is
// created.
const alreadyLoggedInReason = "A valid session was detected and thus login is not possible"

// uiErrorText maps a Kratos UI message ID to the text returned to the user.
// To handle a new code, add a constant above and one entry here.
//
// The frontend matches on some of these strings (ui/pages/login.tsx,
// ui/components/NodeInputPassword.tsx); change them in both places.
var uiErrorText = map[int64]func(kClient.UiText) string{
	IncorrectCredentials:         static("incorrect username or password"),
	IncorrectAccountIdentifier:   static("account does not exist or has no login method configured"),
	AddressNotVerified:           static("account not active yet, verify your email address first"),
	IdentityDisabled:             static("this account has been disabled"),
	PropertyMissing:              fromContext("invalid %s", "property"), // NodeInputPassword.tsx matches "invalid password"
	NotEnoughCharacters:          fromContext("at least %v characters required", "min_length"),
	TooManyCharacters:            fromContext("maximum %v characters allowed", "max_length"),
	PasswordTooShort:             fromContext("password must be at least %v characters long", "min_length"),
	PasswordTooLong:              fromContext("maximum %v characters allowed", "max_length"),
	PasswordPolicyViolation:      fromContext("password can not be used because %v", "reason"),
	PasswordBreached:             static("password has been found in data breaches and can not be used"),
	PasswordIdentifierSimilarity: static("password can not be similar to the email"),
	PasswordSameAsOld:            static("new password must be different from the old password"),
	InvalidAuthCode:              static("invalid authentication code"),
	MissingTOTPSetup:             static("no authenticator app set up for this account"),
	MissingSecurityKey:           static("no security key set up for this account"),
	MissingSecurityKeySetup:      static("choose a different login method"),
	BackupCodeAlreadyUsed:        static("this backup code was already used"),
	InvalidBackupCode:            static("invalid backup code"),
	MissingBackupCodesSetup:      static("login with backup codes unavailable"),
	DuplicateIdentifier:          static("an account with the same identifier already exists, contact support"),
	DuplicateIdentifierOIDCLink:  static("an account with the same identifier already exists, sign in with your existing credentials"),
	DuplicateIdentifierWithHints: static("an account with the same identifier already exists, sign in with your existing credentials"),
	InvalidRecoveryCode:          static("the recovery code is invalid or has already been used"),
}

func static(text string) func(kClient.UiText) string {
	return func(kClient.UiText) string { return text }
}

func fromContext(format, key string) func(kClient.UiText) string {
	return func(m kClient.UiText) string { return fmt.Sprintf(format, m.Context[key]) }
}

// uiError returns the user-facing error for the first recognised message in msgs,
// or nil if none of them has a known ID.
func uiError(msgs []kClient.UiText) error {
	for _, m := range msgs {
		if text, ok := uiErrorText[m.GetId()]; ok {
			return errors.New(text(m))
		}
	}

	return nil
}

// alreadyLoggedIn reports whether msgs say that the session already satisfies
// the login flow that was submitted.
func alreadyLoggedIn(msgs []kClient.UiText) bool {
	for _, m := range msgs {
		if m.GetId() == ValidationGeneric && strings.HasPrefix(m.GetText(), alreadyLoggedInReason) {
			return true
		}
	}

	return false
}

// registrationRejection returns the text of the first message with which a
// webhook rejected the registration.
func registrationRejection(ui kClient.UiContainer) (string, bool) {
	rejection := func(msgs []kClient.UiText) (string, bool) {
		for _, m := range msgs {
			if id := m.GetId(); id >= RegistrationRejectedFirst && id <= RegistrationRejectedLast {
				return m.GetText(), true
			}
		}

		return "", false
	}

	if text, ok := rejection(ui.GetMessages()); ok {
		return text, true
	}

	for _, n := range ui.GetNodes() {
		if text, ok := rejection(n.GetMessages()); ok {
			return text, true
		}
	}

	return "", false
}

// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package kratos

import (
	"context"
	"errors"
	"net/http"

	client "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
	"github.com/canonical/identity-platform-login-ui/internal/logging"
)

// Option configures optional API behaviour.
type Option func(*API)

// WithExtension plugs an extension into the handlers' hooks.
func WithExtension(ext ExtensionInterface) Option {
	return func(a *API) {
		a.ext = ext
	}
}

// WithBackupCodesRegeneration asks a user who signed in with one of their
// last backup codes to make new ones, also when mfaEnabled is off: MFA may
// be asked for by other means than of everyone.
func WithBackupCodesRegeneration() Option {
	return func(a *API) {
		a.regenerateBackupCodes = true
	}
}

// errResponseWritten signals that a hook has already answered the request.
var errResponseWritten = errors.New("response already written")

// UnsetSessionCookie expires the Kratos session cookie of the browser.
func UnsetSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, kratosSessionUnsetCookie())
}

// WriteGetFlowError answers a flow an extension failed to fetch or create as
// the handlers answer their own.
func WriteGetFlowError(w http.ResponseWriter, logger logging.LoggerInterface, flowType string, err error, fallback string) {
	(&API{logger: logger}).writeGetFlowError(w, flowType, err, fallback)
}

// WriteUpdateFlowError answers a flow update an extension failed to make as
// the handlers answer their own.
func WriteUpdateFlowError(w http.ResponseWriter, logger logging.LoggerInterface, flowType string, err error) {
	(&API{logger: logger}).writeUpdateFlowError(w, flowType, err)
}

// NoOpExtension is used when no extension is configured. No hook answers a
// request, and flows pass through unchanged.
type NoOpExtension struct{}

func NewNoOpExtension() *NoOpExtension {
	return &NoOpExtension{}
}

func (n *NoOpExtension) HydrateLoginFlow(_ http.ResponseWriter, _ *http.Request, flow *client.LoginFlow) (*client.LoginFlow, bool) {
	return flow, true
}

func (n *NoOpExtension) BeforeTenantSelection(_ http.ResponseWriter, _ *http.Request, _ *client.LoginFlow, _ string) bool {
	return false
}

func (n *NoOpExtension) InterceptLoginSubmission(_ http.ResponseWriter, r *http.Request, _ *client.LoginFlow) (*http.Request, bool) {
	return r, false
}

func (n *NoOpExtension) BeforeAcceptLogin(_ http.ResponseWriter, _ *http.Request, _ *client.Session, _ string, _ cookies.FlowStateCookie) bool {
	return false
}

func (n *NoOpExtension) InterceptRegistrationSubmission(_ http.ResponseWriter, _ *http.Request, _ string) bool {
	return false
}

func (n *NoOpExtension) HydrateSettingsFlow(_ context.Context, flow *client.SettingsFlow, _ []*http.Cookie) *client.SettingsFlow {
	return flow
}

func (n *NoOpExtension) InterceptSettingsSubmission(_ http.ResponseWriter, _ *http.Request, _ string) bool {
	return false
}

func (n *NoOpExtension) HandlesSessionLogin() bool { return false }

func (n *NoOpExtension) HandleSessionLogin(_ http.ResponseWriter, _ *http.Request, _ *client.Session, _ string) {
}

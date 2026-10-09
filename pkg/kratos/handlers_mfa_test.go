// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package kratos

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"
	gomock "go.uber.org/mock/gomock"

	kClient "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
	"github.com/canonical/identity-platform-login-ui/pkg/tenants"
)

// The tests in this file pin how the MFA_ENABLED and
// OIDC_WEBAUTHN_SEQUENCING_ENABLED settings drive the login handlers: which
// lookups are made about the user, in which order, and where the user is sent.
// The mocks are strict, so a lookup, log line or span that a row does not
// expect fails it.

const (
	mfaTestIdentityID = "identity-id"
	mfaTestAccept     = "application/json, text/plain, */*"
)

// mfaLookup is the answer of one lookup about the user.
type mfaLookup struct {
	result bool
	err    error
}

// mfaRedirect is the body the handlers answer with when they send the user
// somewhere.
type mfaRedirect struct {
	Error struct {
		ID string `json:"id"`
	} `json:"error"`
	RedirectTo string `json:"redirect_to"`
}

func mfaTestSession(methods ...string) *kClient.Session {
	session := kClient.NewSession("session-id")
	session.Identity = kClient.NewIdentity(mfaTestIdentityID, "test.json", "https://test.com/test.json", map[string]string{"name": "name"})

	for _, method := range methods {
		session.AuthenticationMethods = append(session.AuthenticationMethods, kClient.SessionAuthenticationMethod{Method: &method})
	}

	return session
}

func expectSpan(mockTracer *MockTracingInterface, name string, times int) {
	mockTracer.EXPECT().Start(gomock.Any(), name).Return(context.Background(), trace.SpanFromContext(context.Background())).Times(times)
}

func assertMFARedirect(t *testing.T, res *http.Response, expectedErrorID, expectedRedirectTo string) {
	t.Helper()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP status code 200 got %v: %s", res.StatusCode, data)
	}

	redirect := mfaRedirect{}
	if err := json.Unmarshal(data, &redirect); err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}

	if redirect.Error.ID != expectedErrorID {
		t.Errorf("expected error id %q got %q", expectedErrorID, redirect.Error.ID)
	}

	if redirect.RedirectTo != expectedRedirectTo {
		t.Errorf("expected redirect_to %q got %q", expectedRedirectTo, redirect.RedirectTo)
	}
}

func assertMFAFailure(t *testing.T, res *http.Response, expectedBody string) {
	t.Helper()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected HTTP status code 500 got %v", res.StatusCode)
	}

	if string(data) != expectedBody {
		t.Errorf("expected body %q got %q", expectedBody, data)
	}
}

func TestHandleCreateFlowMFA(t *testing.T) {
	const (
		acceptLogin = iota
		setupAuthenticator
		setupPasskey
		failMFACheck
		failWebAuthnCheck
	)

	loginChallenge := "login_challenge_2341235123231"
	returnTo := BASE_URL + "/ui/login?login_challenge=" + loginChallenge
	acceptRedirect := "https://some/path/to/somewhere"
	lookupErr := fmt.Errorf("lookup failed")

	tests := []struct {
		name                          string
		mfaEnabled                    bool
		oidcWebAuthnSequencingEnabled bool
		methods                       []string
		// the lookups the handler makes, nil when it must not make them
		totp     *mfaLookup
		webAuthn *mfaLookup
		expected int
	}{
		{
			name:       "mfa: password without an authenticator sets one up",
			mfaEnabled: true,
			methods:    []string{"password"},
			totp:       &mfaLookup{result: false},
			expected:   setupAuthenticator,
		},
		{
			name:       "mfa: a failed authenticator lookup is a server error",
			mfaEnabled: true,
			methods:    []string{"password"},
			totp:       &mfaLookup{err: lookupErr},
			expected:   failMFACheck,
		},
		{
			name:       "mfa: oidc is accepted without a lookup",
			mfaEnabled: true,
			methods:    []string{"oidc"},
			expected:   acceptLogin,
		},
		{
			name:       "mfa: password then oidc is accepted without a lookup",
			mfaEnabled: true,
			methods:    []string{"password", "oidc"},
			expected:   acceptLogin,
		},
		{
			name:                          "sequencing: oidc without a key sets one up",
			oidcWebAuthnSequencingEnabled: true,
			methods:                       []string{"oidc"},
			webAuthn:                      &mfaLookup{result: false},
			expected:                      setupPasskey,
		},
		{
			name:                          "sequencing: a failed key lookup is a server error",
			oidcWebAuthnSequencingEnabled: true,
			methods:                       []string{"oidc"},
			webAuthn:                      &mfaLookup{err: lookupErr},
			expected:                      failWebAuthnCheck,
		},
		{
			name:                          "sequencing: password is accepted without a lookup",
			oidcWebAuthnSequencingEnabled: true,
			methods:                       []string{"password"},
			expected:                      acceptLogin,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockService := NewMockServiceInterface(ctrl)
			mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)

			session := mfaTestSession(tt.methods...)
			flowCookie := cookies.FlowStateCookie{LoginChallengeHash: cookies.ChallengeHash(loginChallenge)}

			req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_FLOW_URL, nil)
			values := req.URL.Query()
			values.Add("login_challenge", loginChallenge)
			req.URL.RawQuery = values.Encode()
			req.Header.Set("Accept", mfaTestAccept)

			mockCookieManager.EXPECT().GetStateCookie(gomock.Any()).Return(cookies.FlowStateCookie{}, nil)

			calls := []any{
				mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil),
			}
			if tt.totp != nil {
				calls = append(calls, mockService.EXPECT().HasTOTPAvailable(gomock.Any(), mfaTestIdentityID).Return(tt.totp.result, tt.totp.err))
			}
			if tt.webAuthn != nil {
				calls = append(calls, mockService.EXPECT().HasWebAuthnAvailable(gomock.Any(), mfaTestIdentityID).Return(tt.webAuthn.result, tt.webAuthn.err))
			}

			webAuthnSpans := 1

			switch tt.expected {
			case acceptLogin:
				calls = append(
					calls,
					mockService.EXPECT().MustReAuthenticate(gomock.Any(), loginChallenge, session, cookies.FlowStateCookie{}).Return(false, nil),
					mockService.EXPECT().AcceptLoginRequest(gomock.Any(), session, loginChallenge, "").Return(&BrowserLocationChangeRequired{RedirectTo: &acceptRedirect}, nil, nil),
				)
				mockCookieManager.EXPECT().ClearStateCookie(gomock.Any())
			case setupAuthenticator:
				webAuthnSpans = 0
				flowCookie.TotpSetup = true
				mockCookieManager.EXPECT().SetStateCookie(gomock.Any(), flowCookie).Return(nil)
			case setupPasskey:
				flowCookie.WebauthnSetup = true
				mockCookieManager.EXPECT().SetStateCookie(gomock.Any(), flowCookie).Return(nil)
			case failMFACheck:
				webAuthnSpans = 0
				mockLogger.EXPECT().Errorf("failed to check MFA status: %v", lookupErr)
			case failWebAuthnCheck:
				mockLogger.EXPECT().Errorf("failed to check WebAuthn status: %v", lookupErr)
			}

			gomock.InOrder(calls...)

			expectSpan(mockTracer, "kratos.API.shouldEnforceVerificationWithSession", 1)
			expectSpan(mockTracer, "kratos.API.shouldEnforceMFAWithSession", 1)
			expectSpan(mockTracer, "kratos.API.shouldEnforceWebAuthnWithSession", webAuthnSpans)

			w := httptest.NewRecorder()
			mux := chi.NewMux()
			NewAPI(mockService, false, tt.mfaEnabled, tt.oidcWebAuthnSequencingEnabled, tenants.NewNoOpTenantResolver(), BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

			mux.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()

			switch tt.expected {
			case acceptLogin:
				assertMFARedirect(t, res, "", acceptRedirect)
			case setupAuthenticator:
				assertMFARedirect(t, res, TOTP_REGISTRATION_REQUIRED, "/ui/setup_secure?return_to="+url.QueryEscape(returnTo))
			case setupPasskey:
				assertMFARedirect(t, res, WEBAUTHN_REGISTRATION_REQUIRED, "/ui/setup_passkey?return_to="+url.QueryEscape(returnTo))
			case failMFACheck:
				assertMFAFailure(t, res, "failed to check MFA status\n")
			case failWebAuthnCheck:
				assertMFAFailure(t, res, "failed to check WebAuthn status\n")
			}
		})
	}
}

func TestHandleUpdateFlowMFA(t *testing.T) {
	const (
		followRedirect = iota
		setupAuthenticator
		regenerateBackupCodes
		failMFACheck
		failBackupCodesCheck
	)

	flowId := "test"
	returnTo := "https://some/return/url"
	kratosRedirect := "https://some/path/to/somewhere"
	lookupErr := fmt.Errorf("lookup failed")

	tests := []struct {
		name                          string
		mfaEnabled                    bool
		oidcWebAuthnSequencingEnabled bool
		noSession                     bool
		methods                       []string
		// the lookups the handler makes, nil when it must not make them
		totp        *mfaLookup
		backupCodes *mfaLookup
		// the debug line about a session that has one method only
		oneMethodLogged bool
		expected        int
	}{
		{
			name:       "mfa: no session follows kratos without a lookup",
			mfaEnabled: true,
			noSession:  true,
			expected:   followRedirect,
		},
		{
			name:            "mfa: password without an authenticator sets one up",
			mfaEnabled:      true,
			methods:         []string{"password"},
			totp:            &mfaLookup{result: false},
			oneMethodLogged: true,
			expected:        setupAuthenticator,
		},
		{
			name:       "mfa: a failed authenticator lookup is a server error",
			mfaEnabled: true,
			methods:    []string{"password"},
			totp:       &mfaLookup{err: lookupErr},
			expected:   failMFACheck,
		},
		{
			name:            "mfa: oidc follows kratos without a lookup",
			mfaEnabled:      true,
			methods:         []string{"oidc"},
			oneMethodLogged: true,
			expected:        followRedirect,
		},
		{
			name:        "mfa: regenerating backup codes comes before setting an authenticator up",
			mfaEnabled:  true,
			methods:     []string{"password", "lookup_secret"},
			totp:        &mfaLookup{result: false},
			backupCodes: &mfaLookup{result: true},
			expected:    regenerateBackupCodes,
		},
		{
			name:        "mfa: a failed backup codes lookup is a server error",
			mfaEnabled:  true,
			methods:     []string{"password", "lookup_secret"},
			totp:        &mfaLookup{result: true},
			backupCodes: &mfaLookup{err: lookupErr},
			expected:    failBackupCodesCheck,
		},
		{
			name:       "mfa: a backup code as the first method is not looked up",
			mfaEnabled: true,
			methods:    []string{"lookup_secret", "password"},
			totp:       &mfaLookup{result: true},
			expected:   followRedirect,
		},
		{
			name:                          "sequencing: oidc then a backup code follows kratos without a lookup",
			oidcWebAuthnSequencingEnabled: true,
			methods:                       []string{"oidc", "lookup_secret"},
			expected:                      followRedirect,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockService := NewMockServiceInterface(ctrl)
			mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)

			var session *kClient.Session
			if !tt.noSession {
				session = mfaTestSession(tt.methods...)
			}

			flow := kClient.NewLoginFlowWithDefaults()
			flow.Id = flowId
			flow.ReturnTo = &returnTo

			flowBody := new(kClient.UpdateLoginFlowBody)
			flowBody.UpdateLoginFlowWithPasswordMethod = kClient.NewUpdateLoginFlowWithPasswordMethod("user@example.com", "password", "password")

			flowCookie := cookies.FlowStateCookie{}

			req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_LOGIN_FLOW_URL, nil)
			values := req.URL.Query()
			values.Add("flow", flowId)
			req.URL.RawQuery = values.Encode()
			req.Header.Set("Accept", mfaTestAccept)

			mockCookieManager.EXPECT().GetStateCookie(gomock.Any()).Return(cookies.FlowStateCookie{}, nil)

			calls := []any{
				mockService.EXPECT().GetLoginFlow(gomock.Any(), flowId, req.Cookies()).Return(flow, nil, nil),
				mockService.EXPECT().ParseLoginFlowMethodBody(gomock.Any(), gomock.Any()).Return(flowBody, req.Cookies(), nil),
				mockService.EXPECT().CheckAllowedProvider(gomock.Any(), flow, flowBody).Return(true, nil),
				mockService.EXPECT().UpdateLoginFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(&BrowserLocationChangeRequired{RedirectTo: &kratosRedirect}, nil, req.Cookies(), nil),
				mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil),
			}
			if tt.totp != nil {
				calls = append(calls, mockService.EXPECT().HasTOTPAvailable(gomock.Any(), mfaTestIdentityID).Return(tt.totp.result, tt.totp.err))
			}
			if tt.backupCodes != nil {
				calls = append(calls, mockService.EXPECT().HasNotEnoughLookupSecretsLeft(gomock.Any(), mfaTestIdentityID).Return(tt.backupCodes.result, tt.backupCodes.err))
			}

			gomock.InOrder(calls...)

			if tt.oneMethodLogged {
				mockLogger.EXPECT().Debugf("User has not yet completed 2fa")
			}

			backupCodesSpans := 1

			switch tt.expected {
			case followRedirect:
				mockCookieManager.EXPECT().SetStateCookie(gomock.Any(), flowCookie).Return(nil)
			case setupAuthenticator:
				flowCookie.TotpSetup = true
				mockCookieManager.EXPECT().SetStateCookie(gomock.Any(), flowCookie).Return(nil)
			case regenerateBackupCodes:
				flowCookie.BackupCodeUsed = true
				mockCookieManager.EXPECT().SetStateCookie(gomock.Any(), flowCookie).Return(nil)
			case failMFACheck:
				backupCodesSpans = 0
				mockLogger.EXPECT().Errorf("enforce MFA check error: %v", lookupErr)
			case failBackupCodesCheck:
				mockLogger.EXPECT().Errorf("backup codes check error: %v", lookupErr)
			}

			expectSpan(mockTracer, "kratos.API.shouldEnforceVerificationWithSession", 1)
			expectSpan(mockTracer, "kratos.API.shouldEnforceMFAWithSession", 1)
			expectSpan(mockTracer, "kratos.API.shouldRegenerateBackupCodesWithSession", backupCodesSpans)

			w := httptest.NewRecorder()
			mux := chi.NewMux()
			NewAPI(mockService, false, tt.mfaEnabled, tt.oidcWebAuthnSequencingEnabled, tenants.NewNoOpTenantResolver(), BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

			mux.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()

			switch tt.expected {
			case followRedirect:
				assertMFARedirect(t, res, "", kratosRedirect)
			case setupAuthenticator:
				assertMFARedirect(t, res, TOTP_REGISTRATION_REQUIRED, "/ui/setup_secure?return_to="+url.QueryEscape(returnTo))
			case regenerateBackupCodes:
				assertMFARedirect(t, res, RegenerateBackupCodesError, "/ui/backup_codes_regenerate?flow="+flowId+"&return_to="+url.QueryEscape(returnTo))
			case failMFACheck, failBackupCodesCheck:
				assertMFAFailure(t, res, "internal server error\n")
			}
		})
	}
}

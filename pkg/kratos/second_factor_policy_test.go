// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package kratos

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-chi/chi/v5"
	gomock "go.uber.org/mock/gomock"

	kClient "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
	"github.com/canonical/identity-platform-login-ui/pkg/tenants"
)

func TestPlatformSecondFactorPolicyFor(t *testing.T) {
	tests := []struct {
		name                          string
		mfaEnabled                    bool
		oidcWebAuthnSequencingEnabled bool
		methods                       []string
		expected                      Requirement
	}{
		{
			name:     "no setting: password needs nothing",
			methods:  []string{"password"},
			expected: Requirement{},
		},
		{
			name:       "mfa: password needs a second factor and an authenticator",
			mfaEnabled: true,
			methods:    []string{"password"},
			expected:   Requirement{SecondFactor: true, SetUp: "totp", RegenerateBackupCodes: true},
		},
		{
			name:       "mfa: webauthn needs a second factor and an authenticator",
			mfaEnabled: true,
			methods:    []string{"webauthn"},
			expected:   Requirement{SecondFactor: true, SetUp: "totp", RegenerateBackupCodes: true},
		},
		{
			name:       "mfa: oidc needs no second factor and nothing set up",
			mfaEnabled: true,
			methods:    []string{"oidc"},
			expected:   Requirement{RegenerateBackupCodes: true},
		},
		{
			name:       "mfa: a passkey needs an authenticator and no second factor",
			mfaEnabled: true,
			methods:    []string{"passkey"},
			expected:   Requirement{SetUp: "totp", RegenerateBackupCodes: true},
		},
		{
			name:       "mfa: no method needs an authenticator and no second factor",
			mfaEnabled: true,
			expected:   Requirement{SetUp: "totp", RegenerateBackupCodes: true},
		},
		{
			name:                          "sequencing: oidc needs a second factor and a key",
			oidcWebAuthnSequencingEnabled: true,
			methods:                       []string{"oidc"},
			expected:                      Requirement{SecondFactor: true, SetUp: "webauthn"},
		},
		{
			name:                          "sequencing: password then oidc needs a key and no second factor",
			oidcWebAuthnSequencingEnabled: true,
			methods:                       []string{"password", "oidc"},
			expected:                      Requirement{SetUp: "webauthn"},
		},
		{
			name:                          "mfa and sequencing: oidc needs a second factor and a key",
			mfaEnabled:                    true,
			oidcWebAuthnSequencingEnabled: true,
			methods:                       []string{"oidc"},
			expected:                      Requirement{SecondFactor: true, SetUp: "webauthn", RegenerateBackupCodes: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := NewPlatformSecondFactorPolicy(tt.mfaEnabled, tt.oidcWebAuthnSequencingEnabled)

			requirement := policy.For(SignIn{Methods: tt.methods})

			if requirement != tt.expected {
				t.Fatalf("expected %+v got %+v", tt.expected, requirement)
			}
		})
	}
}

// The tests below give the API a policy of their own, with both settings off,
// and check that the handlers follow what that policy asks.

func TestHandleCreateFlowAsksSecondFactorPolicy(t *testing.T) {
	loginChallenge := "login_challenge_2341235123231"
	returnTo := BASE_URL + "/ui/login?login_challenge=" + loginChallenge

	tests := []struct {
		name    string
		methods []string
		setUp   string
	}{
		{
			name:    "an authenticator that is missing is set up",
			methods: []string{"oidc"},
			setUp:   "totp",
		},
		{
			name:    "a key that is missing is set up",
			methods: []string{"password"},
			setUp:   "webauthn",
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
			mockPolicy := NewMockSecondFactorPolicyInterface(ctrl)

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
				mockPolicy.EXPECT().For(SignIn{Methods: tt.methods}).Return(Requirement{SetUp: tt.setUp}),
			}

			webAuthnSpans := 1

			switch tt.setUp {
			case "totp":
				webAuthnSpans = 0
				flowCookie.TotpSetup = true
				calls = append(calls, mockService.EXPECT().HasTOTPAvailable(gomock.Any(), mfaTestIdentityID).Return(false, nil))
			case "webauthn":
				flowCookie.WebauthnSetup = true
				calls = append(calls, mockService.EXPECT().HasWebAuthnAvailable(gomock.Any(), mfaTestIdentityID).Return(false, nil))
			}

			gomock.InOrder(calls...)

			mockCookieManager.EXPECT().SetStateCookie(gomock.Any(), flowCookie).Return(nil)

			expectSpan(mockTracer, "kratos.API.shouldEnforceVerificationWithSession", 1)
			expectSpan(mockTracer, "kratos.API.shouldEnforceMFAWithSession", 1)
			expectSpan(mockTracer, "kratos.API.shouldEnforceWebAuthnWithSession", webAuthnSpans)

			w := httptest.NewRecorder()
			mux := chi.NewMux()
			NewAPI(mockService, false, false, false, tenants.NewNoOpTenantResolver(), BASE_URL, mockCookieManager, mockTracer, mockLogger, WithSecondFactorPolicy(mockPolicy)).RegisterEndpoints(mux)

			mux.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()

			switch tt.setUp {
			case "totp":
				assertMFARedirect(t, res, TOTP_REGISTRATION_REQUIRED, "/ui/setup_secure?return_to="+url.QueryEscape(returnTo))
			case "webauthn":
				assertMFARedirect(t, res, WEBAUTHN_REGISTRATION_REQUIRED, "/ui/setup_passkey?return_to="+url.QueryEscape(returnTo))
			}
		})
	}
}

func TestHandleUpdateFlowAsksSecondFactorPolicy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockPolicy := NewMockSecondFactorPolicyInterface(ctrl)

	flowId := "test"
	returnTo := "https://some/return/url"
	kratosRedirect := "https://some/path/to/somewhere"

	session := mfaTestSession("password", "lookup_secret")

	flow := kClient.NewLoginFlowWithDefaults()
	flow.Id = flowId
	flow.ReturnTo = &returnTo

	flowBody := new(kClient.UpdateLoginFlowBody)
	flowBody.UpdateLoginFlowWithPasswordMethod = kClient.NewUpdateLoginFlowWithPasswordMethod("user@example.com", "password", "password")

	req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_LOGIN_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("flow", flowId)
	req.URL.RawQuery = values.Encode()
	req.Header.Set("Accept", mfaTestAccept)

	mockCookieManager.EXPECT().GetStateCookie(gomock.Any()).Return(cookies.FlowStateCookie{}, nil)

	gomock.InOrder(
		mockService.EXPECT().GetLoginFlow(gomock.Any(), flowId, req.Cookies()).Return(flow, nil, nil),
		mockService.EXPECT().ParseLoginFlowMethodBody(gomock.Any(), gomock.Any()).Return(flowBody, req.Cookies(), nil),
		mockService.EXPECT().CheckAllowedProvider(gomock.Any(), flow, flowBody).Return(true, nil),
		mockService.EXPECT().UpdateLoginFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(&BrowserLocationChangeRequired{RedirectTo: &kratosRedirect}, nil, req.Cookies(), nil),
		mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil),
		mockPolicy.EXPECT().For(SignIn{Methods: []string{"password", "lookup_secret"}}).Return(Requirement{RegenerateBackupCodes: true}),
		mockService.EXPECT().HasNotEnoughLookupSecretsLeft(gomock.Any(), mfaTestIdentityID).Return(true, nil),
	)

	mockCookieManager.EXPECT().SetStateCookie(gomock.Any(), cookies.FlowStateCookie{BackupCodeUsed: true}).Return(nil)

	expectSpan(mockTracer, "kratos.API.shouldEnforceVerificationWithSession", 1)
	expectSpan(mockTracer, "kratos.API.shouldEnforceMFAWithSession", 1)
	expectSpan(mockTracer, "kratos.API.shouldRegenerateBackupCodesWithSession", 1)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, false, tenants.NewNoOpTenantResolver(), BASE_URL, mockCookieManager, mockTracer, mockLogger, WithSecondFactorPolicy(mockPolicy)).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	assertMFARedirect(t, res, RegenerateBackupCodesError, "/ui/backup_codes_regenerate?flow="+flowId+"&return_to="+url.QueryEscape(returnTo))
}

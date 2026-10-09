// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	kClient "github.com/ory/kratos-client-go/v25"
	"go.uber.org/mock/gomock"

	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

// expectMFA expects a session login at testTenant, whose sign-in context is
// context, to need MFA.
func (m *apiMocks) expectMFA(context *SignInContext) {
	m.expectStateCookie(stateFor(testTenant), testTenant)
	m.hydra.EXPECT().LoginRequest(gomock.Any(), testLoginChallenge).Times(1).Return(&LoginRequest{}, nil)
	m.directory.EXPECT().SignInContext(gomock.Any(), testTenant, "", "iid").Times(1).Return(context, nil)
}

func TestAskMFAWithoutSecondFactor(t *testing.T) {
	tests := []struct {
		name       string
		session    *kClient.Session
		provenance string
		context    *SignInContext
	}{
		{name: "password", session: passwordSession("s1"), context: member(EnforcementOff, MFARequirementNone)},
		{name: "company sign-in at a tenant with MFA", session: companySession("s1"), provenance: connA, context: member(EnforcementRequired, MFARequirementRequired, connA)},
		// the account joins the tenant once MFA is done
		{name: "company sign-in admitted by auto-join at a tenant with MFA", session: companySession("s1"), provenance: connA, context: autoJoinAdmitted(MFARequirementRequired, connA)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)
			r := withCookies(createLoginFlowRequest(), func(w http.ResponseWriter) {
				if tt.provenance != "" {
					_ = mocks.store.SetProvenance(w, ProvenanceCookie{SessionID: "s1", ConnectionID: tt.provenance}, tt.session)
				}
			})

			mocks.expectMFA(tt.context)
			mocks.kratos.EXPECT().HasWebAuthnAvailable(gomock.Any(), "iid").Times(1).Return(false, nil)
			mocks.kratos.EXPECT().HasTOTPAvailable(gomock.Any(), "iid").Times(1).Return(false, nil)
			mocks.identities.EXPECT().HasRecoveryCodes(gomock.Any(), "iid").Times(1).Return(false, nil)

			rec := httptest.NewRecorder()
			api.HandleSessionLogin(rec, r, tt.session, testLoginChallenge)

			// the first setup trusts the first factor
			expectErrorID(t, rec, kratos.TOTP_REGISTRATION_REQUIRED)
			expectedRedirect := "/ui/setup_secure?" + url.Values{"return_to": {BASE_URL + "/ui/login?login_challenge=" + testLoginChallenge}}.Encode()
			expectRedirectTo(t, rec, expectedRedirect)
		})
	}
}

func TestAskMFAWithSecondFactor(t *testing.T) {
	tests := []struct {
		name          string
		totp          bool
		recoveryCodes bool
	}{
		{name: "authenticator app", totp: true},
		// recovery codes are a second factor: no authenticator is set up on the first factor alone
		{name: "recovery codes only", recoveryCodes: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			mocks.expectMFA(member(EnforcementOff, MFARequirementNone))
			mocks.kratos.EXPECT().HasWebAuthnAvailable(gomock.Any(), "iid").Times(1).Return(false, nil)
			mocks.kratos.EXPECT().HasTOTPAvailable(gomock.Any(), "iid").Times(1).Return(tt.totp, nil)
			if !tt.totp {
				mocks.identities.EXPECT().HasRecoveryCodes(gomock.Any(), "iid").Times(1).Return(tt.recoveryCodes, nil)
			}

			rec := httptest.NewRecorder()
			api.HandleSessionLogin(rec, createLoginFlowRequest(), passwordSession("s1"), testLoginChallenge)

			expectErrorID(t, rec, "session_aal2_required")
			expectedRedirect := "/ui/login?" + url.Values{"aal": {"aal2"}, "return_to": {BASE_URL + "/ui/login?login_challenge=" + testLoginChallenge}}.Encode()
			expectRedirectTo(t, rec, expectedRedirect)
		})
	}
}

func TestAskMFAWithAAL2Session(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)
	session := passwordSession("s1")
	session.SetAuthenticatorAssuranceLevel(kClient.AUTHENTICATORASSURANCELEVEL_AAL2)

	mocks.kratos.EXPECT().HasWebAuthnAvailable(gomock.Any(), "iid").Times(1).Return(true, nil)

	rec := httptest.NewRecorder()
	api.askMFA(rec, createLoginFlowRequest(), session, testLoginChallenge)

	// Kratos needs refresh to ask a session at aal2 again
	expectedRedirect := "/ui/login?" + url.Values{"aal": {"aal2"}, "refresh": {"true"}, "return_to": {BASE_URL + "/ui/login?login_challenge=" + testLoginChallenge}}.Encode()
	expectRedirectTo(t, rec, expectedRedirect)
}

func TestAskMFAFailOnHasTOTPAvailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)

	mocks.expectMFA(member(EnforcementOff, MFARequirementNone))
	mocks.kratos.EXPECT().HasWebAuthnAvailable(gomock.Any(), "iid").Times(1).Return(false, nil)
	mocks.kratos.EXPECT().HasTOTPAvailable(gomock.Any(), "iid").Times(1).Return(false, errors.New("error"))

	rec := httptest.NewRecorder()
	api.HandleSessionLogin(rec, createLoginFlowRequest(), passwordSession("s1"), testLoginChallenge)

	expectStatus(t, rec, http.StatusInternalServerError)
}

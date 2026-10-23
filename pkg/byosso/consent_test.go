// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"errors"
	"testing"

	hClient "github.com/ory/hydra-client-go/v26"
	kClient "github.com/ory/kratos-client-go/v25"
	"go.uber.org/mock/gomock"
)

func consentRequest(subject string, loginContext interface{}) *hClient.OAuth2ConsentRequest {
	consent := hClient.NewOAuth2ConsentRequest("test.challenge")
	consent.SetSubject(subject)
	consent.Context = loginContext
	return consent
}

func TestConsentRefusal(t *testing.T) {
	tests := []struct {
		name            string
		session         *kClient.Session
		consent         *hClient.OAuth2ConsentRequest
		expectedRefused bool
	}{
		{
			name:    "login accepted by login-ui",
			session: passwordSession("s1"),
			consent: consentRequest("iid", map[string]interface{}{"tenant_id": testTenant}),
		},
		{
			name:            "login of another account",
			session:         passwordSession("s1"),
			consent:         consentRequest("other", map[string]interface{}{"tenant_id": testTenant}),
			expectedRefused: true,
		},
		{
			// a login Kratos accepted itself carries no context
			name:            "login without context",
			session:         passwordSession("s1"),
			consent:         consentRequest("iid", nil),
			expectedRefused: true,
		},
		{
			name:            "login without tenant",
			session:         passwordSession("s1"),
			consent:         consentRequest("iid", map[string]interface{}{"tenant_id": ""}),
			expectedRefused: true,
		},
		{
			name:            "login with a context of another type",
			session:         passwordSession("s1"),
			consent:         consentRequest("iid", "tenant_id"),
			expectedRefused: true,
		},
		{
			name:            "no session",
			session:         nil,
			consent:         consentRequest("iid", map[string]interface{}{"tenant_id": testTenant}),
			expectedRefused: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if refused := consentRefusal(tt.session, tt.consent) != ""; refused != tt.expectedRefused {
				t.Fatalf("expected refused %v, got %v", tt.expectedRefused, refused)
			}
		})
	}
}

func TestGateConsent(t *testing.T) {
	tests := []struct {
		name             string
		consent          *hClient.OAuth2ConsentRequest
		expectedReason   string
		expectedRedirect string
	}{
		{
			// Hydra is not called for a consent that goes on
			name:    "login accepted by login-ui",
			consent: consentRequest("iid", map[string]interface{}{"tenant_id": testTenant}),
		},
		{
			name:             "login without context",
			consent:          consentRequest("iid", nil),
			expectedReason:   "this sign-in did not go through the portal's sign-in checks",
			expectedRedirect: "https://app.example/callback?error=login_required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			api, mocks := newTestAPI(t, ctrl)

			if tt.expectedReason != "" {
				// a refusal is logged as a warning
				mocks.logger.EXPECT().Warnf(gomock.Any(), gomock.Any()).Times(1)
				mocks.hydra.EXPECT().RejectConsent(underBudget{}, "test.challenge", tt.expectedReason).Times(1).Return(tt.expectedRedirect, nil)
			}

			redirectTo, err := api.GateConsent(context.Background(), passwordSession("s1"), tt.consent)

			if redirectTo != tt.expectedRedirect {
				t.Fatalf("expected redirect to %q, got %q", tt.expectedRedirect, redirectTo)
			}
			expectNoError(t, err)
		})
	}
}

func TestGateConsentFailOnRejectConsent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	api, mocks := newTestAPI(t, ctrl)

	mocks.logger.EXPECT().Warnf(gomock.Any(), gomock.Any()).Times(1)
	mocks.hydra.EXPECT().RejectConsent(gomock.Any(), "test.challenge", "this sign-in belongs to another account").Times(1).Return("", errors.New("error"))

	redirectTo, err := api.GateConsent(context.Background(), passwordSession("s1"), consentRequest("other", map[string]interface{}{"tenant_id": testTenant}))

	if redirectTo != "" {
		t.Fatalf("expected no redirect, got %s", redirectTo)
	}
	if err == nil {
		t.Fatalf("expected error not nil")
	}
}

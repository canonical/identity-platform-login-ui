// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package extra

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/mock/gomock"

	hClient "github.com/ory/hydra-client-go/v26"
	kClient "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

// TestHandleConsentMFA pins the assurance level the MFA_ENABLED and
// OIDC_WEBAUTHN_SEQUENCING_ENABLED settings ask of a session before consent
// is given. The mocks are strict, so a call, log line or span that a row does
// not expect fails it.
func TestHandleConsentMFA(t *testing.T) {
	aal1 := kClient.AUTHENTICATORASSURANCELEVEL_AAL1
	aal2 := kClient.AUTHENTICATORASSURANCELEVEL_AAL2

	tests := []struct {
		name                          string
		mfaEnabled                    bool
		oidcWebAuthnSequencingEnabled bool
		noSession                     bool
		methods                       []string
		aal                           *kClient.AuthenticatorAssuranceLevel
		accepted                      bool
	}{
		{
			name:      "no session is refused",
			noSession: true,
		},
		{
			name:       "mfa: password at aal2 is accepted",
			mfaEnabled: true,
			methods:    []string{"password", "totp"},
			aal:        &aal2,
			accepted:   true,
		},
		{
			name:       "mfa: webauthn at aal1 is refused",
			mfaEnabled: true,
			methods:    []string{"webauthn"},
			aal:        &aal1,
		},
		{
			name:       "mfa: oidc at aal1 is accepted",
			mfaEnabled: true,
			methods:    []string{"oidc"},
			aal:        &aal1,
			accepted:   true,
		},
		{
			name:       "mfa: password then oidc at aal1 is refused",
			mfaEnabled: true,
			methods:    []string{"password", "oidc"},
			aal:        &aal1,
		},
		{
			name:                          "mfa and sequencing: a passkey at aal1 is accepted",
			mfaEnabled:                    true,
			oidcWebAuthnSequencingEnabled: true,
			methods:                       []string{"passkey"},
			aal:                           &aal1,
			accepted:                      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockService := NewMockServiceInterface(ctrl)
			mockKratosService := kratos.NewMockServiceInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)

			var session *kClient.Session
			if !tt.noSession {
				session = kClient.NewSession("test")
				session.Identity = kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
				session.AuthenticatorAssuranceLevel = tt.aal
				for _, method := range tt.methods {
					session.AuthenticationMethods = append(session.AuthenticationMethods, kClient.SessionAuthenticationMethod{Method: &method})
				}
			}

			consent := hClient.NewOAuth2ConsentRequest("challenge")
			accept := hClient.NewOAuth2RedirectTo("test")

			req := httptest.NewRequest(http.MethodGet, "/api/consent", nil)
			values := req.URL.Query()
			values.Add("consent_challenge", "7bb518c4eec2454dbb289f5fdb4c0ee2")
			req.URL.RawQuery = values.Encode()

			mockKratosService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil)
			if tt.accepted {
				mockService.EXPECT().GetConsent(gomock.Any(), "7bb518c4eec2454dbb289f5fdb4c0ee2").Return(consent, nil)
				mockService.EXPECT().AcceptConsent(gomock.Any(), *session.Identity, consent, "").Return(accept, nil)
			} else {
				mockLogger.EXPECT().Errorf("insufficient session aal, this indicates a misconfiguration in kratos")
			}

			w := httptest.NewRecorder()
			mux := chi.NewMux()
			NewAPI(mockService, mockKratosService, BASE_URL, tt.mfaEnabled, tt.oidcWebAuthnSequencingEnabled, mockTracer, mockLogger).RegisterEndpoints(mux)

			mux.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()

			data, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatalf("expected error to be nil got %v", err)
			}

			if tt.accepted {
				if res.StatusCode != http.StatusOK {
					t.Fatalf("expected HTTP status code 200 got %v", res.StatusCode)
				}
				return
			}

			if res.StatusCode != http.StatusForbidden {
				t.Fatalf("expected HTTP status code 403 got %v", res.StatusCode)
			}

			if string(data) != "insufficient session aal\n" {
				t.Errorf("expected body %q got %q", "insufficient session aal\n", data)
			}
		})
	}
}

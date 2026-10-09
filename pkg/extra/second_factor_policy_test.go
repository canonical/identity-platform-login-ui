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

// TestHandleConsentAsksSecondFactorPolicy gives the API a policy of its own,
// with both settings off, and checks that the consent handler follows what
// that policy asks.
func TestHandleConsentAsksSecondFactorPolicy(t *testing.T) {
	tests := []struct {
		name     string
		methods  []string
		aal      kClient.AuthenticatorAssuranceLevel
		accepted bool
	}{
		{
			name:    "a required second factor refuses aal1",
			methods: []string{"oidc"},
			aal:     kClient.AUTHENTICATORASSURANCELEVEL_AAL1,
		},
		{
			name:     "a required second factor accepts aal2",
			methods:  []string{"password", "totp"},
			aal:      kClient.AUTHENTICATORASSURANCELEVEL_AAL2,
			accepted: true,
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
			mockPolicy := NewMockSecondFactorPolicyInterface(ctrl)

			session := kClient.NewSession("test")
			session.Identity = kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
			session.AuthenticatorAssuranceLevel = &tt.aal
			for _, method := range tt.methods {
				session.AuthenticationMethods = append(session.AuthenticationMethods, kClient.SessionAuthenticationMethod{Method: &method})
			}

			consent := hClient.NewOAuth2ConsentRequest("challenge")
			accept := hClient.NewOAuth2RedirectTo("test")

			req := httptest.NewRequest(http.MethodGet, "/api/consent", nil)
			values := req.URL.Query()
			values.Add("consent_challenge", "7bb518c4eec2454dbb289f5fdb4c0ee2")
			req.URL.RawQuery = values.Encode()

			calls := []any{
				mockKratosService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil),
				mockPolicy.EXPECT().For(kratos.SignIn{Methods: tt.methods}).Return(kratos.Requirement{SecondFactor: true}),
			}

			if tt.accepted {
				calls = append(
					calls,
					mockService.EXPECT().GetConsent(gomock.Any(), "7bb518c4eec2454dbb289f5fdb4c0ee2").Return(consent, nil),
					mockService.EXPECT().AcceptConsent(gomock.Any(), *session.Identity, consent, "").Return(accept, nil),
				)
			} else {
				mockLogger.EXPECT().Errorf("insufficient session aal, this indicates a misconfiguration in kratos")
			}

			gomock.InOrder(calls...)

			w := httptest.NewRecorder()
			mux := chi.NewMux()
			NewAPI(mockService, mockKratosService, BASE_URL, false, false, mockTracer, mockLogger, WithSecondFactorPolicy(mockPolicy)).RegisterEndpoints(mux)

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

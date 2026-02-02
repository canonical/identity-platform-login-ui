// Copyright 2024 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

// Package kratos provides unit tests for Kratos service functionality.
package kratos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	reflect "reflect"
	"strings"
	"testing"
	"time"

	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go/v25"
	"go.opentelemetry.io/otel/trace"
	gomock "go.uber.org/mock/gomock"

	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
)

//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_interfaces.go -source=./interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_monitor.go -source=../../internal/monitoring/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_tracing.go -source=../../internal/tracing/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_kratos.go github.com/ory/kratos-client-go/v25 FrontendAPI
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_identity.go github.com/ory/kratos-client-go/v25 IdentityAPI
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_hydra.go -source=../../internal/hydra/interfaces.go

func TestCheckSession(t *testing.T) {
	tests := []struct {
		name           string
		cookies        []*http.Cookie
		mockError      error
		mockSession    *kClient.Session
		expectError    bool
		expectNilRes   bool
		validateCookie bool
	}{
		{
			name:    "success with valid cookies",
			cookies: []*http.Cookie{{Name: "test", Value: "test"}},
			mockSession: func() *kClient.Session {
				s := kClient.NewSession("test")
				s.Identity = kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
				return s
			}(),
			mockError:      nil,
			expectError:    false,
			expectNilRes:   false,
			validateCookie: true,
		},
		{
			name:           "failure on API error",
			cookies:        []*http.Cookie{{Name: "test", Value: "test"}},
			mockSession:    nil,
			mockError:      fmt.Errorf("error"),
			expectError:    true,
			expectNilRes:   true,
			validateCookie: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
			mockKratosFrontendApi := NewMockFrontendAPI(ctrl)

			ctx := context.Background()
			sessionRequest := kClient.FrontendAPIToSessionRequest{
				ApiService: mockKratosFrontendApi,
			}
			resp := http.Response{
				Header: http.Header{"Set-Cookie": []string{tt.cookies[0].Raw}},
			}

			mockTracer.EXPECT().Start(ctx, "kratos.Service.ToSession").Times(1).Return(ctx, trace.SpanFromContext(ctx))
			mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
			mockKratosFrontendApi.EXPECT().ToSession(ctx).Times(1).Return(sessionRequest)
			mockKratosFrontendApi.EXPECT().ToSessionExecute(gomock.Any()).Times(1).DoAndReturn(
				func(r kClient.FrontendAPIToSessionRequest) (*kClient.Session, *http.Response, error) {
					if tt.validateCookie {
						if cookie := (*string)(reflect.ValueOf(r).FieldByName("cookie").UnsafePointer()); *cookie != "test=test" {
							t.Fatalf("expected cookie string as test=test, got %s", *cookie)
						}
					}

					if tt.mockError != nil {
						return nil, new(http.Response), tt.mockError
					}
					return tt.mockSession, &resp, nil
				},
			)

			s, c, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).CheckSession(ctx, tt.cookies)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error but got %v", err)
				}
			}

			if tt.expectNilRes {
				if s != nil {
					t.Fatalf("expected session to be nil but got %v", s)
				}
				if c != nil {
					t.Fatalf("expected cookies to be nil but got %v", c)
				}
			} else {
				if s != tt.mockSession {
					t.Fatalf("expected session to be %v but got %v", tt.mockSession, s)
				}
				if !reflect.DeepEqual(c, resp.Cookies()) {
					t.Fatalf("expected cookies to be %v but got %v", resp.Cookies(), c)
				}
			}
		})
	}
}

func TestAcceptLoginRequest(t *testing.T) {
	tests := []struct {
		name              string
		loginChallenge    string
		identityID        string
		sessionExpiry     *time.Time
		mockError         error
		expectError       bool
		validateTimestamp bool
	}{
		{
			name:           "success with session expiry",
			loginChallenge: "123456",
			identityID:     "id",
			sessionExpiry: func() *time.Time {
				t := time.Now().Add(300 * time.Second)
				return &t
			}(),
			mockError:         nil,
			expectError:       false,
			validateTimestamp: true,
		},
		{
			name:              "failure on API error",
			loginChallenge:    "123456",
			identityID:        "test",
			sessionExpiry:     nil,
			mockError:         fmt.Errorf("error"),
			expectError:       true,
			validateTimestamp: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
			mockHydraOauthApi := NewMockOAuth2API(ctrl)

			ctx := context.Background()
			redirectTo := hClient.NewOAuth2RedirectTo("http://redirect/to/path")
			acceptLoginRequest := hClient.OAuth2APIAcceptOAuth2LoginRequestRequest{
				ApiService: mockHydraOauthApi,
			}
			session := kClient.NewSession("test")
			session.Identity = kClient.NewIdentity(tt.identityID, "test.json", "https://test.com/test.json", map[string]string{"name": "name"})

			if tt.sessionExpiry != nil {
				session.SetExpiresAt(*tt.sessionExpiry)
			}

			resp := new(http.Response)

			mockTracer.EXPECT().Start(ctx, gomock.Any()).Times(1).Return(ctx, trace.SpanFromContext(ctx))
			mockHydra.EXPECT().OAuth2API().Times(1).Return(mockHydraOauthApi)
			mockHydraOauthApi.EXPECT().AcceptOAuth2LoginRequest(ctx).Times(1).Return(acceptLoginRequest)
			mockHydraOauthApi.EXPECT().AcceptOAuth2LoginRequestExecute(gomock.Any()).Times(1).DoAndReturn(
				func(r hClient.OAuth2APIAcceptOAuth2LoginRequestRequest) (*hClient.OAuth2RedirectTo, *http.Response, error) {
					if tt.mockError != nil {
						return nil, nil, tt.mockError
					}

					lc := (*string)(reflect.ValueOf(r).FieldByName("loginChallenge").UnsafePointer())
					if *lc != tt.loginChallenge {
						t.Fatalf("expected loginChallenge to be %s, got %s", tt.loginChallenge, *lc)
					}

					acceptReq := (*hClient.AcceptOAuth2LoginRequest)(reflect.ValueOf(r).FieldByName("acceptOAuth2LoginRequest").UnsafePointer())
					if acceptReq.Subject != tt.identityID {
						t.Fatalf("expected identityID to be %s, got %s", tt.identityID, acceptReq.Subject)
					}

					if tt.validateTimestamp {
						leeway := int64(2)
						if 300-acceptReq.GetRememberFor() > leeway {
							t.Fatalf("expected RememberFor to be close to 300, got %v", acceptReq.GetRememberFor())
						}
					}

					if acceptReq.GetIdentityProviderSessionId() != session.GetId() {
						t.Fatalf("expected session ID to be %s, got %s", session.GetId(), acceptReq.GetIdentityProviderSessionId())
					}

					return redirectTo, resp, nil
				},
			)

			rt, c, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).AcceptLoginRequest(ctx, session, tt.loginChallenge)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if rt != nil || c != nil {
					t.Fatal("expected nil response on error")
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error but got %v", err)
				}
				if rt == nil {
					t.Fatal("expected redirect but got nil")
				}
				if !reflect.DeepEqual(c, resp.Cookies()) {
					t.Fatalf("expected cookies to match")
				}
			}
		})
	}
}

func TestAcceptLoginRequestWithPopForWebAuthn(t *testing.T) {
	tests := []struct {
		name                          string
		authMethods                   []string
		oidcWebAuthnSequencingEnabled bool
		expectedAmr                   map[string]bool
	}{
		{
			name:                          "TOTP with flag enabled should not include pop",
			authMethods:                   []string{"totp"},
			oidcWebAuthnSequencingEnabled: true,
			expectedAmr:                   map[string]bool{"totp": true},
		},
		{
			name:                          "WebAuthn with flag enabled should include pop",
			authMethods:                   []string{"webauthn"},
			oidcWebAuthnSequencingEnabled: true,
			expectedAmr:                   map[string]bool{"webauthn": true, "pop": true},
		},
		{
			name:                          "TOTP and WebAuthn with flag enabled should include pop",
			authMethods:                   []string{"totp", "webauthn"},
			oidcWebAuthnSequencingEnabled: true,
			expectedAmr:                   map[string]bool{"totp": true, "webauthn": true, "pop": true},
		},
		{
			name:                          "TOTP with flag disabled should not include pop",
			authMethods:                   []string{"totp"},
			oidcWebAuthnSequencingEnabled: false,
			expectedAmr:                   map[string]bool{"totp": true},
		},
		{
			name:                          "WebAuthn with flag disabled should not include pop",
			authMethods:                   []string{"webauthn"},
			oidcWebAuthnSequencingEnabled: false,
			expectedAmr:                   map[string]bool{"webauthn": true},
		},
		{
			name:                          "Password should not include pop even with flag enabled",
			authMethods:                   []string{"password"},
			oidcWebAuthnSequencingEnabled: true,
			expectedAmr:                   map[string]bool{"password": true},
		},
		{
			name:                          "OIDC should not include pop even with flag enabled",
			authMethods:                   []string{"oidc"},
			oidcWebAuthnSequencingEnabled: true,
			expectedAmr:                   map[string]bool{"oidc": true},
		},
		{
			name:                          "Multiple methods with TOTP should not include pop",
			authMethods:                   []string{"password", "totp"},
			oidcWebAuthnSequencingEnabled: true,
			expectedAmr:                   map[string]bool{"password": true, "totp": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
			mockHydraOauthApi := NewMockOAuth2API(ctrl)

			ctx := context.Background()
			loginChallenge := "123456"
			identityID := "id"
			redirectTo := hClient.NewOAuth2RedirectTo("http://redirect/to/path")
			acceptLoginRequest := hClient.OAuth2APIAcceptOAuth2LoginRequestRequest{
				ApiService: mockHydraOauthApi,
			}
			session := kClient.NewSession("test")
			session.Identity = kClient.NewIdentity(identityID, "test.json", "https://test.com/test.json", map[string]string{"name": "name"})

			// Set up authentication methods
			var authMethods []kClient.SessionAuthenticationMethod
			for _, method := range tt.authMethods {
				m := method
				authMethods = append(authMethods, kClient.SessionAuthenticationMethod{Method: &m})
			}
			session.AuthenticationMethods = authMethods

			resp := new(http.Response)

			mockTracer.EXPECT().Start(ctx, gomock.Any()).Times(1).Return(ctx, trace.SpanFromContext(ctx))
			mockHydra.EXPECT().OAuth2API().Times(1).Return(mockHydraOauthApi)
			mockHydraOauthApi.EXPECT().AcceptOAuth2LoginRequest(ctx).Times(1).Return(acceptLoginRequest)
			mockHydraOauthApi.EXPECT().AcceptOAuth2LoginRequestExecute(gomock.Any()).Times(1).DoAndReturn(
				func(r hClient.OAuth2APIAcceptOAuth2LoginRequestRequest) (*hClient.OAuth2RedirectTo, *http.Response, error) {
					acceptReq := (*hClient.AcceptOAuth2LoginRequest)(reflect.ValueOf(r).FieldByName("acceptOAuth2LoginRequest").UnsafePointer())

					// Verify all expected AMR values are present
					amrMap := make(map[string]bool)
					for _, amr := range acceptReq.Amr {
						amrMap[amr] = true
					}

					if len(amrMap) != len(tt.expectedAmr) {
						t.Fatalf("expected %d AMR values, got %d. Expected: %v, Got: %v", len(tt.expectedAmr), len(amrMap), tt.expectedAmr, amrMap)
					}

					for expectedAmr := range tt.expectedAmr {
						if !amrMap[expectedAmr] {
							t.Fatalf("expected AMR to contain %s, got %v", expectedAmr, acceptReq.Amr)
						}
					}

					return redirectTo, resp, nil
				},
			)

			_, _, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, tt.oidcWebAuthnSequencingEnabled, mockTracer, mockMonitor, mockLogger).AcceptLoginRequest(ctx, session, loginChallenge)

			if err != nil {
				t.Fatalf("expected error to be nil not %v", err)
			}
		})
	}
}

func TestGetLoginRequest(t *testing.T) {
	tests := []struct {
		name            string
		loginChallenge  string
		mockError       error
		expectError     bool
		validateRequest bool
	}{
		{
			name:            "success with valid login challenge",
			loginChallenge:  "123456",
			mockError:       nil,
			expectError:     false,
			validateRequest: true,
		},
		{
			name:            "failure on API error",
			loginChallenge:  "123456",
			mockError:       fmt.Errorf("error"),
			expectError:     true,
			validateRequest: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
			mockHydraOauthApi := NewMockOAuth2API(ctrl)

			ctx := context.Background()
			getLoginRequest := hClient.OAuth2APIGetOAuth2LoginRequestRequest{
				ApiService: mockHydraOauthApi,
			}
			lr := hClient.NewOAuth2LoginRequestWithDefaults()
			resp := new(http.Response)

			mockTracer.EXPECT().Start(ctx, gomock.Any()).Times(1).Return(ctx, trace.SpanFromContext(ctx))
			mockHydra.EXPECT().OAuth2API().Times(1).Return(mockHydraOauthApi)
			mockHydraOauthApi.EXPECT().GetOAuth2LoginRequest(ctx).Times(1).Return(getLoginRequest)
			mockHydraOauthApi.EXPECT().GetOAuth2LoginRequestExecute(gomock.Any()).Times(1).DoAndReturn(
				func(r hClient.OAuth2APIGetOAuth2LoginRequestRequest) (*hClient.OAuth2LoginRequest, *http.Response, error) {
					if tt.mockError != nil {
						return nil, nil, tt.mockError
					}

					if tt.validateRequest {
						lc := (*string)(reflect.ValueOf(r).FieldByName("loginChallenge").UnsafePointer())
						if *lc != tt.loginChallenge {
							t.Fatalf("expected loginChallenge to be %s, got %s", tt.loginChallenge, *lc)
						}
					}
					return lr, resp, nil
				},
			)

			ret, c, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).GetLoginRequest(ctx, tt.loginChallenge)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if ret != nil || c != nil {
					t.Fatal("expected nil response on error")
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error but got %v", err)
				}
				if ret != lr {
					t.Fatalf("expected response to be %v but got %v", lr, ret)
				}
				if !reflect.DeepEqual(c, resp.Cookies()) {
					t.Fatalf("expected cookies to match")
				}
			}
		})
	}
}

func TestMustReAuthenticate(t *testing.T) {
	tests := []struct {
		name           string
		loginChallenge string
		session        *kClient.Session
		state          FlowStateCookie
		hydraSkip      bool
		mockError      error
		expectResult   bool
		expectError    bool
		needsHydraCall bool
	}{
		{
			name:           "skip with totp setup",
			loginChallenge: "123456",
			session: func() *kClient.Session {
				s := kClient.NewSession("test")
				s.Identity = kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
				return s
			}(),
			state:          FlowStateCookie{LoginChallengeHash: "1234", TotpSetup: false},
			hydraSkip:      true,
			mockError:      nil,
			expectResult:   false,
			expectError:    false,
			needsHydraCall: true,
		},
		{
			name:           "skip with backup code used",
			loginChallenge: "123456",
			session: func() *kClient.Session {
				s := kClient.NewSession("test")
				s.Identity = kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
				return s
			}(),
			state:          FlowStateCookie{LoginChallengeHash: "1234", BackupCodeUsed: true},
			hydraSkip:      true,
			mockError:      nil,
			expectResult:   false,
			expectError:    false,
			needsHydraCall: true,
		},
		{
			name:           "no skip required",
			loginChallenge: "123456",
			session: func() *kClient.Session {
				s := kClient.NewSession("test")
				s.Identity = kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
				return s
			}(),
			state:          FlowStateCookie{LoginChallengeHash: "1234", TotpSetup: false},
			hydraSkip:      false,
			mockError:      nil,
			expectResult:   true,
			expectError:    false,
			needsHydraCall: true,
		},
		{
			name:           "no login challenge",
			loginChallenge: "",
			session: func() *kClient.Session {
				s := kClient.NewSession("test")
				s.Identity = kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
				return s
			}(),
			state:          FlowStateCookie{},
			hydraSkip:      false,
			mockError:      nil,
			expectResult:   true,
			expectError:    false,
			needsHydraCall: false,
		},
		{
			name:           "no session",
			loginChallenge: "123456",
			session:        nil,
			state:          FlowStateCookie{},
			hydraSkip:      false,
			mockError:      nil,
			expectResult:   true,
			expectError:    false,
			needsHydraCall: false,
		},
		{
			name:           "API error",
			loginChallenge: "123456",
			session: func() *kClient.Session {
				s := kClient.NewSession("test")
				s.Identity = kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
				return s
			}(),
			state:          FlowStateCookie{},
			hydraSkip:      false,
			mockError:      fmt.Errorf("error"),
			expectResult:   true,
			expectError:    true,
			needsHydraCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
			mockHydraOauthApi := NewMockOAuth2API(ctrl)

			ctx := context.Background()

			mockTracer.EXPECT().Start(ctx, "kratos.Service.MustReAuthenticate").Times(1).Return(ctx, trace.SpanFromContext(ctx))

			if tt.needsHydraCall {
				sessionId := "1234"
				getLoginRequest := hClient.OAuth2APIGetOAuth2LoginRequestRequest{
					ApiService: mockHydraOauthApi,
				}
				lr := hClient.NewOAuth2LoginRequestWithDefaults()
				lr.Skip = tt.hydraSkip
				lr.SessionId = &sessionId
				resp := new(http.Response)

				mockTracer.EXPECT().Start(ctx, gomock.Any()).Times(1).Return(ctx, trace.SpanFromContext(ctx))
				mockHydra.EXPECT().OAuth2API().Times(1).Return(mockHydraOauthApi)
				mockHydraOauthApi.EXPECT().GetOAuth2LoginRequest(ctx).Times(1).Return(getLoginRequest)
				mockHydraOauthApi.EXPECT().GetOAuth2LoginRequestExecute(gomock.Any()).Times(1).DoAndReturn(
					func(r hClient.OAuth2APIGetOAuth2LoginRequestRequest) (*hClient.OAuth2LoginRequest, *http.Response, error) {
						if tt.mockError != nil {
							return nil, nil, tt.mockError
						}
						lc := (*string)(reflect.ValueOf(r).FieldByName("loginChallenge").UnsafePointer())
						if *lc != tt.loginChallenge {
							t.Fatalf("expected loginChallenge to be %s, got %s", tt.loginChallenge, *lc)
						}
						return lr, resp, nil
					},
				)
			}

			ret, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).
				MustReAuthenticate(ctx, tt.loginChallenge, tt.session, tt.state)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error but got %v", err)
				}
			}

			if ret != tt.expectResult {
				t.Fatalf("expected result to be %v but got %v", tt.expectResult, ret)
			}
		})
	}
}

func TestCreateBrowserLoginFlow(t *testing.T) {
	tests := []struct {
		name                          string
		aal                           string
		returnTo                      string
		loginChallenge                string
		refresh                       bool
		oidcWebAuthnSequencingEnabled bool
		shouldHydrate                 bool
		shouldFail                    bool
		expectNil                     bool
		expectLoginChallengeNil       bool
	}{
		{
			name:                          "With login challenge success",
			aal:                           "aal",
			returnTo:                      "https://return/to/somewhere",
			loginChallenge:                "123456",
			refresh:                       false,
			oidcWebAuthnSequencingEnabled: false,
			shouldHydrate:                 true,
			shouldFail:                    false,
			expectNil:                     false,
			expectLoginChallengeNil:       false,
		},
		{
			name:                          "With return to success",
			aal:                           "aal",
			returnTo:                      "https://return/to/somewhere",
			loginChallenge:                "",
			refresh:                       false,
			oidcWebAuthnSequencingEnabled: false,
			shouldHydrate:                 true,
			shouldFail:                    false,
			expectNil:                     false,
			expectLoginChallengeNil:       false,
		},
		{
			name:                          "With sequencing and login challenge",
			aal:                           "aal",
			returnTo:                      "https://return/to/somewhere",
			loginChallenge:                "123456",
			refresh:                       false,
			oidcWebAuthnSequencingEnabled: true,
			shouldHydrate:                 true,
			shouldFail:                    false,
			expectNil:                     false,
			expectLoginChallengeNil:       true,
		},
		{
			name:                          "Without return to and login challenge",
			aal:                           "aal",
			returnTo:                      "",
			loginChallenge:                "",
			refresh:                       false,
			oidcWebAuthnSequencingEnabled: false,
			shouldHydrate:                 false,
			shouldFail:                    false,
			expectNil:                     true,
			expectLoginChallengeNil:       false,
		},
		{
			name:                          "Fail case",
			aal:                           "aal",
			returnTo:                      "https://return/to/somewhere",
			loginChallenge:                "123456",
			refresh:                       false,
			oidcWebAuthnSequencingEnabled: false,
			shouldHydrate:                 false,
			shouldFail:                    true,
			expectNil:                     true,
			expectLoginChallengeNil:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
			mockKratosFrontendApi := NewMockFrontendAPI(ctrl)

			ctx := context.Background()
			cookies := make([]*http.Cookie, 0)
			cookie := &http.Cookie{Name: "test", Value: "test"}
			cookies = append(cookies, cookie)
			flow := kClient.NewLoginFlowWithDefaults()
			request := kClient.FrontendAPICreateBrowserLoginFlowRequest{
				ApiService: mockKratosFrontendApi,
			}
			resp := http.Response{
				Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
			}

			mockTracer.EXPECT().Start(ctx, "kratos.Service.CreateBrowserLoginFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
			if tt.shouldHydrate {
				mockTracer.EXPECT().Start(ctx, "kratos.Service.hydrateKratosLoginFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
			}
			mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
			mockKratosFrontendApi.EXPECT().CreateBrowserLoginFlow(ctx).Times(1).Return(request)

			if tt.shouldFail || !tt.expectNil {
				mockKratosFrontendApi.EXPECT().CreateBrowserLoginFlowExecute(gomock.Any()).Times(1).DoAndReturn(
					func(r kClient.FrontendAPICreateBrowserLoginFlowRequest) (*kClient.LoginFlow, *http.Response, error) {
						if _aal := (*string)(reflect.ValueOf(r).FieldByName("aal").UnsafePointer()); *_aal != tt.aal {
							t.Fatalf("expected aal to be %s, got %s", tt.aal, *_aal)
						}
						if rt := (*string)(reflect.ValueOf(r).FieldByName("returnTo").UnsafePointer()); *rt != tt.returnTo {
							t.Fatalf("expected returnTo to be %s, got %s", tt.returnTo, *rt)
						}
						if tt.expectLoginChallengeNil {
							if lc := (*string)(reflect.ValueOf(r).FieldByName("loginChallenge").UnsafePointer()); lc != nil {
								t.Fatalf("expected loginChallenge to be nil, got %s", *lc)
							}
						} else if tt.loginChallenge != "" {
							if lc := (*string)(reflect.ValueOf(r).FieldByName("loginChallenge").UnsafePointer()); *lc != tt.loginChallenge {
								t.Fatalf("expected loginChallenge to be %s, got %s", tt.loginChallenge, *lc)
							}
						}
						if ref := (*bool)(reflect.ValueOf(r).FieldByName("refresh").UnsafePointer()); *ref != tt.refresh {
							t.Fatalf("expected refresh to be %v, got %v", tt.refresh, *ref)
						}
						if cookie := (*string)(reflect.ValueOf(r).FieldByName("cookie").UnsafePointer()); *cookie != "test=test" {
							t.Fatalf("expected cookie string as test=test, got %s", *cookie)
						}

						if tt.shouldFail {
							return nil, &resp, fmt.Errorf("error")
						}
						return flow, &resp, nil
					},
				)
			}

			f, c, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, tt.oidcWebAuthnSequencingEnabled, mockTracer, mockMonitor, mockLogger).CreateBrowserLoginFlow(ctx, tt.aal, tt.returnTo, tt.loginChallenge, tt.refresh, cookies)

			if tt.expectNil {
				if f != nil {
					t.Fatalf("expected flow to be %v not  %v", nil, f)
				}
				if c != nil {
					t.Fatalf("expected cookies to be %v not  %v", nil, c)
				}
				if err == nil {
					t.Fatalf("expected error not nil")
				}
			} else {
				if f != flow {
					t.Fatalf("expected flow to be %v not  %v", flow, f)
				}
				if !reflect.DeepEqual(c, resp.Cookies()) {
					t.Fatalf("expected cookies to be %v not  %v", resp.Cookies(), c)
				}
				if tt.shouldFail {
					if err == nil {
						t.Fatalf("expected error not nil")
					}
				} else {
					if err != nil {
						t.Fatalf("expected error to be nil not  %v", err)
					}
				}
			}
		})
	}
}

func TestGetLoginFlow(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		shouldFail bool
		expectNil  bool
	}{
		{
			name:       "Success",
			id:         "id",
			shouldFail: false,
			expectNil:  false,
		},
		{
			name:       "Fail",
			id:         "id",
			shouldFail: true,
			expectNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
			mockKratosFrontendApi := NewMockFrontendAPI(ctrl)

			ctx := context.Background()
			cookies := make([]*http.Cookie, 0)
			cookie := &http.Cookie{Name: "test", Value: "test"}
			cookies = append(cookies, cookie)
			flow := kClient.NewLoginFlowWithDefaults()
			request := kClient.FrontendAPIGetLoginFlowRequest{
				ApiService: mockKratosFrontendApi,
			}
			resp := http.Response{
				Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
			}

			mockTracer.EXPECT().Start(ctx, "kratos.Service.GetLoginFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
			if !tt.shouldFail {
				mockTracer.EXPECT().Start(ctx, "kratos.Service.hydrateKratosLoginFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
			}
			mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
			mockKratosFrontendApi.EXPECT().GetLoginFlow(ctx).Times(1).Return(request)
			mockKratosFrontendApi.EXPECT().GetLoginFlowExecute(gomock.Any()).Times(1).DoAndReturn(
				func(r kClient.FrontendAPIGetLoginFlowRequest) (*kClient.LoginFlow, *http.Response, error) {
					if _id := (*string)(reflect.ValueOf(r).FieldByName("id").UnsafePointer()); *_id != tt.id {
						t.Fatalf("expected id to be %s, got %s", tt.id, *_id)
					}
					if cookie := (*string)(reflect.ValueOf(r).FieldByName("cookie").UnsafePointer()); *cookie != "test=test" {
						t.Fatalf("expected cookie string as test=test, got %s", *cookie)
					}

					if tt.shouldFail {
						return nil, &resp, fmt.Errorf("error")
					}
					return flow, &resp, nil
				},
			)

			f, c, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).GetLoginFlow(ctx, tt.id, cookies)

			if tt.expectNil {
				if f != nil {
					t.Fatalf("expected flow to be %v not  %v", nil, f)
				}
				if c != nil {
					t.Fatalf("expected header to be %v not  %v", nil, c)
				}
				if err == nil {
					t.Fatalf("expected error not nil")
				}
			} else {
				if f != flow {
					t.Fatalf("expected flow to be %v not  %v", flow, f)
				}
				if !reflect.DeepEqual(c, resp.Cookies()) {
					t.Fatalf("expected cookies to be %v not  %v", resp.Cookies(), c)
				}
				if err != nil {
					t.Fatalf("expected error to be nil not  %v", err)
				}
			}
		})
	}
}

func TestUpdateIdentifierFirstLoginFlow(t *testing.T) {
	tests := []struct {
		name            string
		csrfToken       *string
		identifier      string
		responseStatus  int
		redirectTo      string
		expectErr       bool
		expectedErrText string
		expectLog       bool
		skipExecute     bool
	}{
		{
			name:           "Success",
			csrfToken:      stringPtr("csrf_token_1234"),
			identifier:     "test@example.com",
			responseStatus: http.StatusSeeOther,
			redirectTo:     "https://redirect/to/path",
			expectErr:      false,
		},
		{
			name:            "Missing CSRF token",
			csrfToken:       nil,
			identifier:      "test@example.com",
			expectErr:       true,
			expectedErrText: "missing csrf token",
			skipExecute:     true,
		},
		{
			name:           "Status bad request",
			csrfToken:      stringPtr("csrf_token_1234"),
			identifier:     "test@example.com",
			responseStatus: http.StatusBadRequest,
			expectErr:      true,
			expectLog:      true,
		},
		{
			name:            "Unexpected status",
			csrfToken:       stringPtr("csrf_token_1234"),
			identifier:      "test@example.com",
			responseStatus:  http.StatusGone,
			expectErr:       true,
			expectedErrText: "unexpected status: 410",
			expectLog:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

			ctx := context.Background()
			cookies := make([]*http.Cookie, 0)
			cookie := &http.Cookie{Name: "test", Value: "test"}
			cookies = append(cookies, cookie)
			flowId := "flow"

			body := kClient.UpdateLoginFlowWithIdentifierFirstMethod{
				CsrfToken:  tt.csrfToken,
				Identifier: tt.identifier,
			}

			mockTracer.EXPECT().Start(ctx, "kratos.Service.UpdateIdentifierFirstLoginFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))

			if !tt.skipExecute {
				resp := &http.Response{
					StatusCode: tt.responseStatus,
					Body:       io.NopCloser(strings.NewReader("")),
				}
				if tt.responseStatus == http.StatusSeeOther {
					resp.Header = http.Header{
						"Location":   []string{tt.redirectTo},
						"Set-Cookie": []string{cookie.String()},
					}
				}

				mockKratos.EXPECT().
					ExecuteIdentifierFirstUpdateLoginRequest(ctx, flowId, *tt.csrfToken, tt.identifier, cookies).
					Return(resp, nil).
					Times(1)
			}

			if tt.expectLog {
				mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)
			}

			r, c, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).UpdateIdentifierFirstLoginFlow(ctx, flowId, body, cookies)

			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.expectedErrText != "" && !strings.Contains(err.Error(), tt.expectedErrText) {
					t.Fatalf("expected %s error, got %v", tt.expectedErrText, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected error to be nil not  %v", err)
				}
				if *r.RedirectTo != tt.redirectTo {
					t.Fatalf("expected redirect URL %s, got %s", tt.redirectTo, *r.RedirectTo)
				}
				if len(c) != len(cookies) {
					t.Fatalf("expected %d cookies, got %d", len(cookies), len(c))
				}
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

func TestUpdateLoginFlow(t *testing.T) {
	tests := []struct {
		name              string
		errorMessageId    int64
		expectedError     string
		statusCode        int
		expectSuccess     bool
		expectLog         bool
		expectParseTracer bool
	}{
		{
			name:              "Success",
			statusCode:        http.StatusUnprocessableEntity,
			expectSuccess:     true,
			expectParseTracer: true,
		},
		{
			name:           "Error WebAuthn not set",
			errorMessageId: MissingSecurityKeySetup,
			expectedError:  "choose a different login method",
			statusCode:     400,
			expectSuccess:  false,
		},
		{
			name:           "Error backup codes not set",
			errorMessageId: MissingBackupCodesSetup,
			expectedError:  "login with backup codes unavailable",
			statusCode:     400,
			expectSuccess:  false,
		},
		{
			name:          "Fail generic",
			statusCode:    200,
			expectSuccess: false,
			expectLog:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
			mockKratosFrontendApi := NewMockFrontendAPI(ctrl)

			ctx := context.Background()
			cookies := make([]*http.Cookie, 0)
			cookie := &http.Cookie{Name: "test", Value: "test"}
			cookies = append(cookies, cookie)
			flowId := "flow"
			body := new(kClient.UpdateLoginFlowBody)
			request := kClient.FrontendAPIUpdateLoginFlowRequest{
				ApiService: mockKratosFrontendApi,
			}

			mockTracer.EXPECT().Start(ctx, "kratos.Service.UpdateLoginFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
			if tt.expectParseTracer {
				mockTracer.EXPECT().Start(ctx, "kratos.Service.parseKratosRedirectResponse").Times(1).Return(ctx, trace.SpanFromContext(ctx))
			}
			mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
			mockKratosFrontendApi.EXPECT().UpdateLoginFlow(ctx).Times(1).Return(request)

			if tt.expectSuccess {
				_redirectTo := "https://redirect/to/path"
				flow := ErrorBrowserLocationChangeRequired{
					RedirectBrowserTo: &_redirectTo,
				}
				flowJson, _ := json.Marshal(flow)
				resp := http.Response{
					Header:     http.Header{"Set-Cookie": []string{cookie.Raw}},
					Body:       io.NopCloser(bytes.NewBuffer(flowJson)),
					StatusCode: tt.statusCode,
				}
				mockKratosFrontendApi.EXPECT().UpdateLoginFlowExecute(gomock.Any()).Times(1).DoAndReturn(
					func(r kClient.FrontendAPIUpdateLoginFlowRequest) (*ErrorBrowserLocationChangeRequired, *http.Response, error) {
						if _flow := (*string)(reflect.ValueOf(r).FieldByName("flow").UnsafePointer()); *_flow != flowId {
							t.Fatalf("expected id to be %s, got %s", flowId, *_flow)
						}
						if _body := (*kClient.UpdateLoginFlowBody)(reflect.ValueOf(r).FieldByName("updateLoginFlowBody").UnsafePointer()); *_body != *body {
							t.Fatalf("expected id to be %v, got %v", *body, *_body)
						}
						if cookie := (*string)(reflect.ValueOf(r).FieldByName("cookie").UnsafePointer()); *cookie != "test=test" {
							t.Fatalf("expected cookie string as test=test, got %s", *cookie)
						}
						return &flow, &resp, nil
					},
				)
			} else {
				var respBody []byte
				if tt.errorMessageId != 0 {
					errorBody := &UiErrorMessages{
						Ui: kClient.UiContainer{
							Messages: []kClient.UiText{{Id: tt.errorMessageId}},
						},
					}
					respBody, _ = json.Marshal(errorBody)
				} else {
					_redirectTo := "https://redirect/to/path"
					flow := ErrorBrowserLocationChangeRequired{
						RedirectBrowserTo: &_redirectTo,
					}
					respBody, _ = json.Marshal(flow)
				}
				resp := http.Response{
					Header:     http.Header{"Set-Cookie": []string{cookie.Raw}},
					Body:       io.NopCloser(bytes.NewBuffer(respBody)),
					StatusCode: tt.statusCode,
				}
				mockKratosFrontendApi.EXPECT().UpdateLoginFlowExecute(gomock.Any()).Times(1).Return(nil, &resp, fmt.Errorf("error"))
			}

			if tt.expectLog {
				mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)
			}

			r, _, c, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).UpdateLoginFlow(ctx, flowId, *body, cookies)

			if tt.expectSuccess {
				if err != nil {
					t.Fatalf("expected error to be nil not  %v", err)
				}
				if *r.RedirectTo != "https://redirect/to/path" {
					t.Fatalf("expected redirectTo to be https://redirect/to/path not %s", *r.RedirectTo)
				}
				if c == nil {
					t.Fatalf("expected cookies not to be nil")
				}
			} else {
				if err == nil {
					t.Fatalf("expected error not nil")
				}
				if tt.expectedError != "" && err.Error() != tt.expectedError {
					t.Fatalf("expected error to be %s not %v", tt.expectedError, err)
				}
				if r != nil {
					t.Fatalf("expected flow to be %v not %+v", nil, r)
				}
				if c != nil {
					t.Fatalf("expected header to be %v not  %v", nil, c)
				}
			}
		})
	}
}

func TestUpdateLoginFlowSuccessNative(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendAPI(ctrl)

	ctx := context.Background()
	flowId := "flow"
	body := kClient.UpdateLoginFlowWithPasswordMethodAsUpdateLoginFlowBody(kClient.NewUpdateLoginFlowWithPasswordMethodWithDefaults())
	request := kClient.FrontendAPIUpdateLoginFlowRequest{ApiService: mockKratosFrontendApi}
	respCookie := &http.Cookie{Name: "test", Value: "test"}
	resp := http.Response{Header: http.Header{"Set-Cookie": []string{respCookie.String()}}, StatusCode: http.StatusOK}
	login := kClient.NewSuccessfulNativeLogin(*kClient.NewSession("session-id"))
	login.ContinueWith = []kClient.ContinueWith{{}}

	mockTracer.EXPECT().Start(ctx, "kratos.Service.UpdateLoginFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().UpdateLoginFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().UpdateLoginFlowExecute(gomock.Any()).Times(1).DoAndReturn(
		func(r kClient.FrontendAPIUpdateLoginFlowRequest) (*kClient.SuccessfulNativeLogin, *http.Response, error) {
			if _flow := (*string)(reflect.ValueOf(r).FieldByName("flow").UnsafePointer()); *_flow != flowId {
				t.Fatalf("expected id to be %s, got %s", flowId, *_flow)
			}
			return login, &resp, nil
		},
	)

	redirect, nativeLogin, cookies, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).UpdateLoginFlow(ctx, flowId, body, []*http.Cookie{})

	if err != nil {
		t.Fatalf("expected error to be nil not %v", err)
	}
	if redirect != nil {
		t.Fatalf("expected redirect to be nil not %v", redirect)
	}
	if nativeLogin == nil {
		t.Fatalf("expected native login not nil")
	}
	if nativeLogin.ContinueWith != nil {
		t.Fatalf("expected continue_with to be nil")
	}
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
}

func TestUpdateLoginFlowOidcAddsSessionUnsetCookie(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendAPI(ctrl)

	ctx := context.Background()
	flowId := "flow"
	oidcBody := kClient.NewUpdateLoginFlowWithOidcMethod("oidc", "google")
	body := kClient.UpdateLoginFlowWithOidcMethodAsUpdateLoginFlowBody(oidcBody)
	request := kClient.FrontendAPIUpdateLoginFlowRequest{ApiService: mockKratosFrontendApi}
	respCookie := &http.Cookie{Name: "test", Value: "test"}
	resp := http.Response{Header: http.Header{"Set-Cookie": []string{respCookie.String()}}, StatusCode: http.StatusOK}
	login := kClient.NewSuccessfulNativeLogin(*kClient.NewSession("session-id"))

	mockTracer.EXPECT().Start(ctx, "kratos.Service.UpdateLoginFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().UpdateLoginFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().UpdateLoginFlowExecute(gomock.Any()).Times(1).Return(login, &resp, nil)

	redirect, nativeLogin, cookies, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).UpdateLoginFlow(ctx, flowId, body, []*http.Cookie{})

	if err != nil {
		t.Fatalf("expected error to be nil not %v", err)
	}
	if redirect != nil {
		t.Fatalf("expected redirect to be nil not %v", redirect)
	}
	if nativeLogin == nil {
		t.Fatalf("expected native login not nil")
	}
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}
	unset := kratosSessionUnsetCookie()
	found := false
	for _, c := range cookies {
		if c.Name == unset.Name {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected session unset cookie to be added")
	}
}

func TestUpdateLoginFlowParseRedirectError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
	mockKratosFrontendApi := NewMockFrontendAPI(ctrl)

	ctx := context.Background()
	flowId := "flow"
	body := kClient.UpdateLoginFlowWithPasswordMethodAsUpdateLoginFlowBody(kClient.NewUpdateLoginFlowWithPasswordMethodWithDefaults())
	request := kClient.FrontendAPIUpdateLoginFlowRequest{ApiService: mockKratosFrontendApi}
	resp := http.Response{Body: io.NopCloser(strings.NewReader("not-json")), StatusCode: http.StatusUnprocessableEntity}

	mockTracer.EXPECT().Start(ctx, "kratos.Service.UpdateLoginFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockTracer.EXPECT().Start(ctx, "kratos.Service.parseKratosRedirectResponse").Times(1).Return(ctx, trace.SpanFromContext(ctx))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)
	mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
	mockKratosFrontendApi.EXPECT().UpdateLoginFlow(ctx).Times(1).Return(request)
	mockKratosFrontendApi.EXPECT().UpdateLoginFlowExecute(gomock.Any()).Times(1).Return(nil, &resp, nil)

	redirect, nativeLogin, cookies, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).UpdateLoginFlow(ctx, flowId, body, []*http.Cookie{})

	if err == nil {
		t.Fatalf("expected error not nil")
	}
	if redirect != nil {
		t.Fatalf("expected redirect to be nil not %v", redirect)
	}
	if nativeLogin != nil {
		t.Fatalf("expected native login to be nil")
	}
	if cookies != nil {
		t.Fatalf("expected cookies to be nil")
	}
}

func TestGetUiError(t *testing.T) {
	tests := []struct {
		name      string
		messages  []kClient.UiText
		expectErr string
		expectLog bool
	}{
		{
			name:      "incorrect credentials",
			messages:  []kClient.UiText{{Id: IncorrectCredentials}},
			expectErr: "incorrect username or password",
		},
		{
			name:      "incorrect account identifier",
			messages:  []kClient.UiText{{Id: IncorrectAccountIdentifier}},
			expectErr: "account does not exist or has no login method configured",
		},
		{
			name:      "inactive account",
			messages:  []kClient.UiText{{Id: InactiveAccount}},
			expectErr: "inactive account",
		},
		{
			name:      "invalid property",
			messages:  []kClient.UiText{{Id: InvalidProperty, Context: map[string]interface{}{"property": "email"}}},
			expectErr: "invalid email",
		},
		{
			name:      "password policy violation",
			messages:  []kClient.UiText{{Id: NewPasswordPolicyViolation, Text: "password must contain uppercase and numbers"}},
			expectErr: "new password does not meet the password policy requirements: password must contain uppercase and numbers",
		},
		{
			name:      "not enough characters",
			messages:  []kClient.UiText{{Id: NotEnoughCharacters, Context: map[string]interface{}{"min_length": 8}}},
			expectErr: "at least 8 characters required",
		},
		{
			name:      "too many characters",
			messages:  []kClient.UiText{{Id: TooManyCharacters, Context: map[string]interface{}{"max_length": 64}}},
			expectErr: "maximum 64 characters allowed",
		},
		{
			name:      "password too long",
			messages:  []kClient.UiText{{Id: PasswordTooLong, Context: map[string]interface{}{"max_length": 64}}},
			expectErr: "maximum 64 characters allowed",
		},
		{
			name:      "invalid backup code",
			messages:  []kClient.UiText{{Id: InvalidBackupCode}},
			expectErr: "invalid backup code",
		},
		{
			name:      "backup code already used",
			messages:  []kClient.UiText{{Id: BackupCodeAlreadyUsed}},
			expectErr: "this backup code was already used",
		},
		{
			name:      "invalid auth code",
			messages:  []kClient.UiText{{Id: InvalidAuthCode}},
			expectErr: "invalid authentication code",
		},
		{
			name:      "missing security key setup",
			messages:  []kClient.UiText{{Id: MissingSecurityKeySetup}},
			expectErr: "choose a different login method",
		},
		{
			name:      "missing backup codes setup",
			messages:  []kClient.UiText{{Id: MissingBackupCodesSetup}},
			expectErr: "login with backup codes unavailable",
		},
		{
			name:      "password identifier similarity",
			messages:  []kClient.UiText{{Id: PasswordIdentifierSimilarity}},
			expectErr: "password can not be similar to the email",
		},
		{
			name:      "unknown code logs and returns server error",
			messages:  []kClient.UiText{{Id: 9999999}},
			expectErr: "server error",
			expectLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

			if tt.expectLog {
				mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)
			}

			errorResp := UiErrorMessages{Ui: kClient.UiContainer{Messages: tt.messages}}
			body, _ := json.Marshal(errorResp)
			resp := io.NopCloser(bytes.NewBuffer(body))

			err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).getUiError(resp)

			if err == nil || err.Error() != tt.expectErr {
				t.Fatalf("expected error '%s', got %v", tt.expectErr, err)
			}
		})
	}
}

func TestGetFlowError(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		shouldFail bool
		expectNil  bool
	}{
		{
			name:       "Success",
			id:         "id",
			shouldFail: false,
			expectNil:  false,
		},
		{
			name:       "Fail",
			id:         "id",
			shouldFail: true,
			expectNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
			mockKratosFrontendApi := NewMockFrontendAPI(ctrl)

			ctx := context.Background()
			flow := kClient.NewFlowError(tt.id)
			request := kClient.FrontendAPIGetFlowErrorRequest{
				ApiService: mockKratosFrontendApi,
			}
			resp := http.Response{
				Header: http.Header{"K": []string{"V"}},
			}

			mockTracer.EXPECT().Start(ctx, "kratos.Service.GetFlowError").Times(1).Return(ctx, trace.SpanFromContext(ctx))
			mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
			mockKratosFrontendApi.EXPECT().GetFlowError(ctx).Times(1).Return(request)

			if tt.shouldFail {
				mockKratosFrontendApi.EXPECT().GetFlowErrorExecute(gomock.Any()).Times(1).Return(nil, &resp, fmt.Errorf("error"))
			} else {
				mockKratosFrontendApi.EXPECT().GetFlowErrorExecute(gomock.Any()).Times(1).DoAndReturn(
					func(r kClient.FrontendAPIGetFlowErrorRequest) (*kClient.FlowError, *http.Response, error) {
						if _id := (*string)(reflect.ValueOf(r).FieldByName("id").UnsafePointer()); *_id != tt.id {
							t.Fatalf("expected id to be %s, got %s", tt.id, *_id)
						}
						return flow, &resp, nil
					},
				)
			}

			f, c, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).GetFlowError(ctx, tt.id)

			if tt.expectNil {
				if f != nil {
					t.Fatalf("expected flow to be %v not %+v", nil, f)
				}
				if c != nil {
					t.Fatalf("expected cookies to be %v not  %v", nil, c)
				}
				if err == nil {
					t.Fatalf("expected error not nil")
				}
			} else {
				if !reflect.DeepEqual(f, flow) {
					t.Fatalf("expected flow to be %+v not %+v", flow, f)
				}
				if !reflect.DeepEqual(c, resp.Cookies()) {
					t.Fatalf("expected cookies to be %v not  %v", resp.Cookies(), c)
				}
				if err != nil {
					t.Fatalf("expected error to be nil not  %v", err)
				}
			}
		})
	}
}

func TestCheckAllowedProvider(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		allowedList   []string
		shouldFail    bool
		expectAllowed bool
	}{
		{
			name:          "Allowed success",
			provider:      "provider",
			allowedList:   []string{"provider"},
			shouldFail:    false,
			expectAllowed: true,
		},
		{
			name:          "Not allowed success",
			provider:      "provider",
			allowedList:   []string{"other_provider"},
			shouldFail:    false,
			expectAllowed: false,
		},
		{
			name:          "Fail",
			provider:      "provider",
			allowedList:   []string{},
			shouldFail:    true,
			expectAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

			ctx := context.Background()
			oidcBody := kClient.NewUpdateLoginFlowWithOidcMethod("oidc", tt.provider)
			body := kClient.UpdateLoginFlowWithOidcMethodAsUpdateLoginFlowBody(oidcBody)

			client_name := "foo"
			client := kClient.NewOAuth2ClientWithDefaults()
			client.ClientName = &client_name
			loginReq := kClient.NewOAuth2LoginRequestWithDefaults()
			loginReq.Client = client
			flow := kClient.NewLoginFlowWithDefaults()
			flow.Oauth2LoginRequest = loginReq

			mockTracer.EXPECT().Start(ctx, "kratos.Service.CheckAllowedProvider").Times(1).Return(ctx, trace.SpanFromContext(ctx))

			if tt.shouldFail {
				mockAuthz.EXPECT().ListObjects(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(tt.allowedList, fmt.Errorf("oh no"))
			} else {
				mockAuthz.EXPECT().ListObjects(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(tt.allowedList, nil)
			}

			allowed, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).CheckAllowedProvider(ctx, flow, &body)

			if tt.shouldFail {
				if err == nil {
					t.Fatalf("expected error not nil")
				}
			} else {
				if err != nil {
					t.Fatalf("expected error to be nil not  %v", err)
				}
				if allowed != tt.expectAllowed {
					t.Fatalf("expected allowed to be %v not %v", tt.expectAllowed, allowed)
				}
			}
		})
	}
}

func TestGetClientName(t *testing.T) {
	tests := []struct {
		name               string
		loginFlow          *kClient.LoginFlow
		expectedClientName string
	}{
		{
			name:               "Oathkeeper",
			loginFlow:          &kClient.LoginFlow{},
			expectedClientName: "",
		},
		{
			name: "OAuth2Request",
			loginFlow: func() *kClient.LoginFlow {
				clientName := "mockClientName"
				return &kClient.LoginFlow{Oauth2LoginRequest: &kClient.OAuth2LoginRequest{Client: &kClient.OAuth2Client{ClientName: &clientName}}}
			}(),
			expectedClientName: "mockClientName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(nil, nil, nil, nil, false, nil, nil, nil)

			actualClientName := service.getClientName(tt.loginFlow)

			if tt.expectedClientName != actualClientName {
				t.Fatalf("Expected client name %s, got %s", tt.expectedClientName, actualClientName)
			}
		})
	}
}

func TestFilterFlowProviderList(t *testing.T) {
	tests := []struct {
		name                string
		kratosProviders     []string
		allowedProviders    []string
		authzError          error
		expectedUiNodeCount int
		expectedError       bool
		checkUiMatches      bool
	}{
		{
			name:                "AllowAll",
			kratosProviders:     []string{"1", "2", "3", "4"},
			allowedProviders:    []string{"1", "2", "3", "4"},
			authzError:          nil,
			expectedUiNodeCount: 4,
			expectedError:       false,
			checkUiMatches:      true,
		},
		{
			name:                "AllowSome",
			kratosProviders:     []string{"1", "2", "3", "4"},
			allowedProviders:    []string{"1", "ab", "ba", "4"},
			authzError:          nil,
			expectedUiNodeCount: 2,
			expectedError:       false,
			checkUiMatches:      false,
		},
		{
			name:                "AllowNone",
			kratosProviders:     []string{"1", "2", "3", "4"},
			allowedProviders:    []string{},
			authzError:          nil,
			expectedUiNodeCount: 4,
			expectedError:       false,
			checkUiMatches:      true,
		},
		{
			name:             "Fail",
			kratosProviders:  []string{"1", "2", "3", "4"},
			allowedProviders: nil,
			authzError:       fmt.Errorf("oh no"),
			expectedError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

			ctx := context.Background()

			client_name := "foo"
			client := kClient.NewOAuth2ClientWithDefaults()
			client.ClientName = &client_name
			loginReq := kClient.NewOAuth2LoginRequestWithDefaults()
			loginReq.Client = client
			ui := *kClient.NewUiContainerWithDefaults()
			for _, p := range tt.kratosProviders {
				node := kClient.NewUiNodeWithDefaults()
				attributes := kClient.NewUiNodeInputAttributesWithDefaults()
				attributes.Value = p
				node.Attributes = kClient.UiNodeInputAttributesAsUiNodeAttributes(attributes)
				node.Group = "oidc"
				ui.Nodes = append(ui.Nodes, *node)
			}
			flow := kClient.NewLoginFlowWithDefaults()
			flow.Oauth2LoginRequest = loginReq
			flow.Ui = ui

			mockTracer.EXPECT().Start(ctx, "kratos.Service.FilterFlowProviderList").Times(1).Return(ctx, trace.SpanFromContext(ctx))
			mockAuthz.EXPECT().ListObjects(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(tt.allowedProviders, tt.authzError)

			f, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).FilterFlowProviderList(ctx, flow)

			if tt.expectedError {
				if err == nil {
					t.Fatalf("expected error to be not nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("expected error to be nil not  %v", err)
			}

			if tt.checkUiMatches {
				if !reflect.DeepEqual(f.Ui, ui) {
					t.Fatalf("expected ui to be %v not  %v", ui, f.Ui)
				}
			} else {
				expectedUi := *kClient.NewUiContainerWithDefaults()
				expectedUi.Nodes = []kClient.UiNode{ui.Nodes[0], ui.Nodes[3]}
				if !reflect.DeepEqual(f.Ui, expectedUi) {
					t.Fatalf("expected Ui to be %v not  %v", expectedUi, f.Ui)
				}
			}
		})
	}
}

func TestParseLoginFlowMethodBody(t *testing.T) {
	tests := []struct {
		name   string
		body   kClient.UpdateLoginFlowBody
		method string
	}{
		{
			name:   "Oidc",
			body:   kClient.UpdateLoginFlowWithOidcMethodAsUpdateLoginFlowBody(kClient.NewUpdateLoginFlowWithOidcMethodWithDefaults()),
			method: "oidc",
		},
		{
			name: "Password",
			body: func() kClient.UpdateLoginFlowBody {
				flow := kClient.NewUpdateLoginFlowWithPasswordMethodWithDefaults()
				flow.SetMethod("password")
				return kClient.UpdateLoginFlowWithPasswordMethodAsUpdateLoginFlowBody(flow)
			}(),
			method: "password",
		},
		{
			name: "Totp",
			body: func() kClient.UpdateLoginFlowBody {
				flow := kClient.NewUpdateLoginFlowWithTotpMethodWithDefaults()
				flow.SetMethod("totp")
				return kClient.UpdateLoginFlowWithTotpMethodAsUpdateLoginFlowBody(flow)
			}(),
			method: "totp",
		},
		{
			name: "LookupSecret",
			body: func() kClient.UpdateLoginFlowBody {
				flow := kClient.NewUpdateLoginFlowWithLookupSecretMethodWithDefaults()
				flow.SetMethod("lookup_secret")
				return kClient.UpdateLoginFlowWithLookupSecretMethodAsUpdateLoginFlowBody(flow)
			}(),
			method: "lookup_secret",
		},
		{
			name: "WebAuthn",
			body: func() kClient.UpdateLoginFlowBody {
				flow := kClient.NewUpdateLoginFlowWithWebAuthnMethodWithDefaults()
				flow.SetMethod("webauthn")
				return kClient.UpdateLoginFlowWithWebAuthnMethodAsUpdateLoginFlowBody(flow)
			}(),
			method: "webauthn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

			jsonBody, _ := tt.body.MarshalJSON()
			req := httptest.NewRequest(http.MethodPost, "http://some/path", io.NopCloser(bytes.NewBuffer(jsonBody)))

			b, _, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).ParseLoginFlowMethodBody(req)

			actual, _ := b.MarshalJSON()
			expected, _ := tt.body.MarshalJSON()
			if !reflect.DeepEqual(string(actual), string(expected)) {
				t.Fatalf("expected flow to be %s not %s", string(expected), string(actual))
			}
			if err != nil {
				t.Fatalf("expected error to be nil not  %v", err)
			}
		})
	}
}

func TestParseLoginFlowMethodBody_ErrorsAndForm(t *testing.T) {
	tests := []struct {
		name           string
		setupRequest   func() *http.Request
		expectError    bool
		expectedErrMsg string
		assertBody     func(*testing.T, *kClient.UpdateLoginFlowBody, []*http.Cookie)
	}{
		{
			name: "ReadBodyError",
			setupRequest: func() *http.Request {
				errReader := io.NopCloser(readerFunc(func([]byte) (int, error) {
					return 0, fmt.Errorf("read error")
				}))
				return httptest.NewRequest(http.MethodPost, "http://some/path", errReader)
			},
			expectError:    true,
			expectedErrMsg: "unable to read body",
		},
		{
			name: "WebAuthnFormRemovesSessionCookie",
			setupRequest: func() *http.Request {
				form := "csrf_token=csrf&webauthn_login=login&identifier=user@example.com"
				req := httptest.NewRequest(http.MethodPost, "http://some/path", io.NopCloser(strings.NewReader(form)))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.AddCookie(&http.Cookie{Name: KRATOS_SESSION_COOKIE_NAME, Value: "session"})
				req.AddCookie(&http.Cookie{Name: "other", Value: "value"})
				return req
			},
			expectError: false,
			assertBody: func(t *testing.T, body *kClient.UpdateLoginFlowBody, cookies []*http.Cookie) {
				if body == nil || body.UpdateLoginFlowWithWebAuthnMethod == nil {
					t.Fatalf("expected webauthn body to be set")
				}
				if body.UpdateLoginFlowWithWebAuthnMethod.CsrfToken == nil || *body.UpdateLoginFlowWithWebAuthnMethod.CsrfToken != "csrf" {
					t.Fatalf("expected csrf token to be %s", "csrf")
				}
				if body.UpdateLoginFlowWithWebAuthnMethod.WebauthnLogin == nil || *body.UpdateLoginFlowWithWebAuthnMethod.WebauthnLogin != "login" {
					t.Fatalf("expected login to be %s", "login")
				}
				if body.UpdateLoginFlowWithWebAuthnMethod.Identifier != "user@example.com" {
					t.Fatalf("expected identifier to be %s", "user@example.com")
				}
				for _, c := range cookies {
					if c.Name == KRATOS_SESSION_COOKIE_NAME {
						t.Fatalf("expected session cookie to be removed")
					}
				}
				if len(cookies) != 1 || cookies[0].Name != "other" {
					t.Fatalf("expected only non-session cookies to remain")
				}
			},
		},
		{
			name: "InvalidJSONFallbackWebAuthn",
			setupRequest: func() *http.Request {
				body := []byte("{not-json")
				return httptest.NewRequest(http.MethodPost, "http://some/path", io.NopCloser(bytes.NewBuffer(body)))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

			req := tt.setupRequest()
			body, cookies, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).ParseLoginFlowMethodBody(req)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error not nil")
				}
				if tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
					t.Fatalf("expected error %s got %v", tt.expectedErrMsg, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected error to be nil not %v", err)
			}
			if tt.assertBody != nil {
				tt.assertBody(t, body, cookies)
			}
		})
	}
}

func TestGetProviderName(t *testing.T) {
	tests := []struct {
		name                 string
		setupBody            func() kClient.UpdateLoginFlowBody
		expectedProviderName string
	}{
		{
			name: "WhenNotOidcMethod",
			setupBody: func() kClient.UpdateLoginFlowBody {
				body := kClient.UpdateLoginFlowBody{}
				return body
			},
			expectedProviderName: "",
		},
		{
			name: "Oidc",
			setupBody: func() kClient.UpdateLoginFlowBody {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockLogger := NewMockLoggerInterface(ctrl)
				mockHydra := NewMockHydraClientInterface(ctrl)
				mockKratos := NewMockKratosClientInterface(ctrl)
				mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
				mockAuthz := NewMockAuthorizerInterface(ctrl)
				mockTracer := NewMockTracingInterface(ctrl)
				mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

				expectedProviderName := "someProvider"
				flow := kClient.NewUpdateLoginFlowWithOidcMethod("", expectedProviderName)
				body := kClient.UpdateLoginFlowWithOidcMethodAsUpdateLoginFlowBody(flow)
				jsonBody, _ := body.MarshalJSON()

				req := httptest.NewRequest(http.MethodPost, "http://some/path", io.NopCloser(bytes.NewBuffer(jsonBody)))
				b, _, _ := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).ParseLoginFlowMethodBody(req)

				return *b
			},
			expectedProviderName: "someProvider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(nil, nil, nil, nil, false, nil, nil, nil)
			body := tt.setupBody()

			actualProviderName := service.getProviderName(&body)

			if tt.expectedProviderName != actualProviderName {
				t.Fatalf("Expected the provider to be %v, not %v", tt.expectedProviderName, actualProviderName)
			}
		})
	}
}

func TestParseRecoveryFlowCodeMethodBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockHydra := NewMockHydraClientInterface(ctrl)
	mockKratos := NewMockKratosClientInterface(ctrl)
	mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
	mockAuthz := NewMockAuthorizerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

	flow := kClient.NewUpdateRecoveryFlowWithCodeMethodWithDefaults()
	flow.SetMethod("code")

	body := kClient.UpdateRecoveryFlowWithCodeMethodAsUpdateRecoveryFlowBody(flow)

	jsonBody, _ := body.MarshalJSON()

	req := httptest.NewRequest(http.MethodPost, "http://some/path", io.NopCloser(bytes.NewBuffer(jsonBody)))

	b, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).ParseRecoveryFlowMethodBody(req)

	actual, _ := b.MarshalJSON()
	expected, _ := body.MarshalJSON()

	if !reflect.DeepEqual(string(actual), string(expected)) {
		t.Fatalf("expected flow to be %s not %s", string(expected), string(actual))
	}
	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestGetRecoveryFlow(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*MockFrontendAPI, *kClient.RecoveryFlow, *http.Response)
		expectedError bool
	}{
		{
			name: "Success",
			setupMocks: func(mockKratosFrontendApi *MockFrontendAPI, flow *kClient.RecoveryFlow, resp *http.Response) {
				mockKratosFrontendApi.EXPECT().GetRecoveryFlowExecute(gomock.Any()).Times(1).DoAndReturn(
					func(r kClient.FrontendAPIGetRecoveryFlowRequest) (*kClient.RecoveryFlow, *http.Response, error) {
						id := "id"
						if _id := (*string)(reflect.ValueOf(r).FieldByName("id").UnsafePointer()); *_id != id {
							t.Fatalf("expected id to be %s, got %s", id, *_id)
						}
						if cookie := (*string)(reflect.ValueOf(r).FieldByName("cookie").UnsafePointer()); *cookie != "test=test" {
							t.Fatalf("expected cookie string as test=test, got %s", *cookie)
						}
						return flow, resp, nil
					},
				)
			},
			expectedError: false,
		},
		{
			name: "Fail",
			setupMocks: func(mockKratosFrontendApi *MockFrontendAPI, flow *kClient.RecoveryFlow, resp *http.Response) {
				mockKratosFrontendApi.EXPECT().GetRecoveryFlowExecute(gomock.Any()).Times(1).Return(nil, resp, fmt.Errorf("error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
			mockKratosFrontendApi := NewMockFrontendAPI(ctrl)

			ctx := context.Background()
			cookies := make([]*http.Cookie, 0)
			cookie := &http.Cookie{Name: "test", Value: "test"}
			cookies = append(cookies, cookie)
			id := "id"
			flow := kClient.NewRecoveryFlowWithDefaults()
			request := kClient.FrontendAPIGetRecoveryFlowRequest{
				ApiService: mockKratosFrontendApi,
			}
			resp := http.Response{
				Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
			}

			mockTracer.EXPECT().Start(ctx, "kratos.Service.GetRecoveryFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
			mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
			mockKratosFrontendApi.EXPECT().GetRecoveryFlow(ctx).Times(1).Return(request)
			tt.setupMocks(mockKratosFrontendApi, flow, &resp)

			s, c, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).GetRecoveryFlow(ctx, id, cookies)

			if tt.expectedError {
				if err == nil {
					t.Fatalf("expected error not nil")
				}
				if s != nil {
					t.Fatalf("expected flow to be %v not  %v", nil, s)
				}
				if c != nil {
					t.Fatalf("expected header to be %v not  %v", nil, c)
				}
			} else {
				if s != flow {
					t.Fatalf("expected flow to be %v not  %v", flow, s)
				}
				if !reflect.DeepEqual(c, resp.Cookies()) {
					t.Fatalf("expected cookies to be %v not  %v", resp.Cookies(), c)
				}
				if err != nil {
					t.Fatalf("expected error to be nil not  %v", err)
				}
			}
		})
	}
}

func TestCreateBrowserRecoveryFlow(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*MockFrontendAPI, *kClient.RecoveryFlow, *http.Response, string)
		expectedError bool
	}{
		{
			name: "Success",
			setupMocks: func(mockKratosFrontendApi *MockFrontendAPI, flow *kClient.RecoveryFlow, resp *http.Response, returnTo string) {
				mockKratosFrontendApi.EXPECT().CreateBrowserRecoveryFlowExecute(gomock.Any()).Times(1).DoAndReturn(
					func(r kClient.FrontendAPICreateBrowserRecoveryFlowRequest) (*kClient.RecoveryFlow, *http.Response, error) {
						if rt := (*string)(reflect.ValueOf(r).FieldByName("returnTo").UnsafePointer()); *rt != returnTo {
							t.Fatalf("expected returnTo to be %s, got %s", returnTo, *rt)
						}
						return flow, resp, nil
					},
				)
			},
			expectedError: false,
		},
		{
			name: "Fail",
			setupMocks: func(mockKratosFrontendApi *MockFrontendAPI, flow *kClient.RecoveryFlow, resp *http.Response, returnTo string) {
				mockKratosFrontendApi.EXPECT().CreateBrowserRecoveryFlowExecute(gomock.Any()).Times(1).Return(nil, nil, fmt.Errorf("error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
			mockKratosFrontendApi := NewMockFrontendAPI(ctrl)

			ctx := context.Background()
			cookies := make([]*http.Cookie, 0)
			cookie := &http.Cookie{Name: "test", Value: "test"}
			cookies = append(cookies, cookie)
			returnTo := "https://example.com/ui/reset_email"
			flow := kClient.NewRecoveryFlowWithDefaults()
			request := kClient.FrontendAPICreateBrowserRecoveryFlowRequest{
				ApiService: mockKratosFrontendApi,
			}
			resp := http.Response{
				Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
			}

			mockTracer.EXPECT().Start(ctx, "kratos.Service.CreateBrowserRecoveryFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
			mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
			mockKratosFrontendApi.EXPECT().CreateBrowserRecoveryFlow(ctx).Times(1).Return(request)
			tt.setupMocks(mockKratosFrontendApi, flow, &resp, returnTo)

			f, c, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).CreateBrowserRecoveryFlow(ctx, returnTo, cookies)

			if tt.expectedError {
				if f != nil {
					t.Fatalf("expected flow to be %v not  %v", nil, f)
				}
				if c != nil {
					t.Fatalf("expected cookies to be %v not  %v", nil, c)
				}
				if err == nil {
					t.Fatalf("expected error not nil")
				}
			} else {
				if f != flow {
					t.Fatalf("expected flow to be %v not  %v", flow, f)
				}
				if !reflect.DeepEqual(c, resp.Cookies()) {
					t.Fatalf("expected cookies to be %v not  %v", resp.Cookies(), c)
				}
				if err != nil {
					t.Fatalf("expected error to be nil not  %v", err)
				}
			}
		})
	}
}

func TestUpdateRecoveryFlow(t *testing.T) {
	tests := []struct {
		name                 string
		setupMocks           func(*MockFrontendAPI, *MockLoggerInterface, *MockTracingInterface, *http.Response)
		expectedError        bool
		expectedErrorMessage string
		checkRedirect        bool
		checkCookies         bool
	}{
		{
			name: "Success",
			setupMocks: func(mockKratosFrontendApi *MockFrontendAPI, mockLogger *MockLoggerInterface, mockTracer *MockTracingInterface, resp *http.Response) {
				mockTracer.EXPECT().Start(gomock.Any(), "kratos.Service.parseKratosRedirectResponse").Times(1).Return(context.Background(), trace.SpanFromContext(context.Background()))
				mockKratosFrontendApi.EXPECT().UpdateRecoveryFlowExecute(gomock.Any()).Times(1).DoAndReturn(
					func(r kClient.FrontendAPIUpdateRecoveryFlowRequest) (*ErrorBrowserLocationChangeRequired, *http.Response, error) {
						flowId := "flow"
						if _flow := (*string)(reflect.ValueOf(r).FieldByName("flow").UnsafePointer()); *_flow != flowId {
							t.Fatalf("expected id to be %s, got %s", flowId, *_flow)
						}
						_redirectTo := "https://redirect/to/path"
						flow := ErrorBrowserLocationChangeRequired{
							RedirectBrowserTo: &_redirectTo,
						}
						return &flow, resp, nil
					},
				)
			},
			expectedError: false,
			checkRedirect: true,
			checkCookies:  true,
		},
		{
			name: "FailOnExecute",
			setupMocks: func(mockKratosFrontendApi *MockFrontendAPI, mockLogger *MockLoggerInterface, mockTracer *MockTracingInterface, resp *http.Response) {
				mockKratosFrontendApi.EXPECT().UpdateRecoveryFlowExecute(gomock.Any()).Times(1).Return(nil, resp, fmt.Errorf("error"))
				mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)
			},
			expectedError: true,
		},
		{
			name: "FailOnInvalidCode",
			setupMocks: func(mockKratosFrontendApi *MockFrontendAPI, mockLogger *MockLoggerInterface, mockTracer *MockTracingInterface, resp *http.Response) {
				flow := &kClient.RecoveryFlow{
					Ui: kClient.UiContainer{
						Messages: []kClient.UiText{
							{
								Id: InvalidRecoveryCode,
							},
						},
					},
				}
				resp.StatusCode = 200
				flowJson, _ := json.Marshal(flow)
				resp.Body = io.NopCloser(bytes.NewBuffer(flowJson))
				mockKratosFrontendApi.EXPECT().UpdateRecoveryFlowExecute(gomock.Any()).Times(1).Return(flow, resp, nil)
			},
			expectedError:        true,
			expectedErrorMessage: "the recovery code is invalid or has already been used",
		},
		{
			name: "BadRequestParseSuccess",
			setupMocks: func(mockKratosFrontendApi *MockFrontendAPI, mockLogger *MockLoggerInterface, mockTracer *MockTracingInterface, resp *http.Response) {
				resp.StatusCode = http.StatusBadRequest
				_redirectTo := "https://redirect/to/path"
				flow := ErrorBrowserLocationChangeRequired{
					RedirectBrowserTo: &_redirectTo,
				}
				flowJson, _ := json.Marshal(flow)
				resp.Body = io.NopCloser(bytes.NewBuffer(flowJson))
				mockKratosFrontendApi.EXPECT().UpdateRecoveryFlowExecute(gomock.Any()).Times(1).Return(nil, resp, fmt.Errorf("error"))
				mockTracer.EXPECT().Start(gomock.Any(), "kratos.Service.parseKratosRedirectResponse").Times(1).Return(context.Background(), trace.SpanFromContext(context.Background()))
			},
			expectedError: false,
			checkRedirect: true,
			checkCookies:  false,
		},
		{
			name: "BadRequestParseError",
			setupMocks: func(mockKratosFrontendApi *MockFrontendAPI, mockLogger *MockLoggerInterface, mockTracer *MockTracingInterface, resp *http.Response) {
				resp.StatusCode = http.StatusBadRequest
				resp.Body = io.NopCloser(strings.NewReader("not-json"))
				mockKratosFrontendApi.EXPECT().UpdateRecoveryFlowExecute(gomock.Any()).Times(1).Return(nil, resp, fmt.Errorf("error"))
				mockTracer.EXPECT().Start(gomock.Any(), "kratos.Service.parseKratosRedirectResponse").Times(1).Return(context.Background(), trace.SpanFromContext(context.Background()))
				mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)
			},
			expectedError: true,
			checkCookies:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
			mockKratosFrontendApi := NewMockFrontendAPI(ctrl)

			ctx := context.Background()
			cookies := make([]*http.Cookie, 0)
			cookie := &http.Cookie{Name: "test", Value: "test"}
			cookies = append(cookies, cookie)
			flowId := "flow"
			_redirectTo := "https://redirect/to/path"
			flow := ErrorBrowserLocationChangeRequired{
				RedirectBrowserTo: &_redirectTo,
			}
			flowJson, _ := json.Marshal(flow)
			body := new(kClient.UpdateRecoveryFlowBody)
			request := kClient.FrontendAPIUpdateRecoveryFlowRequest{
				ApiService: mockKratosFrontendApi,
			}
			resp := http.Response{
				Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
				Body:   io.NopCloser(bytes.NewBuffer(flowJson)),
			}

			mockTracer.EXPECT().Start(ctx, "kratos.Service.UpdateRecoveryFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
			mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
			mockKratosFrontendApi.EXPECT().UpdateRecoveryFlow(ctx).Times(1).Return(request)
			tt.setupMocks(mockKratosFrontendApi, mockLogger, mockTracer, &resp)

			f, c, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).UpdateRecoveryFlow(ctx, flowId, *body, cookies)

			if tt.expectedError {
				if f != nil {
					t.Fatalf("expected flow to be %v not %+v", nil, f)
				}
				if c != nil {
					t.Fatalf("expected header to be %v not %v", nil, c)
				}
				if err == nil {
					t.Fatalf("expected error not nil")
				}
				if tt.expectedErrorMessage != "" && err.Error() != tt.expectedErrorMessage {
					t.Fatalf("expected error to be %v not %v", tt.expectedErrorMessage, err)
				}
			} else {
				if tt.checkRedirect {
					if *f.RedirectTo != _redirectTo {
						t.Fatalf("expected redirectTo to be %s not %s", _redirectTo, *f.RedirectTo)
					}
				}
				if tt.checkCookies && !reflect.DeepEqual(c, resp.Cookies()) {
					t.Fatalf("expected cookies to be %v not  %v", resp.Cookies(), c)
				}
				if err != nil {
					t.Fatalf("expected error to be nil not  %v", err)
				}
			}
		})
	}
}

func TestParseSettingsFlowMethodBody(t *testing.T) {
	tests := []struct {
		name   string
		body   kClient.UpdateSettingsFlowBody
		method string
	}{
		{
			name: "Password",
			body: func() kClient.UpdateSettingsFlowBody {
				flow := kClient.NewUpdateSettingsFlowWithPasswordMethodWithDefaults()
				flow.SetMethod("password")
				return kClient.UpdateSettingsFlowWithPasswordMethodAsUpdateSettingsFlowBody(flow)
			}(),
			method: "password",
		},
		{
			name: "Oidc",
			body: func() kClient.UpdateSettingsFlowBody {
				flow := kClient.NewUpdateSettingsFlowWithOidcMethodWithDefaults()
				flow.SetMethod("oidc")
				return kClient.UpdateSettingsFlowWithOidcMethodAsUpdateSettingsFlowBody(flow)
			}(),
			method: "oidc",
		},
		{
			name: "Totp",
			body: func() kClient.UpdateSettingsFlowBody {
				flow := kClient.NewUpdateSettingsFlowWithTotpMethodWithDefaults()
				flow.SetMethod("totp")
				return kClient.UpdateSettingsFlowWithTotpMethodAsUpdateSettingsFlowBody(flow)
			}(),
			method: "totp",
		},
		{
			name: "Lookup",
			body: func() kClient.UpdateSettingsFlowBody {
				flow := kClient.NewUpdateSettingsFlowWithLookupMethodWithDefaults()
				flow.SetMethod("lookup_secret")
				return kClient.UpdateSettingsFlowWithLookupMethodAsUpdateSettingsFlowBody(flow)
			}(),
			method: "lookup_secret",
		},
		{
			name: "WebAuthn",
			body: func() kClient.UpdateSettingsFlowBody {
				flow := kClient.NewUpdateSettingsFlowWithWebAuthnMethodWithDefaults()
				flow.SetMethod("webauthn")
				return kClient.UpdateSettingsFlowWithWebAuthnMethodAsUpdateSettingsFlowBody(flow)
			}(),
			method: "webauthn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

			jsonBody, _ := tt.body.MarshalJSON()
			req := httptest.NewRequest(http.MethodPost, "http://some/path", io.NopCloser(bytes.NewBuffer(jsonBody)))

			b, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).ParseSettingsFlowMethodBody(req)

			actual, _ := b.MarshalJSON()
			expected, _ := tt.body.MarshalJSON()

			if !reflect.DeepEqual(string(actual), string(expected)) {
				t.Fatalf("expected flow to be %s not %s", string(expected), string(actual))
			}
			if err != nil {
				t.Fatalf("expected error to be nil not  %v", err)
			}
		})
	}
}

func TestParseSettingsFlowMethodBody_ErrorsAndForm(t *testing.T) {
	tests := []struct {
		name           string
		setupRequest   func() *http.Request
		expectError    bool
		expectedErrMsg string
		assertBody     func(*testing.T, *kClient.UpdateSettingsFlowBody)
	}{
		{
			name: "ReadBodyError",
			setupRequest: func() *http.Request {
				errReader := io.NopCloser(readerFunc(func([]byte) (int, error) {
					return 0, fmt.Errorf("read error")
				}))
				return httptest.NewRequest(http.MethodPost, "http://some/path", errReader)
			},
			expectError:    true,
			expectedErrMsg: "unable to read body",
		},
		{
			name: "UnsupportedMethod",
			setupRequest: func() *http.Request {
				body := []byte(`{"method":"unknown"}`)
				return httptest.NewRequest(http.MethodPost, "http://some/path", io.NopCloser(bytes.NewBuffer(body)))
			},
			expectError:    true,
			expectedErrMsg: "upsupported method: unknown",
		},
		{
			name: "WebAuthnForm",
			setupRequest: func() *http.Request {
				form := "csrf_token=csrf&webauthn_register_displayname=John&webauthn_register=reg&webauthn_remove=rem"
				req := httptest.NewRequest(http.MethodPost, "http://some/path", io.NopCloser(strings.NewReader(form)))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req
			},
			expectError: false,
			assertBody: func(t *testing.T, body *kClient.UpdateSettingsFlowBody) {
				if body == nil || body.UpdateSettingsFlowWithWebAuthnMethod == nil {
					t.Fatalf("expected webauthn body to be set")
				}
				if body.UpdateSettingsFlowWithWebAuthnMethod.CsrfToken == nil || *body.UpdateSettingsFlowWithWebAuthnMethod.CsrfToken != "csrf" {
					t.Fatalf("expected csrf token to be %s", "csrf")
				}
				if body.UpdateSettingsFlowWithWebAuthnMethod.WebauthnRegisterDisplayname == nil || *body.UpdateSettingsFlowWithWebAuthnMethod.WebauthnRegisterDisplayname != "John" {
					t.Fatalf("expected display name to be %s", "John")
				}
				if body.UpdateSettingsFlowWithWebAuthnMethod.WebauthnRegister == nil || *body.UpdateSettingsFlowWithWebAuthnMethod.WebauthnRegister != "reg" {
					t.Fatalf("expected register to be %s", "reg")
				}
				if body.UpdateSettingsFlowWithWebAuthnMethod.WebauthnRemove == nil || *body.UpdateSettingsFlowWithWebAuthnMethod.WebauthnRemove != "rem" {
					t.Fatalf("expected remove to be %s", "rem")
				}
			},
		},
		{
			name: "InvalidJSONFallbackWebAuthn",
			setupRequest: func() *http.Request {
				body := []byte("{not-json")
				return httptest.NewRequest(http.MethodPost, "http://some/path", io.NopCloser(bytes.NewBuffer(body)))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)

			req := tt.setupRequest()
			body, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).ParseSettingsFlowMethodBody(req)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error not nil")
				}
				if tt.expectedErrMsg != "" && err.Error() != tt.expectedErrMsg {
					t.Fatalf("expected error %s got %v", tt.expectedErrMsg, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected error to be nil not %v", err)
			}
			if tt.assertBody != nil {
				tt.assertBody(t, body)
			}
		})
	}
}

type readerFunc func([]byte) (int, error)

func (r readerFunc) Read(p []byte) (int, error) {
	return r(p)
}

func TestGetSettingsFlow(t *testing.T) {
	tests := []struct {
		name                 string
		setupFlow            func() *kClient.SettingsFlow
		setupMocks           func(*MockFrontendAPI, *kClient.SettingsFlow)
		expectedError        bool
		expectedErrorMessage string
		expectNilResponse    bool
	}{
		{
			name: "Success",
			setupFlow: func() *kClient.SettingsFlow {
				return kClient.NewSettingsFlowWithDefaults()
			},
			setupMocks: func(mockKratosFrontendApi *MockFrontendAPI, flow *kClient.SettingsFlow) {
				mockKratosFrontendApi.EXPECT().GetSettingsFlowExecute(gomock.Any()).Times(1).DoAndReturn(
					func(r kClient.FrontendAPIGetSettingsFlowRequest) (*kClient.SettingsFlow, *http.Response, error) {
						id := "id"
						if _id := (*string)(reflect.ValueOf(r).FieldByName("id").UnsafePointer()); *_id != id {
							t.Fatalf("expected id to be %s, got %s", id, *_id)
						}
						return flow, &http.Response{StatusCode: http.StatusOK}, nil
					},
				)
			},
			expectedError:     false,
			expectNilResponse: true,
		},
		{
			name: "DuplicateIdentifier",
			setupFlow: func() *kClient.SettingsFlow {
				duplicateIdentifierMsg := kClient.UiText{
					Id:   4000007,
					Text: "duplicate identifier",
					Type: "error",
				}
				flow := kClient.NewSettingsFlowWithDefaults()
				flow.Ui = kClient.UiContainer{
					Messages: []kClient.UiText{duplicateIdentifierMsg},
				}
				return flow
			},
			setupMocks: func(mockKratosFrontendApi *MockFrontendAPI, flow *kClient.SettingsFlow) {
				mockKratosFrontendApi.EXPECT().GetSettingsFlowExecute(gomock.Any()).Times(1).DoAndReturn(
					func(r kClient.FrontendAPIGetSettingsFlowRequest) (*kClient.SettingsFlow, *http.Response, error) {
						id := "id"
						if _id := (*string)(reflect.ValueOf(r).FieldByName("id").UnsafePointer()); *_id != id {
							t.Fatalf("expected id to be %s, got %s", id, *_id)
						}
						return flow, &http.Response{StatusCode: http.StatusOK}, nil
					},
				)
			},
			expectedError:        true,
			expectedErrorMessage: "an account with the same identifier already exists, contact support",
		},
		{
			name: "Fail",
			setupFlow: func() *kClient.SettingsFlow {
				return kClient.NewSettingsFlowWithDefaults()
			},
			setupMocks: func(mockKratosFrontendApi *MockFrontendAPI, flow *kClient.SettingsFlow) {
				cookie := &http.Cookie{Name: "test", Value: "test"}
				resp := http.Response{
					Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
				}
				mockKratosFrontendApi.EXPECT().GetSettingsFlowExecute(gomock.Any()).Times(1).Return(nil, &resp, fmt.Errorf("error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
			mockKratosFrontendApi := NewMockFrontendAPI(ctrl)

			ctx := context.Background()
			cookies := make([]*http.Cookie, 0)
			cookie := &http.Cookie{Name: "test", Value: "test"}
			cookies = append(cookies, cookie)
			id := "id"

			flow := tt.setupFlow()
			request := kClient.FrontendAPIGetSettingsFlowRequest{
				ApiService: mockKratosFrontendApi,
			}

			mockTracer.EXPECT().Start(ctx, "kratos.Service.GetSettingsFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
			mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
			mockKratosFrontendApi.EXPECT().GetSettingsFlow(ctx).Times(1).Return(request)
			tt.setupMocks(mockKratosFrontendApi, flow)

			s, r, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).GetSettingsFlow(ctx, id, cookies)

			if tt.expectedError {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if tt.expectedErrorMessage != "" && !strings.Contains(err.Error(), tt.expectedErrorMessage) {
					t.Fatalf("unexpected error: %v", err)
				}
				if s != nil {
					t.Fatalf("expected flow to be %v not  %v", nil, s)
				}
				if r != nil {
					t.Fatalf("expected response to be %v not  %v", nil, r)
				}
			} else {
				if s != flow {
					t.Fatalf("expected flow to be %v not  %v", flow, s)
				}
				if tt.expectNilResponse && r != nil {
					t.Fatalf("expected response to be nil not  %v", r)
				}
				if err != nil {
					t.Fatalf("expected error to be nil not  %v", err)
				}
			}
		})
	}
}

func TestCreateBrowserSettingsFlow(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*MockFrontendAPI, *kClient.SettingsFlow, string)
		expectedError bool
	}{
		{
			name: "Success",
			setupMocks: func(mockKratosFrontendApi *MockFrontendAPI, flow *kClient.SettingsFlow, returnTo string) {
				mockKratosFrontendApi.EXPECT().CreateBrowserSettingsFlowExecute(gomock.Any()).Times(1).DoAndReturn(
					func(r kClient.FrontendAPICreateBrowserSettingsFlowRequest) (*kClient.SettingsFlow, *BrowserLocationChangeRequired, error) {
						if rt := (*string)(reflect.ValueOf(r).FieldByName("returnTo").UnsafePointer()); *rt != returnTo {
							t.Fatalf("expected returnTo to be %s, got %s", returnTo, *rt)
						}
						return flow, nil, nil
					},
				)
			},
			expectedError: false,
		},
		{
			name: "Fail",
			setupMocks: func(mockKratosFrontendApi *MockFrontendAPI, flow *kClient.SettingsFlow, returnTo string) {
				resp := http.Response{
					StatusCode: http.StatusNotFound,
				}
				mockKratosFrontendApi.EXPECT().CreateBrowserSettingsFlowExecute(gomock.Any()).Times(1).Return(nil, &resp, fmt.Errorf(""))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
			mockKratosFrontendApi := NewMockFrontendAPI(ctrl)

			ctx := context.Background()
			cookies := make([]*http.Cookie, 0)
			cookie := &http.Cookie{Name: "test", Value: "test"}
			cookies = append(cookies, cookie)
			returnTo := "https://example.com/ui/reset_complete"
			flow := kClient.NewSettingsFlowWithDefaults()
			request := kClient.FrontendAPICreateBrowserSettingsFlowRequest{
				ApiService: mockKratosFrontendApi,
			}

			mockTracer.EXPECT().Start(ctx, "kratos.Service.CreateBrowserSettingsFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
			mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
			mockKratosFrontendApi.EXPECT().CreateBrowserSettingsFlow(ctx).Times(1).Return(request)
			tt.setupMocks(mockKratosFrontendApi, flow, returnTo)

			f, r, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).CreateBrowserSettingsFlow(ctx, returnTo, cookies)

			if tt.expectedError {
				if f != nil {
					t.Fatalf("expected flow to be %v not  %v", nil, f)
				}
				if r != nil {
					t.Fatalf("expected response to be %v not  %v", nil, r)
				}
				if err == nil {
					t.Fatalf("expected error not nil")
				}
			} else {
				if f != flow {
					t.Fatalf("expected flow to be %v not  %v", flow, f)
				}
				if r != nil {
					t.Fatalf("expected response to be nil not %v", r)
				}
				if err != nil {
					t.Fatalf("expected error to be nil not  %v", err)
				}
			}
		})
	}
}

func TestUpdateSettingsFlow(t *testing.T) {
	tests := []struct {
		name             string
		setupMocks       func(*MockFrontendAPI, *MockLoggerInterface, *MockTracingInterface, *http.Response, *kClient.SettingsFlow)
		expectedError    bool
		checkRedirect    bool
		expectedRedirect string
	}{
		{
			name: "Success",
			setupMocks: func(mockKratosFrontendApi *MockFrontendAPI, mockLogger *MockLoggerInterface, mockTracer *MockTracingInterface, resp *http.Response, flow *kClient.SettingsFlow) {
				mockKratosFrontendApi.EXPECT().UpdateSettingsFlowExecute(gomock.Any()).Times(1).DoAndReturn(
					func(r kClient.FrontendAPIUpdateSettingsFlowRequest) (*kClient.SettingsFlow, *http.Response, error) {
						flowId := "flow"
						if _flow := (*string)(reflect.ValueOf(r).FieldByName("flow").UnsafePointer()); *_flow != flowId {
							t.Fatalf("expected id to be %s, got %s", flowId, *_flow)
						}
						return flow, resp, nil
					},
				)
			},
			expectedError: false,
		},
		{
			name: "FailOnExecute",
			setupMocks: func(mockKratosFrontendApi *MockFrontendAPI, mockLogger *MockLoggerInterface, mockTracer *MockTracingInterface, resp *http.Response, flow *kClient.SettingsFlow) {
				mockKratosFrontendApi.EXPECT().UpdateSettingsFlowExecute(gomock.Any()).Times(1).Return(nil, resp, fmt.Errorf("error"))
				mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)
			},
			expectedError: true,
		},
		{
			name: "ForbiddenStatus",
			setupMocks: func(mockKratosFrontendApi *MockFrontendAPI, mockLogger *MockLoggerInterface, mockTracer *MockTracingInterface, resp *http.Response, flow *kClient.SettingsFlow) {
				cookie := &http.Cookie{Name: "test", Value: "test"}
				resp.StatusCode = http.StatusForbidden
				resp.Header = http.Header{
					"Set-Cookie": []string{cookie.String()},
				}
				redirectTo := "http://kratos/self-service/login/browser?refresh=true"
				sessionRequiredErrorId := "session_refresh_required"
				errorPayload := ErrorBrowserLocationChangeRequired{
					Error: &kClient.GenericError{
						Id: &sessionRequiredErrorId,
					},
					RedirectBrowserTo: &redirectTo,
				}
				errorBodyJson, _ := json.Marshal(errorPayload)
				resp.Body = io.NopCloser(bytes.NewBuffer(errorBodyJson))

				mockTracer.EXPECT().Start(gomock.Any(), gomock.Any()).Times(1).Return(context.Background(), trace.SpanFromContext(context.Background()))
				mockKratosFrontendApi.EXPECT().UpdateSettingsFlowExecute(gomock.Any()).Times(1).Return(nil, resp, fmt.Errorf("forbidden"))
			},
			expectedError:    false,
			checkRedirect:    true,
			expectedRedirect: "http://kratos/self-service/login/browser?refresh=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
			mockKratosFrontendApi := NewMockFrontendAPI(ctrl)

			ctx := context.Background()
			cookies := make([]*http.Cookie, 0)
			cookie := &http.Cookie{Name: "test", Value: "test"}
			cookies = append(cookies, cookie)
			flowId := "flow"

			flow := kClient.NewSettingsFlowWithDefaults()
			flowJson, _ := json.Marshal(flow)
			body := new(kClient.UpdateSettingsFlowBody)
			request := kClient.FrontendAPIUpdateSettingsFlowRequest{
				ApiService: mockKratosFrontendApi,
			}
			resp := http.Response{
				Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
				Body:   io.NopCloser(bytes.NewBuffer(flowJson)),
			}

			mockTracer.EXPECT().Start(ctx, "kratos.Service.UpdateSettingsFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
			mockKratos.EXPECT().FrontendApi().Times(1).Return(mockKratosFrontendApi)
			mockKratosFrontendApi.EXPECT().UpdateSettingsFlow(ctx).Times(1).Return(request)
			tt.setupMocks(mockKratosFrontendApi, mockLogger, mockTracer, &resp, flow)

			f, r, c, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).UpdateSettingsFlow(ctx, flowId, *body, cookies)

			if tt.expectedError {
				if f != nil {
					t.Fatalf("expected flow to be %v not %+v", nil, f)
				}
				if r != nil {
					t.Fatalf("expected redirect info to be %v not %+v", nil, r)
				}
				if c != nil {
					t.Fatalf("expected header to be %v not %v", nil, c)
				}
				if err == nil {
					t.Fatalf("expected error not nil")
				}
			} else if tt.checkRedirect {
				if f != nil {
					t.Fatalf("expected flow to be %v, not %v", nil, f)
				}
				if r == nil {
					t.Fatalf("expected redirect info to be not nil")
				}
				if *r.RedirectTo != tt.expectedRedirect {
					t.Errorf("expected redirect url %s, got %s", tt.expectedRedirect, *r.RedirectTo)
				}
				if len(c) == 0 {
					t.Fatalf("expected cookies, got empty list")
				}
				if err != nil {
					t.Fatalf("expected error to be nil, got %v", err)
				}
			} else {
				if !reflect.DeepEqual(c, resp.Cookies()) {
					t.Fatalf("expected cookies to be %v not  %v", resp.Cookies(), c)
				}
				if err != nil {
					t.Fatalf("expected error to be nil not  %v", err)
				}
			}
		})
	}
}

func TestHasNotEnoughLookupSecretsLeft(t *testing.T) {
	tests := []struct {
		name           string
		identity       *kClient.Identity
		executeError   error
		expectedResult bool
		expectedError  bool
		expectDebug    bool
		expectErrorf   bool
	}{
		{
			name: "EnoughCodes",
			identity: &kClient.Identity{
				Id: "test",
				Credentials: func() *map[string]kClient.IdentityCredentials {
					creds := map[string]kClient.IdentityCredentials{
						"lookup_secret": {
							Config: map[string]interface{}{
								"recovery_codes": []map[string]interface{}{
									{"code": "a"},
									{"code": "b"},
									{"code": "c"},
									{"code": "d"},
								},
							},
						},
					}
					return &creds
				}(),
			},
			expectedResult: false,
			expectedError:  false,
		},
		{
			name: "NotEnoughCodes",
			identity: &kClient.Identity{
				Id: "test",
				Credentials: func() *map[string]kClient.IdentityCredentials {
					creds := map[string]kClient.IdentityCredentials{
						"lookup_secret": {
							Config: map[string]interface{}{
								"recovery_codes": []map[string]interface{}{
									{"code": "a"},
									{"code": "b"},
								},
							},
						},
					}
					return &creds
				}(),
			},
			expectedResult: true,
			expectedError:  false,
			expectDebug:    true,
		},
		{
			name: "MissingLookupSecret",
			identity: &kClient.Identity{
				Id: "test",
				Credentials: func() *map[string]kClient.IdentityCredentials {
					creds := map[string]kClient.IdentityCredentials{}
					return &creds
				}(),
			},
			expectedResult: false,
			expectedError:  false,
			expectDebug:    true,
		},
		{
			name: "MissingRecoveryCodes",
			identity: &kClient.Identity{
				Id: "test",
				Credentials: func() *map[string]kClient.IdentityCredentials {
					creds := map[string]kClient.IdentityCredentials{
						"lookup_secret": {
							Config: map[string]interface{}{},
						},
					}
					return &creds
				}(),
			},
			expectedResult: false,
			expectedError:  false,
			expectDebug:    true,
		},
		{
			name: "MarshalError",
			identity: &kClient.Identity{
				Id: "test",
				Credentials: func() *map[string]kClient.IdentityCredentials {
					creds := map[string]kClient.IdentityCredentials{
						"lookup_secret": {
							Config: map[string]interface{}{
								"recovery_codes": func() {},
							},
						},
					}
					return &creds
				}(),
			},
			expectedResult: false,
			expectedError:  true,
			expectErrorf:   true,
		},
		{
			name: "UnmarshalError",
			identity: &kClient.Identity{
				Id: "test",
				Credentials: func() *map[string]kClient.IdentityCredentials {
					creds := map[string]kClient.IdentityCredentials{
						"lookup_secret": {
							Config: map[string]interface{}{
								"recovery_codes": "invalid",
							},
						},
					}
					return &creds
				}(),
			},
			expectedResult: false,
			expectedError:  true,
			expectErrorf:   true,
		},
		{
			name:          "FailonGetIdentityExecute",
			identity:      nil,
			executeError:  fmt.Errorf("error"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
			mockKratosIdentityApi := NewMockIdentityAPI(ctrl)

			ctx := context.Background()
			cookie := &http.Cookie{Name: "test", Value: "test"}
			resp := http.Response{
				Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
			}
			identityRequest := kClient.IdentityAPIGetIdentityRequest{
				ApiService: mockKratosIdentityApi,
			}

			mockTracer.EXPECT().Start(ctx, "kratos.Service.HasNotEnoughLookupSecretsLeft").Times(1).Return(ctx, trace.SpanFromContext(ctx))
			mockAdminKratos.EXPECT().IdentityApi().Times(1).Return(mockKratosIdentityApi)
			mockKratosIdentityApi.EXPECT().GetIdentity(ctx, gomock.Any()).Times(1).Return(identityRequest)
			if tt.expectDebug {
				mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(1)
			}
			if tt.expectErrorf {
				mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)
			}
			identity := tt.identity
			if identity == nil {
				identity = &kClient.Identity{Id: "test"}
			}
			mockKratosIdentityApi.EXPECT().GetIdentityExecute(gomock.Any()).Times(1).Return(identity, &resp, tt.executeError)

			hasNotEnoughLookupSecretsLeft, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).HasNotEnoughLookupSecretsLeft(ctx, "test")

			if hasNotEnoughLookupSecretsLeft != tt.expectedResult {
				t.Fatalf("expected return value to be %v not %v", tt.expectedResult, hasNotEnoughLookupSecretsLeft)
			}
			if tt.expectedError {
				if err == nil {
					t.Fatalf("expected error not nil")
				}
			} else {
				if err != nil {
					t.Fatalf("expected error to be nil not %v", err)
				}
			}
		})
	}
}

func TestHasTOTPAvailable(t *testing.T) {
	tests := []struct {
		name          string
		identity      *kClient.Identity
		executeError  error
		expected      bool
		expectedError bool
	}{
		{
			name: "HasTotp",
			identity: &kClient.Identity{
				Id: "test",
				Credentials: func() *map[string]kClient.IdentityCredentials {
					creds := map[string]kClient.IdentityCredentials{
						"totp": {},
					}
					return &creds
				}(),
			},
			expected:      true,
			expectedError: false,
		},
		{
			name: "NoTotp",
			identity: &kClient.Identity{
				Id: "test",
				Credentials: func() *map[string]kClient.IdentityCredentials {
					creds := map[string]kClient.IdentityCredentials{}
					return &creds
				}(),
			},
			expected:      false,
			expectedError: false,
		},
		{
			name:          "ExecuteError",
			identity:      nil,
			executeError:  fmt.Errorf("error"),
			expected:      false,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
			mockKratosIdentityApi := NewMockIdentityAPI(ctrl)

			ctx := context.Background()
			cookie := &http.Cookie{Name: "test", Value: "test"}
			resp := http.Response{
				Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
			}
			identityRequest := kClient.IdentityAPIGetIdentityRequest{
				ApiService: mockKratosIdentityApi,
			}

			mockTracer.EXPECT().Start(ctx, "kratos.Service.HasTOTPAvailable").Times(1).Return(ctx, trace.SpanFromContext(ctx))
			mockAdminKratos.EXPECT().IdentityApi().Times(1).Return(mockKratosIdentityApi)
			mockKratosIdentityApi.EXPECT().GetIdentity(ctx, gomock.Any()).Times(1).Return(identityRequest)

			identity := tt.identity
			if identity == nil {
				identity = &kClient.Identity{Id: "test"}
			}
			mockKratosIdentityApi.EXPECT().GetIdentityExecute(gomock.Any()).Times(1).Return(identity, &resp, tt.executeError)

			hasTotp, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).HasTOTPAvailable(ctx, "test")

			if hasTotp != tt.expected {
				t.Fatalf("expected return value to be %v not %v", tt.expected, hasTotp)
			}
			if tt.expectedError {
				if err == nil {
					t.Fatalf("expected error not nil")
				}
			} else if err != nil {
				t.Fatalf("expected error to be nil not %v", err)
			}
		})
	}
}

func TestHydrateKratosLoginFlow(t *testing.T) {
	newLoginFlow := func(returnTo string) *kClient.LoginFlow {
		now := time.Now()
		ui := kClient.NewUiContainer("https://example.com/ui", "POST", []kClient.UiNode{})
		flow := kClient.NewLoginFlow(now.Add(time.Hour), "flow-id", now, "https://example.com/login", "state", "browser", *ui)
		flow.ReturnTo = &returnTo
		return flow
	}

	tests := []struct {
		name            string
		flow            *kClient.LoginFlow
		setupMocks      func(context.Context, *MockHydraClientInterface, *MockOAuth2API)
		expectChallenge string
		expectOauth2Req bool
		expectError     bool
		expectGetReq    bool
	}{
		{
			name: "NoLoginChallenge",
			flow: func() *kClient.LoginFlow {
				return newLoginFlow("https://example.com")
			}(),
			setupMocks:      func(context.Context, *MockHydraClientInterface, *MockOAuth2API) {},
			expectChallenge: "",
			expectOauth2Req: false,
			expectError:     false,
			expectGetReq:    false,
		},
		{
			name: "AlreadyHydrated",
			flow: func() *kClient.LoginFlow {
				flow := newLoginFlow("https://example.com?login_challenge=abc")
				flow.Oauth2LoginRequest = kClient.NewOAuth2LoginRequest()
				return flow
			}(),
			setupMocks:      func(context.Context, *MockHydraClientInterface, *MockOAuth2API) {},
			expectChallenge: "",
			expectOauth2Req: true,
			expectError:     false,
			expectGetReq:    false,
		},
		{
			name: "HydrateError",
			flow: func() *kClient.LoginFlow {
				return newLoginFlow("https://example.com?login_challenge=abc")
			}(),
			setupMocks: func(ctx context.Context, mockHydra *MockHydraClientInterface, mockHydraOauthApi *MockOAuth2API) {
				getLoginRequest := hClient.OAuth2APIGetOAuth2LoginRequestRequest{ApiService: mockHydraOauthApi}
				mockHydra.EXPECT().OAuth2API().Times(1).Return(mockHydraOauthApi)
				mockHydraOauthApi.EXPECT().GetOAuth2LoginRequest(ctx).Times(1).Return(getLoginRequest)
				mockHydraOauthApi.EXPECT().GetOAuth2LoginRequestExecute(gomock.Any()).Times(1).Return(nil, &http.Response{}, fmt.Errorf("error"))
			},
			expectChallenge: "abc",
			expectOauth2Req: false,
			expectError:     true,
			expectGetReq:    true,
		},
		{
			name: "HydrateSuccess",
			flow: func() *kClient.LoginFlow {
				return newLoginFlow("https://example.com?login_challenge=abc")
			}(),
			setupMocks: func(ctx context.Context, mockHydra *MockHydraClientInterface, mockHydraOauthApi *MockOAuth2API) {
				getLoginRequest := hClient.OAuth2APIGetOAuth2LoginRequestRequest{ApiService: mockHydraOauthApi}
				hydraLoginRequest := hClient.NewOAuth2LoginRequestWithDefaults()
				mockHydra.EXPECT().OAuth2API().Times(1).Return(mockHydraOauthApi)
				mockHydraOauthApi.EXPECT().GetOAuth2LoginRequest(ctx).Times(1).Return(getLoginRequest)
				mockHydraOauthApi.EXPECT().GetOAuth2LoginRequestExecute(gomock.Any()).Times(1).Return(hydraLoginRequest, &http.Response{}, nil)
			},
			expectChallenge: "abc",
			expectOauth2Req: true,
			expectError:     false,
			expectGetReq:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
			mockHydraOauthApi := NewMockOAuth2API(ctrl)

			ctx := context.Background()
			mockTracer.EXPECT().Start(ctx, "kratos.Service.hydrateKratosLoginFlow").Times(1).Return(ctx, trace.SpanFromContext(ctx))
			if tt.expectGetReq {
				mockTracer.EXPECT().Start(ctx, "kratos.Service.GetLoginRequest").Times(1).Return(ctx, trace.SpanFromContext(ctx))
			}
			tt.setupMocks(ctx, mockHydra, mockHydraOauthApi)

			flow, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).hydrateKratosLoginFlow(ctx, tt.flow)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error not nil")
				}
			} else if err != nil {
				t.Fatalf("expected error to be nil not %v", err)
			}
			if flow == nil {
				t.Fatalf("expected flow not nil")
			}
			if tt.expectChallenge != "" {
				if flow.Oauth2LoginChallenge == nil || *flow.Oauth2LoginChallenge != tt.expectChallenge {
					t.Fatalf("expected login challenge %s", tt.expectChallenge)
				}
			}
			if tt.expectOauth2Req {
				if flow.Oauth2LoginRequest == nil {
					t.Fatalf("expected oauth2 login request to be set")
				}
			} else if flow.Oauth2LoginRequest != nil && tt.name != "AlreadyHydrated" {
				t.Fatalf("expected oauth2 login request to be nil")
			}
		})
	}
}

func TestHasWebAuthnAvailable(t *testing.T) {
	tests := []struct {
		name           string
		identity       *kClient.Identity
		getIdentityErr error
		expected       bool
		expectedError  bool
		debugCalls     int
	}{
		{
			name:          "NoCredentials",
			identity:      &kClient.Identity{Id: "test"},
			expected:      false,
			expectedError: false,
			debugCalls:    1,
		},
		{
			name: "NoWebauthnCredentials",
			identity: func() *kClient.Identity {
				identity := kClient.Identity{Id: "test"}
				credentials := map[string]kClient.IdentityCredentials{}
				identity.Credentials = &credentials
				return &identity
			}(),
			expected:      false,
			expectedError: false,
			debugCalls:    1,
		},
		{
			name: "NoCredentialsList",
			identity: func() *kClient.Identity {
				identity := kClient.Identity{Id: "test"}
				credentials := map[string]kClient.IdentityCredentials{
					"webauthn": {Config: map[string]interface{}{"credentials": "invalid"}},
				}
				identity.Credentials = &credentials
				return &identity
			}(),
			expected:      false,
			expectedError: false,
			debugCalls:    1,
		},
		{
			name: "PasswordlessOnly",
			identity: func() *kClient.Identity {
				identity := kClient.Identity{Id: "test"}
				credentials := map[string]kClient.IdentityCredentials{
					"webauthn": {Config: map[string]interface{}{"credentials": []interface{}{map[string]interface{}{"is_passwordless": true}}}},
				}
				identity.Credentials = &credentials
				return &identity
			}(),
			expected:      false,
			expectedError: false,
			debugCalls:    0,
		},
		{
			name: "HasTwoFactorWebauthn",
			identity: func() *kClient.Identity {
				identity := kClient.Identity{Id: "test"}
				credentials := map[string]kClient.IdentityCredentials{
					"webauthn": {Config: map[string]interface{}{"credentials": []interface{}{map[string]interface{}{"is_passwordless": false}}}},
				}
				identity.Credentials = &credentials
				return &identity
			}(),
			expected:      true,
			expectedError: false,
			debugCalls:    1,
		},
		{
			name:           "FailOnGetIdentityExecute",
			identity:       nil,
			getIdentityErr: fmt.Errorf("error"),
			expected:       false,
			expectedError:  true,
			debugCalls:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockKratos := NewMockKratosClientInterface(ctrl)
			mockAdminKratos := NewMockKratosAdminClientInterface(ctrl)
			mockAuthz := NewMockAuthorizerInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockMonitor := monitoring.NewMockMonitorInterface(ctrl)
			mockKratosIdentityApi := NewMockIdentityAPI(ctrl)

			ctx := context.Background()
			cookie := &http.Cookie{Name: "test", Value: "test"}
			resp := http.Response{
				Header: http.Header{"Set-Cookie": []string{cookie.Raw}},
			}
			identityRequest := kClient.IdentityAPIGetIdentityRequest{
				ApiService: mockKratosIdentityApi,
			}

			mockTracer.EXPECT().Start(ctx, "kratos.Service.HasWebAuthnAvailable").Times(1).Return(ctx, trace.SpanFromContext(ctx))
			mockAdminKratos.EXPECT().IdentityApi().Times(1).Return(mockKratosIdentityApi)
			mockKratosIdentityApi.EXPECT().GetIdentity(ctx, gomock.Any()).Times(1).Return(identityRequest)
			if tt.debugCalls > 0 {
				mockLogger.EXPECT().Debugf(gomock.Any(), gomock.Any()).Times(tt.debugCalls)
			}
			if tt.expectedError {
				mockKratosIdentityApi.EXPECT().GetIdentityExecute(gomock.Any()).Times(1).Return(nil, &resp, tt.getIdentityErr)
			} else {
				mockKratosIdentityApi.EXPECT().GetIdentityExecute(gomock.Any()).Times(1).DoAndReturn(
					func(r kClient.IdentityAPIGetIdentityRequest) (*kClient.Identity, *http.Response, error) {
						return tt.identity, &resp, nil
					},
				)
			}

			hasWebAuthnAvailable, err := NewService(mockKratos, mockAdminKratos, mockHydra, mockAuthz, false, mockTracer, mockMonitor, mockLogger).HasWebAuthnAvailable(ctx, "test")

			if hasWebAuthnAvailable != tt.expected {
				t.Fatalf("expected return value to be %v not %v", tt.expected, hasWebAuthnAvailable)
			}
			if tt.expectedError {
				if err == nil {
					t.Fatalf("expected error not nil")
				}
			} else {
				if err != nil {
					t.Fatalf("expected error to be nil not %v", err)
				}
			}
		})
	}
}

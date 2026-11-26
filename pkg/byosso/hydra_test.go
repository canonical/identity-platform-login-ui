// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"

	hClient "github.com/ory/hydra-client-go/v26"
	kClient "github.com/ory/kratos-client-go/v25"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
)

//go:generate mockgen -build_flags=--mod=mod -package byosso -destination ./mock_hydra.go -source=../../internal/hydra/interfaces.go

func TestHydraSuccess(t *testing.T) {
	ctx := context.Background()
	requestURL := "https://hydra.example/oauth2/auth?client_id=app&prompt=login&max_age=60"
	maxAge := 60 * time.Second
	loginRequest := &LoginRequest{PromptLogin: true, MaxAge: &maxAge, Subject: "test.subject", SessionID: "test.sid", RequestURL: requestURL}

	// each call is made once with what it was given, and its answer is returned
	tests := []struct {
		name     string
		span     string
		call     func(t *testing.T, api *MockOAuth2API, h *Hydra) (any, error)
		expected any
	}{
		{name: "LoginRequest", span: "hydra.OAuth2API.GetOAuth2LoginRequest", expected: loginRequest, call: func(t *testing.T, api *MockOAuth2API, h *Hydra) (any, error) {
			login := hClient.NewOAuth2LoginRequestWithDefaults()
			login.SetRequestUrl(requestURL)
			login.SetSubject("test.subject")
			login.SetSessionId("test.sid")
			api.EXPECT().GetOAuth2LoginRequest(ctx).Times(1).Return(hClient.OAuth2APIGetOAuth2LoginRequestRequest{ApiService: api})
			api.EXPECT().GetOAuth2LoginRequestExecute(gomock.Any()).Times(1).DoAndReturn(
				func(r hClient.OAuth2APIGetOAuth2LoginRequestRequest) (*hClient.OAuth2LoginRequest, *http.Response, error) {
					// use reflect as loginChallenge is a private attribute, also is a string pointer so need to cast it multiple times
					if challenge := (*string)(reflect.ValueOf(r).FieldByName("loginChallenge").UnsafePointer()); *challenge != "test.challenge" {
						t.Fatalf("expected challenge string as test.challenge, got %s", *challenge)
					}
					return login, new(http.Response), nil
				},
			)
			return h.LoginRequest(ctx, "test.challenge")
		}},
		{name: "RevokeLoginSession", span: "hydra.OAuth2API.RevokeOAuth2LoginSessions", call: func(t *testing.T, api *MockOAuth2API, h *Hydra) (any, error) {
			api.EXPECT().RevokeOAuth2LoginSessions(ctx).Times(1).Return(hClient.OAuth2APIRevokeOAuth2LoginSessionsRequest{ApiService: api})
			api.EXPECT().RevokeOAuth2LoginSessionsExecute(gomock.Any()).Times(1).DoAndReturn(
				func(r hClient.OAuth2APIRevokeOAuth2LoginSessionsRequest) (*http.Response, error) {
					if sid := (*string)(reflect.ValueOf(r).FieldByName("sid").UnsafePointer()); *sid != "test.sid" {
						t.Fatalf("expected sid as test.sid, got %s", *sid)
					}
					// only the session with that sid is revoked
					if subject := (*string)(reflect.ValueOf(r).FieldByName("subject").UnsafePointer()); subject != nil {
						t.Fatalf("expected subject to be nil, got %s", *subject)
					}
					return new(http.Response), nil
				},
			)
			return nil, h.RevokeLoginSession(ctx, "test.sid")
		}},
		{name: "RejectConsent", span: "hydra.OAuth2API.RejectOAuth2ConsentRequest", expected: "https://test.com/test", call: func(t *testing.T, api *MockOAuth2API, h *Hydra) (any, error) {
			api.EXPECT().RejectOAuth2ConsentRequest(ctx).Times(1).Return(hClient.OAuth2APIRejectOAuth2ConsentRequestRequest{ApiService: api})
			api.EXPECT().RejectOAuth2ConsentRequestExecute(gomock.Any()).Times(1).DoAndReturn(
				func(r hClient.OAuth2APIRejectOAuth2ConsentRequestRequest) (*hClient.OAuth2RedirectTo, *http.Response, error) {
					if challenge := (*string)(reflect.ValueOf(r).FieldByName("consentChallenge").UnsafePointer()); *challenge != "test.challenge" {
						t.Fatalf("expected challenge string as test.challenge, got %s", *challenge)
					}
					// use reflect as rejectOAuth2Request is a private attribute, also is a pointer so need to cast it multiple times
					reject := (*hClient.RejectOAuth2Request)(reflect.ValueOf(r).FieldByName("rejectOAuth2Request").UnsafePointer())
					if reject.GetError() != "login_required" || reject.GetErrorDescription() != "test.description" {
						t.Fatalf("expected login_required with the description, got %s: %s", reject.GetError(), reject.GetErrorDescription())
					}
					return hClient.NewOAuth2RedirectTo("https://test.com/test"), new(http.Response), nil
				},
			)
			return h.RejectConsent(ctx, "test.challenge", "test.description")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockLogger := NewMockLoggerInterface(ctrl)
			mockHydra := NewMockHydraClientInterface(ctrl)
			mockTracer := NewMockTracingInterface(ctrl)
			mockHydraOAuth2API := NewMockOAuth2API(ctrl)

			mockTracer.EXPECT().Start(ctx, tt.span).Times(1).Return(ctx, trace.SpanFromContext(ctx))
			mockHydra.EXPECT().OAuth2API().Times(1).Return(mockHydraOAuth2API)

			got, err := tt.call(t, mockHydraOAuth2API, NewHydra(mockHydra, mockTracer, mockLogger))

			expectNoError(t, err)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Fatalf("expected %+v, got %+v", tt.expected, got)
			}
		})
	}
}

func TestParseLoginRequestURL(t *testing.T) {
	maxAge := 300 * time.Second

	tests := []struct {
		name       string
		requestURL string
		expected   LoginRequest
	}{
		{name: "prompt login and max age", requestURL: "https://hydra.example/oauth2/auth?client_id=app&prompt=login%20consent&max_age=300", expected: LoginRequest{PromptLogin: true, MaxAge: &maxAge}},
		{name: "another prompt", requestURL: "https://hydra.example/oauth2/auth?prompt=consent"},
		{name: "no prompt and no max age", requestURL: "https://hydra.example/oauth2/auth?client_id=app"},
		{name: "negative max age", requestURL: "https://hydra.example/oauth2/auth?max_age=-1"},
		{name: "invalid url", requestURL: "://"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if lr := parseLoginRequestURL(tt.requestURL); !reflect.DeepEqual(*lr, tt.expected) {
				t.Fatalf("expected login request %+v, got %+v", tt.expected, *lr)
			}
		})
	}
}

func TestRemembersAnother(t *testing.T) {
	tests := []struct {
		name     string
		lr       *LoginRequest
		expected bool
	}{
		{name: "another subject", lr: &LoginRequest{Subject: "other", SessionID: "sid", RequestURL: "https://hydra.example/oauth2/auth"}, expected: true},
		{name: "same subject", lr: &LoginRequest{Subject: "iid", SessionID: "sid", RequestURL: "https://hydra.example/oauth2/auth"}, expected: false},
		{name: "no subject", lr: &LoginRequest{SessionID: "sid", RequestURL: "https://hydra.example/oauth2/auth"}, expected: false},
		{name: "no session id", lr: &LoginRequest{Subject: "other", RequestURL: "https://hydra.example/oauth2/auth"}, expected: false},
		{name: "no login request", lr: nil, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if remembers := tt.lr.RemembersAnother("iid"); remembers != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, remembers)
			}
		})
	}
}

func TestReauthenticate(t *testing.T) {
	tests := []struct {
		name       string
		requestURL string
		session    *kClient.Session
		now        time.Time
		expected   bool
	}{
		{name: "prompt login without session", requestURL: "https://hydra.example/oauth2/auth?prompt=login", session: nil, now: t0, expected: true},
		{name: "max age zero without session", requestURL: "https://hydra.example/oauth2/auth?max_age=0", session: nil, now: t0, expected: true},
		{name: "max age without session", requestURL: "https://hydra.example/oauth2/auth?max_age=60", session: nil, now: t0, expected: false},
		{name: "max age not exceeded", requestURL: "https://hydra.example/oauth2/auth?max_age=60", session: passwordSession("s"), now: t0.Add(59 * time.Second), expected: false},
		{name: "max age exceeded", requestURL: "https://hydra.example/oauth2/auth?max_age=60", session: passwordSession("s"), now: t0.Add(61 * time.Second), expected: true},
		{name: "no prompt and no max age", requestURL: "https://hydra.example/oauth2/auth", session: passwordSession("s"), now: t0.Add(24 * time.Hour), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if reauthenticate := parseLoginRequestURL(tt.requestURL).Reauthenticate(tt.session, tt.now); reauthenticate != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, reauthenticate)
			}
		})
	}
}

func TestNeedsFreshFirstFactor(t *testing.T) {
	tests := []struct {
		name       string
		requestURL string
		session    *kClient.Session
		now        time.Time
		expected   bool
	}{
		{name: "prompt login", requestURL: "https://hydra.example/oauth2/auth?prompt=login", session: passwordSession("s"), now: t0, expected: true},
		{name: "max age not exceeded", requestURL: "https://hydra.example/oauth2/auth?max_age=60", session: passwordSession("s"), now: t0.Add(59 * time.Second), expected: false},
		{name: "max age exceeded", requestURL: "https://hydra.example/oauth2/auth?max_age=60", session: passwordSession("s"), now: t0.Add(61 * time.Second), expected: true},
		{name: "max age and session without first factor", requestURL: "https://hydra.example/oauth2/auth?max_age=60", session: sessionWith("s"), now: t0, expected: true},
		{name: "no prompt and no max age", requestURL: "https://hydra.example/oauth2/auth", session: passwordSession("s"), now: t0.Add(24 * time.Hour), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if needed := parseLoginRequestURL(tt.requestURL).NeedsFreshFirstFactor(tt.session, tt.now); needed != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, needed)
			}
		})
	}
}

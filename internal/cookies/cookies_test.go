// Copyright 2024 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package cookies

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
)

//go:generate mockgen -build_flags=--mod=mod -package cookies -destination ./mock_logger.go -source=../logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package cookies -destination ./mock_cookies.go -source=./interfaces.go

func findCookie(name string, cookies []*http.Cookie) (*http.Cookie, bool) {
	for _, cookie := range cookies {
		if name == cookie.Name {
			return cookie, true
		}
	}
	return nil, false
}

func TestAuthCookieManager_ClearStateCookie(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockLogger := NewMockLoggerInterface(ctrl)
	mockEncrypt := NewMockEncryptInterface(ctrl)

	mockRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	mockRequest.AddCookie(&http.Cookie{Name: "state"})

	mockResponse := httptest.NewRecorder()

	manager := NewAuthCookieManager(5, mockEncrypt, mockLogger)
	manager.ClearStateCookie(mockResponse)

	c, _ := findCookie("login_ui_state", mockResponse.Result().Cookies())

	if c.Expires != time.Unix(0, 0).UTC() {
		t.Fatal("did not clear state cookie")
	}
}

func TestAuthCookieManager_GetStateCookie(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockLogger := NewMockLoggerInterface(ctrl)
	mockEncrypt := NewMockEncryptInterface(ctrl)

	state := FlowStateCookie{}
	sj, _ := json.Marshal(state)

	mockEncrypt.EXPECT().Decrypt("mock-state").Return(string(sj), nil)

	mockRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	mockRequest.AddCookie(&http.Cookie{Name: "login_ui_state", Value: "mock-state"})

	manager := NewAuthCookieManager(5, mockEncrypt, mockLogger)
	cookie, err := manager.GetStateCookie(mockRequest)

	if cookie != state {
		t.Fatal("state cookie value does not match expected")
	}

	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestAuthCookieManager_GetStateCookieNoCookie(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockLogger := NewMockLoggerInterface(ctrl)
	mockRequest := httptest.NewRequest(http.MethodGet, "/", nil)

	manager := NewAuthCookieManager(5, nil, mockLogger)
	cookie, err := manager.GetStateCookie(mockRequest)

	state := FlowStateCookie{}
	if cookie != state {
		t.Fatal("state cookie value does not match expected")
	}

	if err != nil {
		t.Fatalf("expected error to be nil, not %v", err)
	}
}

func TestAuthCookieManager_GetStateCookieDecryptFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockError := errors.New("mock-error")

	mockLogger := NewMockLoggerInterface(ctrl)
	mockLogger.EXPECT().Errorf("cannot decrypt cookie value: %v", mockError).Times(1)
	mockEncrypt := NewMockEncryptInterface(ctrl)
	mockEncrypt.EXPECT().Decrypt("mock-state").Return("", mockError)

	mockRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	mockRequest.AddCookie(&http.Cookie{Name: "login_ui_state", Value: "mock-state"})

	manager := NewAuthCookieManager(5, mockEncrypt, mockLogger)
	cookie, err := manager.GetStateCookie(mockRequest)

	state := FlowStateCookie{}
	if cookie != state {
		t.Fatal("state cookie value does not match expected")
	}

	if err == nil {
		t.Fatalf("expected error to be not nil")
	}
}

func TestAuthCookieManager_SetStateCookie(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockLogger := NewMockLoggerInterface(ctrl)
	mockEncrypt := NewMockEncryptInterface(ctrl)

	state := FlowStateCookie{}
	js, _ := json.Marshal(state)

	mockEncrypt.EXPECT().Encrypt(string(js)).Return("mock-state", nil)

	mockResponse := httptest.NewRecorder()

	manager := NewAuthCookieManager(5, mockEncrypt, mockLogger)
	err := manager.SetStateCookie(mockResponse, state)

	c, found := findCookie("login_ui_state", mockResponse.Result().Cookies())
	if !found {
		t.Fatal("did not set state cookie")
	}

	if c.Value != "mock-state" {
		t.Fatal("state cookie value does not match expected")
	}

	if err != nil {
		t.Fatalf("expected error to be nil not  %v", err)
	}
}

func TestAuthCookieManager_SetStateCookieFailure(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockError := errors.New("mock-error")
	state := FlowStateCookie{}
	js, _ := json.Marshal(state)

	mockLogger := NewMockLoggerInterface(ctrl)
	mockLogger.EXPECT().Errorf("cannot encrypt cookie value: %v", mockError).Times(1)
	mockEncrypt := NewMockEncryptInterface(ctrl)
	mockEncrypt.EXPECT().Encrypt(string(js)).Return("", mockError)

	mockResponse := httptest.NewRecorder()

	manager := NewAuthCookieManager(5, mockEncrypt, mockLogger)
	err := manager.SetStateCookie(mockResponse, state)

	if err == nil {
		t.Fatalf("expected error to be not nil")
	}
}

func TestSignedInFor(t *testing.T) {
	started := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	after, before := started.Add(time.Second), started.Add(-time.Second)

	// through JSON, as the cookie stores it
	var c FlowStateCookie
	raw, _ := json.Marshal(FlowStateCookie{}.StartLogin("ch-1", started))
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		name            string
		cookie          FlowStateCookie
		challenge       string
		authenticatedAt *time.Time
		expect          bool
	}{
		{name: "authenticated after the login started", cookie: c, challenge: "ch-1", authenticatedAt: &after, expect: true},
		{name: "authenticated before the login started", cookie: c, challenge: "ch-1", authenticatedAt: &before, expect: false},
		{name: "authenticated when the login started", cookie: c, challenge: "ch-1", authenticatedAt: &started, expect: false},
		{name: "no login started", cookie: FlowStateCookie{LoginChallengeHash: ChallengeHash("ch-1")}, challenge: "ch-1", authenticatedAt: &after, expect: false},
		{name: "another challenge", cookie: c, challenge: "ch-2", authenticatedAt: &after, expect: false},
		{name: "no authentication time", cookie: c, challenge: "ch-1", expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cookie.SignedInFor(tt.challenge, tt.authenticatedAt); got != tt.expect {
				t.Fatalf("expected %v, got %v", tt.expect, got)
			}
		})
	}
}

func TestStartLoginAndRenewForChallenge(t *testing.T) {
	started := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	c := FlowStateCookie{}.StartLogin("ch-1", started)
	c.TenantID = "t1"
	c.TotpSetup = true

	if again := c.StartLogin("ch-1", started.Add(time.Minute)); !again.LoginStartedAt.Equal(started) || again.TenantID != "t1" {
		t.Fatalf("expected the first start and the tenant to be kept for the same challenge, got %+v", again)
	}
	if other := c.RenewForChallenge("ch-2"); !other.LoginStartedAt.IsZero() || other.TenantID != "" || other.TotpSetup {
		t.Fatalf("expected nothing to be carried to another challenge, got %+v", other)
	}
}

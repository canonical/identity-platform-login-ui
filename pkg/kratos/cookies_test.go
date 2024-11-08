// Copyright 2024 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

package kratos

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"
)

//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_interfaces.go -source=./interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_tracing.go go.opentelemetry.io/otel/trace Tracer
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_monitor.go -source=../../internal/monitoring/interfaces.go

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

	if c.Expires != epoch {
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
	mockLogger.EXPECT().Errorf("can't decrypt cookie value, %v", mockError).Times(1)

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
	mockLogger.EXPECT().Errorf("can't encrypt cookie value, %v", mockError).Times(1)

	mockEncrypt := NewMockEncryptInterface(ctrl)
	mockEncrypt.EXPECT().Encrypt(string(js)).Return("", mockError)

	mockResponse := httptest.NewRecorder()

	manager := NewAuthCookieManager(5, mockEncrypt, mockLogger)
	err := manager.SetStateCookie(mockResponse, state)

	if err == nil {
		t.Fatalf("expected error to be not nil")
	}
}

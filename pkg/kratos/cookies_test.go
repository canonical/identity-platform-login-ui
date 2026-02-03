// Copyright 2024 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

// Package kratos provides unit tests for cookie management functionality.
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
	tests := []struct {
		name string
	}{
		{
			name: "ClearState",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		})
	}
}

func TestAuthCookieManager_GetStateCookie(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockEncryptInterface, *MockLoggerInterface)
		requestCookie  *http.Cookie
		expectedCookie FlowStateCookie
		expectedErr    bool
	}{
		{
			name: "Success",
			setupMocks: func(mockEncrypt *MockEncryptInterface, mockLogger *MockLoggerInterface) {
				state := FlowStateCookie{}
				sj, _ := json.Marshal(state)
				mockEncrypt.EXPECT().Decrypt("mock-state").Return(string(sj), nil)
			},
			requestCookie:  &http.Cookie{Name: "login_ui_state", Value: "mock-state"},
			expectedCookie: FlowStateCookie{},
			expectedErr:    false,
		},
		{
			name:           "NoCookie",
			setupMocks:     func(mockEncrypt *MockEncryptInterface, mockLogger *MockLoggerInterface) {},
			requestCookie:  nil,
			expectedCookie: FlowStateCookie{},
			expectedErr:    false,
		},
		{
			name: "DecryptFailure",
			setupMocks: func(mockEncrypt *MockEncryptInterface, mockLogger *MockLoggerInterface) {
				mockError := errors.New("mock-error")
				mockLogger.EXPECT().Errorf("can't decrypt cookie value, %v", mockError).Times(1)
				mockEncrypt.EXPECT().Decrypt("mock-state").Return("", mockError)
			},
			requestCookie:  &http.Cookie{Name: "login_ui_state", Value: "mock-state"},
			expectedCookie: FlowStateCookie{},
			expectedErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockLogger := NewMockLoggerInterface(ctrl)
			mockEncrypt := NewMockEncryptInterface(ctrl)

			if tt.requestCookie == nil {
				mockEncrypt = nil
			}
			if mockEncrypt != nil {
				tt.setupMocks(mockEncrypt, mockLogger)
			} else {
				tt.setupMocks(nil, mockLogger)
			}

			mockRequest := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.requestCookie != nil {
				mockRequest.AddCookie(tt.requestCookie)
			}

			manager := NewAuthCookieManager(5, mockEncrypt, mockLogger)
			cookie, err := manager.GetStateCookie(mockRequest)

			if cookie != tt.expectedCookie {
				t.Fatal("state cookie value does not match expected")
			}

			if tt.expectedErr {
				if err == nil {
					t.Fatalf("expected error to be not nil")
				}
			} else if err != nil {
				t.Fatalf("expected error to be nil not  %v", err)
			}
		})
	}
}

func TestAuthCookieManager_SetStateCookie(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*MockEncryptInterface, *MockLoggerInterface)
		expectedErr bool
	}{
		{
			name: "Success",
			setupMocks: func(mockEncrypt *MockEncryptInterface, mockLogger *MockLoggerInterface) {
				state := FlowStateCookie{}
				js, _ := json.Marshal(state)
				mockEncrypt.EXPECT().Encrypt(string(js)).Return("mock-state", nil)
			},
			expectedErr: false,
		},
		{
			name: "Failure",
			setupMocks: func(mockEncrypt *MockEncryptInterface, mockLogger *MockLoggerInterface) {
				mockError := errors.New("mock-error")
				state := FlowStateCookie{}
				js, _ := json.Marshal(state)
				mockLogger.EXPECT().Errorf("can't encrypt cookie value, %v", mockError).Times(1)
				mockEncrypt.EXPECT().Encrypt(string(js)).Return("", mockError)
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mockLogger := NewMockLoggerInterface(ctrl)
			mockEncrypt := NewMockEncryptInterface(ctrl)

			tt.setupMocks(mockEncrypt, mockLogger)

			state := FlowStateCookie{}
			mockResponse := httptest.NewRecorder()

			manager := NewAuthCookieManager(5, mockEncrypt, mockLogger)
			err := manager.SetStateCookie(mockResponse, state)

			if tt.expectedErr {
				if err == nil {
					t.Fatalf("expected error to be not nil")
				}
				return
			}

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
		})
	}
}

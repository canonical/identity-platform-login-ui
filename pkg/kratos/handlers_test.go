// Copyright 2024 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

// Package kratos provides unit tests for Kratos handlers.
package kratos

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"
	gomock "go.uber.org/mock/gomock"

	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go/v25"
)

const (
	BASE_URL                                      = "https://example.com"
	HANDLE_CREATE_FLOW_URL                        = BASE_URL + "/api/kratos/self-service/login/browser"
	HANDLE_UPDATE_LOGIN_FLOW_URL                  = BASE_URL + "/api/kratos/self-service/login"
	HANDLE_UPDATE_IDENTIFIER_FIRST_LOGIN_FLOW_URL = BASE_URL + "/api/kratos/self-service/login/id-first"
	HANDLE_GET_LOGIN_FLOW_URL                     = BASE_URL + "/api/kratos/self-service/login/flows"
	HANDLE_ERROR_URL                              = BASE_URL + "/api/kratos/self-service/errors"
	HANDLE_CREATE_RECOVERY_FLOW_URL               = BASE_URL + "/api/kratos/self-service/recovery/browser"
	HANDLE_UPDATE_RECOVERY_FLOW_URL               = BASE_URL + "/api/kratos/self-service/recovery"
	HANDLE_GET_RECOVERY_FLOW_URL                  = BASE_URL + "/api/kratos/self-service/recovery/flows"
	HANDLE_CREATE_SETTINGS_FLOW_URL               = BASE_URL + "/api/kratos/self-service/settings/browser"
	HANDLE_UPDATE_SETTINGS_FLOW_URL               = BASE_URL + "/api/kratos/self-service/settings"
	HANDLE_GET_SETTINGS_FLOW_URL                  = BASE_URL + "/api/kratos/self-service/settings/flows"
)

//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_kratos.go -source=./interfaces.go

func TestHandleCreateFlowWithoutParams(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	flow := kClient.NewLoginFlowWithDefaults()
	flow.Id = "test"

	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_FLOW_URL, nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected HTTP status code 400 got %v", res.StatusCode)
	}
}

func TestHandleCreateFlowWithoutSession(t *testing.T) {
	tests := []struct {
		name           string
		acceptHeader   string
		createFlowErr  error
		filterErr      error
		expectStatus   int
		expectJSON     bool
		expectLocation bool
		expectLog      bool
	}{
		{
			name:         "AcceptJSON",
			acceptHeader: "application/json, text/plain, */*",
			expectStatus: http.StatusOK,
			expectJSON:   true,
		},
		{
			name:           "NotAcceptJSON",
			acceptHeader:   "application/x-www-form-urlencoded",
			expectStatus:   http.StatusSeeOther,
			expectLocation: true,
		},
		{
			name:          "FailOnCreateBrowserLoginFlow",
			createFlowErr: fmt.Errorf("error"),
			expectStatus:  http.StatusInternalServerError,
			expectLog:     true,
		},
		{
			name:         "FailOnFilterProviders",
			filterErr:    fmt.Errorf("oh no"),
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:         "NoProvidersAllowedAcceptJSON",
			acceptHeader: "application/json, text/plain, */*",
			expectStatus: http.StatusOK,
			expectJSON:   true,
		},
		{
			name:           "NoProvidersAllowedNotAcceptJSON",
			acceptHeader:   "application/x-www-form-urlencoded",
			expectStatus:   http.StatusSeeOther,
			expectLocation: true,
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

			flow := kClient.NewLoginFlowWithDefaults()
			flow.Id = "test"
			flow.State = "passed_challenge"

			loginChallenge := "login_challenge_2341235123231"
			returnTo, _ := url.JoinPath(BASE_URL, "ui/login")
			returnTo = returnTo + "?login_challenge=" + loginChallenge

			req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_FLOW_URL, nil)
			values := req.URL.Query()
			values.Add("login_challenge", loginChallenge)
			req.URL.RawQuery = values.Encode()
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}

			mockCookieManager.EXPECT().GetStateCookie(gomock.Any()).Return(FlowStateCookie{}, nil)
			mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(nil, nil, nil)
			mockService.EXPECT().MustReAuthenticate(gomock.Any(), loginChallenge, nil, FlowStateCookie{}).Return(true, nil)

			if tt.createFlowErr != nil {
				if tt.expectLog {
					mockLogger.EXPECT().Errorf("failed to create login flow, err: error")
				}
				mockService.EXPECT().CreateBrowserLoginFlow(gomock.Any(), gomock.Any(), returnTo, loginChallenge, gomock.Any(), req.Cookies()).Return(nil, nil, tt.createFlowErr)
			} else {
				mockService.EXPECT().CreateBrowserLoginFlow(gomock.Any(), gomock.Any(), returnTo, loginChallenge, gomock.Any(), req.Cookies()).Return(flow, req.Cookies(), nil)
				if tt.filterErr != nil {
					mockService.EXPECT().FilterFlowProviderList(gomock.Any(), flow).Return(nil, tt.filterErr)
					mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).AnyTimes()
				} else {
					mockService.EXPECT().FilterFlowProviderList(gomock.Any(), flow).Return(flow, nil)
				}
			}

			w := httptest.NewRecorder()
			mux := chi.NewMux()
			NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

			mux.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.expectStatus {
				t.Fatalf("expected HTTP status code %d got %v", tt.expectStatus, res.Status)
			}

			if tt.expectJSON {
				data, err := io.ReadAll(res.Body)
				if err != nil {
					t.Errorf("expected error to be nil got %v", err)
				}
				loginFlow := kClient.NewLoginFlowWithDefaults()
				if err := json.Unmarshal(data, loginFlow); err != nil {
					t.Errorf("expected error to be nil got %v", err)
				}
				if loginFlow.Id != flow.Id {
					t.Fatalf("Invalid flow id, expected: %s, got: %s", flow.Id, loginFlow.Id)
				}
			}

			if tt.expectLocation {
				location, _ := url.JoinPath(BASE_URL, "ui/login")
				location = fmt.Sprintf("%s?flow=%s", location, flow.Id)
				if res.Header.Get("Location") != location {
					t.Fatalf("Invalid location, expected: %s, got: %s", location, res.Header.Get("Location"))
				}
			}
		})
	}
}

func TestHandleCreateFlowRedirectToSetupWebauthn(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	flow := kClient.NewLoginFlowWithDefaults()
	flow.Id = "test"
	flow.State = "passed_challenge"

	loginChallenge := "login_challenge_2341235123231"

	session := kClient.NewSession("test")
	session.Identity = kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
	method := "oidc"
	aal := kClient.AUTHENTICATORASSURANCELEVEL_AAL1
	session.AuthenticationMethods = []kClient.SessionAuthenticationMethod{{Method: &method}}
	session.AuthenticatorAssuranceLevel = &aal

	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("login_challenge", loginChallenge)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil)
	mockTracer.EXPECT().Start(gomock.Any(), "kratos.API.shouldEnforceMFAWithSession").Return(context.Background(), trace.SpanFromContext(context.Background())).AnyTimes()
	mockTracer.EXPECT().Start(gomock.Any(), "kratos.API.shouldEnforceWebAuthnWithSession").Return(context.Background(), trace.SpanFromContext(context.Background())).AnyTimes()
	mockService.EXPECT().HasWebAuthnAvailable(gomock.Any(), session.Id).Return(false, nil)
	mockCookieManager.EXPECT().SetStateCookie(gomock.Any(), gomock.Any()).Return(nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, true, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP status code 200 got %v", res.StatusCode)
	}
	loginFlow := BrowserLocationChangeRequired{}
	if err := json.Unmarshal(data, &loginFlow); err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	if !strings.HasPrefix(*loginFlow.RedirectTo, "/ui/setup_passkey") {
		t.Errorf("expected redirect_to to start with '/ui/setup_passkey' got %v", *loginFlow.RedirectTo)
	}
}

func TestHandleCreateFlowWithSession(t *testing.T) {
	tests := []struct {
		name          string
		acceptHeader  string
		acceptErr     error
		expectStatus  int
		expectSuccess bool
	}{
		{
			name:          "AcceptJSON",
			acceptHeader:  "application/json, text/plain, */*",
			expectStatus:  http.StatusOK,
			expectSuccess: true,
		},
		{
			name:          "NotAcceptJSON",
			acceptHeader:  "application/x-www-form-urlencoded",
			expectStatus:  http.StatusOK,
			expectSuccess: true,
		},
		{
			name:          "FailOnAcceptLoginRequest",
			acceptErr:     fmt.Errorf("error"),
			expectStatus:  http.StatusInternalServerError,
			expectSuccess: false,
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

			session := kClient.NewSession("test")
			redirect := "https://some/path/to/somewhere"
			redirectTo := BrowserLocationChangeRequired{RedirectTo: &redirect}

			loginChallenge := "login_challenge_2341235123231"

			req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_FLOW_URL, nil)
			values := req.URL.Query()
			values.Add("login_challenge", loginChallenge)
			req.URL.RawQuery = values.Encode()
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}

			mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil)
			mockTracer.EXPECT().Start(gomock.Any(), "kratos.API.shouldEnforceMFAWithSession").Return(context.Background(), trace.SpanFromContext(context.Background())).AnyTimes()
			mockTracer.EXPECT().Start(gomock.Any(), "kratos.API.shouldEnforceWebAuthnWithSession").Return(context.Background(), trace.SpanFromContext(context.Background())).AnyTimes()
			mockService.EXPECT().MustReAuthenticate(gomock.Any(), loginChallenge, session, FlowStateCookie{}).Return(false, nil)
			mockCookieManager.EXPECT().GetStateCookie(gomock.Any()).Return(FlowStateCookie{}, nil)

			if tt.acceptErr != nil {
				mockService.EXPECT().AcceptLoginRequest(gomock.Any(), session, loginChallenge).Return(nil, nil, tt.acceptErr)
				mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).AnyTimes()
			} else {
				mockService.EXPECT().AcceptLoginRequest(gomock.Any(), session, loginChallenge).Return(&redirectTo, req.Cookies(), nil)
				mockCookieManager.EXPECT().ClearStateCookie(gomock.Any()).Return()
			}

			w := httptest.NewRecorder()
			mux := chi.NewMux()
			NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

			mux.ServeHTTP(w, req)

			res := w.Result()

			if res.StatusCode != tt.expectStatus {
				t.Fatalf("Expected HTTP status code %d, got: %v", tt.expectStatus, res.Status)
			}

			if tt.expectSuccess {
				data, err := io.ReadAll(res.Body)
				if err != nil {
					t.Fatalf("Expected error to be nil got %v", err)
				}
				redirectResp := hClient.NewOAuth2RedirectToWithDefaults()
				if err := json.Unmarshal(data, redirectResp); err != nil {
					t.Fatalf("Expected error to be nil got %v", err)
				}
				if redirectResp.RedirectTo != redirect {
					t.Fatalf("Expected redirect to %s, got: %s", redirect, res.Header["Location"][0])
				}
			}
		})
	}
}

func TestHandleGetLoginFlow(t *testing.T) {
	tests := []struct {
		name         string
		serviceError error
		expectStatus int
	}{
		{
			name:         "Success",
			serviceError: nil,
			expectStatus: http.StatusOK,
		},
		{
			name:         "Fail",
			serviceError: fmt.Errorf("error"),
			expectStatus: http.StatusInternalServerError,
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

			id := "test"
			flow := kClient.NewLoginFlowWithDefaults()
			flow.SetId(id)
			flow.SetState("choose_method")

			req := httptest.NewRequest(http.MethodGet, HANDLE_GET_LOGIN_FLOW_URL, nil)
			values := req.URL.Query()
			values.Add("id", id)
			req.URL.RawQuery = values.Encode()

			if tt.serviceError != nil {
				mockService.EXPECT().GetLoginFlow(gomock.Any(), id, req.Cookies()).Return(nil, nil, tt.serviceError)
				mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).AnyTimes()
			} else {
				mockService.EXPECT().GetLoginFlow(gomock.Any(), id, req.Cookies()).Return(flow, req.Cookies(), nil)
			}

			w := httptest.NewRecorder()
			mux := chi.NewMux()
			NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

			mux.ServeHTTP(w, req)

			res := w.Result()

			if res.StatusCode != tt.expectStatus {
				t.Fatalf("Expected HTTP status code %d, got: %v", tt.expectStatus, res.Status)
			}

			if tt.serviceError == nil {
				data, err := io.ReadAll(res.Body)
				if err != nil {
					t.Fatalf("Expected error to be nil got %v", err)
				}
				flowResponse := kClient.NewLoginFlowWithDefaults()
				if err := json.Unmarshal(data, flowResponse); err != nil {
					t.Fatalf("Expected error to be nil got %v", err)
				}
				if flowResponse.Id != flow.Id {
					t.Fatalf("Expected id to be: %s, got: %s", flow.Id, flowResponse.Id)
				}
			}
		})
	}
}

func TestHandleUpdateIdentifierFirstFlow(t *testing.T) {
	tests := []struct {
		name          string
		parseError    error
		updateError   error
		expectStatus  int
		expectSuccess bool
	}{
		{
			name:          "Success",
			parseError:    nil,
			updateError:   nil,
			expectStatus:  http.StatusOK,
			expectSuccess: true,
		},
		{
			name:         "FailOnParseLoginFlowMethodBody",
			parseError:   fmt.Errorf("error"),
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:         "FailOnUpdateIdLoginFlow",
			updateError:  fmt.Errorf("error"),
			expectStatus: http.StatusInternalServerError,
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

			flowId := "test"
			redirectTo := "https://some/path/to/somewhere"
			redirectFlow := new(BrowserLocationChangeRequired)
			redirectFlow.RedirectTo = &redirectTo

			flowBody := new(kClient.UpdateLoginFlowWithIdentifierFirstMethod)
			flowBody.SetIdentifier("test@example.com")

			req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_IDENTIFIER_FIRST_LOGIN_FLOW_URL, nil)
			values := req.URL.Query()
			values.Add("flow", flowId)
			req.URL.RawQuery = values.Encode()

			if tt.parseError != nil {
				mockService.EXPECT().ParseIdentifierFirstLoginFlowMethodBody(gomock.Any()).Return(flowBody, nil, tt.parseError)
				mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).AnyTimes()
			} else {
				mockService.EXPECT().ParseIdentifierFirstLoginFlowMethodBody(gomock.Any()).Return(flowBody, req.Cookies(), nil)
				if tt.updateError != nil {
					mockService.EXPECT().UpdateIdentifierFirstLoginFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(nil, nil, tt.updateError)
					mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).AnyTimes()
				} else {
					mockService.EXPECT().UpdateIdentifierFirstLoginFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(redirectFlow, req.Cookies(), nil)
				}
			}

			w := httptest.NewRecorder()
			mux := chi.NewMux()
			NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)
			mux.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.expectStatus {
				t.Fatalf("Expected HTTP status code %d, got: %v", tt.expectStatus, res.Status)
			}

			if tt.expectSuccess {
				data, err := io.ReadAll(res.Body)
				if err != nil {
					t.Fatalf("Expected error to be nil got %v", err)
				}
				flowResponse := new(BrowserLocationChangeRequired)
				if err := json.Unmarshal(data, flowResponse); err != nil {
					t.Fatalf("Expected error to be nil got %v", err)
				}
				if *flowResponse.RedirectTo != redirectTo {
					t.Fatalf("Expected redirectTo to be %v not %v", redirectTo, flowResponse.RedirectTo)
				}
			}
		})
	}
}

func TestHandleUpdateFlow(t *testing.T) {
	tests := []struct {
		name            string
		parseError      error
		checkAllowedErr error
		providerAllowed bool
		updateError     error
		expectStatus    int
		expectSuccess   bool
	}{
		{
			name:            "Success",
			providerAllowed: true,
			expectStatus:    http.StatusOK,
			expectSuccess:   true,
		},
		{
			name:            "WhenProviderNotAllowed",
			providerAllowed: false,
			expectStatus:    http.StatusForbidden,
		},
		{
			name:         "FailOnParseLoginFlowMethodBody",
			parseError:   fmt.Errorf("error"),
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:            "FailOnUpdateOIDCLoginFlow",
			providerAllowed: true,
			updateError:     fmt.Errorf("error"),
			expectStatus:    http.StatusInternalServerError,
		},
		{
			name:            "FailOnCheckAllowedProvider",
			providerAllowed: false,
			checkAllowedErr: fmt.Errorf("error"),
			expectStatus:    http.StatusInternalServerError,
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

			flowId := "test"
			flow := kClient.NewLoginFlowWithDefaults()
			flow.Id = flowId
			flow.ExpiresAt = time.Now().UTC()
			redirectTo := "https://some/path/to/somewhere"
			redirectFlow := new(BrowserLocationChangeRequired)
			redirectFlow.RedirectTo = &redirectTo

			flowBody := new(kClient.UpdateLoginFlowBody)
			flowBody.UpdateLoginFlowWithOidcMethod = kClient.NewUpdateLoginFlowWithOidcMethod("oidc", "oidc")

			req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_LOGIN_FLOW_URL, nil)
			values := req.URL.Query()
			values.Add("flow", flowId)
			req.URL.RawQuery = values.Encode()

			if tt.parseError != nil {
				mockService.EXPECT().ParseLoginFlowMethodBody(gomock.Any()).Return(flowBody, nil, tt.parseError)
				mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).AnyTimes()
			} else {
				mockService.EXPECT().ParseLoginFlowMethodBody(gomock.Any()).Return(flowBody, req.Cookies(), nil)
				mockService.EXPECT().GetLoginFlow(gomock.Any(), flowId, req.Cookies()).Return(flow, nil, nil)
				mockService.EXPECT().CheckAllowedProvider(gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.providerAllowed, tt.checkAllowedErr)

				if tt.checkAllowedErr != nil {
					mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).AnyTimes()
				} else if !tt.providerAllowed {
					// forbidden case
				} else if tt.updateError != nil {
					mockService.EXPECT().UpdateLoginFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(nil, nil, nil, tt.updateError)
					mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).AnyTimes()
				} else {
					mockTracer.EXPECT().Start(gomock.Any(), "kratos.API.shouldEnforceMFA").Return(context.Background(), trace.SpanFromContext(context.Background())).AnyTimes()
					mockTracer.EXPECT().Start(gomock.Any(), "kratos.API.shouldRegenerateBackupCodes").Return(context.Background(), trace.SpanFromContext(context.Background())).AnyTimes()
					mockCookieManager.EXPECT().SetStateCookie(gomock.Any(), gomock.Any()).Return(nil)
					mockService.EXPECT().UpdateLoginFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(redirectFlow, nil, req.Cookies(), nil)
				}
			}

			w := httptest.NewRecorder()
			mux := chi.NewMux()
			NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)
			mux.ServeHTTP(w, req)

			res := w.Result()

			if res.StatusCode != tt.expectStatus {
				t.Fatalf("Expected HTTP status code %d, got: %v", tt.expectStatus, res.Status)
			}

			if tt.expectSuccess {
				data, err := io.ReadAll(res.Body)
				if err != nil {
					t.Fatalf("Expected error to be nil got %v", err)
				}
				flowResponse := new(BrowserLocationChangeRequired)
				if err := json.Unmarshal(data, flowResponse); err != nil {
					t.Fatalf("Expected error to be nil got %v", err)
				}
			}
		})
	}
}

func TestHandleUpdateLoginFlowRedirectToRegenerateBackupCodes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	session := kClient.NewSession("test")

	lookupMethod := kClient.NewSessionAuthenticationMethodWithDefaults()
	lookupMethod.SetMethod("lookup_secret")

	pwdMethod := kClient.NewSessionAuthenticationMethodWithDefaults()
	pwdMethod.SetMethod("password")

	session.SetAuthenticatorAssuranceLevel("aal2")
	session.AuthenticationMethods = []kClient.SessionAuthenticationMethod{*pwdMethod, *lookupMethod}

	flowId := "test"
	redirectTo := "https://some/path/to/somewhere"
	redirectFlow := new(BrowserLocationChangeRequired)
	redirectFlow.RedirectTo = &redirectTo

	flow := kClient.NewLoginFlowWithDefaults()
	flow.Id = flowId
	returnTo := "https://some/return/url"
	flow.ReturnTo = &returnTo

	flowBody := new(kClient.UpdateLoginFlowBody)
	flowBody.UpdateLoginFlowWithLookupSecretMethod = kClient.NewUpdateLoginFlowWithLookupSecretMethod("xt879l1a", "lookup_secret")

	req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_LOGIN_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("flow", flowId)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseLoginFlowMethodBody(gomock.Any()).Return(flowBody, req.Cookies(), nil)
	mockService.EXPECT().GetLoginFlow(gomock.Any(), flowId, req.Cookies()).Return(flow, nil, nil)
	mockService.EXPECT().CheckAllowedProvider(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
	mockService.EXPECT().UpdateLoginFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(redirectFlow, nil, req.Cookies(), nil)

	mockTracer.EXPECT().Start(gomock.Any(), "kratos.API.shouldEnforceMFA").Return(context.Background(), trace.SpanFromContext(context.Background())).AnyTimes()
	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil)
	mockTracer.EXPECT().Start(gomock.Any(), "kratos.API.shouldEnforceMFAWithSession").Return(context.Background(), trace.SpanFromContext(context.Background())).AnyTimes()
	mockTracer.EXPECT().Start(gomock.Any(), "kratos.Service.HasTOTPAvailable").Return(context.Background(), trace.SpanFromContext(context.Background())).AnyTimes()
	mockService.EXPECT().HasTOTPAvailable(gomock.Any(), gomock.Any()).Return(true, nil)

	mockTracer.EXPECT().Start(gomock.Any(), "kratos.API.shouldRegenerateBackupCodes").Return(context.Background(), trace.SpanFromContext(context.Background())).AnyTimes()
	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil)
	mockService.EXPECT().HasNotEnoughLookupSecretsLeft(gomock.Any(), session.Identity.GetId()).Return(true, nil)
	mockCookieManager.EXPECT().SetStateCookie(gomock.Any(), gomock.Any()).Return(nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, true, true, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if _, err := json.Marshal(flow); err != nil {
		t.Fatalf("Expected error to be nil got %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatal("Expected HTTP status code 200, got: ", res.Status)
	}
}

func TestHandleCreateRecoveryFlow(t *testing.T) {
	tests := []struct {
		name         string
		withSession  bool
		serviceError error
		expectStatus int
		expectDelete bool
	}{
		{
			name:         "WithoutSession",
			expectStatus: http.StatusOK,
		},
		{
			name:         "WithSession",
			withSession:  true,
			expectStatus: http.StatusOK,
			expectDelete: true,
		},
		{
			name:         "FailOnCreateBrowserRecoveryFlow",
			serviceError: fmt.Errorf("error"),
			expectStatus: http.StatusInternalServerError,
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

			redirect := "https://example.com/ui/reset_email"

			req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_RECOVERY_FLOW_URL, nil)
			values := req.URL.Query()
			req.URL.RawQuery = values.Encode()

			if tt.withSession {
				sessionCookie := &http.Cookie{
					Name:     KRATOS_SESSION_COOKIE_NAME,
					Value:    "some_value",
					Path:     "/",
					HttpOnly: true,
					Secure:   true,
				}
				req.AddCookie(sessionCookie)
			}

			flow := kClient.NewRecoveryFlowWithDefaults()
			if tt.serviceError != nil {
				mockService.EXPECT().CreateBrowserRecoveryFlow(gomock.Any(), redirect, req.Cookies()).Return(nil, nil, tt.serviceError)
				mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).AnyTimes()
			} else {
				mockService.EXPECT().CreateBrowserRecoveryFlow(gomock.Any(), redirect, req.Cookies()).Return(flow, req.Cookies(), nil)
			}

			w := httptest.NewRecorder()
			mux := chi.NewMux()
			NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

			mux.ServeHTTP(w, req)

			res := w.Result()

			if res.StatusCode != tt.expectStatus {
				t.Fatalf("Expected HTTP status code %d, got: %v", tt.expectStatus, res.Status)
			}
			if tt.serviceError == nil {
				if _, err := json.Marshal(flow); err != nil {
					t.Fatalf("Expected error to be nil got %v", err)
				}
			}
			if tt.expectDelete {
				deleted := false
				for _, c := range res.Cookies() {
					if c.Name == KRATOS_SESSION_COOKIE_NAME {
						if c.Expires.Equal(time.Unix(0, 0)) {
							deleted = true
						} else {
							t.Fatal("Kratos session cookie was set")
						}
					}
				}
				if !deleted {
					t.Fatal("Kratos session cookie was not deleted")
				}
			}
		})
	}
}

func TestHandleGetRecoveryFlow(t *testing.T) {
	tests := []struct {
		name         string
		serviceError error
		expectStatus int
	}{
		{
			name:         "Success",
			serviceError: nil,
			expectStatus: http.StatusOK,
		},
		{
			name:         "Fail",
			serviceError: fmt.Errorf("error"),
			expectStatus: http.StatusInternalServerError,
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

			id := "test"
			flow := kClient.NewRecoveryFlowWithDefaults()
			flow.SetId(id)
			flow.SetState("choose_method")

			req := httptest.NewRequest(http.MethodGet, HANDLE_GET_RECOVERY_FLOW_URL, nil)
			values := req.URL.Query()
			values.Add("id", id)
			req.URL.RawQuery = values.Encode()

			if tt.serviceError != nil {
				mockService.EXPECT().GetRecoveryFlow(gomock.Any(), id, req.Cookies()).Return(nil, nil, tt.serviceError)
				mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).AnyTimes()
			} else {
				mockService.EXPECT().GetRecoveryFlow(gomock.Any(), id, req.Cookies()).Return(flow, req.Cookies(), nil)
			}

			w := httptest.NewRecorder()
			mux := chi.NewMux()
			NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

			mux.ServeHTTP(w, req)

			res := w.Result()

			if res.StatusCode != tt.expectStatus {
				t.Fatalf("Expected HTTP status code %d, got: %v", tt.expectStatus, res.Status)
			}

			if tt.serviceError == nil {
				data, err := io.ReadAll(res.Body)
				if err != nil {
					t.Fatalf("Expected error to be nil got %v", err)
				}
				flowResponse := kClient.NewRecoveryFlowWithDefaults()
				if err := json.Unmarshal(data, flowResponse); err != nil {
					t.Fatalf("Expected error to be nil got %v", err)
				}
				if flowResponse.Id != flow.Id {
					t.Fatalf("Expected id to be: %s, got: %s", flow.Id, flowResponse.Id)
				}
			}
		})
	}
}

func TestHandleUpdateRecoveryFlow(t *testing.T) {
	tests := []struct {
		name         string
		parseError   error
		expectStatus int
	}{
		{
			name:         "Success",
			parseError:   nil,
			expectStatus: http.StatusOK,
		},
		{
			name:         "FailOnParseRecoveryFlowMethodBody",
			parseError:   fmt.Errorf("error"),
			expectStatus: http.StatusInternalServerError,
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

			flowId := "test"
			flow := kClient.NewRecoveryFlowWithDefaults()
			flow.Id = flowId
			flow.ExpiresAt = time.Now().UTC()

			redirectTo := "https://example.com/ui/reset_email"
			redirectFlow := new(BrowserLocationChangeRequired)
			redirectFlow.RedirectTo = &redirectTo

			flowBody := new(kClient.UpdateRecoveryFlowBody)
			flowBody.UpdateRecoveryFlowWithCodeMethod = kClient.NewUpdateRecoveryFlowWithCodeMethod("code")

			req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_RECOVERY_FLOW_URL, nil)
			values := req.URL.Query()
			values.Add("flow", flowId)
			req.URL.RawQuery = values.Encode()

			if tt.parseError != nil {
				mockService.EXPECT().ParseRecoveryFlowMethodBody(gomock.Any()).Return(flowBody, tt.parseError)
				mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).AnyTimes()
			} else {
				mockService.EXPECT().ParseRecoveryFlowMethodBody(gomock.Any()).Return(flowBody, nil)
				mockService.EXPECT().UpdateRecoveryFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(redirectFlow, req.Cookies(), nil)
			}

			w := httptest.NewRecorder()
			mux := chi.NewMux()
			NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

			mux.ServeHTTP(w, req)

			res := w.Result()

			if res.StatusCode != tt.expectStatus {
				t.Fatalf("Expected HTTP status code %d, got: %v", tt.expectStatus, res.Status)
			}

			if tt.parseError == nil {
				data, err := io.ReadAll(res.Body)
				if err != nil {
					t.Fatalf("Expected error to be nil got %v", err)
				}
				flowResponse := new(BrowserLocationChangeRequired)
				if err := json.Unmarshal(data, flowResponse); err != nil {
					t.Fatalf("Expected error to be nil got %v", err)
				}
			}
		})
	}
}

func TestHandleCreateSettingsFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	redirect := "https://example.com/ui/setup_complete"

	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_SETTINGS_FLOW_URL, nil)
	values := req.URL.Query()
	values.Set("return_to", redirect)
	req.URL.RawQuery = values.Encode()

	flow := kClient.NewSettingsFlowWithDefaults()
	mockService.EXPECT().CreateBrowserSettingsFlow(gomock.Any(), redirect, req.Cookies()).Return(flow, nil, nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if _, err := json.Marshal(flow); err != nil {
		t.Fatalf("Expected error to be nil got %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatal("Expected HTTP status code 200, got: ", res.Status)
	}
}

func TestHandleCreateSettingsFlowWithRedirect(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	redirect := "https://example.com/ui/setup_complete"

	redirectErrorBrowserTo := "https://some/path/to/somewhere"
	redirectFlow := new(BrowserLocationChangeRequired)
	redirectFlow.RedirectTo = &redirectErrorBrowserTo
	redirectFlow.Error = kClient.NewGenericErrorWithDefaults()
	redirectFlow.Error.Code = new(int64)
	*redirectFlow.Error.Code = 403

	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_SETTINGS_FLOW_URL, nil)
	values := req.URL.Query()
	values.Set("return_to", redirect)
	req.URL.RawQuery = values.Encode()

	flow := kClient.NewSettingsFlowWithDefaults()
	mockService.EXPECT().CreateBrowserSettingsFlow(gomock.Any(), redirect, req.Cookies()).Return(flow, redirectFlow, nil)
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).AnyTimes()

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if _, err := json.Marshal(flow); err != nil {
		t.Fatalf("Expected error to be nil got %v", err)
	}
	if res.StatusCode != http.StatusForbidden {
		t.Fatal("Expected HTTP status code 403, got: ", res.Status)
	}
}

func TestHandleCreateSettingsFlowFailOnCreateBrowserSettingsFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	redirect := "https://example.com/ui/setup_complete"

	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_SETTINGS_FLOW_URL, nil)
	values := req.URL.Query()
	values.Set("return_to", redirect)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().CreateBrowserSettingsFlow(gomock.Any(), redirect, req.Cookies()).Return(nil, nil, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).AnyTimes()

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatal("Expected HTTP status code 500, got: ", res.Status)
	}
}

func TestHandleGetSettingsFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	id := "test"
	flow := kClient.NewSettingsFlowWithDefaults()
	flow.SetId(id)
	flow.SetState("show_form")
	flow.Identity.SetTraits(map[string]string{"name": "name"})

	req := httptest.NewRequest(http.MethodGet, HANDLE_GET_SETTINGS_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("id", id)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().GetSettingsFlow(gomock.Any(), id, req.Cookies()).Return(flow, nil, nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusOK {
		t.Fatal("Expected HTTP status code 200, got: ", res.Status)
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("Expected error to be nil got %v", err)
	}
	flowResponse := kClient.NewSettingsFlowWithDefaults()
	if err := json.Unmarshal(data, flowResponse); err != nil {
		t.Fatalf("Expected error to be nil got %v", err)
	}
	if flowResponse.Id != flow.Id {
		t.Fatalf("Expected id to be: %s, got: %s", flow.Id, flowResponse.Id)
	}
}

func TestHandleGetSettingsFlowWithRedirect(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	id := "test"
	flow := kClient.NewSettingsFlowWithDefaults()
	flow.SetId(id)
	flow.SetState("show_form")

	req := httptest.NewRequest(http.MethodGet, HANDLE_GET_SETTINGS_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("id", id)
	req.URL.RawQuery = values.Encode()

	redirectErrorBrowserTo := "https://some/path/to/somewhere"
	redirectFlow := new(BrowserLocationChangeRequired)
	redirectFlow.RedirectTo = &redirectErrorBrowserTo
	redirectFlow.Error = kClient.NewGenericErrorWithDefaults()
	redirectFlow.Error.Code = new(int64)
	*redirectFlow.Error.Code = 403

	mockService.EXPECT().GetSettingsFlow(gomock.Any(), id, req.Cookies()).Return(flow, redirectFlow, nil)
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).AnyTimes()

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusForbidden {
		t.Fatal("Expected HTTP status code 403, got: ", res.Status)
	}
	_, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("Expected error to be nil got %v", err)
	}
}

func TestHandleGetSettingsFlowFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	id := "test"
	flow := kClient.NewSettingsFlowWithDefaults()
	flow.SetId(id)

	req := httptest.NewRequest(http.MethodGet, HANDLE_GET_SETTINGS_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("id", id)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().GetSettingsFlow(gomock.Any(), id, req.Cookies()).Return(nil, nil, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).AnyTimes()

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatal("Expected HTTP status code 500, got: ", res.Status)
	}
}

func TestHandleUpdateSettingsFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	flowId := "test"
	flow := kClient.NewSettingsFlowWithDefaults()
	flow.Id = flowId
	flow.State = "show_form"
	flow.Identity.SetTraits(map[string]string{"name": "name"})

	flowBody := new(kClient.UpdateSettingsFlowBody)
	flowBody.UpdateSettingsFlowWithPasswordMethod = kClient.NewUpdateSettingsFlowWithPasswordMethod("password", "password")

	req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_SETTINGS_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("flow", flowId)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseSettingsFlowMethodBody(gomock.Any()).Return(flowBody, nil)
	mockService.EXPECT().UpdateSettingsFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(flow, nil, req.Cookies(), nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusOK {
		t.Fatal("Expected HTTP status code 200, got: ", res.Status)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("Expected error to be nil got %v", err)
	}
	flowResponse := kClient.NewSettingsFlowWithDefaults()
	if err := json.Unmarshal(data, flowResponse); err != nil {
		t.Fatalf("Expected error to be nil got %v", err)
	}
}

func TestHandleUpdateSettingsFlowPrivilegedSessionRequired(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	flowId := "test"
	returnTo := "https://example.com/settings"

	currentFlow := kClient.NewSettingsFlowWithDefaults()
	currentFlow.Id = flowId
	currentFlow.ReturnTo = &returnTo

	redirectBase := "http://kratos/self-service/login/browser?refresh=true"
	sessionRequiredErrorId := "session_refresh_required"

	redirectFlow := new(BrowserLocationChangeRequired)
	redirectFlow.RedirectTo = &redirectBase
	redirectFlow.Error = &kClient.GenericError{
		Id: &sessionRequiredErrorId,
	}

	flowBody := new(kClient.UpdateSettingsFlowBody)
	flowBody.UpdateSettingsFlowWithPasswordMethod = kClient.NewUpdateSettingsFlowWithPasswordMethod("password", "password")

	req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_SETTINGS_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("flow", flowId)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseSettingsFlowMethodBody(gomock.Any()).Return(flowBody, nil)
	mockService.EXPECT().UpdateSettingsFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(nil, redirectFlow, req.Cookies(), nil)
	mockService.EXPECT().GetSettingsFlow(gomock.Any(), flowId, req.Cookies()).Return(currentFlow, nil, nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected HTTP status code 200, got: %d", res.StatusCode)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("Expected error to be nil, got %v", err)
	}

	flowResponse := new(BrowserLocationChangeRequired)
	if err := json.Unmarshal(data, flowResponse); err != nil {
		t.Fatalf("Expected error to be nil, got %v", err)
	}

	expectedRedirect := fmt.Sprintf("%s&return_to=%s", redirectBase, url.QueryEscape(returnTo))
	if flowResponse.RedirectTo == nil || *flowResponse.RedirectTo != expectedRedirect {
		t.Fatalf("Expected redirect_to to be %s, got %v", expectedRedirect, *flowResponse.RedirectTo)
	}
}

func TestHandleUpdateSettingsFlowWithRedirect(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	flowId := "test"
	flow := kClient.NewSettingsFlowWithDefaults()
	flow.Id = flowId
	flow.ExpiresAt = time.Now().UTC()

	redirectTo := "https://example.com/sign_in"
	redirectFlow := new(BrowserLocationChangeRequired)
	redirectFlow.RedirectTo = &redirectTo

	flowBody := new(kClient.UpdateSettingsFlowBody)
	flowBody.UpdateSettingsFlowWithOidcMethod = kClient.NewUpdateSettingsFlowWithOidcMethod("oidc")

	req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_SETTINGS_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("flow", flowId)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseSettingsFlowMethodBody(gomock.Any()).Return(flowBody, nil)
	mockService.EXPECT().UpdateSettingsFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(nil, redirectFlow, req.Cookies(), nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatal("Expected HTTP status code 200, got: ", res.Status)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("Expected error to be nil got %v", err)
	}
	flowResponse := new(BrowserLocationChangeRequired)
	if err := json.Unmarshal(data, flowResponse); err != nil {
		t.Fatalf("Expected error to be nil got %v", err)
	}
	if flowResponse.RedirectTo == nil || *flowResponse.RedirectTo != redirectTo {
		t.Fatalf("Expected redirect_to to be %v got %v", redirectTo, flowResponse.RedirectTo)
	}
}

func TestHandleUpdateWebAuthnSettingsFlowWithReturnTo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	returnTo := "https://example.com/ui/login?login_challenge=test"
	flowId := "test"
	flow := kClient.NewSettingsFlowWithDefaults()
	flow.Id = flowId
	flow.State = "success"
	flow.Identity.SetTraits(map[string]string{"name": "name"})
	flow.ReturnTo = &returnTo

	flowBody := new(kClient.UpdateSettingsFlowBody)
	flowBody.UpdateSettingsFlowWithWebAuthnMethod = kClient.NewUpdateSettingsFlowWithWebAuthnMethod("webauthn")

	req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_SETTINGS_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("flow", flowId)
	req.URL.RawQuery = values.Encode()
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	mockService.EXPECT().ParseSettingsFlowMethodBody(gomock.Any()).Return(flowBody, nil)
	mockService.EXPECT().UpdateSettingsFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(flow, nil, req.Cookies(), nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusSeeOther {
		t.Fatal("Expected HTTP status code 303, got: ", res.Status)
	}

	if res.Header.Get("Location") != returnTo {
		t.Fatalf("Invalid location, expected: %s, got: %s", returnTo, res.Header.Get("Location"))
	}
}

func TestHandleUpdateWebAuthnSettingsFlowWithoutReturnTo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	returnTo := "https://example.com/setup_passkey"
	flowId := "test"
	flow := kClient.NewSettingsFlowWithDefaults()
	flow.Id = flowId
	flow.State = "success"
	flow.Identity.SetTraits(map[string]string{"name": "name"})

	continueRedirect := &kClient.ContinueWithRedirectBrowserTo{
		Action:            "redirect_browser_to",
		RedirectBrowserTo: returnTo,
	}
	flow.ContinueWith = []kClient.ContinueWith{
		{
			ContinueWithRedirectBrowserTo: continueRedirect,
		},
	}

	flowBody := new(kClient.UpdateSettingsFlowBody)
	flowBody.UpdateSettingsFlowWithWebAuthnMethod = kClient.NewUpdateSettingsFlowWithWebAuthnMethod("webauthn")

	req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_SETTINGS_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("flow", flowId)
	req.URL.RawQuery = values.Encode()
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	mockService.EXPECT().ParseSettingsFlowMethodBody(gomock.Any()).Return(flowBody, nil)
	mockService.EXPECT().UpdateSettingsFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(flow, nil, req.Cookies(), nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusSeeOther {
		t.Fatal("Expected HTTP status code 303, got: ", res.Status)
	}

	if res.Header.Get("Location") != returnTo {
		t.Fatalf("Invalid location, expected: %s, got: %s", returnTo, res.Header.Get("Location"))
	}
}

func TestHandleUpdateSettingsFlowFailOnParseSettingsFlowMethodBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	flowId := "test"

	flowBody := new(kClient.UpdateSettingsFlowBody)
	flowBody.UpdateSettingsFlowWithPasswordMethod = kClient.NewUpdateSettingsFlowWithPasswordMethod("password", "password")

	req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_SETTINGS_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("flow", flowId)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseSettingsFlowMethodBody(gomock.Any()).Return(flowBody, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).AnyTimes()

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatal("Expected HTTP status code 500, got: ", res.Status)
	}
}

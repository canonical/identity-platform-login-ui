package kratos

import (
	"context"
	"encoding/json"
    "errors"
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

func TestHandleCreateFlowWithoutSessionAcceptJSON(t *testing.T) {
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
	req.Header.Set("Accept", "application/json, text/plain, */*")
	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(nil, nil, nil)
	mockService.EXPECT().MustReAuthenticate(gomock.Any(), loginChallenge, nil, FlowStateCookie{}).Return(true, nil)
	mockService.EXPECT().CreateBrowserLoginFlow(gomock.Any(), gomock.Any(), returnTo, loginChallenge, gomock.Any(), req.Cookies()).Return(flow, req.Cookies(), nil)
	mockService.EXPECT().FilterFlowProviderList(gomock.Any(), flow).Return(flow, nil)
	mockCookieManager.EXPECT().GetStateCookie(gomock.Any()).Return(FlowStateCookie{}, nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

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
	loginFlow := kClient.NewLoginFlowWithDefaults()
	if err := json.Unmarshal(data, loginFlow); err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}

	if loginFlow.Id != flow.Id {
		t.Fatalf("Invalid flow id, expected: %s, got: %s", flow.Id, loginFlow.Id)
	}
}

func TestHandleCreateFlowWithoutSessionNotAcceptJSON(t *testing.T) {
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
	req.Header.Set("Accept", "application/x-www-form-urlencoded")
	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(nil, nil, nil)
	mockService.EXPECT().MustReAuthenticate(gomock.Any(), loginChallenge, nil, FlowStateCookie{}).Return(true, nil)
	mockService.EXPECT().CreateBrowserLoginFlow(gomock.Any(), gomock.Any(), returnTo, loginChallenge, gomock.Any(), req.Cookies()).Return(flow, req.Cookies(), nil)
	mockService.EXPECT().FilterFlowProviderList(gomock.Any(), flow).Return(flow, nil)
	mockCookieManager.EXPECT().GetStateCookie(gomock.Any()).Return(FlowStateCookie{}, nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected HTTP status code 303 got %v", res.StatusCode)
	}

	location, _ := url.JoinPath(BASE_URL, "ui/login")
	location = fmt.Sprintf("%s?flow=%s", location, flow.Id)

	if res.Header.Get("Location") != location {
		t.Fatalf("Invalid location, expected: %s, got: %s", location, res.Header.Get("Location"))
	}
}

func TestHandleCreateFlowWithoutSessionFailOnCreateBrowserLoginFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	flow := kClient.NewLoginFlowWithDefaults()
	flow.Id = "test"

	loginChallenge := "login_challenge_2341235123231"
	returnTo, _ := url.JoinPath(BASE_URL, "ui/login")
	returnTo = returnTo + "?login_challenge=" + loginChallenge

	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("login_challenge", loginChallenge)
	req.URL.RawQuery = values.Encode()

	mockCookieManager.EXPECT().GetStateCookie(gomock.Any()).Return(FlowStateCookie{}, nil)
	mockLogger.EXPECT().Errorf("failed to create login flow, err: error")
	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(nil, nil, nil)
	mockService.EXPECT().MustReAuthenticate(gomock.Any(), loginChallenge, nil, FlowStateCookie{}).Return(true, nil)
	mockService.EXPECT().CreateBrowserLoginFlow(gomock.Any(), gomock.Any(), returnTo, loginChallenge, gomock.Any(), req.Cookies()).Return(nil, nil, fmt.Errorf("error"))

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected HTTP status code 500 got %v", res.StatusCode)
	}
}

func TestHandleCreateFlowWithoutSessionFailOnFilterProviders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	flow := kClient.NewLoginFlowWithDefaults()
	flow.Id = "test"

	loginChallenge := "login_challenge_2341235123231"
	returnTo, _ := url.JoinPath(BASE_URL, "ui/login")
	returnTo = returnTo + "?login_challenge=" + loginChallenge

	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("login_challenge", loginChallenge)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(nil, nil, nil)
	mockService.EXPECT().MustReAuthenticate(gomock.Any(), loginChallenge, nil, FlowStateCookie{}).Return(true, nil)
	mockService.EXPECT().CreateBrowserLoginFlow(gomock.Any(), gomock.Any(), returnTo, loginChallenge, gomock.Any(), req.Cookies()).Return(flow, req.Cookies(), nil)
	mockService.EXPECT().FilterFlowProviderList(gomock.Any(), flow).Return(nil, fmt.Errorf("oh no"))
	mockCookieManager.EXPECT().GetStateCookie(gomock.Any()).Return(FlowStateCookie{}, nil)
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).AnyTimes()

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected HTTP status code 500 got %v", res.StatusCode)
	}
}

func TestHandleCreateFlowWithoutSessionWhenNoProvidersAllowedAcceptJSON(t *testing.T) {
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
	req.Header.Set("Accept", "application/json, text/plain, */*")

	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(nil, nil, nil)
	mockService.EXPECT().MustReAuthenticate(gomock.Any(), loginChallenge, nil, FlowStateCookie{}).Return(true, nil)
	mockService.EXPECT().CreateBrowserLoginFlow(gomock.Any(), gomock.Any(), returnTo, loginChallenge, gomock.Any(), req.Cookies()).Return(flow, req.Cookies(), nil)
	mockService.EXPECT().FilterFlowProviderList(gomock.Any(), flow).Return(flow, nil)
	mockCookieManager.EXPECT().GetStateCookie(gomock.Any()).Return(FlowStateCookie{}, nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

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
	loginFlow := kClient.NewLoginFlowWithDefaults()
	if err := json.Unmarshal(data, loginFlow); err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}

	if loginFlow.Id != flow.Id {
		t.Fatalf("Invalid flow id, expected: %s, got: %s", flow.Id, loginFlow.Id)
	}
}

func TestHandleCreateFlowWithoutSessionWhenNoProvidersAllowedNotAcceptJSON(t *testing.T) {
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
	req.Header.Set("Accept", "application/x-www-form-urlencoded")

	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(nil, nil, nil)
	mockService.EXPECT().MustReAuthenticate(gomock.Any(), loginChallenge, nil, FlowStateCookie{}).Return(true, nil)
	mockService.EXPECT().CreateBrowserLoginFlow(gomock.Any(), gomock.Any(), returnTo, loginChallenge, gomock.Any(), req.Cookies()).Return(flow, req.Cookies(), nil)
	mockService.EXPECT().FilterFlowProviderList(gomock.Any(), flow).Return(flow, nil)
	mockCookieManager.EXPECT().GetStateCookie(gomock.Any()).Return(FlowStateCookie{}, nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected HTTP status code 303 got %v", res.StatusCode)
	}

	location, _ := url.JoinPath(BASE_URL, "ui/login")
	location = fmt.Sprintf("%s?flow=%s", location, flow.Id)

	if res.Header.Get("Location") != location {
		t.Fatalf("Invalid location, expected: %s, got: %s", location, res.Header.Get("Location"))
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

func TestHandleCreateFlowWithSessionAcceptJSON(t *testing.T) {
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
	req.Header.Set("Accept", "application/json, text/plain, */*")

	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil)
	mockTracer.EXPECT().Start(gomock.Any(), "kratos.API.shouldEnforceMFAWithSession").Return(context.Background(), trace.SpanFromContext(context.Background())).AnyTimes()
	mockTracer.EXPECT().Start(gomock.Any(), "kratos.API.shouldEnforceWebAuthnWithSession").Return(context.Background(), trace.SpanFromContext(context.Background())).AnyTimes()
	mockService.EXPECT().MustReAuthenticate(gomock.Any(), loginChallenge, session, FlowStateCookie{}).Return(false, nil)
	mockService.EXPECT().AcceptLoginRequest(gomock.Any(), session, loginChallenge).Return(&redirectTo, req.Cookies(), nil)
	mockCookieManager.EXPECT().GetStateCookie(gomock.Any()).Return(FlowStateCookie{}, nil)
	mockCookieManager.EXPECT().ClearStateCookie(gomock.Any()).Return()

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("Expected error to be nil got %v", err)
	}
	redirectResp := hClient.NewOAuth2RedirectToWithDefaults()
	if err := json.Unmarshal(data, redirectResp); err != nil {
		t.Fatalf("Expected error to be nil got %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatal("Expected HTTP status code 200, got: ", res.Status)
	}
	if redirectResp.RedirectTo != redirect {
		t.Fatalf("Expected redirect to %s, got: %s", redirect, res.Header["Location"][0])
	}
}

func TestHandleCreateFlowWithSessionNotAcceptJSON(t *testing.T) {
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
	req.Header.Set("Accept", "application/x-www-form-urlencoded")

	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil)
	mockTracer.EXPECT().Start(gomock.Any(), "kratos.API.shouldEnforceMFAWithSession").Return(context.Background(), trace.SpanFromContext(context.Background())).AnyTimes()
	mockTracer.EXPECT().Start(gomock.Any(), "kratos.API.shouldEnforceWebAuthnWithSession").Return(context.Background(), trace.SpanFromContext(context.Background())).AnyTimes()
	mockService.EXPECT().MustReAuthenticate(gomock.Any(), loginChallenge, session, FlowStateCookie{}).Return(false, nil)
	mockService.EXPECT().AcceptLoginRequest(gomock.Any(), session, loginChallenge).Return(&redirectTo, req.Cookies(), nil)
	mockCookieManager.EXPECT().GetStateCookie(gomock.Any()).Return(FlowStateCookie{}, nil)
	mockCookieManager.EXPECT().ClearStateCookie(gomock.Any()).Return()

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("Expected error to be nil got %v", err)
	}
	redirectResp := hClient.NewOAuth2RedirectToWithDefaults()
	if err := json.Unmarshal(data, redirectResp); err != nil {
		t.Fatalf("Expected error to be nil got %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatal("Expected HTTP status code 200, got: ", res.Status)
	}
	if redirectResp.RedirectTo != redirect {
		t.Fatalf("Expected redirect to %s, got: %s", redirect, res.Header["Location"][0])
	}
}

func TestHandleCreateFlowWithSessionFailOnAcceptLoginRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	session := kClient.NewSession("test")

	loginChallenge := "login_challenge_2341235123231"

	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("login_challenge", loginChallenge)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil)
	mockTracer.EXPECT().Start(gomock.Any(), "kratos.API.shouldEnforceMFAWithSession").Return(context.Background(), trace.SpanFromContext(context.Background())).AnyTimes()
	mockTracer.EXPECT().Start(gomock.Any(), "kratos.API.shouldEnforceWebAuthnWithSession").Return(context.Background(), trace.SpanFromContext(context.Background())).AnyTimes()
	mockService.EXPECT().MustReAuthenticate(gomock.Any(), loginChallenge, session, FlowStateCookie{}).Return(false, nil)
	mockService.EXPECT().AcceptLoginRequest(gomock.Any(), session, loginChallenge).Return(nil, nil, fmt.Errorf("error"))
	mockCookieManager.EXPECT().GetStateCookie(gomock.Any()).Return(FlowStateCookie{}, nil)
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

func TestHandleGetLoginFlow(t *testing.T) {
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

	mockService.EXPECT().GetLoginFlow(gomock.Any(), id, req.Cookies()).Return(flow, req.Cookies(), nil)

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
	flowResponse := kClient.NewLoginFlowWithDefaults()
	if err := json.Unmarshal(data, flowResponse); err != nil {
		t.Fatalf("Expected error to be nil got %v", err)
	}
	if flowResponse.Id != flow.Id {
		t.Fatalf("Expected id to be: %s, got: %s", flow.Id, flowResponse.Id)
	}
}

func TestHandleGetLoginFlowFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	id := "test"
	flow := kClient.NewLoginFlowWithDefaults()
	flow.SetId(id)

	req := httptest.NewRequest(http.MethodGet, HANDLE_GET_LOGIN_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("id", id)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().GetLoginFlow(gomock.Any(), id, req.Cookies()).Return(nil, nil, fmt.Errorf("error"))
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

func TestHandleCreateRegistrationFlow(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockLogger := NewMockLoggerInterface(ctrl)
    mockService := NewMockServiceInterface(ctrl)
    mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
    mockTracer := NewMockTracingInterface(ctrl)

    api := NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger)

    t.Run("service.CreateBrowserRegistrationFlow returns error", func(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/registration/create?return_to=/error", nil)
        w := httptest.NewRecorder()

        mockService.EXPECT().CreateBrowserRegistrationFlow(gomock.Any(), "/error").
            Return(nil, nil, errors.New("create failed"))
        mockLogger.EXPECT().Errorf("Failed to create registration flow: %v", gomock.Any())

        api.handleCreateRegistrationFlow(w, req)

        res := w.Result()
        defer res.Body.Close()

        if res.StatusCode != http.StatusInternalServerError {
            t.Fatalf("expected %d, got %d", http.StatusInternalServerError, res.StatusCode)
        }

        data, _ := io.ReadAll(res.Body)
        if !strings.Contains(string(data), "Failed to create registration flow") {
            t.Fatalf("expected failure message, got %q", string(data))
        }
    })

    t.Run("success - custom return_to", func(t *testing.T) {
        flowID := "flow-abc-123"
        req := httptest.NewRequest(http.MethodGet, "/registration/create?return_to=/welcome", nil)
        w := httptest.NewRecorder()

        flow := kClient.NewRegistrationFlowWithDefaults()
        flow.SetId(flowID)
        cookies := []*http.Cookie{{Name: "session", Value: "xyz"}}

        mockService.EXPECT().CreateBrowserRegistrationFlow(gomock.Any(), "/welcome").
            Return(flow, cookies, nil)

        api.handleCreateRegistrationFlow(w, req)

        res := w.Result()
        defer res.Body.Close()

        if res.StatusCode != http.StatusOK {
            t.Fatalf("expected %d, got %d", http.StatusOK, res.StatusCode)
        }

        foundCookie := false
        for _, c := range res.Cookies() {
            if c.Name == "session" && c.Value == "xyz" {
                foundCookie = true
            }
        }
        if !foundCookie {
            t.Fatalf("expected cookie 'session=xyz' to be set")
        }

        var body map[string]interface{}
        if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
            t.Fatalf("unexpected decode error: %v", err)
        }
        if body["id"] != flowID {
            t.Fatalf("expected id %s, got %v", flowID, body["id"])
        }
    })

    t.Run("success - no return_to", func(t *testing.T) {
        flowID := "flow-789"
        req := httptest.NewRequest(http.MethodGet, "/registration/create", nil)
        w := httptest.NewRecorder()

        flow := kClient.NewRegistrationFlowWithDefaults()
        flow.SetId(flowID)
        cookies := []*http.Cookie{{Name: "def", Value: "ok"}}

        mockService.EXPECT().CreateBrowserRegistrationFlow(gomock.Any(), "").
            Return(flow, cookies, nil)

        api.handleCreateRegistrationFlow(w, req)

        res := w.Result()
        defer res.Body.Close()

        if res.StatusCode != http.StatusOK {
            t.Fatalf("expected %d, got %d", http.StatusOK, res.StatusCode)
        }

        foundCookie := false
        for _, c := range res.Cookies() {
            if c.Name == "def" && c.Value == "ok" {
                foundCookie = true
            }
        }
        if !foundCookie {
            t.Fatalf("expected cookie 'def=ok' to be set")
        }

        var body map[string]interface{}
        if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
            t.Fatalf("unexpected decode error: %v", err)
        }
        if body["id"] != flowID {
            t.Fatalf("expected id %s, got %v", flowID, body["id"])
        }
    })
}


func TestHandleGetRegistrationFlow(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockLogger := NewMockLoggerInterface(ctrl)
    mockService := NewMockServiceInterface(ctrl)
    mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
    mockTracer := NewMockTracingInterface(ctrl)

    api := NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger)

    t.Run("Missing id parameter", func(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/registration", nil)
        w := httptest.NewRecorder()

        mockLogger.EXPECT().Errorf("ID parameter not present")

        api.handleGetRegistrationFlow(w, req)

        res := w.Result()
        defer res.Body.Close()

        if res.StatusCode != http.StatusBadRequest {
            t.Fatalf("expected %d, got %d", http.StatusBadRequest, res.StatusCode)
        }

        var body string
        if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
            t.Fatalf("unexpected decode error: %v", err)
        }
        if body != "ID parameter not present" {
            t.Fatalf("expected error message 'ID parameter not present', got %q", body)
        }
    })

    t.Run("GetRegistrationFlow returns error", func(t *testing.T) {
        id := "flow123"
        req := httptest.NewRequest(http.MethodGet, "/registration?id="+id, nil)
        w := httptest.NewRecorder()

        mockService.EXPECT().GetRegistrationFlow(gomock.Any(), id, req.Cookies()).
            Return(nil, nil, errors.New("some error"))
        mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any())

        api.handleGetRegistrationFlow(w, req)

        res := w.Result()
        defer res.Body.Close()

        if res.StatusCode != http.StatusInternalServerError {
            t.Fatalf("expected %d, got %d", http.StatusInternalServerError, res.StatusCode)
        }

        data, _ := io.ReadAll(res.Body)
        if !strings.Contains(string(data), "Failed to get registration flow") {
            t.Fatalf("expected failure message, got %q", string(data))
        }
    })

    t.Run("Success", func(t *testing.T) {
        id := "flow456"
        req := httptest.NewRequest(http.MethodGet, "/registration?id="+id, nil)
        w := httptest.NewRecorder()

        flow := kClient.NewRegistrationFlowWithDefaults()
        flow.SetId(id)
        cookies := []*http.Cookie{{Name: "test", Value: "ok"}}

        mockService.EXPECT().GetRegistrationFlow(gomock.Any(), id, req.Cookies()).
            Return(flow, cookies, nil)

        api.handleGetRegistrationFlow(w, req)

        res := w.Result()
        defer res.Body.Close()

        if res.StatusCode != http.StatusOK {
            t.Fatalf("expected %d, got %d", http.StatusOK, res.StatusCode)
        }

        foundCookie := false
        for _, c := range res.Cookies() {
            if c.Name == "test" && c.Value == "ok" {
                foundCookie = true
            }
        }
        if !foundCookie {
            t.Fatalf("expected cookie 'test=ok' to be set")
        }

        var body map[string]interface{}
        if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
            t.Fatalf("unexpected decode error: %v", err)
        }

        if body["id"] != id {
            t.Fatalf("expected id %s, got %v", id, body["id"])
        }
    })
}

func TestHandleUpdateRegistrationFlow(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockLogger := NewMockLoggerInterface(ctrl)
    mockService := NewMockServiceInterface(ctrl)
    mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
    mockTracer := NewMockTracingInterface(ctrl)

    api := NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger)

    t.Run("ParseRegistrationFlowMethodBody returns error", func(t *testing.T) {
        req := httptest.NewRequest(http.MethodPost, "/registration/update?flow=e2c802141dc51a06676974687562", nil)
        w := httptest.NewRecorder()

        mockService.EXPECT().ParseRegistrationFlowMethodBody(req).
            Return(nil, errors.New("parse error"))
        mockLogger.EXPECT().Errorf("Error when parsing request body: %v\n", gomock.Any())

        api.handleUpdateRegistrationFlow(w, req)

        res := w.Result()
        defer res.Body.Close()

        if res.StatusCode != http.StatusInternalServerError {
            t.Fatalf("expected %d, got %d", http.StatusInternalServerError, res.StatusCode)
        }

        data, _ := io.ReadAll(res.Body)
        if !strings.Contains(string(data), "Failed to parse registration flow") {
            t.Fatalf("expected parse failure message, got %q", string(data))
        }
    })

    t.Run("UpdateRegistrationFlow returns error", func(t *testing.T) {
        req := httptest.NewRequest(http.MethodPost, "/registration/update?flow=e2c802141dc51a06676974687562", nil)
        w := httptest.NewRecorder()

        body := &kClient.UpdateRegistrationFlowBody{}
        mockService.EXPECT().ParseRegistrationFlowMethodBody(req).
            Return(body, nil)
        mockService.EXPECT().UpdateRegistrationFlow(gomock.Any(), "e2c802141dc51a06676974687562", *body, req.Cookies()).
            Return(nil, nil, errors.New("update failed"))
        mockLogger.EXPECT().Errorf("Error when updating registration flow: %v\n", gomock.Any())

        api.handleUpdateRegistrationFlow(w, req)

        res := w.Result()
        defer res.Body.Close()

        if res.StatusCode != http.StatusInternalServerError {
            t.Fatalf("expected %d, got %d", http.StatusInternalServerError, res.StatusCode)
        }

        data, _ := io.ReadAll(res.Body)
        if !strings.Contains(string(data), "update failed") {
            t.Fatalf("expected update failure message, got %q", string(data))
        }
    })

    t.Run("success", func(t *testing.T) {
        flowID := "e2c802141dc51a06676974687562"
        req := httptest.NewRequest(http.MethodPost, "/registration/update?flow="+flowID, nil)
        w := httptest.NewRecorder()

        body := &kClient.UpdateRegistrationFlowBody{}
        mockService.EXPECT().ParseRegistrationFlowMethodBody(req).
            Return(body, nil)

        mockRegistration := &RegistrationFlowResponse{changeRequired: &BrowserLocationChangeRequired{}}

        cookies := []*http.Cookie{{Name: "updated", Value: "ok"}}

        mockService.EXPECT().UpdateRegistrationFlow(gomock.Any(), flowID, *body, req.Cookies()).
            Return(mockRegistration, cookies, nil)

        api.handleUpdateRegistrationFlow(w, req)

        res := w.Result()
        defer res.Body.Close()

        if res.StatusCode != http.StatusUnprocessableEntity {
            t.Fatalf("expected %d, got %d", http.StatusOK, res.StatusCode)
        }

        foundCookie := false
        for _, c := range res.Cookies() {
            if c.Name == "updated" && c.Value == "ok" {
                foundCookie = true
            }
        }
        if !foundCookie {
            t.Fatalf("expected cookie 'updated=ok' to be set")
        }
    })
}



func TestHandleUpdateIdentifierFirstFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	flowId := "test"
	flow := kClient.NewLoginFlowWithDefaults()
	flow.Id = flowId
	redirectTo := "https://some/path/to/somewhere"
	redirectFlow := new(BrowserLocationChangeRequired)
	redirectFlow.RedirectTo = &redirectTo

	flowBody := new(kClient.UpdateLoginFlowWithIdentifierFirstMethod)
	flowBody.SetIdentifier("test@example.com")

	req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_IDENTIFIER_FIRST_LOGIN_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("flow", flowId)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseIdentifierFirstLoginFlowMethodBody(gomock.Any()).Return(flowBody, req.Cookies(), nil)
	mockService.EXPECT().UpdateIdentifierFirstLoginFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(redirectFlow, req.Cookies(), nil)
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
	if *flowResponse.RedirectTo != redirectTo {
		t.Fatalf("Expected redirectTo to be %v not %v", redirectTo, flowResponse.RedirectTo)
	}
}

func TestHandleUpdateIdentifierFirstFlowFailOnParseLoginFlowMethodBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	flowId := "test"
	flowBody := new(kClient.UpdateLoginFlowWithIdentifierFirstMethod)
	flowBody.SetIdentifier("test@example.com")

	req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_IDENTIFIER_FIRST_LOGIN_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("flow", flowId)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseIdentifierFirstLoginFlowMethodBody(gomock.Any()).Return(flowBody, nil, fmt.Errorf("error"))
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

func TestHandleUpdateIdentifierFirstFlowFailOnUpdateIdLoginFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	flowId := "test"
	flow := kClient.NewLoginFlowWithDefaults()
	flow.Id = flowId

	flowBody := new(kClient.UpdateLoginFlowWithIdentifierFirstMethod)
	flowBody.SetIdentifier("test@example.com")

	req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_IDENTIFIER_FIRST_LOGIN_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("flow", flowId)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseIdentifierFirstLoginFlowMethodBody(gomock.Any()).Return(flowBody, req.Cookies(), nil)
	mockService.EXPECT().UpdateIdentifierFirstLoginFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(nil, nil, fmt.Errorf("error"))
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

func TestHandleUpdateFlow(t *testing.T) {
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

	mockTracer.EXPECT().Start(gomock.Any(), "kratos.API.shouldEnforceMFA").Return(context.Background(), trace.SpanFromContext(context.Background())).AnyTimes()
	mockTracer.EXPECT().Start(gomock.Any(), "kratos.API.shouldRegenerateBackupCodes").Return(context.Background(), trace.SpanFromContext(context.Background())).AnyTimes()
	mockCookieManager.EXPECT().SetStateCookie(gomock.Any(), gomock.Any()).Return(nil)
	mockService.EXPECT().ParseLoginFlowMethodBody(gomock.Any()).Return(flowBody, req.Cookies(), nil)
	mockService.EXPECT().UpdateLoginFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(redirectFlow, nil, req.Cookies(), nil)
	mockService.EXPECT().GetLoginFlow(gomock.Any(), flowId, req.Cookies()).Return(flow, nil, nil)
	mockService.EXPECT().CheckAllowedProvider(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)

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
	flowResponse := new(BrowserLocationChangeRequired)
	if err := json.Unmarshal(data, flowResponse); err != nil {
		t.Fatalf("Expected error to be nil got %v", err)
	}
}

func TestHandleUpdateFlowWhenProviderNotAllowed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	flowId := "test"
	flow := kClient.NewLoginFlowWithDefaults()
	flow.Id = flowId
	redirectTo := "https://some/path/to/somewhere"
	redirectFlow := new(BrowserLocationChangeRequired)
	redirectFlow.RedirectTo = &redirectTo

	flowBody := new(kClient.UpdateLoginFlowBody)
	flowBody.UpdateLoginFlowWithOidcMethod = kClient.NewUpdateLoginFlowWithOidcMethod("oidc", "oidc")

	req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_LOGIN_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("flow", flowId)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseLoginFlowMethodBody(gomock.Any()).Return(flowBody, req.Cookies(), nil)
	mockService.EXPECT().GetLoginFlow(gomock.Any(), flowId, req.Cookies()).Return(flow, nil, nil)
	mockService.EXPECT().CheckAllowedProvider(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusForbidden {
		t.Fatal("Expected HTTP status code 403, got: ", res.Status)
	}
}

func TestHandleUpdateFlowFailOnParseLoginFlowMethodBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	flowId := "test"

	flowBody := new(kClient.UpdateLoginFlowBody)
	flowBody.UpdateLoginFlowWithOidcMethod = kClient.NewUpdateLoginFlowWithOidcMethod("oidc", "oidc")

	req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_LOGIN_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("flow", flowId)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseLoginFlowMethodBody(gomock.Any()).Return(flowBody, nil, fmt.Errorf("error"))
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

func TestHandleUpdateFlowFailOnUpdateOIDCLoginFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	flowId := "test"
	flow := kClient.NewLoginFlowWithDefaults()
	flow.Id = flowId

	flowBody := new(kClient.UpdateLoginFlowBody)
	flowBody.UpdateLoginFlowWithOidcMethod = kClient.NewUpdateLoginFlowWithOidcMethod("oidc", "oidc")

	req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_LOGIN_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("flow", flowId)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseLoginFlowMethodBody(gomock.Any()).Return(flowBody, req.Cookies(), nil)
	mockService.EXPECT().UpdateLoginFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(nil, nil, nil, fmt.Errorf("error"))
	mockService.EXPECT().GetLoginFlow(gomock.Any(), flowId, req.Cookies()).Return(flow, nil, nil)
	mockService.EXPECT().CheckAllowedProvider(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
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

func TestHandleUpdateFlowFailOnCheckAllowedProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	flowId := "test"
	flow := kClient.NewLoginFlowWithDefaults()
	flow.Id = flowId

	flowBody := new(kClient.UpdateLoginFlowBody)
	flowBody.UpdateLoginFlowWithOidcMethod = kClient.NewUpdateLoginFlowWithOidcMethod("oidc", "oidc")

	req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_LOGIN_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("flow", flowId)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseLoginFlowMethodBody(gomock.Any()).Return(flowBody, req.Cookies(), nil)
	mockService.EXPECT().GetLoginFlow(gomock.Any(), flowId, req.Cookies()).Return(flow, nil, nil)
	mockService.EXPECT().CheckAllowedProvider(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, fmt.Errorf("error"))
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

func TestHandleCreateRecoveryFlow(t *testing.T) {
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

	flow := kClient.NewRecoveryFlowWithDefaults()
	mockService.EXPECT().CreateBrowserRecoveryFlow(gomock.Any(), redirect).Return(flow, req.Cookies(), nil)

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

func TestHandleCreateRecoveryFlowWithSession(t *testing.T) {
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

	sessionCookie := &http.Cookie{
		Name:     KRATOS_SESSION_COOKIE_NAME,
		Value:    "some_value",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
	}
	req.AddCookie(sessionCookie)

	flow := kClient.NewRecoveryFlowWithDefaults()
	mockService.EXPECT().CreateBrowserRecoveryFlow(gomock.Any(), redirect).Return(flow, req.Cookies(), nil)

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

func TestHandleCreateRecoveryFlowFailOnCreateBrowserRecoveryFlow(t *testing.T) {
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

	mockService.EXPECT().CreateBrowserRecoveryFlow(gomock.Any(), redirect).Return(nil, nil, fmt.Errorf("error"))
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

func TestHandleGetRecoveryFlow(t *testing.T) {
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

	mockService.EXPECT().GetRecoveryFlow(gomock.Any(), id, req.Cookies()).Return(flow, req.Cookies(), nil)

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
	flowResponse := kClient.NewRecoveryFlowWithDefaults()
	if err := json.Unmarshal(data, flowResponse); err != nil {
		t.Fatalf("Expected error to be nil got %v", err)
	}
	if flowResponse.Id != flow.Id {
		t.Fatalf("Expected id to be: %s, got: %s", flow.Id, flowResponse.Id)
	}
}

func TestHandleGetRecoveryFlowFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	id := "test"
	flow := kClient.NewRecoveryFlowWithDefaults()
	flow.SetId(id)

	req := httptest.NewRequest(http.MethodGet, HANDLE_GET_RECOVERY_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("id", id)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().GetRecoveryFlow(gomock.Any(), id, req.Cookies()).Return(nil, nil, fmt.Errorf("error"))
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

func TestHandleUpdateRecoveryFlow(t *testing.T) {
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

	mockService.EXPECT().ParseRecoveryFlowMethodBody(gomock.Any()).Return(flowBody, nil)
	mockService.EXPECT().UpdateRecoveryFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(redirectFlow, req.Cookies(), nil)

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
	flowResponse := new(BrowserLocationChangeRequired)
	if err := json.Unmarshal(data, flowResponse); err != nil {
		t.Fatalf("Expected error to be nil got %v", err)
	}
}

func TestHandleUpdateRecoveryFlowFailOnParseRecoveryFlowMethodBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	flowId := "test"

	flowBody := new(kClient.UpdateRecoveryFlowBody)
	flowBody.UpdateRecoveryFlowWithCodeMethod = kClient.NewUpdateRecoveryFlowWithCodeMethod("code")

	req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_RECOVERY_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("flow", flowId)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseRecoveryFlowMethodBody(gomock.Any()).Return(flowBody, fmt.Errorf("error"))
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

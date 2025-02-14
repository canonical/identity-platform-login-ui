package kratos

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	gomock "go.uber.org/mock/gomock"

	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go"
)

const (
	BASE_URL                        = "https://example.com"
	HANDLE_CREATE_FLOW_URL          = BASE_URL + "/api/kratos/self-service/login/browser"
	HANDLE_UPDATE_LOGIN_FLOW_URL    = BASE_URL + "/api/kratos/self-service/login"
	HANDLE_GET_LOGIN_FLOW_URL       = BASE_URL + "/api/kratos/self-service/login/flows"
	HANDLE_ERROR_URL                = BASE_URL + "/api/kratos/self-service/errors"
	HANDLE_CREATE_RECOVERY_FLOW_URL = BASE_URL + "/api/kratos/self-service/recovery/browser"
	HANDLE_UPDATE_RECOVERY_FLOW_URL = BASE_URL + "/api/kratos/self-service/recovery"
	HANDLE_GET_RECOVERY_FLOW_URL    = BASE_URL + "/api/kratos/self-service/recovery/flows"
	HANDLE_CREATE_SETTINGS_FLOW_URL = BASE_URL + "/api/kratos/self-service/settings/browser"
	HANDLE_UPDATE_SETTINGS_FLOW_URL = BASE_URL + "/api/kratos/self-service/settings"
	HANDLE_GET_SETTINGS_FLOW_URL    = BASE_URL + "/api/kratos/self-service/settings/flows"
)

//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_kratos.go -source=./interfaces.go

func TestHandleCreateFlowWithoutParams(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)

	flow := kClient.NewLoginFlowWithDefaults()
	flow.Id = "test"

	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_FLOW_URL, nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected HTTP status code 400 got %v", res.StatusCode)
	}
}

func TestHandleCreateFlowWithoutSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)

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

	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(nil, nil, nil)
	mockService.EXPECT().MustReAuthenticate(gomock.Any(), loginChallenge, nil, FlowStateCookie{}).Return(true, nil)
	mockService.EXPECT().CreateBrowserLoginFlow(gomock.Any(), gomock.Any(), returnTo, loginChallenge, gomock.Any(), req.Cookies()).Return(flow, req.Cookies(), nil)
	mockService.EXPECT().FilterFlowProviderList(gomock.Any(), flow).Return(flow, nil)
	mockCookieManager.EXPECT().GetStateCookie(gomock.Any()).Return(FlowStateCookie{}, nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
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

func TestHandleCreateFlowWithoutSessionFailOnCreateBrowserLoginFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)

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
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

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
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected HTTP status code 500 got %v", res.StatusCode)
	}
}

func TestHandleCreateFlowWithoutSessionWhenNoProvidersAllowed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)

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

	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(nil, nil, nil)
	mockService.EXPECT().MustReAuthenticate(gomock.Any(), loginChallenge, nil, FlowStateCookie{}).Return(true, nil)
	mockService.EXPECT().CreateBrowserLoginFlow(gomock.Any(), gomock.Any(), returnTo, loginChallenge, gomock.Any(), req.Cookies()).Return(flow, req.Cookies(), nil)
	mockService.EXPECT().FilterFlowProviderList(gomock.Any(), flow).Return(flow, nil)
	mockCookieManager.EXPECT().GetStateCookie(gomock.Any()).Return(FlowStateCookie{}, nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
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

func TestHandleCreateFlowRedirectToSetupWebauthn(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)

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
	mockService.EXPECT().HasWebAuthnAvailable(gomock.Any(), session.Id).Return(false, nil)
	mockCookieManager.EXPECT().SetStateCookie(gomock.Any(), gomock.Any()).Return(nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, true, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP status code 303 got %v", res.StatusCode)
	}
	loginFlow := ErrorBrowserLocationChangeRequired{}
	if err := json.Unmarshal(data, &loginFlow); err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	if !strings.HasPrefix(*loginFlow.RedirectBrowserTo, "/ui/setup_passkey") {
		t.Errorf("expected redirect_browser_to to start with '/ui/setup_passkey' got %v", *loginFlow.RedirectBrowserTo)
	}
}

func TestHandleCreateFlowWithSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)

	session := kClient.NewSession("test")
	redirect := "https://some/path/to/somewhere"
	redirectTo := hClient.NewOAuth2RedirectTo(redirect)

	loginChallenge := "login_challenge_2341235123231"

	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("login_challenge", loginChallenge)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil)
	mockService.EXPECT().MustReAuthenticate(gomock.Any(), loginChallenge, session, FlowStateCookie{}).Return(false, nil)
	mockService.EXPECT().AcceptLoginRequest(gomock.Any(), session, loginChallenge).Return(redirectTo, req.Cookies(), nil)
	mockCookieManager.EXPECT().GetStateCookie(gomock.Any()).Return(FlowStateCookie{}, nil)
	mockCookieManager.EXPECT().ClearStateCookie(gomock.Any()).Return()

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	data, err := ioutil.ReadAll(res.Body)
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

	session := kClient.NewSession("test")

	loginChallenge := "login_challenge_2341235123231"

	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("login_challenge", loginChallenge)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil)
	mockService.EXPECT().MustReAuthenticate(gomock.Any(), loginChallenge, session, FlowStateCookie{}).Return(false, nil)
	mockService.EXPECT().AcceptLoginRequest(gomock.Any(), session, loginChallenge).Return(nil, nil, fmt.Errorf("error"))
	mockCookieManager.EXPECT().GetStateCookie(gomock.Any()).Return(FlowStateCookie{}, nil)
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

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
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusOK {
		t.Fatal("Expected HTTP status code 200, got: ", res.Status)
	}

	data, err := ioutil.ReadAll(res.Body)
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

	id := "test"
	flow := kClient.NewLoginFlowWithDefaults()
	flow.SetId(id)

	req := httptest.NewRequest(http.MethodGet, HANDLE_GET_LOGIN_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("id", id)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().GetLoginFlow(gomock.Any(), id, req.Cookies()).Return(nil, nil, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

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

	mockCookieManager.EXPECT().SetStateCookie(gomock.Any(), gomock.Any()).Return(nil)
	mockService.EXPECT().ParseLoginFlowMethodBody(gomock.Any()).Return(flowBody, req.Cookies(), nil)
	mockService.EXPECT().UpdateLoginFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(redirectFlow, nil, req.Cookies(), nil)
	mockService.EXPECT().GetLoginFlow(gomock.Any(), flowId, req.Cookies()).Return(flow, nil, nil)
	mockService.EXPECT().CheckAllowedProvider(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusOK {
		t.Fatal("Expected HTTP status code 200, got: ", res.Status)
	}

	data, err := ioutil.ReadAll(res.Body)
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
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

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

	flowId := "test"
	redirectTo := "https://some/path/to/somewhere"
	flow := new(ErrorBrowserLocationChangeRequired)
	flow.RedirectBrowserTo = &redirectTo

	flowBody := new(kClient.UpdateLoginFlowBody)
	flowBody.UpdateLoginFlowWithOidcMethod = kClient.NewUpdateLoginFlowWithOidcMethod("oidc", "oidc")

	req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_LOGIN_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("flow", flowId)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseLoginFlowMethodBody(gomock.Any()).Return(flowBody, nil, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

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

	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil)
	mockService.EXPECT().HasTOTPAvailable(gomock.Any(), gomock.Any()).Return(true, nil)

	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil)
	mockService.EXPECT().HasNotEnoughLookupSecretsLeft(gomock.Any(), session.Identity.GetId()).Return(true, nil)
	mockCookieManager.EXPECT().SetStateCookie(gomock.Any(), gomock.Any()).Return(nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, true, true, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if _, err := json.Marshal(flow); err != nil {
		t.Fatalf("Expected error to be nil got %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatal("Expected HTTP status code 303, got: ", res.Status)
	}
}

func TestHandleUpdateFlowFailOnUpdateOIDCLoginFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)

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
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

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
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

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

	redirect := "https://example.com/ui/reset_email"

	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_RECOVERY_FLOW_URL, nil)
	values := req.URL.Query()
	req.URL.RawQuery = values.Encode()

	flow := kClient.NewRecoveryFlowWithDefaults()
	mockService.EXPECT().CreateBrowserRecoveryFlow(gomock.Any(), redirect, req.Cookies()).Return(flow, req.Cookies(), nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

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
	mockService.EXPECT().CreateBrowserRecoveryFlow(gomock.Any(), redirect, req.Cookies()).Return(flow, req.Cookies(), nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

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

	redirect := "https://example.com/ui/reset_email"

	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_RECOVERY_FLOW_URL, nil)
	values := req.URL.Query()
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().CreateBrowserRecoveryFlow(gomock.Any(), redirect, req.Cookies()).Return(nil, nil, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

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
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

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

	id := "test"
	flow := kClient.NewRecoveryFlowWithDefaults()
	flow.SetId(id)

	req := httptest.NewRequest(http.MethodGet, HANDLE_GET_RECOVERY_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("id", id)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().GetRecoveryFlow(gomock.Any(), id, req.Cookies()).Return(nil, nil, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

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
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

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

	flowId := "test"
	redirectTo := "https://example.com/ui/reset_email"
	flow := new(ErrorBrowserLocationChangeRequired)
	flow.RedirectBrowserTo = &redirectTo

	flowBody := new(kClient.UpdateRecoveryFlowBody)
	flowBody.UpdateRecoveryFlowWithCodeMethod = kClient.NewUpdateRecoveryFlowWithCodeMethod("code")

	req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_RECOVERY_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("flow", flowId)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseRecoveryFlowMethodBody(gomock.Any()).Return(flowBody, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

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

	redirect := "https://example.com/ui/setup_complete"

	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_SETTINGS_FLOW_URL, nil)
	values := req.URL.Query()
	values.Set("return_to", redirect)
	req.URL.RawQuery = values.Encode()

	flow := kClient.NewSettingsFlowWithDefaults()
	mockService.EXPECT().CreateBrowserSettingsFlow(gomock.Any(), redirect, req.Cookies()).Return(flow, nil, nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

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

	redirect := "https://example.com/ui/setup_complete"

	redirectErrorBrowserTo := "https://some/path/to/somewhere"
	redirectFlow := new(BrowserLocationChangeRequired)
	redirectFlow.RedirectTo = &redirectErrorBrowserTo

	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_SETTINGS_FLOW_URL, nil)
	values := req.URL.Query()
	values.Set("return_to", redirect)
	req.URL.RawQuery = values.Encode()

	flow := kClient.NewSettingsFlowWithDefaults()
	mockService.EXPECT().CreateBrowserSettingsFlow(gomock.Any(), redirect, req.Cookies()).Return(flow, redirectFlow, nil)
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

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

	redirect := "https://example.com/ui/setup_complete"

	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_SETTINGS_FLOW_URL, nil)
	values := req.URL.Query()
	values.Set("return_to", redirect)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().CreateBrowserSettingsFlow(gomock.Any(), redirect, req.Cookies()).Return(nil, nil, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

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
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

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

	mockService.EXPECT().GetSettingsFlow(gomock.Any(), id, req.Cookies()).Return(flow, redirectFlow, nil)
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

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

	id := "test"
	flow := kClient.NewSettingsFlowWithDefaults()
	flow.SetId(id)

	req := httptest.NewRequest(http.MethodGet, HANDLE_GET_SETTINGS_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("id", id)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().GetSettingsFlow(gomock.Any(), id, req.Cookies()).Return(nil, nil, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

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
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseSettingsFlowMethodBody(gomock.Any()).Return(flowBody, nil)
	mockService.EXPECT().UpdateSettingsFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(flow, req.Cookies(), nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

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

func TestHandleUpdateSettingsFlowFailOnParseSettingsFlowMethodBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockCookieManager := NewMockAuthCookieManagerInterface(ctrl)

	flowId := "test"

	flowBody := new(kClient.UpdateSettingsFlowBody)
	flowBody.UpdateSettingsFlowWithPasswordMethod = kClient.NewUpdateSettingsFlowWithPasswordMethod("password", "password")

	req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_SETTINGS_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("flow", flowId)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().ParseSettingsFlowMethodBody(gomock.Any()).Return(flowBody, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, false, false, BASE_URL, mockCookieManager, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatal("Expected HTTP status code 500, got: ", res.Status)
	}
}

package kratos

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-chi/chi/v5"
	gomock "github.com/golang/mock/gomock"

	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go"
)

const (
	BASE_URL                     = "https://example.com"
	HANDLE_CREATE_FLOW_URL       = BASE_URL + "/api/kratos/self-service/login/browser"
	HANDLE_UPDATE_LOGIN_FLOW_URL = BASE_URL + "/api/kratos/self-service/login"
	HANDLE_GET_LOGIN_FLOW_URL    = BASE_URL + "/api/kratos/self-service/login/flows"
	HANDLE_ERROR_URL             = BASE_URL + "/api/kratos/self-service/errors"
)

//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package kratos -destination ./mock_kratos.go -source=./interfaces.go

func TestHandleCreateFlowWithoutSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)

	flow := kClient.NewLoginFlowWithDefaults()
	flow.Id = "test"

	loginChallenge := "login_challenge_2341235123231"
	returnTo, _ := url.JoinPath(BASE_URL, "login")
	returnTo = returnTo + "?login_challenge=" + loginChallenge

	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("login_challenge", loginChallenge)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(nil, nil, nil)
	mockService.EXPECT().CreateBrowserLoginFlow(gomock.Any(), gomock.Any(), returnTo, loginChallenge, gomock.Any(), req.Cookies()).Return(flow, req.Cookies(), nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, BASE_URL, mockLogger).RegisterEndpoints(mux)

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

	flow := kClient.NewLoginFlowWithDefaults()
	flow.Id = "test"

	loginChallenge := "login_challenge_2341235123231"
	returnTo, _ := url.JoinPath(BASE_URL, "login")
	returnTo = returnTo + "?login_challenge=" + loginChallenge

	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("login_challenge", loginChallenge)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(nil, nil, nil)
	mockService.EXPECT().CreateBrowserLoginFlow(gomock.Any(), gomock.Any(), returnTo, loginChallenge, gomock.Any(), req.Cookies()).Return(nil, nil, fmt.Errorf("error"))

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, BASE_URL, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected HTTP status code 500 got %v", res.StatusCode)
	}
}

func TestHandleCreateFlowWithSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)

	session := kClient.NewSession("test", *kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"}))
	redirect := "https://some/path/to/somewhere"
	redirectTo := hClient.NewOAuth2RedirectTo(redirect)

	loginChallenge := "login_challenge_2341235123231"

	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("login_challenge", loginChallenge)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil)
	mockService.EXPECT().AcceptLoginRequest(gomock.Any(), "test", loginChallenge).Return(redirectTo, req.Cookies(), nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, BASE_URL, mockLogger).RegisterEndpoints(mux)

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

	session := kClient.NewSession("test", *kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"}))

	loginChallenge := "login_challenge_2341235123231"

	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("login_challenge", loginChallenge)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil)
	mockService.EXPECT().AcceptLoginRequest(gomock.Any(), "test", loginChallenge).Return(nil, nil, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, BASE_URL, mockLogger).RegisterEndpoints(mux)

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

	id := "test"
	flow := kClient.NewLoginFlowWithDefaults()
	flow.SetId(id)

	req := httptest.NewRequest(http.MethodGet, HANDLE_GET_LOGIN_FLOW_URL, nil)
	values := req.URL.Query()
	values.Add("id", id)
	req.URL.RawQuery = values.Encode()

	mockService.EXPECT().GetLoginFlow(gomock.Any(), id, req.Cookies()).Return(flow, req.Cookies(), nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, BASE_URL, mockLogger).RegisterEndpoints(mux)

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
	NewAPI(mockService, BASE_URL, mockLogger).RegisterEndpoints(mux)

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

	mockService.EXPECT().ParseLoginFlowMethodBody(gomock.Any()).Return(flowBody, nil)
	mockService.EXPECT().UpdateOIDCLoginFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(flow, req.Cookies(), nil)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, BASE_URL, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatal("Expected HTTP status code 422, got: ", res.Status)
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("Expected error to be nil got %v", err)
	}
	flowResponse := kClient.NewLoginFlowWithDefaults()
	if err := json.Unmarshal(data, flowResponse); err != nil {
		t.Fatalf("Expected error to be nil got %v", err)
	}
}

func TestHandleUpdateFlowFailOnParseLoginFlowMethodBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)

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

	mockService.EXPECT().ParseLoginFlowMethodBody(gomock.Any()).Return(flowBody, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, BASE_URL, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatal("Expected HTTP status code 500, got: ", res.Status)
	}
}

func TestHandleUpdateFlowFailOnUpdateOIDCLoginFlow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)

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

	mockService.EXPECT().ParseLoginFlowMethodBody(gomock.Any()).Return(flowBody, nil)
	mockService.EXPECT().UpdateOIDCLoginFlow(gomock.Any(), flowId, *flowBody, req.Cookies()).Return(nil, nil, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	w := httptest.NewRecorder()
	mux := chi.NewMux()
	NewAPI(mockService, BASE_URL, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatal("Expected HTTP status code 500, got: ", res.Status)
	}
}

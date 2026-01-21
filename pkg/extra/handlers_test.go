package extra

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/mock/gomock"

	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

const BASE_URL = "https://example.com"

//go:generate mockgen -build_flags=--mod=mod -package extra -destination ./mock_logger.go -source=../../internal/logging/interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package extra -destination ./mock_extra.go -source=./interfaces.go
//go:generate mockgen -build_flags=--mod=mod -package extra -destination ./mock_tracing.go -source=../../internal/tracing/interfaces.go

func TestHandleConsentSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockKratosService := kratos.NewMockServiceInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	session := kClient.NewSession("test")
	session.Identity = kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
	session.SetAuthenticatorAssuranceLevel(kClient.AUTHENTICATORASSURANCELEVEL_AAL1)
	consent := hClient.NewOAuth2ConsentRequest("challenge")
	accept := hClient.NewOAuth2RedirectTo("test")

	req := httptest.NewRequest(http.MethodGet, "/api/consent", nil)

	values := req.URL.Query()
	values.Add("consent_challenge", "7bb518c4eec2454dbb289f5fdb4c0ee2")
	req.URL.RawQuery = values.Encode()

	w := httptest.NewRecorder()

	mockKratosService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil)
	mockService.EXPECT().GetConsent(gomock.Any(), "7bb518c4eec2454dbb289f5fdb4c0ee2").Return(consent, nil)
	mockService.EXPECT().AcceptConsent(gomock.Any(), *session.Identity, consent).Return(accept, nil)

	mux := chi.NewMux()
	NewAPI(mockService, mockKratosService, BASE_URL, false, false, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP status code 200 got %v", res.StatusCode)
	}

	data, err := io.ReadAll(res.Body)
	defer res.Body.Close()

	if err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}

	redirect := hClient.NewOAuth2RedirectToWithDefaults()
	if err := json.Unmarshal(data, redirect); err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}

	if redirect.RedirectTo != accept.RedirectTo {
		t.Fatalf("expected %s, got %s.", accept.RedirectTo, redirect.RedirectTo)
	}
}

func TestHandleConsentWhenOIDCSequencingEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockKratosService := kratos.NewMockServiceInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	session := kClient.NewSessionWithDefaults()
	session.SetId("test")
	session.Identity = kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
	session.SetAuthenticatorAssuranceLevel(kClient.AUTHENTICATORASSURANCELEVEL_AAL2)

	method := "oidc"
	var authnMethods []kClient.SessionAuthenticationMethod
	authnMethods = append(authnMethods, kClient.SessionAuthenticationMethod{Method: &method})
	session.AuthenticationMethods = authnMethods

	consent := hClient.NewOAuth2ConsentRequest("challenge")
	accept := hClient.NewOAuth2RedirectTo("test")

	req := httptest.NewRequest(http.MethodGet, "/api/consent", nil)

	values := req.URL.Query()
	values.Add("consent_challenge", "7bb518c4eec2454dbb289f5fdb4c0ee2")
	req.URL.RawQuery = values.Encode()

	w := httptest.NewRecorder()

	mockKratosService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil)
	mockService.EXPECT().GetConsent(gomock.Any(), "7bb518c4eec2454dbb289f5fdb4c0ee2").Return(consent, nil)
	mockService.EXPECT().AcceptConsent(gomock.Any(), *session.Identity, consent).Return(accept, nil)

	mux := chi.NewMux()
	NewAPI(mockService, mockKratosService, BASE_URL, true, true, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP status code 200 got %v", res.StatusCode)
	}

	data, err := io.ReadAll(res.Body)
	defer res.Body.Close()

	if err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}

	redirect := hClient.NewOAuth2RedirectToWithDefaults()
	if err := json.Unmarshal(data, redirect); err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}

	if redirect.RedirectTo != accept.RedirectTo {
		t.Fatalf("expected %s, got %s.", accept.RedirectTo, redirect.RedirectTo)
	}
}

func TestHandleConsentInvalidPasswordAAL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockKratosService := kratos.NewMockServiceInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	session := kClient.NewSessionWithDefaults()
	session.SetId("test")
	session.Identity = kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
	session.SetAuthenticatorAssuranceLevel(kClient.AUTHENTICATORASSURANCELEVEL_AAL1)

	method := "password"
	var authnMethods []kClient.SessionAuthenticationMethod
	authnMethods = append(authnMethods, kClient.SessionAuthenticationMethod{Method: &method})
	session.AuthenticationMethods = authnMethods

	req := httptest.NewRequest(http.MethodGet, "/api/consent", nil)

	values := req.URL.Query()
	values.Add("consent_challenge", "7bb518c4eec2454dbb289f5fdb4c0ee2")
	req.URL.RawQuery = values.Encode()

	w := httptest.NewRecorder()

	mockKratosService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil)
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	mux := chi.NewMux()
	NewAPI(mockService, mockKratosService, BASE_URL, true, true, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected HTTP status code 200 got %v", res.StatusCode)
	}
}

func TestHandleConsentInvalidOIDCAAL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockKratosService := kratos.NewMockServiceInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	session := kClient.NewSessionWithDefaults()
	session.SetId("test")
	session.Identity = kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
	session.SetAuthenticatorAssuranceLevel(kClient.AUTHENTICATORASSURANCELEVEL_AAL1)

	method := "oidc"
	var authnMethods []kClient.SessionAuthenticationMethod
	authnMethods = append(authnMethods, kClient.SessionAuthenticationMethod{Method: &method})
	session.AuthenticationMethods = authnMethods

	req := httptest.NewRequest(http.MethodGet, "/api/consent", nil)

	values := req.URL.Query()
	values.Add("consent_challenge", "7bb518c4eec2454dbb289f5fdb4c0ee2")
	req.URL.RawQuery = values.Encode()

	w := httptest.NewRecorder()

	mockKratosService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil)
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	mux := chi.NewMux()
	NewAPI(mockService, mockKratosService, BASE_URL, true, true, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected HTTP status code 200 got %v", res.StatusCode)
	}
}

func TestHandleConsentFailOnAcceptConsent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockKratosService := kratos.NewMockServiceInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	session := kClient.NewSession("test")
	session.Identity = kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
	session.SetAuthenticatorAssuranceLevel(kClient.AUTHENTICATORASSURANCELEVEL_AAL1)
	consent := hClient.NewOAuth2ConsentRequest("challenge")

	req := httptest.NewRequest(http.MethodGet, "/api/consent", nil)

	values := req.URL.Query()
	values.Add("consent_challenge", "7bb518c4eec2454dbb289f5fdb4c0ee2")
	req.URL.RawQuery = values.Encode()

	w := httptest.NewRecorder()

	mockKratosService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil)
	mockService.EXPECT().GetConsent(gomock.Any(), "7bb518c4eec2454dbb289f5fdb4c0ee2").Return(consent, nil)
	mockService.EXPECT().AcceptConsent(gomock.Any(), *session.Identity, consent).Return(nil, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	mux := chi.NewMux()
	NewAPI(mockService, mockKratosService, BASE_URL, false, false, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected HTTP status code 403 got %v", res.StatusCode)
	}
}

func TestHandleConsentFailOnGetConsent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockKratosService := kratos.NewMockServiceInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	session := kClient.NewSession("test")
	session.Identity = kClient.NewIdentity("test", "test.json", "https://test.com/test.json", map[string]string{"name": "name"})
	session.SetAuthenticatorAssuranceLevel(kClient.AUTHENTICATORASSURANCELEVEL_AAL1)

	req := httptest.NewRequest(http.MethodGet, "/api/consent", nil)

	values := req.URL.Query()
	values.Add("consent_challenge", "7bb518c4eec2454dbb289f5fdb4c0ee2")
	req.URL.RawQuery = values.Encode()

	w := httptest.NewRecorder()

	mockKratosService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(session, nil, nil)
	mockService.EXPECT().GetConsent(gomock.Any(), "7bb518c4eec2454dbb289f5fdb4c0ee2").Return(nil, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	mux := chi.NewMux()
	NewAPI(mockService, mockKratosService, BASE_URL, false, false, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected HTTP status code 403 got %v", res.StatusCode)
	}
}

func TestHandleConsentFailOnCheckSession(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockKratosService := kratos.NewMockServiceInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)

	req := httptest.NewRequest(http.MethodGet, "/api/consent", nil)

	values := req.URL.Query()
	values.Add("consent_challenge", "7bb518c4eec2454dbb289f5fdb4c0ee2")
	req.URL.RawQuery = values.Encode()

	w := httptest.NewRecorder()

	mockKratosService.EXPECT().CheckSession(gomock.Any(), req.Cookies()).Return(nil, nil, fmt.Errorf("error"))
	mockLogger.EXPECT().Errorf(gomock.Any(), gomock.Any()).Times(1)

	mux := chi.NewMux()
	NewAPI(mockService, mockKratosService, BASE_URL, false, false, mockTracer, mockLogger).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected HTTP status code 403 got %v", res.StatusCode)
	}
}

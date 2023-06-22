package mocks

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const DEFAULT_SCHEMA_SERVER_URL = "test_default.json"

var schema_server_url string = DEFAULT_SCHEMA_SERVER_URL

func NewKratosServerStub() *httptest.Server {
	m := http.NewServeMux()
	m.HandleFunc("/self-service/errors", SelfServiceErrorsHandler)
	m.HandleFunc("/sessions/whoami", SessionWhoAmIHandler)
	m.HandleFunc("/self-service/login/browser", SelfServiceLoginBrowserHandler)
	m.HandleFunc("/self-service/login/flows", SelfServiceGetLoginHandler)
	m.HandleFunc("/self-service/login", SelfServiceLoginHandler)
	m.HandleFunc("/health/alive", GetOKStatus)
	m.HandleFunc("/health/ready", GetOKStatus)

	return httptest.NewServer(m)
}

func NewHydraServerStub() *httptest.Server {
	m := http.NewServeMux()
	m.HandleFunc("/admin/oauth2/auth/requests/login/accept", Oauth2AuthRequestLoginAcceptHandler)
	m.HandleFunc("/admin/oauth2/auth/requests/consent", Oauth2AuthRequestConsentHandler)
	m.HandleFunc("/admin/oauth2/auth/requests/consent/accept", Oauth2AuthRequestConsentAcceptHandler)
	m.HandleFunc("/health/alive", GetOKStatus)
	m.HandleFunc("/health/ready", GetOKStatus)

	return httptest.NewServer(m)
}

func NewSchemaServerStub() *httptest.Server {
	m := http.NewServeMux()
	m.HandleFunc("/testschema", SchemaHandler)

	return httptest.NewServer(m)
}

func createKratosMockServer() *httptest.Server {
	s := NewKratosServerStub()
	os.Setenv("KRATOS_PUBLIC_URL", s.URL)
	return s
}
func createHydraMockServer() *httptest.Server {
	s := NewKratosServerStub()
	os.Setenv("HYDRA_ADMIN_URL", s.URL)
	return s
}

// Function is kept for future unit tests where validation of Identity Traits Object Schema is needed
func createSchemaMockServer() *httptest.Server {
	s := NewSchemaServerStub()
	schema_server_url = s.URL
	SetSchemaServerURL(s.URL)
	SetSchemaServerURL(s.URL)
	return s
}

// This is a helper function to speed up development
func CreateGenericTest(t *testing.T, serverCreater func(t *testing.T), HttpMethod string, reqHTTPEndpoint string, RequestBody io.Reader, testFunction func(w http.ResponseWriter, r *http.Request)) ([]byte, error) {
	serverCreater(t)
	req := httptest.NewRequest(http.MethodGet, reqHTTPEndpoint, nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testFunction(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func GetSchemaUrl() string {
	return schema_server_url
}

func SchemaHandler(w http.ResponseWriter, r *http.Request) {
	schema, err := ioutil.ReadFile("./internal/ory/mocks/test_identity.schema.json")

	if err != nil {
		log.Fatalf("Error when opening file: %s", err)
	}

	fmt.Fprint(w, string(schema))
}

func CreateTestServers(t *testing.T) {
	kratos := createKratosMockServer()
	hydra := createHydraMockServer()
	t.Cleanup(kratos.Close)
	t.Cleanup(hydra.Close)
}

func createKratosTimeoutMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/self-service/errors", TimeoutHandler)
	mux.HandleFunc("/sessions/whoami", TimeoutHandler)
	mux.HandleFunc("/self-service/login/browser", TimeoutHandler)
	mux.HandleFunc("/self-service/login", TimeoutHandler)
	mux.HandleFunc("/health/alive", TimeoutHandler)
	mux.HandleFunc("/health/ready", TimeoutHandler)

	s := httptest.NewServer(mux)
	os.Setenv("KRATOS_PUBLIC_URL", s.URL+"/")
	return s
}
func createHydraTimeoutMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/admin/oauth2/auth/requests/login/accept", TimeoutHandler)
	mux.HandleFunc("/admin/oauth2/auth/requests/consent", TimeoutHandler)
	mux.HandleFunc("/admin/oauth2/auth/requests/consent/accept", TimeoutHandler)
	mux.HandleFunc("/health/alive", TimeoutHandler)
	mux.HandleFunc("/health/ready", TimeoutHandler)
	s := httptest.NewServer(mux)
	os.Setenv("HYDRA_ADMIN_URL", s.URL)
	return s
}

func CreateTimeoutServers(t *testing.T) {
	tkratos := createKratosTimeoutMockServer()
	thydra := createHydraTimeoutMockServer()
	t.Cleanup(tkratos.Close)
	t.Cleanup(thydra.Close)
}

func createKratosErrorMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/self-service/errors", CreateHandlerWithError("SelfServiceErrorsHandler"))
	mux.HandleFunc("/sessions/whoami", CreateHandlerWithError("SessionWhoAmIHandler"))
	mux.HandleFunc("/self-service/login/browser", CreateHandlerWithError("SelfServiceLoginBrowserHandler"))
	mux.HandleFunc("/self-service/login/flows", CreateHandlerWithError("SelfServiceGetLoginHandler"))
	mux.HandleFunc("/self-service/login", CreateHandlerWithError("SelfServiceLoginHandler"))
	mux.HandleFunc("/health/alive", GetErrorStatus)
	mux.HandleFunc("/health/ready", GetErrorStatus)

	s := httptest.NewServer(mux)
	os.Setenv("KRATOS_PUBLIC_URL", s.URL+"/")
	return s
}
func createHydraErrorMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/admin/oauth2/auth/requests/login/accept", CreateHandlerWithError("Oauth2AuthRequestLoginAcceptHandler"))
	mux.HandleFunc("/admin/oauth2/auth/requests/consent", CreateHandlerWithError("Oauth2AuthRequestConsentHandler"))
	mux.HandleFunc("/admin/oauth2/auth/requests/consent/accept", CreateHandlerWithError("Oauth2AuthRequestConsentAcceptHandler"))
	mux.HandleFunc("/health/alive", GetErrorStatus)
	mux.HandleFunc("/health/ready", GetErrorStatus)
	s := httptest.NewServer(mux)
	os.Setenv("HYDRA_ADMIN_URL", s.URL)
	return s
}
func CreateErrorServers(t *testing.T) {
	ekratos := createKratosErrorMockServer()
	ehydra := createHydraErrorMockServer()
	t.Cleanup(ekratos.Close)
	t.Cleanup(ehydra.Close)
}

func ClearEnvars(t *testing.T) {
	if _, ok := os.LookupEnv("HYDRA_ADMIN_URL"); ok {
		err := os.Unsetenv("HYDRA_ADMIN_URL")
		if err != nil {
			t.Errorf("expected error to be nil got %v", err)
		}
	}
	if _, ok := os.LookupEnv("KRATOS_PUBLIC_URL"); ok {
		err := os.Unsetenv("KRATOS_PUBLIC_URL")
		if err != nil {
			t.Errorf("expected error to be nil got %v", err)
		}
	}

	return
}

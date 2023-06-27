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

	"github.com/go-chi/chi/v5"
)

const DEFAULT_SCHEMA_SERVER_URL = "test_default.json"

var schema_server_url string = DEFAULT_SCHEMA_SERVER_URL

func NewKratosServerStub() *httptest.Server {
	m := chi.NewMux()
	m.Get("/self-service/errors", SelfServiceErrorsHandler)
	m.Get("/sessions/whoami", SessionWhoAmIHandler)
	m.Get("/self-service/login/browser", SelfServiceLoginBrowserHandler)
	m.Get("/self-service/login/flows", SelfServiceGetLoginHandler)
	m.Post("/self-service/login", SelfServiceLoginHandler)
	m.Get("/health/alive", GetOKStatus)
	m.Get("/health/ready", GetOKStatus)

	return httptest.NewServer(m)
}

func NewHydraServerStub() *httptest.Server {
	m := chi.NewMux()
	m.Put("/admin/oauth2/auth/requests/login/accept", Oauth2AuthRequestLoginAcceptHandler)
	m.Get("/admin/oauth2/auth/requests/consent", Oauth2AuthRequestConsentHandler)
	m.Put("/admin/oauth2/auth/requests/consent/accept", Oauth2AuthRequestConsentAcceptHandler)
	m.Get("/health/alive", GetOKStatus)
	m.Get("/health/ready", GetOKStatus)

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

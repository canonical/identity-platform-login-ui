package Testservers

import (
	"fmt"
	handlers "identity_platform_login_ui/ory_mocking/Handlers"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const DEFAULT_SCHEMA_SERVER_URL = "test_default.json"

var schema_server_url string = DEFAULT_SCHEMA_SERVER_URL

func createKratosMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/self-service/errors", handlers.SelfServiceErrorsHandler)
	mux.HandleFunc("/sessions/whoami", handlers.SessionWhoAmIHandler)
	mux.HandleFunc("/self-service/login/browser", handlers.SelfServiceLoginBrowserHandler)
	mux.HandleFunc("/self-service/login/flows", handlers.SelfServiceGetLoginHandler)
	mux.HandleFunc("/self-service/login", handlers.SelfServiceLoginHandler)

	s := httptest.NewServer(mux)
	os.Setenv("KRATOS_PUBLIC_URL", s.URL)
	return s
}
func createHydraMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/admin/oauth2/auth/requests/login/accept", handlers.Oauth2AuthRequestLoginAcceptHandler)
	mux.HandleFunc("/admin/oauth2/auth/requests/consent", handlers.Oauth2AuthRequestConsentHandler)
	mux.HandleFunc("/admin/oauth2/auth/requests/consent/accept", handlers.Oauth2AuthRequestConsentAcceptHandler)
	s := httptest.NewServer(mux)
	os.Setenv("HYDRA_ADMIN_URL", s.URL)
	return s
}

// Function is kept for future unit tests where validation of Identity Traits Object Schema is needed
func createSchemaMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/testschema", SchemaHandler)
	s := httptest.NewServer(mux)
	schema_server_url = s.URL
	handlers.SetSchemaServerURL(s.URL)
	handlers.SetSchemaServerURL(s.URL)
	return s
}

func GetSchemaUrl() string {
	return schema_server_url
}

func SchemaHandler(w http.ResponseWriter, r *http.Request) {
	schema, err := ioutil.ReadFile("./ory_mocking/test_identity.schema.json")
	if err != nil {
		log.Fatal("Error when opening file: ", err)
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
	mux.HandleFunc("/self-service/errors", handlers.TimeoutHandler)
	mux.HandleFunc("/sessions/whoami", handlers.TimeoutHandler)
	mux.HandleFunc("/self-service/login/browser", handlers.TimeoutHandler)
	mux.HandleFunc("/self-service/login", handlers.TimeoutHandler)

	s := httptest.NewServer(mux)
	os.Setenv("KRATOS_PUBLIC_URL", s.URL+"/")
	return s
}
func createHydraTimeoutMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/admin/oauth2/auth/requests/login/accept", handlers.TimeoutHandler)
	mux.HandleFunc("/admin/oauth2/auth/requests/consent", handlers.TimeoutHandler)
	mux.HandleFunc("/admin/oauth2/auth/requests/consent/accept", handlers.TimeoutHandler)
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
	mux.HandleFunc("/self-service/errors", handlers.CreateHandlerWithError("SelfServiceErrorsHandler"))
	mux.HandleFunc("/sessions/whoami", handlers.CreateHandlerWithError("SessionWhoAmIHandler"))
	mux.HandleFunc("/self-service/login/browser", handlers.CreateHandlerWithError("SelfServiceLoginBrowserHandler"))
	mux.HandleFunc("/self-service/login/flows", handlers.CreateHandlerWithError("SelfServiceGetLoginHandler"))
	mux.HandleFunc("/self-service/login", handlers.CreateHandlerWithError("SelfServiceLoginHandler"))

	s := httptest.NewServer(mux)
	os.Setenv("KRATOS_PUBLIC_URL", s.URL+"/")
	return s
}
func createHydraErrorMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/admin/oauth2/auth/requests/login/accept", handlers.CreateHandlerWithError("Oauth2AuthRequestLoginAcceptHandler"))
	mux.HandleFunc("/admin/oauth2/auth/requests/consent", handlers.CreateHandlerWithError("Oauth2AuthRequestConsentHandler"))
	mux.HandleFunc("/admin/oauth2/auth/requests/consent/accept", handlers.CreateHandlerWithError("Oauth2AuthRequestConsentAcceptHandler"))
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

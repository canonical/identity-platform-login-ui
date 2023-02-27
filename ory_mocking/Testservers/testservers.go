package Testservers

import (
	"fmt"
	handlers "identity_platform_login_ui/ory_mocking/Handlers"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
)

var schema_server_url string
var kratos *httptest.Server
var hydra *httptest.Server
var schema *httptest.Server

func createKratosMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/self-service/errors", handlers.SelfServiceErrorsHandler)
	mux.HandleFunc("/sessions/whoami", handlers.SessionWhoAmIHandler)
	mux.HandleFunc("/self-service/login/browser", handlers.SelfServiceLoginBrowserHandler)
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
	schema, err := ioutil.ReadFile("./test_identity.schema.json")
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}
	fmt.Fprint(w, string(schema))
}

func CloseServers() {
	kratos.Close()
	hydra.Close()
	schema.Close()
}

func CreateTestServers() func() {
	kratos = createKratosMockServer()
	hydra = createHydraMockServer()
	schema = createSchemaMockServer()
	return CloseServers
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

func CreateTimeoutServers() func() {
	tkratos := createKratosTimeoutMockServer()
	thydra := createHydraTimeoutMockServer()
	return func() {
		tkratos.Close()
		thydra.Close()
	}
}

func createKratosErrorMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/self-service/errors", handlers.CreateHandlerWithError("SelfServiceErrorsHandler"))
	mux.HandleFunc("/sessions/whoami", handlers.CreateHandlerWithError("SessionWhoAmIHandler"))
	mux.HandleFunc("/self-service/login/browser", handlers.CreateHandlerWithError("SelfServiceLoginBrowserHandler"))
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
func CreateErrorServers() func() {
	ekratos := createKratosErrorMockServer()
	ehydra := createHydraErrorMockServer()
	return func() {
		ekratos.Close()
		ehydra.Close()
	}
}

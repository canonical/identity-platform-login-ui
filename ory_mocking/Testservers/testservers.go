package Testservers

import (
	"fmt"
	testConsent "identity_platform_login_ui/ory_mocking/Consent"
	testErrors "identity_platform_login_ui/ory_mocking/Errors"
	testLoginUpdate "identity_platform_login_ui/ory_mocking/Login"
	testLoginBrowser "identity_platform_login_ui/ory_mocking/LoginBrowser"
	testSession "identity_platform_login_ui/ory_mocking/Session"
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
	mux.HandleFunc("/self-service/errors", testErrors.SelfServiceErrorsHandler)
	mux.HandleFunc("/sessions/whoami", testSession.SessionWhoAmIHandler)
	mux.HandleFunc("/self-service/login/browser", testLoginBrowser.SelfServiceLoginBrowserHandler)
	mux.HandleFunc("/self-service/login", testLoginUpdate.SelfServiceLoginHandler)

	s := httptest.NewServer(mux)
	os.Setenv("KRATOS_PUBLIC_URL", s.URL+"/")
	return s
}
func createHydraMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/admin/oauth2/auth/requests/login/accept", testLoginBrowser.Oauth2AuthRequestLoginAcceptHandler)
	mux.HandleFunc("/admin/oauth2/auth/requests/consent", testConsent.Oauth2AuthRequestConsentHandler)
	mux.HandleFunc("/admin/oauth2/auth/requests/consent/accept", testConsent.Oauth2AuthRequestConsentAcceptHandler)
	s := httptest.NewServer(mux)
	os.Setenv("HYDRA_ADMIN_URL", s.URL)
	return s
}

func createSchemaMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/testschema", SchemaHandler)
	s := httptest.NewServer(mux)
	schema_server_url = s.URL
	testSession.SetSchemaServerURL(s.URL)
	testLoginUpdate.SetSchemaServerURL(s.URL)
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

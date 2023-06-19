package extra

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/canonical/identity_platform_login_ui/internal/hydra"
	"github.com/canonical/identity_platform_login_ui/internal/kratos"
	"github.com/canonical/identity_platform_login_ui/internal/ory/mocks"

	hydra_client "github.com/ory/hydra-client-go/v2"
	"github.com/stretchr/testify/assert"
)

const (
	EXPECTED_NIL_ERROR_MESSAGE   = "expected error to be nil got %v"
	HANDLE_CREATE_FLOW_URL       = "/api/kratos/self-service/login/browser?aal=aal1&login_challenge=&refresh=false&return_to=http://test.test"
	COOKIE_NAME                  = "ory_kratos_session"
	COOKIE_VALUE                 = "test-token"
	UPDATE_LOGIN_FLOW_METHOD     = "oidc"
	UPDATE_LOGIN_FLOW_PROVIDER   = "microsoft"
	HANDLE_UPDATE_LOGIN_FLOW_URL = "/api/kratos/self-service/login?flow=1111"
	HANDLE_GET_LOGIN_FLOW_URL    = "/api/kratos/self-service/login/flows?id=1111"
	HANDLE_ERROR_URL             = "/api/kratos/self-service/errors?id=1111"
	HANDLE_CONSENT_URL           = "/api/consent?consent_challenge=test_challange"
	HANDLE_ALIVE_URL             = "/health/alive"
)

// --------------------------------------------
// TESTING WITH CORRECT SERVERS
// --------------------------------------------

func TestHandleConsent(t *testing.T) {
	kratosStub := mocks.NewKratosServerStub()
	hydraStub := mocks.NewHydraServerStub()

	defer kratosStub.Close()
	defer hydraStub.Close()
	req := httptest.NewRequest(http.MethodGet, HANDLE_CONSENT_URL, nil)
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	NewAPI(kratos.NewClient(kratosStub.URL), hydra.NewClient(hydraStub.URL)).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}
	responseRedirect := hydra_client.NewOAuth2RedirectToWithDefaults()
	if err := json.Unmarshal(data, responseRedirect); err != nil {
		t.Fatalf("expected error to be nil got %v", err)
	}

	assert.Equalf(t, mocks.CONSENT_REDIRECT, responseRedirect.RedirectTo, "Expected %s, got %s.", mocks.CONSENT_REDIRECT, responseRedirect.RedirectTo)
}

// --------------------------------------------
// TESTING WITH TIMEOUT SERVERS
// currently only prints out results main.go needs pr to handle timeouts
// --------------------------------------------
// func TestHandleConsentTimeout(t *testing.T) {
// 	data, err := CreateGenericTest(t, mocks.CreateTimeoutServers, http.MethodGet,
// 		HANDLE_CONSENT_URL,
// 		nil, handleConsent)
// 	if err != nil {
// 		t.Errorf("expected error to be nil got %v", err)
// 	}
// 	t.Logf("Result:\n%s\n", string(data))
// }

// // --------------------------------------------
// // TESTING WITH ERROR SERVERS
// // currently only prints out results main.go needs pr to handle errors
// // --------------------------------------------
// func TestHandleConsentError(t *testing.T) {
// 	data, err := CreateGenericTest(t, mocks.CreateErrorServers, http.MethodGet,
// 		HANDLE_CONSENT_URL,
// 		nil, handleConsent)
// 	if err != nil {
// 		t.Errorf("expected error to be nil got %v", err)
// 	}
// 	t.Logf("Result:\n%s\n", string(data))
// }

package kratos

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/canonical/identity_platform_login_ui/internal/hydra"
	"github.com/canonical/identity_platform_login_ui/internal/kratos"
	"github.com/canonical/identity_platform_login_ui/internal/ory/mocks"

	hydra_client "github.com/ory/hydra-client-go/v2"
	kratos_client "github.com/ory/kratos-client-go"
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
func TestHandleCreateFlowWithoutCookie(t *testing.T) {
	kratosStub := mocks.NewKratosServerStub()
	hydraStub := mocks.NewHydraServerStub()

	defer kratosStub.Close()
	defer hydraStub.Close()
	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_FLOW_URL, nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	NewAPI(kratos.NewClient(kratosStub.URL), hydra.NewClient(hydraStub.URL)).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	loginFlow := kratos_client.NewLoginFlowWithDefaults()
	if err := json.Unmarshal(data, loginFlow); err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	assert.Equalf(t, mocks.BROWSER_LOGIN_ID, loginFlow.Id, "Expected %s, got %s", mocks.BROWSER_LOGIN_ID, loginFlow.Id)
}

func TestHandleCreateFlowWithCookie(t *testing.T) {
	kratosStub := mocks.NewKratosServerStub()
	hydraStub := mocks.NewHydraServerStub()

	defer kratosStub.Close()
	defer hydraStub.Close()

	//create request and response objects
	req := httptest.NewRequest(http.MethodPut, HANDLE_CREATE_FLOW_URL, nil)
	req.Header.Set("Content-Type", "application/json")
	cookie := &http.Cookie{
		Name:   COOKIE_NAME,
		Value:  COOKIE_VALUE,
		MaxAge: 300,
	}
	req.AddCookie(cookie)
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	NewAPI(kratos.NewClient(kratosStub.URL), hydra.NewClient(hydraStub.URL)).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("expected StatusCode to be 200 got %v", res.StatusCode)
	}
	requestLoginResponse := hydra_client.NewOAuth2RedirectToWithDefaults()
	if err := json.Unmarshal(data, requestLoginResponse); err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}

	assert.Equalf(t, mocks.AUTHORIZATION_REDIRECT, requestLoginResponse.RedirectTo, "Expected %s, got %s", mocks.AUTHORIZATION_REDIRECT, requestLoginResponse.RedirectTo)
}

func TestHandleUpdateFlow(t *testing.T) {
	kratosStub := mocks.NewKratosServerStub()
	hydraStub := mocks.NewHydraServerStub()

	defer kratosStub.Close()
	defer hydraStub.Close()

	//create request
	body := kratos_client.NewUpdateLoginFlowWithOidcMethod(UPDATE_LOGIN_FLOW_METHOD, UPDATE_LOGIN_FLOW_PROVIDER)
	bodyJson, err := json.Marshal(*body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	bodyReader := bytes.NewReader(bodyJson)
	req := httptest.NewRequest(http.MethodPost, HANDLE_UPDATE_LOGIN_FLOW_URL, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Session-Token", "test-x-session-token")
	cookie := &http.Cookie{
		Name:   COOKIE_NAME,
		Value:  COOKIE_VALUE,
		MaxAge: 300,
	}
	req.AddCookie(cookie)

	//create response
	w := httptest.NewRecorder()

	mux := http.NewServeMux()
	NewAPI(kratos.NewClient(kratosStub.URL), hydra.NewClient(hydraStub.URL)).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)
	//check results
	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	loginUpdateResponse := kratos_client.NewSuccessfulNativeLoginWithDefaults()
	if err := json.Unmarshal(data, loginUpdateResponse); err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	assert.Equalf(t, mocks.SESSION_ID, loginUpdateResponse.Session.Id, "Expected %s, got %s", mocks.SESSION_ID, loginUpdateResponse.Session.Id)
}

func TestHandleGetLoginFlow(t *testing.T) {
	kratosStub := mocks.NewKratosServerStub()
	hydraStub := mocks.NewHydraServerStub()

	defer kratosStub.Close()
	defer hydraStub.Close()

	//create request
	req := httptest.NewRequest(http.MethodPost, HANDLE_GET_LOGIN_FLOW_URL, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Session-Token", "test-x-session-token")
	cookie := &http.Cookie{
		Name:   COOKIE_NAME,
		Value:  COOKIE_VALUE,
		MaxAge: 300,
	}
	req.AddCookie(cookie)

	//create response
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	NewAPI(kratos.NewClient(kratosStub.URL), hydra.NewClient(hydraStub.URL)).RegisterEndpoints(mux)

	mux.ServeHTTP(w, req)
	//check results
	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	loginFlow := kratos_client.NewLoginFlowWithDefaults()
	if err := json.Unmarshal(data, loginFlow); err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	assert.Equalf(t, mocks.BROWSER_LOGIN_ID, loginFlow.Id, "Expected %s, got %s", mocks.BROWSER_LOGIN_ID, loginFlow.Id)
}

// func TestHandleKratosError(t *testing.T) {
// 	testServers.CreateTestServers(t)

// 	req := httptest.NewRequest(http.MethodGet, HANDLE_ERROR_URL, nil)
// 	w := httptest.NewRecorder()
// 	handleKratosError(w, req)
// 	res := w.Result()
// 	defer res.Body.Close()
// 	data, err := ioutil.ReadAll(res.Body)
// 	if err != nil {
// 		t.Errorf("expected error to be nil got %v", err)
// 	}
// 	te := new(mocks.TestErrorReport)
// 	if err := json.Unmarshal(data, te); err != nil {
// 		t.Errorf("expected error to be nil got %v", err)
// 	}
// 	assert.Equalf(t, mocks.ERROR_MESSAGE, te.Error.Message, "Expected %s, got %s", mocks.ERROR_MESSAGE, te.Error.Message)

// }

// // --------------------------------------------
// // TESTING WITH TIMEOUT SERVERS
// // currently only prints out results main.go needs pr to handle timeouts
// // --------------------------------------------
// func TestHandleCreateFlowTimeout(t *testing.T) {
// 	data, err := CreateGenericTest(t, testServers.CreateTimeoutServers, http.MethodPut,
// 		HANDLE_CREATE_FLOW_URL,
// 		nil, handleCreateFlow)
// 	if err != nil {
// 		t.Errorf("expected error to be nil got %v", err)
// 	}
// 	t.Logf("Result:\n%s\n", string(data))
// }
// func TestHandleUpdateFlowTimeout(t *testing.T) {
// 	//create request
// 	body := kratos_client.NewUpdateLoginFlowWithOidcMethod(UPDATE_LOGIN_FLOW_METHOD, UPDATE_LOGIN_FLOW_PROVIDER)
// 	bodyJson, err := json.Marshal(*body)
// 	if err != nil {
// 		t.Errorf("expected error to be nil got %v", err)
// 	}
// 	bodyReader := bytes.NewReader(bodyJson)
// 	data, err := CreateGenericTest(t, testServers.CreateTimeoutServers, http.MethodPost,
// 		HANDLE_UPDATE_LOGIN_FLOW_URL,
// 		bodyReader, handleUpdateFlow)
// 	if err != nil {
// 		t.Errorf("expected error to be nil got %v", err)
// 	}
// 	t.Logf("Result:\n%s\n", string(data))
// }
// func TestHandleKratosErrorTimeout(t *testing.T) {
// 	data, err := CreateGenericTest(t, testServers.CreateTimeoutServers, http.MethodGet,
// 		HANDLE_ERROR_URL,
// 		nil, handleKratosError)
// 	if err != nil {
// 		t.Errorf("expected error to be nil got %v", err)
// 	}
// 	t.Logf("Result:\n%s\n", string(data))
// }

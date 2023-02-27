package main

import (
	"bytes"
	"encoding/json"
	handlers "identity_platform_login_ui/ory_mocking/Handlers"
	testServers "identity_platform_login_ui/ory_mocking/Testservers"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

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
	HANDLE_ERROR_URL             = "/api/kratos/self-service/errors?id=1111"
	HANDLE_CONSENT_URL           = "/api/consent?consent_challenge=test_challange"
)

// --------------------------------------------
// TESTING WITH CORRECT SERVERS
// --------------------------------------------
func TestHandleCreateFlowWithoutCookie(t *testing.T) {
	//init clients
	t.Cleanup(testServers.CreateTestServers())

	//create request and response objects
	req := httptest.NewRequest(http.MethodGet, HANDLE_CREATE_FLOW_URL, nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	//test and evaluate test
	handleCreateFlow(w, req)
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
	assert.Equalf(t, handlers.BROWSER_LOGIN_ID, loginFlow.Id, "Expected %s, got %s", handlers.BROWSER_LOGIN_ID, loginFlow.Id)
}

func TestHandleCreateFlowWithCookie(t *testing.T) {
	//init clients
	t.Cleanup(testServers.CreateTestServers())

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

	//test and evaluate test
	handleCreateFlow(w, req)
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
	if requestLoginResponse.RedirectTo != handlers.REDIRECT {
		t.Errorf("expected test.test, got %v", string(data))
	}
	assert.Equalf(t, handlers.REDIRECT, requestLoginResponse.RedirectTo, "Expected %s, got %s", handlers.REDIRECT, requestLoginResponse.RedirectTo)
}

func TestHandleUpdateFlow(t *testing.T) {
	//init clients
	t.Cleanup(testServers.CreateTestServers())

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
	//start function
	handleUpdateFlow(w, req)
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
	assert.Equalf(t, handlers.SESSION_ID, loginUpdateResponse.Session.Id, "Expected %s, got %s", handlers.SESSION_ID, loginUpdateResponse.Session.Id)
}

func TestHandleKratosError(t *testing.T) {
	t.Cleanup(testServers.CreateTestServers())

	req := httptest.NewRequest(http.MethodGet, HANDLE_ERROR_URL, nil)
	w := httptest.NewRecorder()
	handleKratosError(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	te := new(handlers.TestErrorReport)
	if err := json.Unmarshal(data, te); err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	assert.Equalf(t, handlers.ERROR_MESSAGE, te.Error.Message, "Expected %s, got %s", handlers.ERROR_MESSAGE, te.Error.Message)

}

func TestHandleConsent(t *testing.T) {
	t.Cleanup(testServers.CreateTestServers())

	req := httptest.NewRequest(http.MethodGet, HANDLE_CONSENT_URL, nil)
	w := httptest.NewRecorder()
	handleConsent(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	responseRedirect := hydra_client.NewOAuth2RedirectToWithDefaults()
	if err := json.Unmarshal(data, responseRedirect); err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}

	if responseRedirect.RedirectTo != "test.test" {
		t.Errorf("expected test.test, got %v", string(data))
	}
	assert.Equalf(t, "test.test", responseRedirect.RedirectTo, "Expected %s, got %s.", "test.test", responseRedirect.RedirectTo)
}

// --------------------------------------------
// TESTING WITH TIMEOUT SERVERS
// currently only prints out results main.go needs pr to handle timeouts
// --------------------------------------------
func TestHandleCreateFlowTimeout(t *testing.T) {
	data, err := CreateGenericTest(testServers.CreateTimeoutServers, http.MethodPut,
		HANDLE_CREATE_FLOW_URL,
		nil, handleCreateFlow)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	t.Logf("Result:\n%s\n", string(data))
}
func TestHandleUpdateFlowTimeout(t *testing.T) {
	//create request
	body := kratos_client.NewUpdateLoginFlowWithOidcMethod(UPDATE_LOGIN_FLOW_METHOD, UPDATE_LOGIN_FLOW_PROVIDER)
	bodyJson, err := json.Marshal(*body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	bodyReader := bytes.NewReader(bodyJson)
	data, err := CreateGenericTest(testServers.CreateTimeoutServers, http.MethodPost,
		HANDLE_UPDATE_LOGIN_FLOW_URL,
		bodyReader, handleUpdateFlow)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	t.Logf("Result:\n%s\n", string(data))
}
func TestHandleKratosErrorTimeout(t *testing.T) {
	data, err := CreateGenericTest(testServers.CreateTimeoutServers, http.MethodGet,
		HANDLE_ERROR_URL,
		nil, handleKratosError)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	t.Logf("Result:\n%s\n", string(data))
}
func TestHandleConsentTimeout(t *testing.T) {
	data, err := CreateGenericTest(testServers.CreateTimeoutServers, http.MethodGet,
		HANDLE_CONSENT_URL,
		nil, handleConsent)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	t.Logf("Result:\n%s\n", string(data))
}

// --------------------------------------------
// TESTING WITH ERROR SERVERS
// currently only prints out results main.go needs pr to handle errors
// --------------------------------------------
func TestHandleCreateFlowError(t *testing.T) {
	data, err := CreateGenericTest(testServers.CreateErrorServers, http.MethodPut,
		HANDLE_CREATE_FLOW_URL,
		nil, handleCreateFlow)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	t.Logf("Result:\n%s\n", string(data))
}
func TestHandleUpdateFlowError(t *testing.T) {
	//create request
	body := kratos_client.NewUpdateLoginFlowWithOidcMethod(UPDATE_LOGIN_FLOW_METHOD, UPDATE_LOGIN_FLOW_PROVIDER)
	bodyJson, err := json.Marshal(*body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	bodyReader := bytes.NewReader(bodyJson)
	data, err := CreateGenericTest(testServers.CreateErrorServers, http.MethodPost,
		HANDLE_UPDATE_LOGIN_FLOW_URL,
		bodyReader, handleUpdateFlow)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	t.Logf("Result:\n%s\n", string(data))
}
func TestHandleKratosErrorError(t *testing.T) {
	data, err := CreateGenericTest(testServers.CreateErrorServers, http.MethodGet,
		HANDLE_ERROR_URL,
		nil, handleKratosError)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	t.Logf("Result:\n%s\n", string(data))
}
func TestHandleConsentError(t *testing.T) {
	data, err := CreateGenericTest(testServers.CreateErrorServers, http.MethodGet,
		HANDLE_CONSENT_URL,
		nil, handleConsent)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	t.Logf("Result:\n%s\n", string(data))
}

// This is a helper function to speed up development
func CreateGenericTest(serverCreater func() func(), HttpMethod string, reqHTTPEndpoint string, RequestBody io.Reader, testFunction func(w http.ResponseWriter, r *http.Request)) ([]byte, error) {
	serverClose := serverCreater()
	defer serverClose()
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

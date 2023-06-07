package main

import (
	"bytes"
	"encoding/json"
	"identity_platform_login_ui/health"
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
	HANDLE_GET_LOGIN_FLOW_URL    = "/api/kratos/self-service/login/flows?id=1111"
	HANDLE_ERROR_URL             = "/api/kratos/self-service/errors?id=1111"
	HANDLE_CONSENT_URL           = "/api/consent?consent_challenge=test_challange"
	HANDLE_ALIVE_URL             = "/health/alive"
	HANDLE_READY_URL             = "/health/ready"
)

// --------------------------------------------
// TESTING WITH CORRECT SERVERS
// --------------------------------------------
func TestHandleCreateFlowWithoutCookie(t *testing.T) {
	//init clients
	testServers.CreateTestServers(t)

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
	testServers.CreateTestServers(t)

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

	assert.Equalf(t, handlers.AUTHORIZATION_REDIRECT, requestLoginResponse.RedirectTo, "Expected %s, got %s", handlers.AUTHORIZATION_REDIRECT, requestLoginResponse.RedirectTo)
}

func TestHandleUpdateFlow(t *testing.T) {
	//init clients
	testServers.CreateTestServers(t)

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

func TestHandleGetLoginFlow(t *testing.T) {
	//init clients
	testServers.CreateTestServers(t)

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
	//start function
	handleLoginFlow(w, req)
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
	assert.Equalf(t, handlers.BROWSER_LOGIN_ID, loginFlow.Id, "Expected %s, got %s", handlers.BROWSER_LOGIN_ID, loginFlow.Id)
}

func TestHandleKratosError(t *testing.T) {
	testServers.CreateTestServers(t)

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
	testServers.CreateTestServers(t)

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

	assert.Equalf(t, handlers.CONSENT_REDIRECT, responseRedirect.RedirectTo, "Expected %s, got %s.", handlers.CONSENT_REDIRECT, responseRedirect.RedirectTo)
}

// --------------------------------------------
// TESTING WITH TIMEOUT SERVERS
// currently only prints out results main.go needs pr to handle timeouts
// --------------------------------------------
func TestHandleCreateFlowTimeout(t *testing.T) {
	data, err := CreateGenericTest(t, testServers.CreateTimeoutServers, http.MethodPut,
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
	data, err := CreateGenericTest(t, testServers.CreateTimeoutServers, http.MethodPost,
		HANDLE_UPDATE_LOGIN_FLOW_URL,
		bodyReader, handleUpdateFlow)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	t.Logf("Result:\n%s\n", string(data))
}
func TestHandleKratosErrorTimeout(t *testing.T) {
	data, err := CreateGenericTest(t, testServers.CreateTimeoutServers, http.MethodGet,
		HANDLE_ERROR_URL,
		nil, handleKratosError)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	t.Logf("Result:\n%s\n", string(data))
}
func TestHandleConsentTimeout(t *testing.T) {
	data, err := CreateGenericTest(t, testServers.CreateTimeoutServers, http.MethodGet,
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
	data, err := CreateGenericTest(t, testServers.CreateErrorServers, http.MethodPut,
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
	data, err := CreateGenericTest(t, testServers.CreateErrorServers, http.MethodPost,
		HANDLE_UPDATE_LOGIN_FLOW_URL,
		bodyReader, handleUpdateFlow)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	t.Logf("Result:\n%s\n", string(data))
}
func TestHandleKratosErrorError(t *testing.T) {
	data, err := CreateGenericTest(t, testServers.CreateErrorServers, http.MethodGet,
		HANDLE_ERROR_URL,
		nil, handleKratosError)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	t.Logf("Result:\n%s\n", string(data))
}
func TestHandleConsentError(t *testing.T) {
	data, err := CreateGenericTest(t, testServers.CreateErrorServers, http.MethodGet,
		HANDLE_CONSENT_URL,
		nil, handleConsent)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	t.Logf("Result:\n%s\n", string(data))
}

// --------------------------------------------
// TESTING HEALTH CHECKS
// --------------------------------------------
func TestAliveOK(t *testing.T) {
	health.TestResetHealth()
	req := httptest.NewRequest(http.MethodGet, HANDLE_ALIVE_URL, nil)
	w := httptest.NewRecorder()
	health.HandleAlive(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	recievedStatus := health.EmptyStatus()
	if err := json.Unmarshal(data, recievedStatus); err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	assert.Equalf(t, "ok", recievedStatus.Status, "Expected %s, got %s", "ok", recievedStatus.Status)
}

func TestAliveFail(t *testing.T) {
	health.TestResetHealth()
	testMessage := "Liveness Check failed for test"
	health.TestSetUnalive(testMessage)

	req := httptest.NewRequest(http.MethodGet, HANDLE_ALIVE_URL, nil)
	w := httptest.NewRecorder()
	health.HandleAlive(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	recievedStatus := health.EmptyStatus()
	if err := json.Unmarshal(data, recievedStatus); err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	assert.Equalf(t, http.StatusText(503), recievedStatus.Status, "Expected %s, got %s", http.StatusText(503), recievedStatus.Status)
	assert.Equalf(t, testMessage, recievedStatus.Message, "Expected %s, got %s", testMessage, recievedStatus.Message)
}

func TestReadyOK(t *testing.T) {
	//init clients
	health.TestResetHealth()

	req := httptest.NewRequest(http.MethodGet, HANDLE_ALIVE_URL, nil)
	w := httptest.NewRecorder()
	health.HandleReady(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	recievedStatus := health.EmptyStatus()
	if err := json.Unmarshal(data, recievedStatus); err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	assert.Equalf(t, "ok", recievedStatus.Status, "Expected %s, got %s", "ok", recievedStatus.Status)
}

func TestReadyFail(t *testing.T) {
	health.TestResetHealth()
	testServers.ClearEnvars(t)
	testMessage := "Readiness Check failed for test"

	req := httptest.NewRequest(http.MethodGet, HANDLE_READY_URL, nil)
	w := httptest.NewRecorder()
	health.TestSetUnready(testMessage)
	health.HandleReady(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	recievedStatus := health.EmptyStatus()
	if err := json.Unmarshal(data, recievedStatus); err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	assert.Equalf(t, http.StatusText(503), recievedStatus.Status, "Expected %s, got %s", http.StatusText(503), recievedStatus.Status)
	assert.Equalf(t, testMessage, recievedStatus.Message, "Expected %s, got %s", testMessage, recievedStatus.Message)
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

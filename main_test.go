package main

import (
	"bytes"
	"encoding/json"
	testConsent "identity_platform_login_ui/ory_mocking/Consent"
	testErrors "identity_platform_login_ui/ory_mocking/Errors"
	testLoginUpdate "identity_platform_login_ui/ory_mocking/Login"
	testLoginBrowser "identity_platform_login_ui/ory_mocking/LoginBrowser"
	testServers "identity_platform_login_ui/ory_mocking/Testservers"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --------------------------------------------
// TESTING WITH CORRECT SERVERS
// --------------------------------------------
func TestHandleCreateFlowWithoutCookie(t *testing.T) {
	//init clients
	serverClose := testServers.CreateTestServers()
	defer serverClose()

	//create request and response objects
	req := httptest.NewRequest(http.MethodGet, "/api/kratos/self-service/login/browser?aal=aal1&login_challenge=&refresh=false&return_to=http://test.test", nil)
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
	lbr := new(testLoginBrowser.LoginBrowserResponse)
	if err := json.Unmarshal(data, lbr); err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	if lbr.Id != "test_id" {
		t.Errorf("expected test_id, got %v", string(data))
	}
}

func TestHandleCreateFlowWithCookie(t *testing.T) {
	//init clients
	serverClose := testServers.CreateTestServers()
	defer serverClose()

	//create request and response objects
	req := httptest.NewRequest(http.MethodPut, "/api/kratos/self-service/login/browser?aal=aal1&login_challenge=test_challange&refresh=false&return_to=http://test.test", nil)
	req.Header.Set("Content-Type", "application/json")
	cookie := &http.Cookie{
		Name:   "ory_kratos_session",
		Value:  "test-token",
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
	requestLoginResponse := new(testLoginBrowser.OAuth2RequestLoginResponse)
	if err := json.Unmarshal(data, requestLoginResponse); err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	if requestLoginResponse.Redirect_to != "test.test" {
		t.Errorf("expected test.test, got %v", string(data))
	}
}

func TestHandleUpdateFlow(t *testing.T) {
	//init clients
	serverClose := testServers.CreateTestServers()
	defer serverClose()

	//create request
	body := testLoginBrowser.LoginBody{
		Method:   "oidc",
		Provider: "microsoft",
	}
	bodyJson, err := json.Marshal(body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	bodyReader := bytes.NewReader(bodyJson)
	req := httptest.NewRequest(http.MethodPost, "/api/kratos/self-service/login?flow=1111", bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Session-Token", "test-x-session-token")
	cookie := &http.Cookie{
		Name:   "ory_kratos_session",
		Value:  "test-token",
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
	loginUpdateResponse := new(testLoginUpdate.LoginUpdateResponse)
	if err := json.Unmarshal(data, loginUpdateResponse); err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	if loginUpdateResponse.Session_token != "test-token" {
		t.Errorf("expected test-token, got %v", string(data))
	}
	if loginUpdateResponse.Session.Id != "test-1111" {
		t.Errorf("expected test-1111, got %v", string(data))
	}
}

func TestHandleKratosError(t *testing.T) {
	serverClose := testServers.CreateTestServers()
	defer serverClose()
	req := httptest.NewRequest(http.MethodGet, "/api/kratos/self-service/errors?id=1111", nil)
	w := httptest.NewRecorder()
	handleKratosError(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	te := new(testErrors.TestErrorReport)
	if err := json.Unmarshal(data, te); err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	if te.Error.Message != "This is a test" {
		t.Errorf("expected This is a test, got %v", string(data))
	}
}

func TestHandleConsent(t *testing.T) {
	serverClose := testServers.CreateTestServers()
	defer serverClose()
	req := httptest.NewRequest(http.MethodGet, "/api/consent?consent_challenge=test_challange", nil)
	w := httptest.NewRecorder()
	handleConsent(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	responseRedirect := new(testConsent.OAuth2ConsentAcceptResponse)
	if err := json.Unmarshal(data, responseRedirect); err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	if responseRedirect.Redirect_to != "test.test" {
		t.Errorf("expected test.test, got %v", string(data))
	}
}

// --------------------------------------------
// TESTING WITH TIMEOUT SERVERS
// currently only prints out results main.go needs pr to handle timeouts
// --------------------------------------------
func TestHandleCreateFlowTimeout(t *testing.T) {
	data, err := CreateGenericTest(testServers.CreateTimeoutServers, http.MethodPut,
		"/api/kratos/self-service/login/browser?aal=aal1&login_challenge=&refresh=false&return_to=http://test.test",
		nil, handleCreateFlow)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	t.Logf("Result:\n%s\n", string(data))
}
func TestHandleUpdateFlowTimeout(t *testing.T) {
	//create request
	body := testLoginBrowser.LoginBody{
		Method:   "oidc",
		Provider: "microsoft",
	}
	bodyJson, err := json.Marshal(body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	bodyReader := bytes.NewReader(bodyJson)
	data, err := CreateGenericTest(testServers.CreateTimeoutServers, http.MethodPost,
		"/api/kratos/self-service/login?flow=1111",
		bodyReader, handleUpdateFlow)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	t.Logf("Result:\n%s\n", string(data))
}
func TestHandleKratosErrorTimeout(t *testing.T) {
	data, err := CreateGenericTest(testServers.CreateTimeoutServers, http.MethodGet,
		"/api/kratos/self-service/errors?id=1111",
		nil, handleKratosError)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	t.Logf("Result:\n%s\n", string(data))
}
func TestHandleConsentTimeout(t *testing.T) {
	data, err := CreateGenericTest(testServers.CreateTimeoutServers, http.MethodGet,
		"/api/consent?consent_challenge=test_challange",
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
		"/api/kratos/self-service/login/browser?aal=aal1&login_challenge=&refresh=false&return_to=http://test.test",
		nil, handleCreateFlow)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	t.Logf("Result:\n%s\n", string(data))
}
func TestHandleUpdateFlowError(t *testing.T) {
	//create request
	body := testLoginBrowser.LoginBody{
		Method:   "oidc",
		Provider: "microsoft",
	}
	bodyJson, err := json.Marshal(body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	bodyReader := bytes.NewReader(bodyJson)
	data, err := CreateGenericTest(testServers.CreateErrorServers, http.MethodPost,
		"/api/kratos/self-service/login?flow=1111",
		bodyReader, handleUpdateFlow)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	t.Logf("Result:\n%s\n", string(data))
}
func TestHandleKratosErrorError(t *testing.T) {
	data, err := CreateGenericTest(testServers.CreateErrorServers, http.MethodGet,
		"/api/kratos/self-service/errors?id=1111",
		nil, handleKratosError)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	t.Logf("Result:\n%s\n", string(data))
}
func TestHandleConsentError(t *testing.T) {
	data, err := CreateGenericTest(testServers.CreateErrorServers, http.MethodGet,
		"/api/consent?consent_challenge=test_challange",
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
	req := httptest.NewRequest(http.MethodGet, "/api/consent?consent_challenge=test_challange", nil)
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

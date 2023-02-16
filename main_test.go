package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func CreateKratosMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/self-service/errors", SelfServiceErrorsHandler)
	mux.HandleFunc("/sessions/whoami", SessionWhoAmIHandler)
	mux.HandleFunc("/self-service/login/browser", SelfServiceLoginBrowserHandler)
	//Post /self-service/login
	mux.HandleFunc("/self-service/login", SelfServiceLoginHandler)

	s := httptest.NewServer(mux)
	os.Setenv("KRATOS_PUBLIC_URL", s.URL+"/")
	return s
}
func CreateHydraMockServer() *httptest.Server {
	mux := http.NewServeMux()

	s := httptest.NewServer(mux)
	os.Setenv("HYDRA_ADMIN_URL", s.URL+"/")
	return s
}

// kratos GET //self-service/login/browser?aal=&login_challenge=&refresh=false&return_to= HTTP/1.1
// hydra Put /oauth2/auth/requests/login/accept
func TestHandleCreateFlow(t *testing.T) {
	kratosClient := CreateKratosMockServer()
	hydraClient := CreateHydraMockServer()
	defer kratosClient.Close()
	defer hydraClient.Close()
	req := httptest.NewRequest(http.MethodGet, "/api/kratos/self-service/login/browser", nil)
	w := httptest.NewRecorder()
	handleCreateFlow(w, req)
	res := w.Result()
	defer res.Body.Close()
	_, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
}

// POST //self-service/login?flow= HTTP/1.1
func TestHandleUpdateFlow(t *testing.T) {
	kratosClient := CreateKratosMockServer()
	hydraClient := CreateHydraMockServer()
	defer kratosClient.Close()
	defer hydraClient.Close()
	req := httptest.NewRequest(http.MethodGet, "/api/kratos/self-service/login", nil)
	w := httptest.NewRecorder()
	handleUpdateFlow(w, req)
	res := w.Result()
	defer res.Body.Close()
	_, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
}

func TestHandleKratosError(t *testing.T) {
	client := CreateKratosMockServer()
	defer client.Close()
	req := httptest.NewRequest(http.MethodGet, "/api/kratos/self-service/errors?id=1111", nil)
	w := httptest.NewRecorder()
	handleKratosError(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	te := new(TestErrorReport)
	if err := json.Unmarshal(data, te); err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	if te.Error.Message != "This is a test" {
		t.Errorf("expected This is a test, got %v", string(data))
	}
}

// GET //sessions/whoami HTTP/1.1
// hydra Get /oauth2/auth/requests/consent
// hydra Put /oauth2/auth/requests/consent/accept
func TestHandleConsent(t *testing.T) {
	kratosClient := CreateKratosMockServer()
	hydraClient := CreateHydraMockServer()
	defer kratosClient.Close()
	defer hydraClient.Close()
	req := httptest.NewRequest(http.MethodGet, "/api/consent", nil)
	w := httptest.NewRecorder()
	handleConsent(w, req)
	res := w.Result()
	defer res.Body.Close()
	_, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
}

type TestErrorReport struct {
	Id         string    `json:"id"`
	Error      TestError `json:"error"`
	Created_at string    `json:"created_at"`
	Updated_at string    `json:"updated_at"`
}

type TestError struct {
	Code    int    `json:"code"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

func SelfServiceErrorsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := q.Get("id")
	e := TestError{
		Code:    400,
		Status:  "Bad Request",
		Reason:  "This is a test",
		Message: "This is a test",
	}
	et := TestErrorReport{
		Id:         id,
		Error:      e,
		Created_at: time.Now().Format("2006-01-02T15:04:05Z07:00"),
		Updated_at: time.Now().Format("2006-01-02T15:04:05Z07:00"),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	response, err := json.Marshal(et)
	if err != nil {
		log.Printf("Bug in test: CreateKratosMockServer\nerror: %s", err.Error())
	}
	w.Write(response)
	return
}

func SessionWhoAmIHandler(w http.ResponseWriter, r *http.Request) {

}
func SelfServiceLoginBrowserHandler(w http.ResponseWriter, r *http.Request) {

}
func SelfServiceLoginHandler(w http.ResponseWriter, r *http.Request) {

}

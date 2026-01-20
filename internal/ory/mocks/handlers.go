package mocks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	hydra_client "github.com/ory/hydra-client-go/v2"
	kratos_client "github.com/ory/kratos-client-go/v25"
)

const (
	BROWSER_LOGIN_ID        = "test_id"
	BROWSER_LOGIN_TYPE      = "browser"
	OAUTH2_SUBJECT          = "test1234"
	CONSENT_REDIRECT        = "test.test"
	AUTHORIZATION_REDIRECT  = "test.test"
	TEST_ERROR_CODE         = 599
	TRAITS_NAME             = "TestName"
	IDENTITY_ID             = "test1234"
	SCHEMA_ID               = "test_identity.schema.json"
	SESSION_ID              = "test-1111"
	ERROR_CODE              = 400
	ERROR_STATUS            = "Bad Request"
	ERROR_REASON            = "This is a test"
	ERROR_MESSAGE           = "This is a test"
	DEFAULT_ERROR_CODE      = 599
	CONSENT_CHALLANGE       = "test_challange"
	AUTHORIZATION_CHALLANGE = "test_authorization_challenge"
)

var schemaServerURL string

type IdentityTraits struct {
	Name string `json:"name"`
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

type GenericError struct {
	Code    int64  `json:"code"`
	Debug   string `json:"debug"`
	Id      string `json:"id"`
	Message string `json:"message"`
	Reason  string `json:"reason"`
	Request string `json:"request"`
	Status  string `json:"status"`
}

type Status struct {
	Status string `json:"status"`
}

func GenericErrorConstructor(testname string) GenericError {
	error := GenericError{
		Code:    DEFAULT_ERROR_CODE,
		Debug:   fmt.Sprintf("Test Debug message from %s handler.", testname),
		Id:      fmt.Sprintf("Test-ID-%s", testname),
		Message: fmt.Sprintf("Test Message from %s handler", testname),
		Reason:  fmt.Sprintf("Test Reason from %s handler", testname),
		Request: fmt.Sprintf("Test Request ID"),
		Status:  http.StatusText(500),
	}
	return error
}

func SetSchemaServerURL(url string) {
	schemaServerURL = url
}

func SelfServiceLoginBrowserHandler(w http.ResponseWriter, r *http.Request) {
	uiContainer := kratos_client.NewUiContainerWithDefaults()
	response := kratos_client.NewLoginFlow(time.Now(), BROWSER_LOGIN_ID, time.Now(), r.URL.Path, "choose_method", BROWSER_LOGIN_TYPE, *uiContainer)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	jsonResp, err := json.Marshal(response)
	if err != nil {
		log.Printf("Bug in test: SelfServiceLoginBrowserHandler\nerror: %s", err.Error())
	}
	w.Write(jsonResp)
	return
}

func SelfServiceGetLoginHandler(w http.ResponseWriter, r *http.Request) {
	uiContainer := kratos_client.NewUiContainerWithDefaults()
	response := kratos_client.NewLoginFlow(time.Now(), BROWSER_LOGIN_ID, time.Now(), r.URL.Path, "choose_method", BROWSER_LOGIN_TYPE, *uiContainer)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	jsonResp, err := json.Marshal(response)
	if err != nil {
		log.Printf("Bug in test: SelfServiceGetLogin\nerror: %s", err.Error())
	}
	w.Write(jsonResp)
	return
}

func Oauth2AuthRequestLoginAcceptHandler(w http.ResponseWriter, r *http.Request) {
	login := hydra_client.NewAcceptOAuth2LoginRequestWithDefaults()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Bug in test: Oauth2AuthRequestLoginAcceptHandler\nerror: %s", err.Error())
	}
	if err = json.Unmarshal(data, login); err != nil {
		log.Printf("Bug in test: Oauth2AuthRequestLoginAcceptHandler\nerror: %s", err.Error())
	}
	if login.Subject != OAUTH2_SUBJECT {
		w.WriteHeader(TEST_ERROR_CODE)
		return
	}
	response := hydra_client.NewOAuth2RedirectTo(AUTHORIZATION_REDIRECT)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	jsonResp, err := json.Marshal(response)
	if err != nil {
		log.Printf("Bug in test: Oauth2AuthRequestLoginAcceptHandler\nerror: %s", err.Error())
	}
	w.Write(jsonResp)
	return
}

func SessionWhoAmIHandler(w http.ResponseWriter, r *http.Request) {
	traits := IdentityTraits{Name: TRAITS_NAME}
	identity := kratos_client.NewIdentity(IDENTITY_ID, SCHEMA_ID, schemaServerURL, traits)
	response := kratos_client.NewSession(SESSION_ID)
	response.SetIdentity(*identity)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	jsonResp, err := json.Marshal(response)
	if err != nil {
		log.Printf("Bug in test: SessionWhoAmIHandler\nerror: %s", err.Error())
	}
	w.Write(jsonResp)
	return
}

func SelfServiceLoginHandler(w http.ResponseWriter, r *http.Request) {
	traits := IdentityTraits{Name: TRAITS_NAME}
	identity := kratos_client.NewIdentity(IDENTITY_ID, SCHEMA_ID, schemaServerURL, traits)
	session := kratos_client.NewSession(SESSION_ID)
	response := kratos_client.NewSuccessfulNativeLogin(*session)
	response.Session.SetIdentity(*identity)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	jsonResp, err := json.Marshal(response)
	if err != nil {
		log.Printf("Bug in test: SelfServiceLoginHandler\nerror: %s", err.Error())
	}
	w.Write(jsonResp)
	return
}

func SelfServiceErrorsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := q.Get("id")
	e := TestError{
		Code:    400,
		Status:  ERROR_STATUS,
		Reason:  ERROR_REASON,
		Message: ERROR_MESSAGE,
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
		log.Printf("Bug in test: SelfServiceErrorsHandler\nerror: %s", err.Error())
	}
	w.Write(response)
	return
}
func Oauth2AuthRequestConsentHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	challange := q.Get("consent_challenge")
	if challange != CONSENT_CHALLANGE {
		w.WriteHeader(TEST_ERROR_CODE)
		return
	}
	response := hydra_client.NewOAuth2ConsentRequest(AUTHORIZATION_CHALLANGE)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	jsonResp, err := json.Marshal(*response)
	if err != nil {
		log.Printf("Bug in test: Oauth2AuthRequestConsentHandler\nerror: %s", err.Error())
	}
	w.Write(jsonResp)
	return
}
func Oauth2AuthRequestConsentAcceptHandler(w http.ResponseWriter, r *http.Request) {
	response := hydra_client.NewOAuth2RedirectTo(CONSENT_REDIRECT)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	jsonResp, err := json.Marshal(*response)
	if err != nil {
		log.Printf("Bug in test: Oauth2AuthRequestConsentHandler\nerror: %s", err.Error())
	}
	w.Write(jsonResp)
	return
}

func CreateHandlerWithError(testname string) func(w http.ResponseWriter, r *http.Request) {
	e := GenericErrorConstructor(testname)
	ret := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(int(e.Code))
		jsonResp, err := json.Marshal(e)
		if err != nil {
			log.Printf("Bug in test: %s\nerror: %s", testname, err.Error())
		}
		w.Write(jsonResp)
		return
	}
	return ret
}

func TimeoutHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusGatewayTimeout)
	return
}

func GetOKStatus(w http.ResponseWriter, r *http.Request) {
	status := Status{
		Status: "ok",
	}
	jsonResp, err := json.Marshal(status)
	if err != nil {
		log.Printf("Bug in test handler: GetOKStatus\nerror: %s", err.Error())
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(jsonResp)
	return
}

func GetErrorStatus(w http.ResponseWriter, r *http.Request) {
	status := Status{
		Status: http.StatusText(503),
	}
	jsonResp, err := json.Marshal(status)
	if err != nil {
		log.Printf("Bug in test handler: GetOKStatus\nerror: %s", err.Error())
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(503)
	w.Write(jsonResp)
	return
}

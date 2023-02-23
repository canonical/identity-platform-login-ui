package LoginBrowser

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type LoginBrowserResponse struct {
	Id          string `json:"id"`
	Expires_at  string `json:"expires_at"`
	Issued_at   string `json:"issues_at"`
	Request_url string `json:"request_url"`
	Type        string `json:"type"`
}

func SelfServiceLoginBrowserHandler(w http.ResponseWriter, r *http.Request) {
	response := LoginBrowserResponse{
		Id:          "test_id",
		Expires_at:  time.Now().Format("2006-01-02T15:04:05Z07:00"),
		Issued_at:   time.Now().Format("2006-01-02T15:04:05Z07:00"),
		Request_url: r.URL.Path,
		Type:        "browser",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	jsonResp, err := json.Marshal(response)
	if err != nil {
		log.Printf("Bug in test: SelfServiceLoginBrowserHandler\nerror: %s", err.Error())
	}
	w.Write(jsonResp)
	return
}

type LoginBody struct {
	Method   string `json:"method"`
	Provider string `json:"provider"`
}

type OAuth2RequestLoginBody struct {
	Acr                      string   `json:"acr"`
	Amr                      []string `json:"amr"`
	Force_subject_identifier string   `json:"force_subject_identifier"`
	Remember                 bool     `json:"remember"`
	Remember_for             int64    `json:"remember_for"`
	Subject                  string   `json:"subject"`
}

type OAuth2RequestLoginResponse struct {
	Redirect_to string `json:"redirect_to"`
}

type OAuth2RequestSubject struct {
	Subject string `json:"subject"`
}

func Oauth2AuthRequestLoginAcceptHandler(w http.ResponseWriter, r *http.Request) {
	login := new(OAuth2RequestLoginBody)
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Bug in test: Oauth2AuthRequestLoginAcceptHandler\nerror: %s", err.Error())
	}
	r.ParseForm()
	log.Printf("form: %s\n\n", r.Form.Get("subject"))
	log.Printf("data: %s\n\n", string(data))
	if err = json.Unmarshal(data, login); err != nil {
		log.Printf("Bug in test: Oauth2AuthRequestLoginAcceptHandler\nerror: %s", err.Error())
	}
	if login.Subject != "test1234" {
		w.WriteHeader(599)
		return
	}
	response := OAuth2RequestLoginResponse{
		Redirect_to: "test.test",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	jsonResp, err := json.Marshal(response)
	if err != nil {
		log.Printf("Bug in test: Oauth2AuthRequestLoginAcceptHandler\nerror: %s", err.Error())
	}
	w.Write(jsonResp)
	return
}

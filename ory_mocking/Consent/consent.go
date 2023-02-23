package Consent

import (
	"encoding/json"
	"log"
	"net/http"
)

type OAuth2ConsentResponse struct {
	Challenge string `json:"challenge"`
}

type OAuth2ConsentAcceptResponse struct {
	Redirect_to string `json:"redirect_to"`
}

func Oauth2AuthRequestConsentHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	challange := q.Get("consent_challenge")
	if challange != "test_challange" {
		w.WriteHeader(599)
		return
	}
	response := OAuth2ConsentResponse{
		Challenge: "test_authorization_challenge",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	jsonResp, err := json.Marshal(response)
	if err != nil {
		log.Printf("Bug in test: Oauth2AuthRequestConsentHandler\nerror: %s", err.Error())
	}
	w.Write(jsonResp)
	return

}
func Oauth2AuthRequestConsentAcceptHandler(w http.ResponseWriter, r *http.Request) {
	response := OAuth2ConsentAcceptResponse{
		Redirect_to: "test.test",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	jsonResp, err := json.Marshal(response)
	if err != nil {
		log.Printf("Bug in test: Oauth2AuthRequestConsentHandler\nerror: %s", err.Error())
	}
	w.Write(jsonResp)
	return

}

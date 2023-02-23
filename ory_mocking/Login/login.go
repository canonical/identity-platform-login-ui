package Login

import (
	"encoding/json"
	testSession "identity_platform_login_ui/ory_mocking/Session"
	"log"
	"net/http"
)

var schemaServerURL string

func SetSchemaServerURL(url string) {
	schemaServerURL = url
}

type LoginUpdateResponse struct {
	Session       testSession.SessionResponse `json:"session"`
	Session_token string                      `json:"session_token"`
}

func SelfServiceLoginHandler(w http.ResponseWriter, r *http.Request) {
	traits := testSession.IdentityTraits{Name: "TestName"}
	identity := testSession.SessionIdentity{
		Id:        "test1234",
		SchemaId:  "test_identity.schema.json",
		SchemaUrl: schemaServerURL,
		Traits:    traits,
	}
	session := testSession.SessionResponse{
		Id:       "test-1111",
		Identity: identity,
	}
	response := LoginUpdateResponse{
		Session:       session,
		Session_token: "test-token",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	jsonResp, err := json.Marshal(response)
	if err != nil {
		log.Printf("Bug in test: SelfServiceLoginHandler\nerror: %s", err.Error())
	}
	w.Write(jsonResp)
	return
}

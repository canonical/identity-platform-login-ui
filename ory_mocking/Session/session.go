package Session

import (
	"encoding/json"
	"log"
	"net/http"
)

var schemaServerURL string

func SetSchemaServerURL(url string) {
	schemaServerURL = url
}

type SessionResponse struct {
	Id       string          `json:"id"`
	Identity SessionIdentity `json:"identity"`
}
type SessionIdentity struct {
	Id        string         `json:"id"`
	SchemaId  string         `json:"schema_id"`
	SchemaUrl string         `json:"schema_url"`
	Traits    IdentityTraits `json:"traits"`
}
type IdentityTraits struct {
	Name string `json:"name"`
}

func SessionWhoAmIHandler(w http.ResponseWriter, r *http.Request) {
	traits := IdentityTraits{Name: "TestName"}
	identity := SessionIdentity{
		Id:        "test1234",
		SchemaId:  "test_identity.schema.json",
		SchemaUrl: schemaServerURL,
		Traits:    traits,
	}
	response := SessionResponse{
		Id:       "test-1111",
		Identity: identity,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	jsonResp, err := json.Marshal(response)
	if err != nil {
		log.Printf("Bug in test: SessionWhoAmIHandler\nerror: %s", err.Error())
	}
	w.Write(jsonResp)
	return
}

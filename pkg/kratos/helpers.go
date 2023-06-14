package kratos

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	hydra_client "github.com/ory/hydra-client-go/v2"
	kratos_client "github.com/ory/kratos-client-go"

	oidc "github.com/canonical/identity_platform_login_ui/pkg/oidc"
)

// dump these in here for now

func getBaseURL(r *http.Request) string {
	if url := os.Getenv("BASE_URL"); url != "" {
		return url
	}
	return fmt.Sprintf("%s://%s/%s", r.URL.Scheme, r.Host, r.URL.Path)
}

func writeResponse(w http.ResponseWriter, r *http.Response) {
	for k, vs := range r.Header {
		for _, v := range vs {
			w.Header().Set(k, v)
		}
	}
	// We need to set the headers before setting the status code, otherwise
	// the response writer freaks out
	w.WriteHeader(r.StatusCode)
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}
	fmt.Fprint(w, string(body))
}

func cookiesToString(cookies []*http.Cookie) string {
	var ret []string
	ret = make([]string, len(cookies))
	for i, c := range cookies {
		ret[i] = fmt.Sprintf("%s=%s", c.Name, c.Value)
	}
	return strings.Join(ret, "; ")
}

func parseBody(r *http.Request, body interface{}) *interface{} {
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(body)
	if err != nil {
		log.Println(err)
	}
	return &body
}

func getUserClaims(i kratos_client.Identity, cr hydra_client.OAuth2ConsentRequest) map[string]interface{} {
	ret := make(map[string]interface{})
	// Export the user claims and filter them based on the requested scopes
	traits, ok := i.Traits.(map[string]interface{})
	if !ok {
		// We should never end up here
		log.Printf("Unexpected traits format: %v\n", ok)
	}
	log.Println(traits)
	for _, s := range cr.RequestedScope {
		cs, ok := oidc.OIDCScopeMapping[s]
		if !ok {
			continue
		}
		log.Println(cs)
		log.Println(s)
		for _, c := range cs {
			val, ok := traits[c]
			if ok {
				ret[c] = val
			}
		}
	}

	return ret
}

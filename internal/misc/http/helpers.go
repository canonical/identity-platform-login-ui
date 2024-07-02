package http

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	hydra_client "github.com/ory/hydra-client-go/v2"
	kratos_client "github.com/ory/kratos-client-go"

	oidc "github.com/canonical/identity-platform-login-ui/pkg/oidc"
)

// dump these in here for now

func CookiesToString(cookies []*http.Cookie) string {
	var ret = make([]string, len(cookies))
	for i, c := range cookies {
		ret[i] = fmt.Sprintf("%s=%s", c.Name, c.Value)
	}
	return strings.Join(ret, "; ")
}

func GetUserClaims(i kratos_client.Identity, cr hydra_client.OAuth2ConsentRequest) map[string]interface{} {
	ret := make(map[string]interface{})
	// Export the user claims and filter them based on the requested scopes
	traits, ok := i.Traits.(map[string]interface{})
	if !ok {
		// We should never end up here
		log.Printf("Unexpected traits format: %v\n", ok)
	}
	for _, s := range cr.RequestedScope {
		cs, ok := oidc.OIDCScopeMapping[s]
		if !ok {
			continue
		}
		for _, c := range cs {
			val, ok := traits[c]
			if ok {
				ret[c] = val
			}
		}
	}

	return ret
}

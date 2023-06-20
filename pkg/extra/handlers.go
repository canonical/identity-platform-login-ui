package extra

import (
	"context"
	"net/http"

	"github.com/canonical/identity_platform_login_ui/internal/logging"
	hydra_client "github.com/ory/hydra-client-go/v2"

	misc "github.com/canonical/identity_platform_login_ui/internal/misc/http"
)

type API struct {
	kratos KratosClientInterface
	hydra  HydraClientInterface

	logger logging.LoggerInterface
}

func (a *API) RegisterEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/api/consent", a.handleConsent)
}

// TODO: Validate response when server error handling is implemented
func (a *API) handleConsent(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// Get the Kratos session to make sure that the user is actually logged in
	session, session_resp, e := a.kratos.FrontendApi().ToSession(context.Background()).
		Cookie(misc.CookiesToString(r.Cookies())).
		Execute()
	if e != nil {
		a.logger.Errorf("Error when calling `FrontendApi.ToSession`: %v\n", e)
		a.logger.Errorf("Full HTTP response: %v\n", session_resp)
		return
	}

	// Get the consent request
	consent, consent_resp, e := a.hydra.OAuth2Api().GetOAuth2ConsentRequest(context.Background()).
		ConsentChallenge(q.Get("consent_challenge")).
		Execute()
	if e != nil {
		a.logger.Errorf("Error when calling `AdminApi.GetConsentRequest`: %v\n", e)
		a.logger.Errorf("Full HTTP response: %v\n", consent_resp)
		return
	}

	consent_session := hydra_client.NewAcceptOAuth2ConsentRequestSession()
	consent_session.SetIdToken(misc.GetUserClaims(session.Identity, *consent))
	accept_consent_req := hydra_client.NewAcceptOAuth2ConsentRequest()
	accept_consent_req.SetGrantScope(consent.RequestedScope)
	accept_consent_req.SetGrantAccessTokenAudience(consent.RequestedAccessTokenAudience)
	accept_consent_req.SetSession(*consent_session)
	accept, accept_resp, e := a.hydra.OAuth2Api().AcceptOAuth2ConsentRequest(context.Background()).
		ConsentChallenge(q.Get("consent_challenge")).
		AcceptOAuth2ConsentRequest(*accept_consent_req).
		Execute()
	if e != nil {
		a.logger.Errorf("Error when calling `AdminApi.AcceptConsentRequest`: %v\n", e)
		a.logger.Errorf("Full HTTP response: %v\n", accept_resp)
		return
	}

	resp, e := accept.MarshalJSON()
	if e != nil {
		a.logger.Errorf("Error when marshalling Json: %v\n", e)
		return
	}
	w.WriteHeader(200)
	w.Write(resp)

	return
}

func NewAPI(kratos KratosClientInterface, hydra HydraClientInterface, logger logging.LoggerInterface) *API {
	a := new(API)

	a.kratos = kratos
	a.hydra = hydra

	a.logger = logger

	return a
}

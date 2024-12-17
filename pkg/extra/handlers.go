package extra

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	client "github.com/ory/kratos-client-go"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	httpHelpers "github.com/canonical/identity-platform-login-ui/internal/misc/http"
	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

type API struct {
	service ServiceInterface
	kratos  kratos.ServiceInterface

	logger                        logging.LoggerInterface
	baseURL                       string
	oidcWebAuthnSequencingEnabled bool
}

func (a *API) RegisterEndpoints(mux *chi.Mux) {
	mux.Get("/api/consent", a.handleConsent)
}

// TODO: Validate response when server error handling is implemented
func (a *API) handleConsent(w http.ResponseWriter, r *http.Request) {
	session, _, err := a.kratos.CheckSession(r.Context(), r.Cookies())

	if err != nil {
		a.logger.Errorf("error when calling kratos: %s", err)
		// TODO @shipperizer evaluate return status
		w.WriteHeader(http.StatusForbidden)

		return
	}

	if a.oidcWebAuthnSequencingEnabled {
		// enforce webauthn setup
		shouldEnforceWebAuthn, err := a.shouldEnforceWebAuthnWithSession(r.Context(), session)
		if err != nil {
			err = fmt.Errorf("webauthn enforce check error: %v", err)
			a.logger.Error(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if shouldEnforceWebAuthn {
			a.webAuthnSettingsRedirect(w, r.Referer())
			return
		}
	}

	// Get the consent request
	consent, err := a.service.GetConsent(r.Context(), r.URL.Query().Get("consent_challenge"))

	if err != nil {
		a.logger.Errorf("error when calling hydra: %s", err)
		// TODO @shipperizer evaluate return status
		w.WriteHeader(http.StatusForbidden)
		return
	}

	accept, err := a.service.AcceptConsent(r.Context(), *session.Identity, consent)
	if err != nil {
		a.logger.Errorf("error when calling hydra: %s", err)
		// TODO @shipperizer evaluate return status
		w.WriteHeader(http.StatusForbidden)
		return
	}

	rr, err := accept.MarshalJSON()
	if err != nil {
		a.logger.Errorf("error when marshalling json: %s", err)
		// TODO @shipperizer evaluate return status
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(rr)
	w.WriteHeader(http.StatusOK)

}

func (a *API) shouldEnforceWebAuthnWithSession(ctx context.Context, session *client.Session) (bool, error) {
	// enforce only if authenticated with OIDC external provider as 1fa
	for _, method := range session.AuthenticationMethods {
		if method.Method != nil && *method.Method != "oidc" {
			return false, nil
		}
	}

	webAuthnAvailable, err := a.kratos.HasWebAuthnAvailable(ctx, session.Identity.GetId())
	if err != nil {
		return false, err
	}

	return !webAuthnAvailable, nil
}

func (a *API) webAuthnSettingsRedirect(w http.ResponseWriter, returnTo string) {
	redirect, err := url.JoinPath(a.baseURL, "/ui/setup_passkey")
	if err != nil {
		err = fmt.Errorf("unable to build webauthn redirect path, possible misconfiguration, err: %v", err)
		a.logger.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	errorId := "session_aal2_required"

	// Set the original consent URL as return_to, to continue the flow after WebAuthn key is set
	r, _ := httpHelpers.AddParamsToURL(redirect, httpHelpers.QueryParam{Name: "return_to", Value: returnTo})

	w.WriteHeader(http.StatusSeeOther)
	_ = json.NewEncoder(w).Encode(
		kratos.ErrorBrowserLocationChangeRequired{
			Error:             &client.GenericError{Id: &errorId},
			RedirectBrowserTo: &r,
		},
	)
}

func NewAPI(service ServiceInterface, kratos kratos.ServiceInterface, logger logging.LoggerInterface, baseURL string, oidcWebAuthnSequencingEnabled bool) *API {
	a := new(API)

	a.service = service
	a.kratos = kratos

	a.logger = logger

	a.baseURL = baseURL
	a.oidcWebAuthnSequencingEnabled = oidcWebAuthnSequencingEnabled

	return a
}

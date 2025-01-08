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
	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

type API struct {
	service ServiceInterface
	kratos  kratos.ServiceInterface

	baseURL                       string
	oidcWebAuthnSequencingEnabled bool
	contextPath                   string
	logger                        logging.LoggerInterface
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

	consentChallenge := r.URL.Query().Get("consent_challenge")
	if consentChallenge == "" {
		err = fmt.Errorf("no consent challenge present")
		a.logger.Errorf(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
			returnTo, _ := url.JoinPath("/", a.contextPath, "/ui/consent")
			returnToConsent, err := url.ParseRequestURI(returnTo)
			if err != nil {
				return
			}

			q := returnToConsent.Query()
			q.Set("consent_challenge", consentChallenge)
			returnToConsent.RawQuery = q.Encode()
			err = a.webAuthnSettingsRedirect(w, returnToConsent.String())
			if err != nil {
				err = fmt.Errorf("unable to build webauthn redirect path, possible misconfiguration, err: %v", err)
				a.logger.Error(err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}

			return
		}

		if session.GetAuthenticatorAssuranceLevel() == "aal1" {
			err = fmt.Errorf("webauthn step was skipped, user has not completed 2fa")
			a.logger.Error(err.Error())
			http.Error(w, err.Error(), http.StatusForbidden)
		}
	}

	// Get the consent request
	consent, err := a.service.GetConsent(r.Context(), consentChallenge)

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
	// enforce only if one of the authentication methods was oidc
	for _, method := range session.AuthenticationMethods {
		if method.GetMethod() == "oidc" {
			webAuthnAvailable, err := a.kratos.HasWebAuthnAvailable(ctx, session.Identity.GetId())
			if err != nil {
				return false, err
			}
			return !webAuthnAvailable, nil
		}
	}
	return false, nil
}

func (a *API) webAuthnSettingsRedirect(w http.ResponseWriter, returnTo string) error {
	redirect, err := url.JoinPath("/", a.contextPath, "/ui/setup_passkey")
	if err != nil {
		return err
	}

	errorId := "session_aal2_required"

	// Set the original consent URL as return_to, to continue the flow after WebAuthn key is set
	redirectTo, err := url.ParseRequestURI(redirect)
	if err != nil {
		return err
	}

	q := redirectTo.Query()
	q.Set("return_to", returnTo)
	redirectTo.RawQuery = q.Encode()
	redirectPath := redirectTo.String()

	w.WriteHeader(http.StatusSeeOther)
	_ = json.NewEncoder(w).Encode(
		kratos.ErrorBrowserLocationChangeRequired{
			Error:             &client.GenericError{Id: &errorId},
			RedirectBrowserTo: &redirectPath,
		},
	)
	return nil
}

func NewAPI(service ServiceInterface, kratos kratos.ServiceInterface, baseURL string, oidcWebAuthnSequencingEnabled bool, logger logging.LoggerInterface) *API {
	a := new(API)

	a.service = service
	a.kratos = kratos

	a.logger = logger

	a.baseURL = baseURL
	a.oidcWebAuthnSequencingEnabled = oidcWebAuthnSequencingEnabled

	fullBaseURL, err := url.Parse(baseURL)
	if err != nil {
		// this should never happen if app is configured properly
		a.logger.Fatalf("Failed to construct API base URL: %v\n", err)
	}
	a.contextPath = fullBaseURL.Path

	return a
}

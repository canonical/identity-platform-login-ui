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
	mfaEnabled  bool
	contextPath string
	service     ServiceInterface
	kratos      kratos.ServiceInterface

	logger logging.LoggerInterface
}

type ConsentRedirectResponse struct {
	RedirectTo string `json:"redirect_to,omitempty"`
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

	// Get the consent request
	consent, err := a.service.GetConsent(r.Context(), r.URL.Query().Get("consent_challenge"))

	if err != nil {
		a.logger.Errorf("error when calling hydra: %s", err)
		// TODO @shipperizer evaluate return status
		w.WriteHeader(http.StatusForbidden)
		return
	}

	shouldEnforceMfa, err := a.shouldEnforceMFA(r.Context(), session)
	if err != nil {
		err = fmt.Errorf("enforce check error: %v", err)
		a.logger.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if shouldEnforceMfa {
		a.mfaSettingsRedirect(w)
		return
	}

	accept, err := a.service.AcceptConsent(r.Context(), session.Identity, consent)
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

func (a *API) shouldEnforceMFA(ctx context.Context, session *client.Session) (bool, error) {
	if !a.mfaEnabled {
		return false, nil
	}

	// if using OIDC external provider, do not enforce MFA
	for _, method := range session.AuthenticationMethods {
		if method.Method != nil && *method.Method == "oidc" {
			return false, nil
		}
	}

	totpAvailable, err := a.kratos.HasTOTPAvailable(ctx, session.Identity.GetId())
	if err != nil {
		return false, err
	}

	return !totpAvailable, nil
}

func (a *API) mfaSettingsRedirect(w http.ResponseWriter) {
	redirect, err := url.JoinPath("/", a.contextPath, "/ui/setup_secure")
	if err != nil {
		err = fmt.Errorf("unable to build mfa redirect path, possible misconfiguration, err: %v", err)
		a.logger.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(
		ConsentRedirectResponse{
			RedirectTo: redirect,
		},
	)
}

func NewAPI(mfaEnabled bool, baseURL string, service ServiceInterface, kratos kratos.ServiceInterface, logger logging.LoggerInterface) *API {
	a := new(API)

	a.mfaEnabled = mfaEnabled

	fullBaseURL, err := url.Parse(baseURL)
	if err != nil {
		// this should never happen if app is configured properly
		a.logger.Fatalf("Failed to construct API base URL: %v\n", err)
	}

	a.contextPath = fullBaseURL.Path

	a.service = service
	a.kratos = kratos

	a.logger = logger

	return a
}

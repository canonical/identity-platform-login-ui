package extra

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go/v25"
)

type API struct {
	service ServiceInterface
	kratos  kratos.ServiceInterface

	baseURL                       string
	oidcWebAuthnSequencingEnabled bool
	mfaEnabled                    bool
	contextPath                   string
	tracer                        tracing.TracingInterface
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

	if session.GetAuthenticatorAssuranceLevel() < a.sessionRequiredAAL(session) {
		a.logger.Errorf("insufficient session aal, this indicates a misconfiguration in kratos")
		http.Error(w, "insufficient session aal", http.StatusForbidden)
		return
	}

	consentChallenge := r.URL.Query().Get("consent_challenge")
	if consentChallenge == "" {
		err = fmt.Errorf("no consent challenge present")
		a.logger.Errorf(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get the consent request
	consent, err := a.service.GetConsent(r.Context(), consentChallenge)

	if err != nil {
		a.logger.Errorf("error when calling hydra: %s", err)
		// TODO @shipperizer evaluate return status
		w.WriteHeader(http.StatusForbidden)
		return
	}

	tenantID := a.resolveTenantID(consent)

	accept, err := a.service.AcceptConsent(r.Context(), *session.Identity, consent, tenantID)
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

// resolveTenantID returns the tenant_id to embed in the token.
// It reads the value from the Hydra login context, which was set when
// the login UI called AcceptLoginRequest with the tenant_id.
func (a *API) resolveTenantID(consent *hClient.OAuth2ConsentRequest) string {
	if ctx, ok := consent.GetContext().(map[string]interface{}); ok {
		if v, ok := ctx["tenant_id"].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// sessionRequiredAAL returns the required aal, based on the session's authentication methods.
func (a *API) sessionRequiredAAL(session *kClient.Session) kClient.AuthenticatorAssuranceLevel {
	var authMethod string
	ret := kClient.AUTHENTICATORASSURANCELEVEL_AAL1

	if methods, ok := session.GetAuthenticationMethodsOk(); ok {
		authMethod = methods[0].GetMethod()
	}

	switch authMethod {
	case "oidc":
		if a.oidcWebAuthnSequencingEnabled {
			ret = kClient.AUTHENTICATORASSURANCELEVEL_AAL2
		}
	case "password", "webauthn":
		if a.mfaEnabled {
			ret = kClient.AUTHENTICATORASSURANCELEVEL_AAL2
		}
	}

	return ret
}

func NewAPI(service ServiceInterface, kratos kratos.ServiceInterface, baseURL string, mfaEnabled, oidcWebAuthnSequencingEnabled bool, tracer tracing.TracingInterface, logger logging.LoggerInterface) *API {
	a := new(API)

	a.service = service
	a.kratos = kratos

	a.logger = logger

	a.baseURL = baseURL
	a.oidcWebAuthnSequencingEnabled = oidcWebAuthnSequencingEnabled
	a.mfaEnabled = mfaEnabled

	fullBaseURL, err := url.Parse(baseURL)
	if err != nil {
		// this should never happen if app is configured properly
		a.logger.Fatalf("Failed to construct API base URL: %v\n", err)
	}
	a.contextPath = fullBaseURL.Path
	a.tracer = tracer

	return a
}

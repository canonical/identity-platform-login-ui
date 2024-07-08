package kratos

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
)

type API struct {
	service ServiceInterface
	baseURL string

	logger logging.LoggerInterface
}

func (a *API) RegisterEndpoints(mux *chi.Mux) {
	mux.Post("/api/kratos/self-service/login", a.handleUpdateFlow)
	mux.Get("/api/kratos/self-service/login/browser", a.handleCreateFlow)
	mux.Get("/api/kratos/self-service/login/flows", a.handleGetLoginFlow)
	mux.Get("/api/kratos/self-service/errors", a.handleKratosError)
	mux.Post("/api/kratos/self-service/recovery", a.handleUpdateRecoveryFlow)
	mux.Get("/api/kratos/self-service/recovery/browser", a.handleCreateRecoveryFlow)
	mux.Get("/api/kratos/self-service/recovery/flows", a.handleGetRecoveryFlow)
	mux.Post("/api/kratos/self-service/settings", a.handleUpdateSettingsFlow)
	mux.Get("/api/kratos/self-service/settings/browser", a.handleCreateSettingsFlow)
	mux.Get("/api/kratos/self-service/settings/flows", a.handleGetSettingsFlow)
}

// TODO: Validate response when server error handling is implemented
func (a *API) handleCreateFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	loginChallenge := q.Get("login_challenge")

	// We try to see if the user is logged in, because if they are the CreateBrowserLoginFlow
	// call will return an empty response
	// TODO: We need to send a different content-type to CreateBrowserLoginFlow in order
	// to avoid this bug.
	session, _, _ := a.service.CheckSession(context.Background(), r.Cookies())
	if session != nil {
		redirectTo, cookies, err := a.service.AcceptLoginRequest(context.Background(), session.Identity.Id, loginChallenge)
		if err != nil {
			a.logger.Errorf("Error when accepting login request: %v\n", err)
			http.Error(w, "Failed to accept login request", http.StatusInternalServerError)
			return
		}
		setCookies(w, cookies)
		resp, err := redirectTo.MarshalJSON()
		if err != nil {
			a.logger.Errorf("Error when marshalling Json: %v\n", err)
			http.Error(w, "Failed to marshall json", http.StatusInternalServerError)
			return
		}
		// The frontend will call this endpoint with an XHR request, so the status code is
		// not that important (the redirect happens based on the response body). But we still send
		// a redirect code response to be consistent with the hydra response.
		w.WriteHeader(http.StatusOK)
		w.Write(resp)
		return
	}

	refresh, err := strconv.ParseBool(q.Get("refresh"))
	if err == nil {
		refresh = false
	}

	returnTo, err := url.JoinPath(a.baseURL, "/ui/login")
	if err != nil {
		a.logger.Errorf("Failed to construct returnTo URL: %v", err)
	}
	returnTo = returnTo + "?login_challenge=" + loginChallenge

	// We redirect the user back to this endpoint with the login_challenge, after they log in, to bypass
	// Kratos bug where the user is not redirected to hydra the first time they log in.
	// Relevant issue https://github.com/ory/kratos/issues/3052
	flow, cookies, err := a.service.CreateBrowserLoginFlow(context.Background(), q.Get("aal"), returnTo, loginChallenge, refresh, r.Cookies())
	if err != nil {
		// TODO: Add more context
		http.Error(w, "Failed to create login flow", http.StatusInternalServerError)
		return
	}

	flow, err = a.service.FilterFlowProviderList(context.Background(), flow)
	if err != nil {
		a.logger.Errorf("Error when filtering providers: %v\n", err)
		http.Error(w, "Unexpected error", http.StatusInternalServerError)
		return
	}

	resp, err := flow.MarshalJSON()
	if err != nil {
		a.logger.Errorf("Error when marshalling Json: %v\n", err)
		http.Error(w, "Failed to marshall json", http.StatusInternalServerError)
		return
	}
	setCookies(w, cookies)
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

// TODO: Validate response when server error handling is implemented
func (a *API) handleGetLoginFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	flow, cookies, err := a.service.GetLoginFlow(context.Background(), q.Get("id"), r.Cookies())
	if err != nil {
		a.logger.Errorf("Error when getting login flow: %v\n", err)
		http.Error(w, "Failed to get login flow", http.StatusInternalServerError)
		return
	}

	resp, err := flow.MarshalJSON()
	if err != nil {
		a.logger.Errorf("Error when marshalling Json: %v\n", err)
		http.Error(w, "Failed to parse login flow", http.StatusInternalServerError)
		return
	}
	setCookies(w, cookies)
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

// TODO: Validate response when server error handling is implemented
func (a *API) handleUpdateFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	flowId := q.Get("flow")

	body, err := a.service.ParseLoginFlowMethodBody(r)
	if err != nil {
		a.logger.Errorf("Error when parsing request body: %v\n", err)
		http.Error(w, "Failed to parse login flow", http.StatusInternalServerError)
		return
	}

	loginFlow, _, err := a.service.GetLoginFlow(context.Background(), flowId, r.Cookies())
	if err != nil {
		a.logger.Errorf("Error when getting login flow: %v\n", err)
		http.Error(w, "Failed to get login flow", http.StatusInternalServerError)
		return
	}

	allowed, err := a.service.CheckAllowedProvider(context.Background(), loginFlow, body)
	if err != nil {
		a.logger.Errorf("Error when authorizing provider: %v\n", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, "Provider not allowed", http.StatusForbidden)
		return
	}

	flow, cookies, err := a.service.UpdateLoginFlow(context.Background(), flowId, *body, r.Cookies())
	if err != nil {
		a.logger.Errorf("Error when updating login flow: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(flow)
	if err != nil {
		a.logger.Errorf("Error when marshalling Json: %v\n", err)
		http.Error(w, "Failed to parse login flow", http.StatusInternalServerError)
		return
	}
	setCookies(w, cookies)
	// Kratos returns us a '422' response but we tranform it to a '200',
	// because this is the expected behavior for us.
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

// TODO: Validate response when server error handling is implemented
func (a *API) handleKratosError(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := q.Get("id")

	flowError, cookies, err := a.service.GetFlowError(context.Background(), id)
	if err != nil {
		a.logger.Errorf("Error when getting flow error: %v\n", err)
		http.Error(w, "Failed to get flow error", http.StatusInternalServerError)
		return
	}

	resp, err := flowError.MarshalJSON()
	if err != nil {
		a.logger.Errorf("Error when marshalling Json: %v\n", err)
		http.Error(w, "Failed to parse flow error", http.StatusInternalServerError)
		return
	}
	setCookies(w, cookies)
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (a *API) handleGetRecoveryFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	flow, cookies, err := a.service.GetRecoveryFlow(context.Background(), q.Get("id"), r.Cookies())
	if err != nil {
		a.logger.Errorf("Error when getting recovery flow: %v\n", err)
		http.Error(w, "Failed to get recovery flow", http.StatusInternalServerError)
		return
	}

	resp, err := flow.MarshalJSON()
	if err != nil {
		a.logger.Errorf("Error when marshalling json: %v\n", err)
		http.Error(w, "Failed to parse recovery flow", http.StatusInternalServerError)
		return
	}
	setCookies(w, cookies)
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (a *API) handleUpdateRecoveryFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	flowId := q.Get("flow")

	body, err := a.service.ParseRecoveryFlowMethodBody(r)
	if err != nil {
		a.logger.Errorf("Error when parsing request body: %v\n", err)
		http.Error(w, "Failed to parse recovery flow", http.StatusInternalServerError)
		return
	}

	flow, cookies, err := a.service.UpdateRecoveryFlow(r.Context(), flowId, *body, r.Cookies())
	if err != nil {
		a.logger.Errorf("Error when updating recovery flow: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if flow.HasNoRedirectTo() && flow.HasError() {
		a.logger.Errorf("Error when updating recovery flow: %v\n", flow)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(flow)
		return
	}

	setCookies(w, cookies)
	// Kratos '422' response maps to out 200 OK, it is expected
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(
		BrowserLocationChangeRequired{
			RedirectTo: flow.RedirectBrowserTo,
		},
	)
}

func (a *API) handleCreateRecoveryFlow(w http.ResponseWriter, r *http.Request) {
	returnTo, err := url.JoinPath(a.baseURL, "/ui/reset_email")
	if err != nil {
		a.logger.Errorf("Failed to construct returnTo URL: ", err)
		http.Error(w, "Failed to construct returnTo URL", http.StatusBadRequest)
	}

	flow, cookies, err := a.service.CreateBrowserRecoveryFlow(context.Background(), returnTo, r.Cookies())
	if err != nil {
		a.logger.Errorf("Failed to create recovery flow: %v\n", err)
		http.Error(w, "Failed to create recovery flow", http.StatusInternalServerError)
		return
	}

	resp, err := flow.MarshalJSON()
	if err != nil {
		a.logger.Errorf("Error when marshalling json: %v\n", err)
		http.Error(w, "Failed to marshal json", http.StatusInternalServerError)
		return
	}

	setCookies(w, cookies)
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (a *API) handleGetSettingsFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	flow, response, err := a.service.GetSettingsFlow(context.Background(), q.Get("id"), r.Cookies())
	if err != nil {
		a.logger.Errorf("Error when getting settings flow: %v\n", err)
		http.Error(w, "Failed to get settings flow", http.StatusInternalServerError)
		return
	}

	// If aal1, redirect to complete second factor auth
	if response.StatusCode == http.StatusForbidden {
		redirectResp := new(ErrorBrowserLocationChangeRequired)
		err = unmarshalByteJson(response.Body, redirectResp)
		if err != nil {
			a.logger.Errorf("Failed to unmarshal JSON: %s", err)
			return
		}

		if !redirectResp.HasNoRedirectTo() && redirectResp.HasError() {
			a.logger.Errorf("Error when updating settings flow: %v\n", flow)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(redirectResp)
			return
		}

	}

	resp, err := flow.MarshalJSON()
	if err != nil {
		a.logger.Errorf("Error when marshalling json: %v\n", err)
		http.Error(w, "Failed to parse settings flow", http.StatusInternalServerError)
		return
	}

	setCookies(w, response.Cookies())
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (a *API) handleUpdateSettingsFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	flowId := q.Get("flow")

	body, err := a.service.ParseSettingsFlowMethodBody(r)
	if err != nil {
		a.logger.Errorf("Error when parsing request body: %v\n", err)
		http.Error(w, "Failed to parse settings flow", http.StatusInternalServerError)
		return
	}

	flow, cookies, err := a.service.UpdateSettingsFlow(context.Background(), flowId, *body, r.Cookies())
	if err != nil {
		a.logger.Errorf("Error when updating settings flow: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(flow)
	if err != nil {
		a.logger.Errorf("Error when marshalling json: %v\n", err)
		http.Error(w, "Failed to parse settings flow", http.StatusInternalServerError)
		return
	}
	setCookies(w, cookies)
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (a *API) handleCreateSettingsFlow(w http.ResponseWriter, r *http.Request) {
	returnTo, err := url.JoinPath(a.baseURL, "/ui/setup_complete")
	if err != nil {
		a.logger.Errorf("Failed to construct returnTo URL: ", err)
		http.Error(w, "Failed to construct returnTo URL", http.StatusBadRequest)
	}

	flow, response, err := a.service.CreateBrowserSettingsFlow(context.Background(), returnTo, r.Cookies())
	if err != nil {
		a.logger.Errorf("Failed to create settings flow: %v", err)
		http.Error(w, "Failed to create settings flow", http.StatusInternalServerError)
		return
	}

	if response.StatusCode == http.StatusForbidden {
		redirectResp := new(ErrorBrowserLocationChangeRequired)
		err = unmarshalByteJson(response.Body, redirectResp)
		if err != nil {
			a.logger.Errorf("Failed to unmarshal JSON: %s", err)
			return
		}

		a.logger.Debugf("Response settings: %s", redirectResp)

		if !redirectResp.HasNoRedirectTo() && redirectResp.HasError() {
			a.logger.Errorf("Error when updating settings flow: %v\n", flow)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(redirectResp)
			return
		}
	}

	resp, err := flow.MarshalJSON()
	if err != nil {
		a.logger.Errorf("Error when marshalling json: %v\n", err)
		http.Error(w, "Failed to marshal json", http.StatusInternalServerError)
		return
	}
	setCookies(w, response.Cookies())
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func NewAPI(service ServiceInterface, baseURL string, logger logging.LoggerInterface) *API {
	a := new(API)

	a.service = service
	a.baseURL = baseURL

	a.logger = logger

	return a
}

func setCookies(w http.ResponseWriter, cookies []*http.Cookie) {
	for _, c := range cookies {
		http.SetCookie(w, c)
	}
}

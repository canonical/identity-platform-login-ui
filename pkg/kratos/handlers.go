package kratos

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/go-chi/chi/v5"
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
}

// TODO: Validate response when server error handling is implemented
func (a *API) handleCreateFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// We try to see if the user is logged in, because if they are the CreateBrowserLoginFlow
	// call will return an empty response
	// TODO: We need to send a different content-type to CreateBrowserLoginFlow in order
	// to avoid this bug.
	session, _, _ := a.service.CheckSession(context.Background(), r.Cookies())
	if session != nil {
		redirectTo, cookies, err := a.service.AcceptLoginRequest(context.Background(), session.Identity.Id, q.Get("login_challenge"))
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

	returnTo, err := url.JoinPath(a.baseURL, "/login")
	if err != nil {
		a.logger.Fatal("Failed to construct returnTo URL: ", err)
	}
	returnTo = returnTo + "?login_challenge=" + q.Get("login_challenge")

	// We redirect the user back to this endpoint with the login_challenge, after they log in, to bypass
	// Kratos bug where the user is not redirected to hydra the first time they log in.
	// Relevant issue https://github.com/ory/kratos/issues/3052
	flow, cookies, err := a.service.CreateBrowserLoginFlow(context.Background(), q.Get("aal"), returnTo, q.Get("login_challenge"), refresh, r.Cookies())
	if err != nil {
		// TODO: Add more context
		http.Error(w, "Failed to create login flow", http.StatusInternalServerError)
		return
	}

	resp, err := flow.MarshalJSON()
	if err != nil {
		a.logger.Errorf("Error when marshalling Json: %v\n", err)
		http.Error(w, "Failed to marshall json", http.StatusInternalServerError)
		return
	}
	setCookies(w, cookies)
	w.WriteHeader(200)
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
	w.WriteHeader(200)
	w.Write(resp)
}

// TODO: Validate response when server error handling is implemented
func (a *API) handleUpdateFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	body, err := a.service.ParseLoginFlowMethodBody(r)
	if err != nil {
		a.logger.Errorf("Error when parsing request body: %v\n", err)
		http.Error(w, "Failed to parse login flow", http.StatusInternalServerError)
		return
	}

	flow, cookies, err := a.service.UpdateOIDCLoginFlow(context.Background(), q.Get("flow"), *body, r.Cookies())
	if err != nil {
		a.logger.Errorf("Error when updating login flow: %v\n", err)
		http.Error(w, "Failed to update login flow", http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(flow)
	if err != nil {
		a.logger.Errorf("Error when marshalling Json: %v\n", err)
		http.Error(w, "Failed to parse login flow", http.StatusInternalServerError)
		return
	}
	setCookies(w, cookies)
	w.WriteHeader(422)
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
	w.WriteHeader(200)
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

func cookiesToString(cookies []*http.Cookie) string {
	var ret = make([]string, len(cookies))
	for i, c := range cookies {
		ret[i] = fmt.Sprintf("%s=%s", c.Name, c.Value)
	}
	return strings.Join(ret, "; ")
}

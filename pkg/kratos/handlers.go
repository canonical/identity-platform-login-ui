package kratos

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	client "github.com/ory/kratos-client-go"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/pkg/ui"
)

const RegenerateBackupCodesError = "regenerate_backup_codes"
const KRATOS_SESSION_COOKIE_NAME = "ory_kratos_session"
const LOGIN_UI_STATE_COOKIE = "login_ui_state"

type API struct {
	mfaEnabled    bool
	service       ServiceInterface
	baseURL       string
	contextPath   string
	cookieManager AuthCookieManagerInterface

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
	var (
		response         any
		shouldEnforceMfa = false
		cookies          []*http.Cookie
		err              error
	)

	q := r.URL.Query()

	loginChallenge := q.Get("login_challenge")
	returnTo := q.Get("return_to")
	aal := q.Get("aal")
	refresh, err := strconv.ParseBool(q.Get("refresh"))
	if err != nil {
		refresh = false
	}

	if returnTo == "" {
		if loginChallenge == "" {
			http.Error(w, "One of return_to or login_challenge must be provided", http.StatusBadRequest)
			return
		}
		returnTo, err = a.returnToUrl(loginChallenge)
		if err != nil {
			// this should never happen if app is properly configured
			a.logger.Errorf("Failed to construct returnTo URL: %v", err)
			http.Error(w, "Failed to construct returnTo URL", http.StatusInternalServerError)
			return
		}
	}

	// if the user is logged in, CreateBrowserLoginFlow call will return an empty response
	// TODO: We need to send a different content-type to CreateBrowserLoginFlow in order to avoid this bug.
	session, _, _ := a.service.CheckSession(r.Context(), r.Cookies())
	if session != nil && a.mfaEnabled {
		shouldEnforceMfa, err = a.shouldEnforceMFAWithSession(r.Context(), session)

		if err != nil {
			a.logger.Errorf("Failed check for MFA: %v", err)
			http.Error(w, "Failed check for MFA", http.StatusInternalServerError)
			return
		}
		if shouldEnforceMfa {
			a.mfaSettingsRedirect(w, returnTo)
			return
		}
	}

	c, err := a.cookieManager.GetStateCookie(r)
	if err != nil {
		a.logger.Errorf("Failed to parse state cookie: %v", err)
		http.Error(w, "Failed to parse state cookie", http.StatusInternalServerError)
		return
	}
	forceLogin, err := a.service.MustReAuthenticate(r.Context(), loginChallenge, session, c)
	if err != nil {
		a.logger.Errorf("Failed to fetch hydra flow: %v", err)
		http.Error(w, "Failed to fetch hydra flow", http.StatusInternalServerError)
		return
	}
	if !forceLogin {
		response, cookies, err = a.handleCreateFlowWithSession(w, r, session, loginChallenge)
	} else {
		if session != nil {
			refresh = true
		}
		flowCookie := FlowStateCookie{LoginChallengeHash: hash(loginChallenge), RequestedAt: strconv.FormatInt(time.Now().Unix(), 10)}
		a.cookieManager.SetStateCookie(w, flowCookie)
		response, cookies, err = a.handleCreateFlowNewSession(r, aal, returnTo, loginChallenge, refresh)
	}

	if err != nil {
		a.logger.Errorf(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	setCookies(w, cookies)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (a *API) handleCreateFlowNewSession(r *http.Request, aal string, returnTo string, loginChallenge string, refresh bool) (*client.LoginFlow, []*http.Cookie, error) {
	// redirect user to this endpoint with the login_challenge after login
	// see https://github.com/ory/kratos/issues/3052
	flow, cookies, err := a.service.CreateBrowserLoginFlow(
		r.Context(),
		aal,
		returnTo,
		loginChallenge,
		refresh,
		filterCookies(r.Cookies(), KRATOS_SESSION_COOKIE_NAME),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create login flow, err: %v", err)
	}

	flow, err = a.service.FilterFlowProviderList(r.Context(), flow)
	if err != nil {
		return nil, nil, fmt.Errorf("Error when filtering providers: %v\n", err)
	}

	return flow, cookies, nil
}

func (a *API) handleCreateFlowWithSession(w http.ResponseWriter, r *http.Request, session *client.Session, loginChallenge string) (any, []*http.Cookie, error) {
	response, cookies, err := a.service.AcceptLoginRequest(r.Context(), session, loginChallenge)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to accept login request: %v", err)
	}
	a.cookieManager.ClearStateCookie(w)
	return response, cookies, nil
}

func (a *API) returnToUrl(loginChallenge string) (string, error) {
	returnTo, err := url.JoinPath(a.baseURL, "/ui/login")
	if err != nil {
		return "", err
	}

	// url.JoinPath already performed this operation, if we get here we're good
	if loginChallenge != "" {
		redirectTo, err := url.ParseRequestURI(returnTo)
		if err != nil {
			return "", err
		}

		q := redirectTo.Query()
		q.Set("login_challenge", loginChallenge)
		redirectTo.RawQuery = q.Encode()
		r := redirectTo.String()
		return r, nil
	}

	return returnTo, nil
}

// TODO: Validate response when server error handling is implemented
func (a *API) handleGetLoginFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	flowId := q.Get("id")
	if flowId == "" {
		a.logger.Errorf("mandatory param id is not present")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode("mandatory param id is not present")
		return
	}

	flow, cookies, err := a.service.GetLoginFlow(r.Context(), flowId, r.Cookies())
	if err != nil {
		a.logger.Errorf("Error when getting login flow: %v\n", err)
		http.Error(w, "Failed to get login flow", http.StatusInternalServerError)
		return
	}

	setCookies(w, cookies)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(flow)
}

// TODO: Validate response when server error handling is implemented
func (a *API) handleUpdateFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	flowId := q.Get("flow")

	body, cookies, err := a.service.ParseLoginFlowMethodBody(r)
	if err != nil {
		a.logger.Errorf("Error when parsing request body: %v\n", err)
		http.Error(w, "Failed to parse login flow", http.StatusInternalServerError)
		return
	}

	loginFlow, _, err := a.service.GetLoginFlow(r.Context(), flowId, r.Cookies())
	if err != nil {
		a.logger.Errorf("Error when getting login flow: %v\n", err)
		http.Error(w, "Failed to get login flow", http.StatusInternalServerError)
		return
	}

	allowed, err := a.service.CheckAllowedProvider(r.Context(), loginFlow, body)
	if err != nil {
		a.logger.Errorf("Error when authorizing provider: %v\n", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	if !allowed {
		http.Error(w, "Provider not allowed", http.StatusForbidden)
		return
	}

	redirectTo, flow, cookies, err := a.service.UpdateLoginFlow(r.Context(), flowId, *body, cookies)
	if err != nil {
		a.logger.Errorf("Error when updating login flow: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	shouldEnforceMfa, err := a.shouldEnforceMFA(r.Context(), cookies)
	if err != nil {
		err = fmt.Errorf("enforce check error: %v", err)
		a.logger.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	shouldRegenerateBackupCodes, err := a.shouldRegenerateBackupCodes(r.Context(), cookies)
	if err != nil {
		err = fmt.Errorf("error when checking backup codes: %v", err)

		a.logger.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	setCookies(w, cookies)

	if shouldRegenerateBackupCodes {
		a.lookupSecretsSettingsRedirect(w, flowId, *loginFlow.ReturnTo)
		return
	}

	if shouldEnforceMfa {
		a.mfaSettingsRedirect(w, *loginFlow.ReturnTo)
		return
	}

	w.WriteHeader(http.StatusOK)
	if redirectTo != nil {
		_ = json.NewEncoder(w).Encode(redirectTo)
	} else {
		u, _ := url.Parse(loginFlow.GetReturnTo())
		response, cookies, err := a.handleCreateFlowWithSession(w, r, &flow.Session, u.Query().Get("login_challenge"))
		if err != nil {
			a.logger.Errorf(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		setCookies(w, cookies)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}
}

func (a *API) shouldRegenerateBackupCodes(ctx context.Context, cookies []*http.Cookie) (bool, error) {
	// skip the check if mfa is not enabled
	if !a.mfaEnabled {
		return false, nil
	}

	session, _, err := a.service.CheckSession(ctx, cookies)

	if err != nil {
		if a.is40xError(err) {
			a.logger.Debugf("check session failed: %v", err)
			return false, nil
		}

		return false, err
	}

	authnMethods := session.AuthenticationMethods
	if len(authnMethods) < 2 {
		a.logger.Debugf("User has not yet completed 2fa")
		return false, nil
	}

	aal2AuthenticationMethod := authnMethods[1].Method

	if aal2AuthenticationMethod == nil || *aal2AuthenticationMethod != "lookup_secret" {
		return false, nil
	}

	// check the backup codes only if aal2 method was lookup_secret
	shouldRegenerateBackupCodes, err := a.service.HasNotEnoughLookupSecretsLeft(ctx, session.Identity.GetId())
	if err != nil {
		return false, err
	}

	return shouldRegenerateBackupCodes, nil
}

func (a *API) shouldEnforceMFA(ctx context.Context, cookies []*http.Cookie) (bool, error) {
	if !a.mfaEnabled {
		return false, nil
	}

	session, _, err := a.service.CheckSession(ctx, cookies)
	if err != nil {
		if a.is40xError(err) {
			a.logger.Debugf("check session failed, err: %v", err)
			return false, nil
		}

		return false, err
	}

	return a.shouldEnforceMFAWithSession(ctx, session)
}

func (a *API) shouldEnforceMFAWithSession(ctx context.Context, session *client.Session) (bool, error) {
	if !a.mfaEnabled {
		return false, nil
	}

	// if using OIDC external provider, do not enforce MFA
	// for _, method := range session.AuthenticationMethods {
	// 	if method.Method != nil && *method.Method == "oidc" {
	// 		return false, nil
	// 	}
	// }

	totpAvailable, err := a.service.HasTOTPAvailable(ctx, session.Identity.GetId())
	if err != nil {
		return false, err
	}

	return !totpAvailable, nil
}

func (a *API) is40xError(err error) bool {
	if openAPIErr, ok := err.(*client.GenericOpenAPIError); ok {
		if genericKratosErr, ok := openAPIErr.Model().(client.ErrorGeneric); ok {
			statusCode := genericKratosErr.Error.GetCode()
			return statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden
		}
	}

	return false
}

func (a *API) mfaSettingsRedirect(w http.ResponseWriter, returnTo string) {
	redirect, err := url.JoinPath("/", a.contextPath, "/ui/setup_secure")

	if err != nil {
		err = fmt.Errorf("unable to build mfa redirect path, possible misconfiguration, err: %v", err)
		a.logger.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	errorId := "session_aal2_required"

	// Set the original login URL as return_to, to continue the flow after mfa
	// has been set.
	redirectTo, err := url.ParseRequestURI(redirect)
	if err != nil {
		a.logger.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	q := redirectTo.Query()
	q.Set("return_to", returnTo)
	redirectTo.RawQuery = q.Encode()
	r := redirectTo.String()

	w.WriteHeader(http.StatusSeeOther)
	_ = json.NewEncoder(w).Encode(
		ErrorBrowserLocationChangeRequired{
			Error:             &client.GenericError{Id: &errorId},
			RedirectBrowserTo: &r,
		},
	)
}

func (a *API) lookupSecretsSettingsRedirect(w http.ResponseWriter, flowId, returnTo string) {
	redirect, err := url.JoinPath("/", a.contextPath, ui.UI, "/backup_codes_regenerate")
	if err != nil {
		err = fmt.Errorf("unable to build backup codes redirect path, possible misconfiguration, err: %v", err)
		a.logger.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	redirectTo, err := url.ParseRequestURI(redirect)
	if err != nil {
		a.logger.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	q := redirectTo.Query()
	q.Add("return_to", returnTo)
	q.Add("flow", flowId)
	redirectTo.RawQuery = q.Encode()
	r := redirectTo.String()
	errorId := RegenerateBackupCodesError

	w.WriteHeader(http.StatusSeeOther)
	_ = json.NewEncoder(w).Encode(
		ErrorBrowserLocationChangeRequired{
			Error:             &client.GenericError{Id: &errorId},
			RedirectBrowserTo: &r,
		},
	)
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

	if !flow.HasRedirectTo() && flow.HasError() {
		a.logger.Errorf("Error when updating recovery flow: %v\n", flow)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(flow)
		return
	}

	setCookies(w, cookies)
	// Kratos '422' response maps to 200 OK, it is expected
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(
		BrowserLocationChangeRequired{
			RedirectTo: flow.RedirectTo,
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

	setCookies(w, cookies, KRATOS_SESSION_COOKIE_NAME)
	// We delete any active Kratos sessions. If there were any active Kratos sessions,
	// recovery wouldn't be needed.
	// See https://github.com/canonical/kratos-operator/issues/259 for more info.
	http.SetCookie(w, kratosSessionUnsetCookie())

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
	if response != nil && response.HasRedirectTo() {
		a.logger.Errorf("Failed to get settings flow due to insufficient aal: %v\n", response)
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(
			ErrorBrowserLocationChangeRequired{
				Error:             response.Error,
				RedirectBrowserTo: response.RedirectTo,
			},
		)
		return
	}

	resp, err := flow.MarshalJSON()
	if err != nil {
		a.logger.Errorf("Error when marshalling json: %v\n", err)
		http.Error(w, "Failed to marshal json", http.StatusInternalServerError)
		return
	}
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
	returnTo := r.URL.Query().Get("return_to")

	flow, response, err := a.service.CreateBrowserSettingsFlow(context.Background(), returnTo, r.Cookies())
	if err != nil {
		a.logger.Errorf("Failed to create settings flow: %v", err)
		http.Error(w, "Failed to create settings flow", http.StatusInternalServerError)
		return
	}

	// If aal1, redirect to complete second factor auth
	if response != nil && response.HasRedirectTo() {
		a.logger.Errorf("Failed to create settings flow due to insufficient aal: %v\n", response)
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(
			ErrorBrowserLocationChangeRequired{
				Error:             response.Error,
				RedirectBrowserTo: response.RedirectTo,
			},
		)
		return
	}

	resp, err := flow.MarshalJSON()
	if err != nil {
		a.logger.Errorf("Error when marshalling json: %v\n", err)
		http.Error(w, "Failed to marshal json", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func kratosSessionUnsetCookie() *http.Cookie {
	return &http.Cookie{
		Name:     KRATOS_SESSION_COOKIE_NAME,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
	}
}

func NewAPI(
	service ServiceInterface,
	mfaEnabled bool,
	baseURL string,
	cookieManager AuthCookieManagerInterface,
	logger logging.LoggerInterface) *API {
	a := new(API)

	a.mfaEnabled = mfaEnabled
	a.service = service
	a.baseURL = baseURL
	a.cookieManager = cookieManager

	fullBaseURL, err := url.Parse(baseURL)
	if err != nil {
		// this should never happen if app is configured properly
		a.logger.Fatalf("Failed to construct API base URL: %v\n", err)
	}

	a.contextPath = fullBaseURL.Path

	a.logger = logger

	return a
}

func setCookies(w http.ResponseWriter, cookies []*http.Cookie, exclude ...string) {
	for _, c := range filterCookies(cookies, exclude...) {
		http.SetCookie(w, c)
	}
}

func filterCookies(cookies []*http.Cookie, exclude ...string) []*http.Cookie {
	ret := []*http.Cookie{}
l1:
	for _, c := range cookies {
		for _, n := range exclude {
			if c.Name == n {
				continue l1
			}
		}
		ret = append(ret, c)
	}
	return ret
}

func hash(plain string) string {
	h := md5.New()
	h.Write([]byte(plain))
	return base64.URLEncoding.EncodeToString(h.Sum(nil))
}

func validateHash(plain, sig string) bool {
	h := md5.New()
	h.Write([]byte(plain))
	return base64.URLEncoding.EncodeToString(h.Sum(nil)) == sig
}

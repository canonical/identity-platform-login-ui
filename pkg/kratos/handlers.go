package kratos

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	httpHelpers "github.com/canonical/identity-platform-login-ui/internal/misc/http"
	"github.com/go-chi/chi/v5"
	client "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
	"github.com/canonical/identity-platform-login-ui/pkg/ui"
)

const TOTP_REGISTRATION_REQUIRED = "totp_registration_required"
const WEBAUTHN_REGISTRATION_REQUIRED = "webauthn_registration_required"
const RegenerateBackupCodesError = "regenerate_backup_codes"
const SESSION_REFRESH_REQUIRED = "session_refresh_required"
const KRATOS_SESSION_COOKIE_NAME = "ory_kratos_session"
const LOGIN_UI_STATE_COOKIE = "login_ui_state"
const SECURITY_CSRF_VIOLATION_ERROR = "security_csrf_violation"

type API struct {
	mfaEnabled                    bool
	oidcWebAuthnSequencingEnabled bool
	service                       ServiceInterface
	baseURL                       string
	contextPath                   string
	cookieManager                 AuthCookieManagerInterface

	tracer tracing.TracingInterface
	logger logging.LoggerInterface
}

type KratosErrorResponse struct {
	Error             *client.GenericError `json:"error,omitempty"`
	RedirectBrowserTo string               `json:"redirect_browser_to,omitempty"`
	RedirectTo        string               `json:"redirect_to,omitempty"`
}

func (a *API) RegisterEndpoints(mux *chi.Mux) {
	mux.Post("/api/kratos/self-service/login", a.handleUpdateFlow)
	mux.Post("/api/kratos/self-service/login/id-first", a.handleUpdateIdentifierFirstFlow)
	mux.Get("/api/kratos/self-service/login/browser", a.handleCreateFlow)
	mux.Get("/api/kratos/self-service/login/flows", a.handleGetLoginFlow)
	mux.Post("/api/kratos/self-service/registration", a.handleUpdateRegistrationFlow)
	mux.Get("/api/kratos/self-service/registration/browser", a.handleCreateRegistrationFlow)
	mux.Get("/api/kratos/self-service/registration/flows", a.handleGetRegistrationFlow)
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
		response              any
		shouldEnforceMfa      = false
		shouldEnforceWebAuthn = false
		cookies               []*http.Cookie
		err                   error
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
	if session != nil {
		shouldEnforceMfa, err = a.shouldEnforceMFAWithSession(r.Context(), session)

		if err != nil {
			a.logger.Errorf("Failed check for MFA: %v", err)
			http.Error(w, "Failed check for MFA", http.StatusInternalServerError)
			return
		}
		if shouldEnforceMfa {
			flowCookie := FlowStateCookie{LoginChallengeHash: hash(loginChallenge)}
			a.mfaSettingsRedirect(w, r, returnTo, flowCookie)
			return
		}

		shouldEnforceWebAuthn, err = a.shouldEnforceWebAuthnWithSession(r.Context(), session)

		if err != nil {
			a.logger.Errorf("Failed check for WebAuthn: %v", err)
			http.Error(w, "Failed check for WebAuthn", http.StatusInternalServerError)
			return
		}
		if shouldEnforceWebAuthn {
			flowCookie := FlowStateCookie{LoginChallengeHash: hash(loginChallenge)}
			a.webAuthnSettingsRedirect(w, r, returnTo, flowCookie)
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
		if response, cookies, err = a.handleCreateFlowWithSession(r, session, loginChallenge); err == nil {
			a.cookieManager.ClearStateCookie(w)
		}
	} else {
		response, cookies, err = a.handleCreateFlowNewSession(r, aal, returnTo, loginChallenge, refresh, session)
	}

	if err != nil {
		// Propagate the KratosErrorResponse so frontend can handle it
		if kratosError, ok := parseGenericError(err); ok {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(kratosError)
			return
		}

		a.logger.Errorf(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	setCookies(w, cookies)

	// this case applies only to when there is a new session, for any other case we proceed with a 200
	switch res := response.(type) {
	case *client.LoginFlow:
		// If the browser was redirected here, we can't return a json object
		// so we redirect the user to the login page with the flow id appended in
		// the query params
		if a.isHTMLRequest(r) {
			u, _ := url.JoinPath(a.baseURL, "/ui/login")
			u = u + "?flow=" + res.Id
			http.Redirect(w, r, u, http.StatusSeeOther)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (a *API) handleCreateFlowNewSession(r *http.Request, aal, returnTo, loginChallenge string, refresh bool, session *client.Session) (*client.LoginFlow, []*http.Cookie, error) {
	// redirect user to this endpoint with the login_challenge after login
	// see https://github.com/ory/kratos/issues/3052

	cookies := r.Cookies()

	// clear cookies if not a refresh request, not aal2, and either:
	// - it's a hydra login (with loginChallenge)
	// - it's a kratos local login without a session
	if !refresh && aal != "aal2" && (session == nil || loginChallenge != "") {
		cookies = httpHelpers.FilterCookies(cookies, KRATOS_SESSION_COOKIE_NAME)
	}

	flow, cookies, err := a.service.CreateBrowserLoginFlow(
		r.Context(),
		aal,
		returnTo,
		loginChallenge,
		refresh,
		cookies,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create login flow, err: %w", err)
	}

	flow, err = a.service.FilterFlowProviderList(r.Context(), flow)
	if err != nil {
		return nil, nil, fmt.Errorf("error when filtering providers: %v\n", err)
	}

	return flow, cookies, nil
}

func (a *API) handleCreateFlowWithSession(r *http.Request, session *client.Session, loginChallenge string) (*BrowserLocationChangeRequired, []*http.Cookie, error) {
	response, cookies, err := a.service.AcceptLoginRequest(r.Context(), session, loginChallenge)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to accept login request: %w", err)
	}

	return response, cookies, nil
}

func parseGenericError(err error) (*KratosErrorResponse, bool) {
	var apiErr *client.GenericOpenAPIError
	if !errors.As(err, &apiErr) {
		return nil, false
	}

	var resp KratosErrorResponse
	if err := json.Unmarshal(apiErr.Body(), &resp); err != nil {
		return nil, false
	}

	return &resp, true
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
		if kratosError, ok := parseGenericError(err); ok {
			if kratosError.Error.GetId() == SECURITY_CSRF_VIOLATION_ERROR {
				a.deleteKratosSession(w)
			}
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(kratosError)
			return
		}

		a.logger.Errorf("Error when getting login flow: %v\n", err)
		http.Error(w, "Failed to get login flow", http.StatusInternalServerError)
		return
	}

	setCookies(w, cookies)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(flow)
}

func (a *API) handleCreateRegistrationFlow(w http.ResponseWriter, r *http.Request) {
	returnTo := r.URL.Query().Get("return_to")

	flow, cookies, err := a.service.CreateBrowserRegistrationFlow(r.Context(), returnTo)
	if err != nil {
		a.logger.Errorf("Failed to create registration flow: %v", err)
		http.Error(w, "Failed to create registration flow", http.StatusInternalServerError)
		return
	}

	setCookies(w, cookies)
	w.WriteHeader(http.StatusOK)

	toMap, _ := flow.ToMap()
	_ = json.NewEncoder(w).Encode(toMap)
}

func (a *API) handleGetRegistrationFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	flowId := q.Get("id")
	if flowId == "" {
		a.logger.Errorf("ID parameter not present")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode("ID parameter not present")
		return
	}

	flow, cookies, err := a.service.GetRegistrationFlow(r.Context(), flowId, r.Cookies())
	if err != nil {
		a.logger.Errorf("Error when getting registration flow: %v\n", err)
		http.Error(w, "Failed to get registration flow", http.StatusInternalServerError)
		return
	}

	setCookies(w, cookies)
	w.WriteHeader(http.StatusOK)
	toMap, _ := flow.ToMap()
	_ = json.NewEncoder(w).Encode(toMap)
}

func (a *API) handleUpdateRegistrationFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	flowId := q.Get("flow")
	if flowId == "" {
		a.logger.Errorf("ID parameter not present")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode("ID parameter not present")
		return
	}

	body, err := a.service.ParseRegistrationFlowMethodBody(r)
	if err != nil {
		a.logger.Errorf("Error when parsing request body: %v\n", err)
		http.Error(w, "Failed to parse registration flow", http.StatusInternalServerError)
		return
	}

	registration, cookies, err := a.service.UpdateRegistrationFlow(r.Context(), flowId, *body, r.Cookies())
	if err != nil {
		a.logger.Errorf("Error when updating registration flow: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	setCookies(w, cookies)
	toEncode, status := registration.GetFlowAndStatus()
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(toEncode)
}

func (a *API) handleUpdateIdentifierFirstFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	flowId := q.Get("flow")

	body, cookies, err := a.service.ParseIdentifierFirstLoginFlowMethodBody(r)
	if err != nil {
		err = fmt.Errorf("error when parsing request body: %w\n", err)
		a.logger.Errorf(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	redirectTo, cookies, err := a.service.UpdateIdentifierFirstLoginFlow(r.Context(), flowId, *body, cookies)
	if err != nil {
		err = fmt.Errorf("error when updating identifier first login flow: %w\n", err)
		a.logger.Errorf(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	setCookies(w, cookies)

	if redirectTo != nil {
		a.redirectResponse(w, r, redirectTo)
		return
	}
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

	u, _ := url.Parse(loginFlow.GetReturnTo())
	lc := u.Query().Get("login_challenge")
	flowCookie := FlowStateCookie{LoginChallengeHash: hash(lc)}

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
		a.lookupSecretsSettingsRedirect(w, r, flowId, *loginFlow.ReturnTo, flowCookie)
		return
	}

	if shouldEnforceMfa {
		a.mfaSettingsRedirect(w, r, *loginFlow.ReturnTo, flowCookie)
		return
	}

	if redirectTo != nil {
		a.cookieManager.SetStateCookie(w, flowCookie)
		a.redirectResponse(w, r, redirectTo)
		return
	}

	// This is a hydra flow
	if lc != "" {
		// User is authenticated, return the session
		response, cookies, err := a.handleCreateFlowWithSession(r, &flow.Session, lc)
		if err != nil {
			a.logger.Errorf(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		a.cookieManager.ClearStateCookie(w)
		setCookies(w, cookies)
		a.redirectResponse(w, r, response)
		return
	}

	if a.isHTMLRequest(r) {
		// redirect to returnTo url instead of returning a json response
		if returnTo, ok := loginFlow.GetReturnToOk(); ok {
			a.redirectResponse(w, r, &BrowserLocationChangeRequired{
				RedirectTo: returnTo,
			})
			return
		}
		// fall back to return a server error when there is no returnTo
		a.logger.Error("Failed to get returnTo")
		http.Error(w, "Failed to get returnTo", http.StatusInternalServerError)
		return
	}

	resp, err := flow.MarshalJSON()
	if err != nil {
		a.logger.Errorf("Error when marshalling Json: %v\n", err)
		http.Error(w, "Failed to parse flow error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (a *API) redirectResponse(w http.ResponseWriter, r *http.Request, resp RedirectToInterface) {
	code := http.StatusOK
	// We differentiate between simple redirects and redirects because of an error to make it easier
	// for the frontend
	if resp.GetCode() == http.StatusForbidden {
		code = http.StatusForbidden
	}
	switch r.Method {
	case http.MethodGet:
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(resp)
	case http.MethodPost:
		// In case of webauthn the user is redirected here and we get a FORM, instead of JSON.
		// TODO: Remove, when the UI fixes this
		if r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
			http.Redirect(w, r, resp.GetRedirectTo(), http.StatusSeeOther)
			return
		}
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(resp)
	default:
		http.Error(w, "unexpected method", http.StatusInternalServerError)
	}
}

func (a *API) shouldRegenerateBackupCodes(ctx context.Context, cookies []*http.Cookie) (bool, error) {
	ctx, span := a.tracer.Start(ctx, "kratos.API.shouldRegenerateBackupCodes")
	defer span.End()

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
	ctx, span := a.tracer.Start(ctx, "kratos.API.shouldEnforceMFA")
	defer span.End()

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
	ctx, span := a.tracer.Start(ctx, "kratos.API.shouldEnforceMFAWithSession")
	defer span.End()

	if !a.mfaEnabled {
		return false, nil
	}

	// if using OIDC external provider, do not enforce MFA
	for _, method := range session.AuthenticationMethods {
		if method.Method != nil && *method.Method == "oidc" {
			return false, nil
		}
	}

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

func (a *API) shouldEnforceWebAuthnWithSession(ctx context.Context, session *client.Session) (bool, error) {
	ctx, span := a.tracer.Start(ctx, "kratos.API.shouldEnforceWebAuthnWithSession")
	defer span.End()

	if !a.oidcWebAuthnSequencingEnabled {
		return false, nil
	}

	// enforce only if one of the authentication methods was oidc
	for _, method := range session.AuthenticationMethods {
		if method.GetMethod() == "oidc" {
			webAuthnAvailable, err := a.service.HasWebAuthnAvailable(ctx, session.Identity.GetId())
			if err != nil {
				return false, err
			}
			return !webAuthnAvailable, nil
		}
	}
	return false, nil
}

func (a *API) webAuthnSettingsRedirect(w http.ResponseWriter, r *http.Request, returnTo string, flowStateCookie FlowStateCookie) {
	redirect, err := url.JoinPath("/", a.contextPath, "/ui/setup_passkey")
	if err != nil {
		return
	}

	errorId := WEBAUTHN_REGISTRATION_REQUIRED

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
	rt := redirectTo.String()

	flowStateCookie.WebauthnSetup = true
	a.cookieManager.SetStateCookie(w, flowStateCookie)
	a.redirectResponse(w, r, &BrowserLocationChangeRequired{
		Error:      &client.GenericError{Id: &errorId},
		RedirectTo: &rt,
	})
}

func (a *API) mfaSettingsRedirect(w http.ResponseWriter, r *http.Request, returnTo string, flowStateCookie FlowStateCookie) {
	redirect, err := url.JoinPath("/", a.contextPath, "/ui/setup_secure")

	if err != nil {
		err = fmt.Errorf("unable to build mfa redirect path, possible misconfiguration, err: %v", err)
		a.logger.Error(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	errorId := TOTP_REGISTRATION_REQUIRED

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
	rt := redirectTo.String()

	flowStateCookie.TotpSetup = true

	a.cookieManager.SetStateCookie(w, flowStateCookie)
	a.redirectResponse(w, r, &BrowserLocationChangeRequired{
		Error:      &client.GenericError{Id: &errorId},
		RedirectTo: &rt,
	})
}

func (a *API) lookupSecretsSettingsRedirect(w http.ResponseWriter, r *http.Request, flowId, returnTo string, flowStateCookie FlowStateCookie) {
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
	rt := redirectTo.String()
	errorId := RegenerateBackupCodesError

	flowStateCookie.BackupCodeUsed = true

	a.cookieManager.SetStateCookie(w, flowStateCookie)
	a.redirectResponse(w, r, &BrowserLocationChangeRequired{
		Error:      &client.GenericError{Id: &errorId},
		RedirectTo: &rt,
	})
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
	a.redirectResponse(w, r, &BrowserLocationChangeRequired{
		RedirectTo: flow.RedirectTo,
	})
}

func (a *API) handleCreateRecoveryFlow(w http.ResponseWriter, r *http.Request) {
	returnTo := r.URL.Query().Get("return_to")

	if returnTo == "" {
		var err error
		returnTo, err = url.JoinPath(a.baseURL, "/ui/reset_email")
		if err != nil {
			a.logger.Errorf("Failed to construct returnTo URL: ", err)
			http.Error(w, "Failed to construct returnTo URL", http.StatusBadRequest)
		}
	}

	flow, cookies, err := a.service.CreateBrowserRecoveryFlow(context.Background(), returnTo)
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
	a.deleteKratosSession(w)

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (a *API) handleGetSettingsFlow(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	flow, response, err := a.service.GetSettingsFlow(context.Background(), q.Get("id"), r.Cookies())
	if err != nil {
		a.logger.Errorf("Error when getting settings flow: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If aal1, redirect to complete second factor auth
	if response != nil && response.HasRedirectTo() {
		a.logger.Errorf("Failed to get settings flow due to insufficient aal: %v\n", response)
		a.redirectResponse(w, r, &BrowserLocationChangeRequired{
			Error:      response.Error,
			RedirectTo: response.RedirectTo,
		})
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

	flow, redirectInfo, cookies, err := a.service.UpdateSettingsFlow(context.Background(), flowId, *body, r.Cookies())
	if err != nil {
		a.logger.Errorf("Error when updating settings flow: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	setCookies(w, cookies)

	if redirectInfo != nil {
		// Check for privileged session request
		if redirectInfo.GetErrorId() == SESSION_REFRESH_REQUIRED && redirectInfo.HasRedirectTo() {
			// Kratos defaults 'return_to' to the endpoint that triggered the error (POST /self-service/settings)
			// We perform a GET request upon redirect after login, but the endpoint expects POST, causing 405 Method Not Allowed.
			// To fix this, we overwrite 'return_to' to the settings UI page so the user can re-submit the form.
			returnTo, err := a.settingsReturnToURL(r, flowId)
			if err != nil {
				a.logger.Errorf("Failed to build settings returnTo URL: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			u, err := url.Parse(redirectInfo.GetRedirectTo())
			if err != nil {
				a.logger.Errorf("Failed to parse redirect url: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			q := u.Query()
			q.Set("return_to", returnTo)
			u.RawQuery = q.Encode()

			newURL := u.String()
			redirectInfo.RedirectTo = &newURL
		}

		a.redirectResponse(w, r, redirectInfo)
		return
	}

	if a.isHTMLRequest(r) {
		var returnTo *string

		if flowReturnTo, ok := flow.GetReturnToOk(); ok {
			returnTo = flowReturnTo
		} else if continueWith, ok := flow.GetContinueWithOk(); ok {
			returnTo = getReturnToFromContinueWith(continueWith)
		}

		// redirect to returnTo url instead of returning a json response
		// this maintains previous kratos behaviour on webauthn registration
		if returnTo != nil {
			a.redirectResponse(w, r, &BrowserLocationChangeRequired{
				RedirectTo: returnTo,
			})
			return
		}

		// fall back to return a server error when there is no returnTo
		// this should never happen as SettingsFlow has at least ContinueWithRedirectBrowserTo
		a.logger.Error("Failed to get returnTo")
		http.Error(w, "Failed to get returnTo", http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(flow)
	if err != nil {
		a.logger.Errorf("Error when marshalling json: %v\n", err)
		http.Error(w, "Failed to parse settings flow", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (a *API) settingsReturnToURL(r *http.Request, flowId string) (string, error) {
	currentFlow, _, err := a.service.GetSettingsFlow(r.Context(), flowId, r.Cookies())
	if err != nil {
		a.logger.Debugf("Failed to get settings flow: %v, falling back to default return_to", err)
	}

	var returnTo string
	if currentFlow != nil {
		if flowReturnTo, ok := currentFlow.GetReturnToOk(); ok {
			returnTo = *flowReturnTo
		}
	}

	if returnTo == "" {
		// fall back to a default redirect path
		returnTo, err = url.JoinPath("/", a.contextPath, ui.UI, "/manage_details")
		if err != nil {
			return "", fmt.Errorf("unable to build settings returnTo path, possible misconfiguration, err: %w", err)
		}
	}
	return returnTo, nil
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
		a.redirectResponse(w, r, &BrowserLocationChangeRequired{
			Error:      response.Error,
			RedirectTo: response.RedirectTo,
		})
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

func (a *API) isHTMLRequest(r *http.Request) bool {
	// Treat requests that don't explicitly accept json as form submissions
	return r.Header.Get("Accept") != "application/json, text/plain, */*"
}

func (a *API) deleteKratosSession(w http.ResponseWriter) {
	// To delete the session we delete the kratos session cookie.
	// This is hacky as it does not call the Kratos API and is likely to break on
	// a new Kratos version, but there is no easy way to delete the session
	// from the Kratos API
	c := kratosSessionUnsetCookie()
	http.SetCookie(w, c)
}

func getReturnToFromContinueWith(continueWith []client.ContinueWith) *string {
	for _, c := range continueWith {
		if r := c.ContinueWithRedirectBrowserTo; r != nil {
			return &r.RedirectBrowserTo
		}
	}
	return nil
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
	mfaEnabled,
	oidcWebAuthnSequencingEnabled bool,
	baseURL string,
	cookieManager AuthCookieManagerInterface,
	tracer tracing.TracingInterface,
	logger logging.LoggerInterface) *API {
	a := new(API)

	a.mfaEnabled = mfaEnabled
	a.oidcWebAuthnSequencingEnabled = oidcWebAuthnSequencingEnabled
	a.service = service
	a.baseURL = baseURL
	a.cookieManager = cookieManager

	fullBaseURL, err := url.Parse(baseURL)
	if err != nil {
		// this should never happen if app is configured properly
		a.logger.Fatalf("Failed to construct API base URL: %v\n", err)
	}

	a.contextPath = fullBaseURL.Path

	a.tracer = tracer
	a.logger = logger

	return a
}

func setCookies(w http.ResponseWriter, cookies []*http.Cookie, exclude ...string) {
	for _, c := range httpHelpers.FilterCookies(cookies, exclude...) {
		http.SetCookie(w, c)
	}
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

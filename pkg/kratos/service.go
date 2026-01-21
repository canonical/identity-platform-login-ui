package kratos

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/internal/logging"
	httpHelpers "github.com/canonical/identity-platform-login-ui/internal/misc/http"
	"github.com/canonical/identity-platform-login-ui/internal/monitoring"
	"github.com/canonical/identity-platform-login-ui/internal/tracing"
)

const (
	MinimumBackupCodesAmount     = 3
	RecoveryCodeSent             = 1060003
	InvalidProperty              = 4000002
	NotEnoughCharacters          = 4000003
	IncorrectCredentials         = 4000006
	DuplicateIdentifier          = 4000007
	InvalidAuthCode              = 4000008
	InactiveAccount              = 4000010
	BackupCodeAlreadyUsed        = 4000012
	MissingBackupCodesSetup      = 4000014
	MissingSecurityKeySetup      = 4000015
	InvalidBackupCode            = 4000016
	TooManyCharacters            = 4000017
	PasswordIdentifierSimilarity = 4000031
	PasswordTooLong              = 4000033
	IncorrectAccountIdentifier   = 4000037
	NewPasswordPolicyViolation   = 4000039
	InvalidRecoveryCode          = 4060006
)

type Service struct {
	kratos      KratosClientInterface
	kratosAdmin KratosAdminClientInterface
	hydra       HydraClientInterface
	authz       AuthorizerInterface

	oidcWebAuthnSequencingEnabled bool

	tracer  tracing.TracingInterface
	monitor monitoring.MonitorInterface
	logger  logging.LoggerInterface
}

// We override the type from the kratos sdk, as it does not
// get marshalled correctly into json.
// For more info see: https://github.com/canonical/identity-platform-login-ui/pull/73/files#r1250460283
type ErrorBrowserLocationChangeRequired struct {
	Error *kClient.GenericError `json:"error,omitempty"`
	// Points to where to redirect the user to next.
	RedirectBrowserTo *string `json:"redirect_browser_to,omitempty"`
}

func (e *BrowserLocationChangeRequired) HasError() bool {
	return e.Error != nil
}

func (e *BrowserLocationChangeRequired) HasRedirectTo() bool {
	return e.RedirectTo != nil
}

func (e *BrowserLocationChangeRequired) GetCode() int {
	if !e.HasError() || e.Error.Code == nil {
		return http.StatusOK
	}
	return int(*e.Error.Code)
}

func (e *BrowserLocationChangeRequired) GetRedirectTo() string {
	if e.RedirectTo == nil {
		return ""
	}
	return *e.RedirectTo
}

type BrowserLocationChangeRequired struct {
	Error *kClient.GenericError `json:"error,omitempty"`
	// Points to where to redirect the user to next.
	RedirectTo *string `json:"redirect_to,omitempty"`
}

type UiErrorMessages struct {
	Ui kClient.UiContainer `json:"ui"`
}

type methodOnly struct {
	Method string `json:"method"`
}

type LookupSecrets []struct {
	Code   string    `json:"code"`
	UsedAt time.Time `json:"used_at,omitempty"`
}

func (s *Service) CheckSession(ctx context.Context, cookies []*http.Cookie) (*kClient.Session, []*http.Cookie, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.ToSession")
	defer span.End()

	session, resp, err := s.kratos.FrontendApi().
		ToSession(ctx).
		Cookie(httpHelpers.CookiesToString(cookies)).
		Execute()

	if err != nil {
		return nil, nil, err
	}
	return session, resp.Cookies(), nil
}

func (s *Service) AcceptLoginRequest(ctx context.Context, session *kClient.Session, lc string) (*BrowserLocationChangeRequired, []*http.Cookie, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.AcceptLoginRequest")
	defer span.End()

	accept := hClient.NewAcceptOAuth2LoginRequest(session.Identity.Id)
	accept.SetRemember(true)
	accept.Amr = []string{}
	for _, r := range session.AuthenticationMethods {
		accept.Amr = append(accept.Amr, r.GetMethod())
	}

	accept.IdentityProviderSessionId = &session.Id

	if session.ExpiresAt != nil {
		expAt := time.Until(*session.ExpiresAt)
		// Set the session to expire when the kratos session expires
		accept.SetRememberFor(int64(expAt.Seconds()))
	}

	redirectTo, resp, err := s.hydra.OAuth2API().
		AcceptOAuth2LoginRequest(ctx).
		LoginChallenge(lc).
		AcceptOAuth2LoginRequest(*accept).
		Execute()

	if err != nil {
		return nil, nil, err
	}

	return &BrowserLocationChangeRequired{RedirectTo: &redirectTo.RedirectTo}, resp.Cookies(), nil
}

func (s *Service) GetLoginRequest(ctx context.Context, loginChallenge string) (*hClient.OAuth2LoginRequest, []*http.Cookie, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.GetLoginRequest")
	defer span.End()

	redirectTo, resp, err := s.hydra.OAuth2API().
		GetOAuth2LoginRequest(ctx).
		LoginChallenge(loginChallenge).
		Execute()

	if err != nil {
		return nil, nil, err
	}

	return redirectTo, resp.Cookies(), nil
}

func (s *Service) MustReAuthenticate(ctx context.Context, hydraLoginChallenge string, session *kClient.Session, c FlowStateCookie) (bool, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.MustReAuthenticate")
	defer span.End()

	if session == nil {
		// No session exists, user is not logged in
		return true, nil
	}

	if hydraLoginChallenge == "" {
		// It's not a hydra flow, let kratos handle it
		return true, nil
	}

	// This is the first user login, they set up their authenticator app
	// Or backup code was used for login, no need to re-auth
	if validateHash(hydraLoginChallenge, c.LoginChallengeHash) && (c.TotpSetup || c.BackupCodeUsed || c.WebauthnSetup) {
		return false, nil
	}

	hydraLoginRequest, _, err := s.GetLoginRequest(ctx, hydraLoginChallenge)
	if err != nil {
		return true, err
	}

	return !hydraLoginRequest.GetSkip(), nil
}

func (s *Service) CreateBrowserLoginFlow(
	ctx context.Context, aal, returnTo, loginChallenge string, refresh bool, cookies []*http.Cookie,
) (*kClient.LoginFlow, []*http.Cookie, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.CreateBrowserLoginFlow")
	defer span.End()

	request := s.kratos.FrontendApi().
		CreateBrowserLoginFlow(ctx).
		Aal(aal).
		ReturnTo(returnTo).
		Refresh(refresh).
		Cookie(httpHelpers.CookiesToString(cookies))

	if !s.oidcWebAuthnSequencingEnabled {
		if loginChallenge != "" {
			request = request.LoginChallenge(loginChallenge)
		} else if loginChallenge == "" && returnTo == "" {
			return nil, nil, fmt.Errorf("no return_to or login_challenge was provided")
		}
	}

	flow, resp, err := request.Execute()
	if err != nil {
		return nil, nil, err
	}

	// Populate the flow with the hydra login req, so that the UI can retrieve this info
	flow, err = s.hydrateKratosLoginFlow(ctx, flow)
	if err != nil {
		s.logger.Warnf("Failed to fetch the hydra oauth2 request: %v", err)
	}

	return flow, resp.Cookies(), nil
}

func (s *Service) CreateBrowserRecoveryFlow(ctx context.Context, returnTo string, cookies []*http.Cookie) (*kClient.RecoveryFlow, []*http.Cookie, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.CreateBrowserRecoveryFlow")
	defer span.End()

	flow, resp, err := s.kratos.FrontendApi().
		CreateBrowserRecoveryFlow(ctx).
		ReturnTo(returnTo).
		Execute()
	if err != nil {
		return nil, nil, err
	}

	return flow, resp.Cookies(), nil
}

func (s *Service) CreateBrowserSettingsFlow(ctx context.Context, returnTo string, cookies []*http.Cookie) (*kClient.SettingsFlow, *BrowserLocationChangeRequired, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.CreateBrowserSettingsFlow")
	defer span.End()

	request := s.kratos.FrontendApi().
		CreateBrowserSettingsFlow(ctx).
		Cookie(httpHelpers.CookiesToString(cookies))

	if returnTo != "" {
		request = request.ReturnTo(returnTo)
	}

	flow, resp, err := request.Execute()

	// 403 means the user must be redirected to complete second factor auth
	// in order to access settings
	if err != nil && resp.StatusCode != http.StatusForbidden {
		return nil, nil, err
	}

	if err == nil {
		return flow, nil, nil
	}

	returnToResp, err := s.parseKratosRedirectResponse(ctx, resp)
	if err != nil {
		return nil, nil, err
	}

	return flow, returnToResp, err
}

func (s *Service) CreateBrowserVerificationFlow(ctx context.Context, cookies []*http.Cookie) (*kClient.VerificationFlow, []*http.Cookie, error) {
    ctx, span := s.tracer.Start(ctx, "kratos.Service.CreateBrowserVerificationFlow")
    defer span.End()

    flow, resp, err := s.kratos.FrontendApi().
        CreateBrowserVerificationFlow(ctx).
        Execute()

    if err != nil {
        s.logger.Errorf("unable to create verification flow: %s", err)
        return nil, nil, fmt.Errorf("unable to create verification flow: %w", err)
    }

    if resp != nil {
        cookies = resp.Cookies()
    }

    return flow, cookies, nil
}

func (s *Service) GetVerificationFlow(ctx context.Context, flowId string, cookies []*http.Cookie) (*kClient.VerificationFlow, []*http.Cookie, error) {
    ctx, span := s.tracer.Start(ctx, "kratos.Service.GetVerificationFlow")
    defer span.End()

    req := s.kratos.FrontendApi().GetVerificationFlow(ctx).
        Id(flowId).
        Cookie(httpHelpers.CookiesToString(cookies))

    flow, response, err := s.kratos.FrontendApi().GetVerificationFlowExecute(req)
    if err != nil {
        return nil, nil, fmt.Errorf("unable to fetch verification flow: %w", err)
    }

    if response != nil {
        cookies = response.Cookies()
    }

    return flow, cookies, nil
}

func (s *Service) UpdateVerificationFlow(ctx context.Context, flowId string, body kClient.UpdateVerificationFlowBody, cookies []*http.Cookie) (*kClient.VerificationFlow, []*http.Cookie, error) {
    ctx, span := s.tracer.Start(ctx, "kratos.Service.UpdateVerificationFlow")
    defer span.End()

    req := s.kratos.FrontendApi().UpdateVerificationFlow(ctx).
        Flow(flowId).
        Cookie(httpHelpers.CookiesToString(cookies)).
        UpdateVerificationFlowBody(body)

    flow, resp, err := s.kratos.FrontendApi().UpdateVerificationFlowExecute(req)

    if err != nil {
        s.logger.Errorf("unable to update verification flow: %s", err)
        return nil, nil, fmt.Errorf("unable to update verification flow: %w", err)
    }

    if resp != nil {
        cookies = resp.Cookies()
    }

    return flow, cookies, err
}

func (s *Service) GetLoginFlow(ctx context.Context, id string, cookies []*http.Cookie) (*kClient.LoginFlow, []*http.Cookie, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.GetLoginFlow")
	defer span.End()

	flow, resp, err := s.kratos.FrontendApi().
		GetLoginFlow(ctx).
		Id(id).
		Cookie(httpHelpers.CookiesToString(cookies)).
		Execute()
	if err != nil {
		return nil, nil, err
	}

	// Populate the flow with the hydra login req, so that the UI can retrieve this info
	flow, err = s.hydrateKratosLoginFlow(ctx, flow)
	if err != nil {
		s.logger.Warnf("Failed to fetch the hydra oauth2 request: %v", err)
	}
	return flow, resp.Cookies(), nil
}

func (s *Service) GetRecoveryFlow(ctx context.Context, id string, cookies []*http.Cookie) (*kClient.RecoveryFlow, []*http.Cookie, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.GetRecoveryFlow")
	defer span.End()

	flow, resp, err := s.kratos.FrontendApi().
		GetRecoveryFlow(ctx).
		Id(id).
		Cookie(httpHelpers.CookiesToString(cookies)).
		Execute()
	if err != nil {
		return nil, nil, err
	}

	return flow, resp.Cookies(), nil
}

func (s *Service) GetSettingsFlow(ctx context.Context, id string, cookies []*http.Cookie) (*kClient.SettingsFlow, *BrowserLocationChangeRequired, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.GetSettingsFlow")
	defer span.End()

	flow, resp, err := s.kratos.FrontendApi().
		GetSettingsFlow(ctx).
		Id(id).
		Cookie(httpHelpers.CookiesToString(cookies)).
		Execute()

	// 403 means the user must be redirected to complete second factor auth
	// in order to access settings
	if err != nil && resp.StatusCode != http.StatusForbidden {
		return nil, nil, err
	}

	// If a duplicate identifier was detected, kratos returns a 200 response
	// with a 4000007 error in the rendered ui messages
	if resp.StatusCode == http.StatusOK {
		uiMsg := flow.GetUi()

		for _, message := range uiMsg.GetMessages() {
			if message.GetId() == DuplicateIdentifier {
				return nil, nil, fmt.Errorf("an account with the same identifier already exists, contact support")
			}
		}
	}

	if err == nil {
		return flow, nil, nil
	}

	returnToResp, err := s.parseKratosRedirectResponse(ctx, resp)
	if err != nil {
		return nil, nil, err
	}

	return flow, returnToResp, nil
}

func (s *Service) UpdateRecoveryFlow(
	ctx context.Context, flow string, body kClient.UpdateRecoveryFlowBody, cookies []*http.Cookie,
) (*BrowserLocationChangeRequired, []*http.Cookie, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.UpdateRecoveryFlow")
	defer span.End()

	recovery, resp, err := s.kratos.FrontendApi().
		UpdateRecoveryFlow(ctx).
		Flow(flow).
		UpdateRecoveryFlowBody(body).
		Cookie(httpHelpers.CookiesToString(cookies)).
		Execute()

	// if the flow responds with 400, it means a session already exists
	if err != nil && resp.StatusCode == http.StatusBadRequest {
		returnToResp, err := s.parseKratosRedirectResponse(ctx, resp)
		return returnToResp, nil, err
	}

	// If the recovery code was invalid, kratos returns a 200 response
	// with a 4060006 error in the rendered ui messages.
	// If the recovery code was valid, we expect to get a 422 response from kratos.
	// That is because the user needs to be redirected to self-service settings page.
	// The sdk forces us to make the request with an 'application/json' content-type, whereas Kratos
	// expects the 'Content-Type' and 'Accept' to be 'application/x-www-form-urlencoded'.
	// This is not a real error, as we still get the URL to which the user needs to be
	// redirected to.

	if err != nil && resp.StatusCode != http.StatusUnprocessableEntity {
		err := s.getUiError(resp.Body)
		return nil, nil, err
	}

	if resp.StatusCode == http.StatusOK {
		uiMsg := recovery.GetUi()

		for _, message := range uiMsg.GetMessages() {
			if message.GetId() == InvalidRecoveryCode {
				return nil, nil, fmt.Errorf("the recovery code is invalid or has already been used")
			}
		}
	}

	returnToResp, err := s.parseKratosRedirectResponse(ctx, resp)
	if err != nil {
		return nil, nil, err
	}

	return returnToResp, resp.Cookies(), nil
}

func (s *Service) UpdateIdentifierFirstLoginFlow(
	ctx context.Context, flow string, body kClient.UpdateLoginFlowWithIdentifierFirstMethod, cookies []*http.Cookie,
) (*BrowserLocationChangeRequired, []*http.Cookie, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.UpdateIdentifierFirstLoginFlow")
	defer span.End()

	csrfToken, ok := body.GetCsrfTokenOk()
	if !ok {
		return nil, nil, fmt.Errorf("missing csrf token")
	}

	identifier, ok := body.GetIdentifierOk()
	if !ok {
		return nil, nil, fmt.Errorf("missing identifier")
	}

	resp, err := s.kratos.ExecuteIdentifierFirstUpdateLoginRequest(ctx, flow, *csrfToken, *identifier, cookies)
	if err != nil {
		s.logger.Errorf("kratos request failed: %s", err)
		return nil, nil, err
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusSeeOther:
		location := resp.Header.Get("Location")
		return &BrowserLocationChangeRequired{RedirectTo: &location}, resp.Cookies(), nil
	case http.StatusBadRequest:
		err = s.getUiError(resp.Body)
		return nil, nil, err
	default:
		s.logger.Errorf("updating identifier first flow %s failed: got unexpected response status %d from kratos", flow, resp.StatusCode)
		return nil, nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
}

func (s *Service) UpdateLoginFlow(
	ctx context.Context, flow string, body kClient.UpdateLoginFlowBody, cookies []*http.Cookie,
) (*BrowserLocationChangeRequired, *kClient.SuccessfulNativeLogin, []*http.Cookie, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.UpdateLoginFlow")
	defer span.End()

	f, resp, err := s.kratos.FrontendApi().
		UpdateLoginFlow(ctx).
		Flow(flow).
		UpdateLoginFlowBody(body).
		Cookie(httpHelpers.CookiesToString(cookies)).
		Execute()
	// We expect to get a 422 response from Kratos. The sdk forces us to
	// make the request with an 'application/json' content-type, whereas Kratos
	// expects the 'Content-Type' and 'Accept' to be 'application/x-www-form-urlencoded'.
	// This is not a real error, as we still get the URL to which the user needs to be
	// redirected to.
	if err != nil && resp.StatusCode != http.StatusUnprocessableEntity {
		err := s.getUiError(resp.Body)
		return nil, nil, nil, err
	}

	c := resp.Cookies()
	if body.UpdateLoginFlowWithOidcMethod != nil {
		// If this is an oidc flow, we need to delete the session cookie
		// A session cookie (probably) means that the user used 1fa, but went back from the 2nd factor screen
		// If the session cookie is set, then Kratos will redirect the user to the default return to URL
		// The only way to avoid this is by setting refresh=true, but the user has no kratos session.
		// This is probably a bug on kratos side, as they check if a session exists to set refresh=false.
		// But in oidc they only check if the session cookie exists, which is not sufficient as the user may not have
		// enough aal
		c = append(c, kratosSessionUnsetCookie())
	}

	if resp.StatusCode == http.StatusUnprocessableEntity {
		returnToResp, err := s.parseKratosRedirectResponse(ctx, resp)
		if err != nil {
			return nil, nil, nil, err
		}

		return returnToResp, nil, c, nil
	}
	// Workaround for marshalling error
	// TODO: Evaluate if we can get rid of that when kratos sdk 1.3 is out
	f.ContinueWith = nil
	return nil, f, c, nil
}

func (s *Service) UpdateSettingsFlow(
	ctx context.Context, flow string, body kClient.UpdateSettingsFlowBody, cookies []*http.Cookie,
) (*kClient.SettingsFlow, *BrowserLocationChangeRequired, []*http.Cookie, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.UpdateSettingsFlow")
	defer span.End()

	settingsFlow, resp, err := s.kratos.FrontendApi().
		UpdateSettingsFlow(ctx).
		Flow(flow).
		UpdateSettingsFlowBody(body).
		Cookie(httpHelpers.CookiesToString(cookies)).
		Execute()

	// Handle 422 response
	if err != nil && resp.StatusCode == http.StatusUnprocessableEntity {
		returnToResp, err := s.parseKratosRedirectResponse(ctx, resp)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to parse redirect response: %w", err)
		}

		if returnToResp.RedirectTo == nil {
			return nil, nil, nil, fmt.Errorf("failed to get redirect url")
		}

		return nil, returnToResp, resp.Cookies(), nil
	}

	if err != nil && resp.StatusCode != http.StatusOK {
		err := s.getUiError(resp.Body)

		return nil, nil, nil, err
	}

	return settingsFlow, nil, resp.Cookies(), nil
}

func (s *Service) getUiError(responseBody io.ReadCloser) (err error) {
	errorMessages := new(UiErrorMessages)
	body, _ := io.ReadAll(responseBody)
	json.Unmarshal([]byte(body), &errorMessages)

	errorCodes := errorMessages.Ui.Messages

	// if no message was found, search through nodes
	if len(errorCodes) == 0 {
		nodes := errorMessages.Ui.GetNodes()
		for _, node := range nodes {
			// look for the node where error appears
			for _, message := range node.Messages {
				if message.Type == "error" {
					errorCodes = node.GetMessages()
				}
			}
		}
	}

	if len(errorCodes) == 0 {
		err = fmt.Errorf("error code not found")
		s.logger.Errorf(err.Error())
		return err
	}

	switch errorCode := errorCodes[0].Id; errorCode {
	case IncorrectCredentials:
		err = fmt.Errorf("incorrect username or password")
	case IncorrectAccountIdentifier:
		err = fmt.Errorf("account does not exist or has no login method configured")
	case InactiveAccount:
		err = fmt.Errorf("inactive account")
	case InvalidProperty:
		err = fmt.Errorf("invalid %s", errorCodes[0].Context["property"])
	case NotEnoughCharacters:
		err = fmt.Errorf("at least %v characters required", errorCodes[0].Context["min_length"])
	case TooManyCharacters, PasswordTooLong:
		err = fmt.Errorf("maximum %v characters allowed", errorCodes[0].Context["max_length"])
	case InvalidAuthCode:
		err = fmt.Errorf("invalid authentication code")
	case MissingSecurityKeySetup:
		err = fmt.Errorf("choose a different login method")
	case BackupCodeAlreadyUsed:
		err = fmt.Errorf("this backup code was already used")
	case InvalidBackupCode:
		err = fmt.Errorf("invalid backup code")
	case MissingBackupCodesSetup:
		err = fmt.Errorf("login with backup codes unavailable")
	case PasswordIdentifierSimilarity:
		err = fmt.Errorf("password can not be similar to the email")
	case NewPasswordPolicyViolation:
		err = fmt.Errorf("new password does not meet the password policy requirements: %s", errorCodes[0].Text)
	default:
		s.logger.Errorf("Unknown kratos error code: %v", errorCode)
		err = fmt.Errorf("server error")
	}
	return err
}

func (s *Service) GetFlowError(ctx context.Context, id string) (*kClient.FlowError, []*http.Cookie, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.GetFlowError")
	defer span.End()

	flowError, resp, err := s.kratos.FrontendApi().GetFlowError(ctx).Id(id).Execute()
	if err != nil {
		return nil, nil, err
	}

	return flowError, resp.Cookies(), nil
}

func (s *Service) CheckAllowedProvider(ctx context.Context, loginFlow *kClient.LoginFlow, updateFlowBody *kClient.UpdateLoginFlowBody) (bool, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.CheckAllowedProvider")
	defer span.End()

	provider := s.getProviderName(updateFlowBody)
	clientName := s.getClientName(loginFlow)

	allowedProviders, err := s.authz.ListObjects(ctx, fmt.Sprintf("app:%s", clientName), "allowed_access", "provider")
	if err != nil {
		return false, err
	}
	// If the user has not configured providers for this app, we allow all providers
	if len(allowedProviders) == 0 {
		return true, nil
	}
	return s.contains(allowedProviders, fmt.Sprintf("%v", provider)), nil
}

func (s *Service) getProviderName(updateFlowBody *kClient.UpdateLoginFlowBody) string {
	if updateFlowBody.GetActualInstance() == updateFlowBody.UpdateLoginFlowWithOidcMethod {
		return updateFlowBody.UpdateLoginFlowWithOidcMethod.Provider
	}
	return ""
}

func (s *Service) getClientName(loginFlow *kClient.LoginFlow) string {
	oauth2LoginRequest := loginFlow.Oauth2LoginRequest
	if oauth2LoginRequest != nil {
		return oauth2LoginRequest.Client.GetClientName()
	}
	// Handle Oathkeeper case
	return ""
}

func (s *Service) FilterFlowProviderList(ctx context.Context, flow *kClient.LoginFlow) (*kClient.LoginFlow, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.FilterFlowProviderList")
	defer span.End()

	clientName := s.getClientName(flow)

	allowedProviders, err := s.authz.ListObjects(ctx, fmt.Sprintf("app:%s", clientName), "allowed_access", "provider")
	if err != nil {
		return nil, err
	}

	// If the user has not configured providers for this app, we allow all providers
	if len(allowedProviders) == 0 {
		return flow, nil
	}

	// Filter UI nodes
	var nodes []kClient.UiNode
	for _, node := range flow.Ui.Nodes {
		switch node.Group {
		case "oidc":
			if s.contains(allowedProviders, fmt.Sprintf("%v", node.Attributes.UiNodeInputAttributes.GetValue())) {
				nodes = append(nodes, node)
			}
		}
	}

	flow.Ui.Nodes = nodes
	return flow, nil
}

func (s *Service) ParseIdentifierFirstLoginFlowMethodBody(r *http.Request) (*kClient.UpdateLoginFlowWithIdentifierFirstMethod, []*http.Cookie, error) {
	defer r.Body.Close()

	cookies := r.Cookies()
	var body kClient.UpdateLoginFlowWithIdentifierFirstMethod
	if err := parseBody(r.Body, &body); err != nil {
		return nil, cookies, err
	}
	return &body, cookies, nil
}

func (s *Service) ParseLoginFlowMethodBody(r *http.Request) (*kClient.UpdateLoginFlowBody, []*http.Cookie, error) {
	defer r.Body.Close()

	cookies := r.Cookies()
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, cookies, errors.New("unable to read body")
	}

	// replace the body that was consumed
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	var methodPayload methodOnly
	if err := json.Unmarshal(bodyBytes, &methodPayload); err != nil {
		// fallback where method might be missing
		methodPayload.Method = "webauthn"
	}

	var ret kClient.UpdateLoginFlowBody
	switch method := methodPayload.Method; method {
	case "password":
		var body kClient.UpdateLoginFlowWithPasswordMethod
		if err := parseBody(r.Body, &body); err != nil {
			return nil, cookies, err
		}
		ret = kClient.UpdateLoginFlowWithPasswordMethodAsUpdateLoginFlowBody(&body)

	case "totp":
		var body kClient.UpdateLoginFlowWithTotpMethod
		if err := parseBody(r.Body, &body); err != nil {
			return nil, cookies, err
		}
		body.SetMethod("totp")
		ret = kClient.UpdateLoginFlowWithTotpMethodAsUpdateLoginFlowBody(
			&body,
		)

	case "webauthn":
		var body kClient.UpdateLoginFlowWithWebAuthnMethod
		if r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
			// TODO: fix me. The UI should start sending that data in json format
			if err := r.ParseForm(); err != nil {
				return nil, cookies, err
			}
			csrf := r.Form.Get("csrf_token")
			login := r.Form.Get("webauthn_login")
			identifier := r.Form.Get("identifier")
			body.CsrfToken = &csrf
			body.WebauthnLogin = &login
			body.Identifier = identifier
		} else {
			if err := parseBody(r.Body, &body); err != nil {
				return nil, cookies, err
			}
		}
		ret = kClient.UpdateLoginFlowWithWebAuthnMethodAsUpdateLoginFlowBody(&body)

	case "lookup_secret":
		var body kClient.UpdateLoginFlowWithLookupSecretMethod
		if err := parseBody(r.Body, &body); err != nil {
			return nil, cookies, err
		}
		ret = kClient.UpdateLoginFlowWithLookupSecretMethodAsUpdateLoginFlowBody(&body)

	default:
		// method field is empty for oidc: https://github.com/ory/kratos/pull/3564
		var body kClient.UpdateLoginFlowWithOidcMethod
		if err := parseBody(r.Body, &body); err != nil {
			return nil, cookies, err
		}
		ret = kClient.UpdateLoginFlowWithOidcMethodAsUpdateLoginFlowBody(&body)
	}

	// Remove session cookie if this is a 1FA method
	if s.is1FAMethod(methodPayload.Method) {
		filtered := make([]*http.Cookie, 0)
		for _, c := range cookies {
			if c.Name != KRATOS_SESSION_COOKIE_NAME {
				filtered = append(filtered, c)
			}
		}
		cookies = filtered
	}

	return &ret, cookies, nil
}

func (s *Service) ParseRecoveryFlowMethodBody(r *http.Request) (*kClient.UpdateRecoveryFlowBody, error) {
	body := new(kClient.UpdateRecoveryFlowWithCodeMethod)

	err := parseBody(r.Body, &body)

	if err != nil {
		return nil, err
	}
	ret := kClient.UpdateRecoveryFlowWithCodeMethodAsUpdateRecoveryFlowBody(
		body,
	)

	ret.UpdateRecoveryFlowWithCodeMethod.Method = "code"

	return &ret, nil
}

func (s *Service) ParseSettingsFlowMethodBody(r *http.Request) (*kClient.UpdateSettingsFlowBody, error) {
	defer r.Body.Close()

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.New("unable to read body")
	}

	// replace the body that was consumed
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	var methodPayload methodOnly
	if err := json.Unmarshal(bodyBytes, &methodPayload); err != nil {
		// fallback where method might be missing
		methodPayload.Method = "webauthn"
	}

	var ret kClient.UpdateSettingsFlowBody
	switch methodPayload.Method {
	case "password":
		var body kClient.UpdateSettingsFlowWithPasswordMethod
		if err := parseBody(r.Body, &body); err != nil {
			return nil, err
		}
		ret = kClient.UpdateSettingsFlowWithPasswordMethodAsUpdateSettingsFlowBody(&body)

	case "oidc":
		var body kClient.UpdateSettingsFlowWithOidcMethod
		if err := parseBody(r.Body, &body); err != nil {
			return nil, err
		}
		ret = kClient.UpdateSettingsFlowWithOidcMethodAsUpdateSettingsFlowBody(&body)

	case "totp":
		var body kClient.UpdateSettingsFlowWithTotpMethod
		if err := parseBody(r.Body, &body); err != nil {
			return nil, err
		}
		ret = kClient.UpdateSettingsFlowWithTotpMethodAsUpdateSettingsFlowBody(&body)

	case "webauthn":
		var body kClient.UpdateSettingsFlowWithWebAuthnMethod
		if r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
			// TODO: fix me. The UI should start sending that data in json format
			if err := r.ParseForm(); err != nil {
				return nil, err
			}

			csrf := r.Form.Get("csrf_token")
			name := r.Form.Get("webauthn_register_displayname")
			register := r.Form.Get("webauthn_register")
			remove := r.Form.Get("webauthn_remove")

			body.CsrfToken = &csrf
			body.WebauthnRegister = &register
			body.WebauthnRegisterDisplayname = &name
			body.WebauthnRemove = &remove

		} else if err := parseBody(r.Body, &body); err != nil {
			return nil, err
		}
		ret = kClient.UpdateSettingsFlowWithWebAuthnMethodAsUpdateSettingsFlowBody(&body)

	case "lookup_secret":
		var body kClient.UpdateSettingsFlowWithLookupMethod
		if err := parseBody(r.Body, &body); err != nil {
			return nil, err
		}
		ret = kClient.UpdateSettingsFlowWithLookupMethodAsUpdateSettingsFlowBody(&body)

	default:
		return nil, fmt.Errorf("upsupported method: %s", methodPayload.Method)
	}

	return &ret, nil
}

func (s *Service) contains(str []string, e string) bool {
	for _, a := range str {
		if a == e {
			return true
		}
	}
	return false
}

func (s *Service) HasTOTPAvailable(ctx context.Context, id string) (bool, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.HasTOTPAvailable")
	defer span.End()

	identity, _, err := s.kratosAdmin.IdentityApi().
		GetIdentity(ctx, id).
		IncludeCredential([]string{"totp"}).
		Execute()

	if err != nil {
		return false, err
	}

	_, ok := identity.GetCredentials()["totp"]
	return ok, nil
}

func (s *Service) HasWebAuthnAvailable(ctx context.Context, id string) (bool, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.HasWebAuthnAvailable")
	defer span.End()

	identity, _, err := s.kratosAdmin.IdentityApi().
		GetIdentity(ctx, id).
		IncludeCredential([]string{"webauthn"}).
		Execute()

	if err != nil {
		return false, err
	}

	var (
		webauthnInfo kClient.IdentityCredentials
		ok           = false
	)

	if webauthnInfo, ok = identity.GetCredentials()["webauthn"]; !ok {
		s.logger.Debugf("Identity %s has no credential entries", id)
		return false, nil
	}

	credentialsSlice, ok := webauthnInfo.Config["credentials"].([]interface{})
	if !ok {
		// user has no webauthn keys
		s.logger.Debugf("Identity %s has no webauthn credentials", id)
		return false, nil
	}

	for _, credentialElem := range credentialsSlice {
		credential, ok := credentialElem.(map[string]interface{})
		if !ok {
			continue
		}

		isPasswordless, ok := credential["is_passwordless"]
		if ok && !isPasswordless.(bool) {
			s.logger.Debugf("Identity %s has a 2fa webauthn key", id)
			return true, nil
		}
	}

	return false, nil
}

func (s *Service) HasNotEnoughLookupSecretsLeft(ctx context.Context, id string) (bool, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.HasNotEnoughLookupSecretsLeft")
	defer span.End()

	identity, _, err := s.kratosAdmin.IdentityApi().
		GetIdentity(ctx, id).
		IncludeCredential([]string{"lookup_secret"}).
		Execute()

	if err != nil {
		return false, err
	}

	lookupSecret, ok := identity.GetCredentials()["lookup_secret"]
	if !ok {
		s.logger.Debugf("User has no lookup secret credentials")
		return false, nil
	}

	lookupCredentials, ok := lookupSecret.Config["recovery_codes"]
	if !ok {
		s.logger.Debugf("Recovery codes unavailable")
		return false, nil
	}

	jsonbody, err := json.Marshal(lookupCredentials)
	if err != nil {
		s.logger.Errorf("Marshalling to json failed: %s", err)
		return false, err
	}

	lookupSecrets := new(LookupSecrets)
	if err := json.Unmarshal(jsonbody, &lookupSecrets); err != nil {
		s.logger.Errorf("Unmarshalling failed: %s", err)
		return false, err
	}

	unusedCodes := 0
	for _, code := range *lookupSecrets {
		if code.UsedAt.IsZero() {
			unusedCodes += 1
		}
	}

	if unusedCodes > MinimumBackupCodesAmount {
		return false, nil
	}

	s.logger.Debugf("Only %d backup codes are left, redirect the user to generate a new set", unusedCodes)

	return true, nil
}

func (s *Service) is1FAMethod(method string) bool {
	switch method {
	case "password", "oidc":
		return true
	case "webauthn":
		return !s.oidcWebAuthnSequencingEnabled
	default:
		return false
	}
}

// hydrateKratosLoginFlow hydrates the kratos login flow with information about the oauth2 flow
// that initiated it. This is usefull only if the login_challenge was not sent to kratos when
// creating the flow.
func (s *Service) hydrateKratosLoginFlow(ctx context.Context, flow *kClient.LoginFlow) (*kClient.LoginFlow, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.hydrateKratosLoginFlow")
	defer span.End()

	u, _ := url.Parse(flow.GetReturnTo())
	loginChallenge := u.Query().Get("login_challenge")

	// This is not a hydra request, nothing to do here
	if loginChallenge == "" {
		return flow, nil
	}

	// The flow already contains information about the oauth2 request
	if flow.Oauth2LoginRequest != nil {
		return flow, nil
	}

	b, err := flow.MarshalJSON()
	if err != nil {
		return nil, err
	}

	newFlow := new(kClient.LoginFlow)
	err = newFlow.UnmarshalJSON(b)
	if err != nil {
		return nil, err
	}

	newFlow.Oauth2LoginChallenge = &loginChallenge
	hydraLoginRequest, _, err := s.GetLoginRequest(ctx, loginChallenge)
	if err != nil {
		return newFlow, err
	}

	b, err = hydraLoginRequest.MarshalJSON()
	if err != nil {
		return newFlow, err
	}

	lr := kClient.NewOAuth2LoginRequest()
	lr.UnmarshalJSON(b)
	newFlow.Oauth2LoginRequest = lr
	return newFlow, nil
}

func (s *Service) parseKratosRedirectResponse(ctx context.Context, resp *http.Response) (*BrowserLocationChangeRequired, error) {
	ctx, span := s.tracer.Start(ctx, "kratos.Service.parseKratosRedirectResponse")
	defer span.End()

	redirectResp := new(ErrorBrowserLocationChangeRequired)
	err := unmarshalByteJson(resp.Body, redirectResp)
	if err != nil {
		s.logger.Errorf("Failed to unmarshal JSON: %s", err)
		return nil, err
	}

	return &BrowserLocationChangeRequired{
		RedirectTo: redirectResp.RedirectBrowserTo,
		Error:      redirectResp.Error,
	}, nil
}

func NewService(kratos KratosClientInterface, kratosAdmin KratosAdminClientInterface, hydra HydraClientInterface, authzClient AuthorizerInterface, oidcWebAuthnSequencingEnabled bool, tracer tracing.TracingInterface, monitor monitoring.MonitorInterface, logger logging.LoggerInterface) *Service {
	s := new(Service)

	s.kratos = kratos
	s.kratosAdmin = kratosAdmin
	s.hydra = hydra
	s.authz = authzClient

	s.oidcWebAuthnSequencingEnabled = oidcWebAuthnSequencingEnabled

	s.monitor = monitor
	s.tracer = tracer
	s.logger = logger

	return s
}

func parseBody(b io.ReadCloser, body interface{}) error {
	decoder := json.NewDecoder(b)
	err := decoder.Decode(body)
	return err
}

func unmarshalByteJson(data io.Reader, v any) error {
	json_data, err := io.ReadAll(data)
	if err != nil {
		return err
	}
	err = json.Unmarshal(json_data, v)
	if err != nil {
		return err
	}
	return nil
}

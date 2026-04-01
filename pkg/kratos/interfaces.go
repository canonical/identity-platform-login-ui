package kratos

import (
	"context"
	"net/http"

	kClient "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/internal/cookies"
	"github.com/canonical/identity-platform-login-ui/internal/hydra"
)

type KratosClientInterface interface {
	FrontendApi() kClient.FrontendAPI
	ExecuteIdentifierFirstUpdateLoginRequest(context.Context, string, string, string, []*http.Cookie) (*http.Response, error)
}

type KratosAdminClientInterface interface {
	IdentityApi() kClient.IdentityAPI
}

type HydraClientInterface interface {
	OAuth2API() hydra.OAuth2API
}

type AuthorizerInterface interface {
	ListObjects(context.Context, string, string, string) ([]string, error)
}

type TenantResolverInterface interface {
	// Enabled reports whether tenant selection is active.
	// When false, handlers skip tenant-selection redirects entirely.
	Enabled() bool
	// TenantID returns the tenant ID bound to the given login challenge, or ""
	// if no tenant has been selected or tenant resolution is disabled.
	TenantID(cookie cookies.FlowStateCookie, loginChallenge string) string
	// StoreTenant persists the tenant selection for the given login challenge.
	// When tenant resolution is disabled this is a no-op.
	StoreTenant(w http.ResponseWriter, r *http.Request, tenantID, loginChallenge string) error
	// HasTenants reports whether the authenticated user has any tenants available.
	// Used to decide whether to redirect to the tenant selection page or proceed
	// silently with NoTenantAvailable. Returns false when tenant selection is disabled.
	HasTenants(ctx context.Context, session *kClient.Session) (bool, error)
}

type ServiceInterface interface {
	CheckSession(context.Context, []*http.Cookie) (*kClient.Session, []*http.Cookie, error)
	AcceptLoginRequest(context.Context, *kClient.Session, string, string) (*BrowserLocationChangeRequired, []*http.Cookie, error)
	MustReAuthenticate(context.Context, string, *kClient.Session, cookies.FlowStateCookie) (bool, error)
	CreateBrowserLoginFlow(context.Context, string, string, string, bool, []*http.Cookie) (*kClient.LoginFlow, []*http.Cookie, error)
	CreateBrowserRegistrationFlow(context.Context, string) (*kClient.RegistrationFlow, []*http.Cookie, error)
	CreateBrowserRecoveryFlow(context.Context, string) (*kClient.RecoveryFlow, []*http.Cookie, error)
	CreateBrowserSettingsFlow(context.Context, string, []*http.Cookie) (*kClient.SettingsFlow, *BrowserLocationChangeRequired, error)
	CreateBrowserVerificationFlow(context.Context, []*http.Cookie) (*kClient.VerificationFlow, []*http.Cookie, error)
	GetLoginFlow(context.Context, string, []*http.Cookie) (*kClient.LoginFlow, []*http.Cookie, error)
	GetRegistrationFlow(context.Context, string, []*http.Cookie) (*kClient.RegistrationFlow, []*http.Cookie, error)
	GetRecoveryFlow(context.Context, string, []*http.Cookie) (*kClient.RecoveryFlow, []*http.Cookie, error)
	GetSettingsFlow(context.Context, string, []*http.Cookie) (*kClient.SettingsFlow, *BrowserLocationChangeRequired, error)
	GetVerificationFlow(context.Context, string, []*http.Cookie) (*kClient.VerificationFlow, []*http.Cookie, error)
	UpdateLoginFlow(context.Context, string, kClient.UpdateLoginFlowBody, []*http.Cookie) (*BrowserLocationChangeRequired, *kClient.SuccessfulNativeLogin, []*http.Cookie, error)
	UpdateIdentifierFirstLoginFlow(context.Context, string, kClient.UpdateLoginFlowWithIdentifierFirstMethod, []*http.Cookie) (*BrowserLocationChangeRequired, []*http.Cookie, error)
	UpdateRegistrationFlow(context.Context, string, kClient.UpdateRegistrationFlowBody, []*http.Cookie) (*RegistrationFlowResponse, []*http.Cookie, error)
	UpdateRecoveryFlow(context.Context, string, kClient.UpdateRecoveryFlowBody, []*http.Cookie) (*BrowserLocationChangeRequired, []*http.Cookie, error)
	UpdateSettingsFlow(context.Context, string, kClient.UpdateSettingsFlowBody, []*http.Cookie) (*kClient.SettingsFlow, *BrowserLocationChangeRequired, []*http.Cookie, error)
	UpdateVerificationFlow(context.Context, string, kClient.UpdateVerificationFlowBody, []*http.Cookie) (*kClient.VerificationFlow, []*http.Cookie, error)
	GetFlowError(context.Context, string) (*kClient.FlowError, []*http.Cookie, error)
	CheckAllowedProvider(context.Context, *kClient.LoginFlow, *kClient.UpdateLoginFlowBody) (bool, error)
	FilterFlowProviderList(context.Context, *kClient.LoginFlow) (*kClient.LoginFlow, error)
	ParseLoginFlowMethodBody(*http.Request) (*kClient.UpdateLoginFlowBody, []*http.Cookie, error)
	ParseIdentifierFirstLoginFlowMethodBody(*http.Request) (*kClient.UpdateLoginFlowWithIdentifierFirstMethod, []*http.Cookie, error)
	ParseRegistrationFlowMethodBody(*http.Request) (*kClient.UpdateRegistrationFlowBody, error)
	ParseRecoveryFlowMethodBody(*http.Request) (*kClient.UpdateRecoveryFlowBody, error)
	ParseSettingsFlowMethodBody(*http.Request) (*kClient.UpdateSettingsFlowBody, error)
	HasTOTPAvailable(context.Context, string) (bool, error)
	HasWebAuthnAvailable(context.Context, string) (bool, error)
	HasNotEnoughLookupSecretsLeft(context.Context, string) (bool, error)
	RequireVerificationForEmail(context.Context, *kClient.Session) (bool, string, error)
}

type AuthCookieManagerInterface interface {
	// SetStateCookie sets the nonce cookie on the response with the specified duration as MaxAge
	SetStateCookie(http.ResponseWriter, cookies.FlowStateCookie) error
	// GetStateCookie returns the string value of the nonce cookie if present, or empty string otherwise
	GetStateCookie(*http.Request) (cookies.FlowStateCookie, error)
	// ClearStateCookie sets the expiration of the cookie to epoch
	ClearStateCookie(http.ResponseWriter)
}

type RedirectToInterface interface {
	GetCode() int
	GetRedirectTo() string
}

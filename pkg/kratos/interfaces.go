package kratos

import (
	"context"
	"net/http"

	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go"

	"github.com/canonical/identity-platform-login-ui/internal/hydra"
)

type KratosClientInterface interface {
	FrontendApi() kClient.FrontendApi
}

type KratosAdminClientInterface interface {
	IdentityApi() kClient.IdentityApi
}

type HydraClientInterface interface {
	OAuth2Api() hydra.OAuth2Api
}

type AuthorizerInterface interface {
	ListObjects(context.Context, string, string, string) ([]string, error)
}

type ServiceInterface interface {
	CheckSession(context.Context, []*http.Cookie) (*kClient.Session, []*http.Cookie, error)
	AcceptLoginRequest(context.Context, string, string) (*hClient.OAuth2RedirectTo, []*http.Cookie, error)
	MustReAuthenticate(context.Context, string, *kClient.Session, FlowStateCookie) (bool, error)
	CreateBrowserLoginFlow(context.Context, string, string, string, bool, []*http.Cookie) (*kClient.LoginFlow, []*http.Cookie, error)
	CreateBrowserRecoveryFlow(context.Context, string, []*http.Cookie) (*kClient.RecoveryFlow, []*http.Cookie, error)
	CreateBrowserSettingsFlow(context.Context, string, []*http.Cookie) (*kClient.SettingsFlow, *BrowserLocationChangeRequired, error)
	GetLoginFlow(context.Context, string, []*http.Cookie) (*kClient.LoginFlow, []*http.Cookie, error)
	GetRecoveryFlow(context.Context, string, []*http.Cookie) (*kClient.RecoveryFlow, []*http.Cookie, error)
	GetSettingsFlow(context.Context, string, []*http.Cookie) (*kClient.SettingsFlow, *BrowserLocationChangeRequired, error)
	UpdateLoginFlow(context.Context, string, kClient.UpdateLoginFlowBody, []*http.Cookie) (*BrowserLocationChangeRequired, []*http.Cookie, error)
	UpdateRecoveryFlow(context.Context, string, kClient.UpdateRecoveryFlowBody, []*http.Cookie) (*BrowserLocationChangeRequired, []*http.Cookie, error)
	UpdateSettingsFlow(context.Context, string, kClient.UpdateSettingsFlowBody, []*http.Cookie) (*kClient.SettingsFlow, []*http.Cookie, error)
	GetFlowError(context.Context, string) (*kClient.FlowError, []*http.Cookie, error)
	CheckAllowedProvider(context.Context, *kClient.LoginFlow, *kClient.UpdateLoginFlowBody) (bool, error)
	FilterFlowProviderList(context.Context, *kClient.LoginFlow) (*kClient.LoginFlow, error)
	ParseLoginFlowMethodBody(*http.Request) (*kClient.UpdateLoginFlowBody, []*http.Cookie, error)
	ParseRecoveryFlowMethodBody(*http.Request) (*kClient.UpdateRecoveryFlowBody, error)
	ParseSettingsFlowMethodBody(*http.Request) (*kClient.UpdateSettingsFlowBody, error)
	HasTOTPAvailable(context.Context, string) (bool, error)
	HasNotEnoughLookupSecretsLeft(context.Context, string) (bool, error)
}

type AuthCookieManagerInterface interface {
	// SetStateCookie sets the nonce cookie on the response with the specified duration as MaxAge
	SetStateCookie(http.ResponseWriter, FlowStateCookie) error
	// GetStateCookie returns the string value of the nonce cookie if present, or empty string otherwise
	GetStateCookie(*http.Request) (FlowStateCookie, error)
	// ClearStateCookie sets the expiration of the cookie to epoch
	ClearStateCookie(http.ResponseWriter)
}

type EncryptInterface interface {
	// Encrypt a plain text string, returns the encrypted string in hex format or an error
	Encrypt(string) (string, error)
	// Decrypt a hex string, returns the decrypted string or an error
	Decrypt(string) (string, error)
}

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

type HydraClientInterface interface {
	OAuth2Api() hydra.OAuth2Api
}

type AuthorizerInterface interface {
	ListObjects(context.Context, string, string, string) ([]string, error)
}

type ServiceInterface interface {
	CheckSession(context.Context, []*http.Cookie) (*kClient.Session, []*http.Cookie, error)
	AcceptLoginRequest(context.Context, string, string) (*hClient.OAuth2RedirectTo, []*http.Cookie, error)
	CreateBrowserLoginFlow(context.Context, string, string, string, bool, []*http.Cookie) (*kClient.LoginFlow, []*http.Cookie, error)
	CreateBrowserRecoveryFlow(context.Context, string, []*http.Cookie) (*kClient.RecoveryFlow, []*http.Cookie, error)
	GetLoginFlow(context.Context, string, []*http.Cookie) (*kClient.LoginFlow, []*http.Cookie, error)
	GetRecoveryFlow(context.Context, string, []*http.Cookie) (*kClient.RecoveryFlow, []*http.Cookie, error)
	UpdateLoginFlow(context.Context, string, kClient.UpdateLoginFlowBody, []*http.Cookie) (*BrowserLocationChangeRequired, []*http.Cookie, error)
	UpdateRecoveryFlow(context.Context, string, kClient.UpdateRecoveryFlowBody, []*http.Cookie) (*BrowserLocationChangeRequired, []*http.Cookie, error)
	GetFlowError(context.Context, string) (*kClient.FlowError, []*http.Cookie, error)
	CheckAllowedProvider(context.Context, *kClient.LoginFlow, *kClient.UpdateLoginFlowBody) (bool, error)
	FilterFlowProviderList(context.Context, *kClient.LoginFlow) (*kClient.LoginFlow, error)
	ParseLoginFlowMethodBody(*http.Request) (*kClient.UpdateLoginFlowBody, error)
	ParseRecoveryFlowMethodBody(*http.Request) (*kClient.UpdateRecoveryFlowBody, error)
}

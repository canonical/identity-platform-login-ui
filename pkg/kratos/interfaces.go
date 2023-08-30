package kratos

import (
	"context"
	"net/http"

	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go"
)

type KratosClientInterface interface {
	FrontendApi() kClient.FrontendApi
}

type HydraClientInterface interface {
	OAuth2Api() hClient.OAuth2Api
}

type ServiceInterface interface {
	CheckSession(context.Context, []*http.Cookie) (*kClient.Session, []*http.Cookie, error)
	AcceptLoginRequest(context.Context, string, string) (*hClient.OAuth2RedirectTo, []*http.Cookie, error)
	CreateBrowserLoginFlow(context.Context, string, string, string, bool, []*http.Cookie) (*kClient.LoginFlow, []*http.Cookie, error)
	GetLoginFlow(context.Context, string, []*http.Cookie) (*kClient.LoginFlow, []*http.Cookie, error)
	UpdateOIDCLoginFlow(context.Context, string, kClient.UpdateLoginFlowBody, []*http.Cookie) (*BrowserLocationChangeRequired, []*http.Cookie, error)
	GetFlowError(context.Context, string) (*kClient.FlowError, []*http.Cookie, error)
	ParseLoginFlowMethodBody(*http.Request) (*kClient.UpdateLoginFlowBody, error)
}

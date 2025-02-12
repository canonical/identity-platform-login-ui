package device

import (
	"context"
	"net/http"

	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go"

	"github.com/canonical/identity-platform-login-ui/internal/hydra"
)

type KratosClientInterface interface {
	FrontendApi() kClient.FrontendAPI
}

type HydraClientInterface interface {
	OAuth2API() hydra.OAuth2API
}

type ServiceInterface interface {
	AcceptUserCode(context.Context, string, *hydra.AcceptDeviceUserCodeRequest) (*hClient.OAuth2RedirectTo, error)
	ParseUserCodeBody(*http.Request) (*hydra.AcceptDeviceUserCodeRequest, error)
}

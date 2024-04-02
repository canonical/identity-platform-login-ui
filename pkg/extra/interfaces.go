package extra

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

type ServiceInterface interface {
	CheckSession(context.Context, []*http.Cookie) (*kClient.Session, error)
	GetConsent(context.Context, string) (*hClient.OAuth2ConsentRequest, error)
	AcceptConsent(context.Context, kClient.Identity, *hClient.OAuth2ConsentRequest) (*hClient.OAuth2RedirectTo, error)
}

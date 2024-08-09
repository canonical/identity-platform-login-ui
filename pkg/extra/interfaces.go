package extra

import (
	"context"

	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go"

	"github.com/canonical/identity-platform-login-ui/internal/hydra"
)

type HydraClientInterface interface {
	OAuth2Api() hydra.OAuth2Api
}

type ServiceInterface interface {
	GetConsent(context.Context, string) (*hClient.OAuth2ConsentRequest, error)
	AcceptConsent(context.Context, kClient.Identity, *hClient.OAuth2ConsentRequest) (*hClient.OAuth2RedirectTo, error)
}

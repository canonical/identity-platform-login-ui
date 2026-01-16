package extra

import (
	"context"

	hClient "github.com/ory/hydra-client-go/v2"
	kClient "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/internal/hydra"
)

type HydraClientInterface interface {
	OAuth2API() hydra.OAuth2API
}

type ServiceInterface interface {
	GetConsent(context.Context, string) (*hClient.OAuth2ConsentRequest, error)
	AcceptConsent(context.Context, kClient.Identity, *hClient.OAuth2ConsentRequest) (*hClient.OAuth2RedirectTo, error)
}

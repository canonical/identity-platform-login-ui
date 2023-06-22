package extra

import (
	hydra_client "github.com/ory/hydra-client-go/v2"
	kratos_client "github.com/ory/kratos-client-go"
)

type KratosClientInterface interface {
	FrontendApi() kratos_client.FrontendApi
}

type HydraClientInterface interface {
	OAuth2Api() hydra_client.OAuth2Api
}

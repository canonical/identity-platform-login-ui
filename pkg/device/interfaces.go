package device

import (
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
	AcceptUserCode(string, *DeviceCodeRequest) (*DeviceCodeResponse, error)
	ParseUserCodeBody(*http.Request) (*DeviceCodeRequest, error)
}

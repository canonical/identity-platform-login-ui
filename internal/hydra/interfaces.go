package hydra

import (
	"context"
	"net/http"

	hClient "github.com/ory/hydra-client-go/v2"
)

// We implement the device API logic, because the upstream sdk does not support it.
// Otherwise we would have to fork the sdk.
// TODO(nsklikas): Remove this once upstream hydra supports the device flow
type DeviceApi interface {
	AcceptUserCodeRequest(context.Context) ApiAcceptUserCodeRequestRequest
	AcceptUserCodeRequestExecute(ApiAcceptUserCodeRequestRequest) (*hClient.OAuth2RedirectTo, *http.Response, error)
}

type OAuth2Api interface {
	hClient.OAuth2Api
	DeviceApi
}

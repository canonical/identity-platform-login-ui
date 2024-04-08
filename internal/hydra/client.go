package hydra

import (
	"net/http"

	hClient "github.com/ory/hydra-client-go/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Client struct {
	c         *hClient.APIClient
	deviceApi *DeviceApiService
}

func (c *Client) OAuth2Api() OAuth2Api {
	return c.deviceApi
}

func (c *Client) MetadataApi() hClient.MetadataApi {
	return c.c.MetadataApi
}

func NewClient(url string, debug bool) *Client {
	c := new(Client)

	configuration := hClient.NewConfiguration()
	configuration.Debug = debug
	configuration.Servers = []hClient.ServerConfiguration{
		{
			URL: url,
		},
	}

	configuration.HTTPClient = &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	c.c = hClient.NewAPIClient(configuration)
	c.deviceApi = newDeviceApiService(c.c)

	return c
}

package hydra

import (
	"net/http"

	client "github.com/ory/hydra-client-go/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Client struct {
	c *client.APIClient
}

func (c *Client) OAuth2Api() client.OAuth2Api {
	return c.c.OAuth2Api
}

func (c *Client) MetadataApi() client.MetadataApi {
	return c.c.MetadataApi
}

func NewClient(url string) *Client {
	c := new(Client)

	configuration := client.NewConfiguration()

	configuration.Debug = true
	configuration.Servers = []client.ServerConfiguration{
		{
			URL: url,
		},
	}

	configuration.HTTPClient = &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	c.c = client.NewAPIClient(configuration)

	return c
}

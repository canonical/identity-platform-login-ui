package kratos

import (
	"net/http"

	client "github.com/ory/kratos-client-go"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Client struct {
	c *client.APIClient
}

func (c *Client) FrontendApi() client.FrontendAPI {
	return c.c.FrontendAPI
}

func (c *Client) IdentityApi() client.IdentityAPI {
	return c.c.IdentityAPI
}

func (c *Client) MetadataApi() client.MetadataAPI {
	return c.c.MetadataAPI
}

func NewClient(url string, debug bool) *Client {
	c := new(Client)

	configuration := client.NewConfiguration()
	configuration.Debug = debug
	configuration.Servers = []client.ServerConfiguration{
		{
			URL: url,
		},
	}

	configuration.HTTPClient = &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	c.c = client.NewAPIClient(configuration)

	return c
}

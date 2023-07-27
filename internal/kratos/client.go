package kratos

import (
	"net/http"

	client "github.com/ory/kratos-client-go"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Client struct {
	c *client.APIClient
}

func (c *Client) FrontendApi() client.FrontendApi {
	return c.c.FrontendApi
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

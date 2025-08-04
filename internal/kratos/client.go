package kratos

import (
	"net/http"

	client "github.com/ory/kratos-client-go"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Client struct {
	c          *client.APIClient
	httpClient *http.Client
}

func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
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
	configuration := client.NewConfiguration()
	configuration.Debug = debug
	configuration.Servers = []client.ServerConfiguration{
		{
			URL: url,
		},
	}

	httpClient := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	configuration.HTTPClient = httpClient

	return &Client{
		c:          client.NewAPIClient(configuration),
		httpClient: httpClient,
	}
}

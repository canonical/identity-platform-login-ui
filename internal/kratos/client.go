package kratos

import (
	client "github.com/ory/kratos-client-go"
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

	c.c = client.NewAPIClient(configuration)

	return c
}

package hydra

import (
	client "github.com/ory/hydra-client-go/v2"
)

type Client struct {
	c *client.APIClient
}

func (c *Client) OAuth2Api() client.OAuth2Api {
	return c.c.OAuth2Api
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

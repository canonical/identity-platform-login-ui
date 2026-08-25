// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package hydra

import (
	"net/http"

	hClient "github.com/ory/hydra-client-go/v26"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Client struct {
	c *hClient.APIClient
}

func (c *Client) OAuth2API() OAuth2API {
	return c.c.OAuth2API
}

func (c *Client) MetadataAPI() hClient.MetadataAPI {
	return c.c.MetadataAPI
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

	return c
}

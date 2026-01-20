package kratos

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	client "github.com/ory/kratos-client-go/v25"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Client struct {
	c          *client.APIClient
	httpClient *http.Client
	loginURL   *url.URL
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

func (c *Client) newIdentifierFirstUpdateLoginRequest(
	ctx context.Context, flow, csrfToken, identifier string, cookies []*http.Cookie,
) (*http.Request, error) {
	form := url.Values{}
	form.Set("csrf_token", csrfToken)
	form.Set("identifier", identifier)
	form.Set("method", "identifier_first")

	// Clone URL and add flow parameter
	u := *c.loginURL
	q := u.Query()
	q.Set("flow", flow)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	return req, nil
}

func (c *Client) ExecuteIdentifierFirstUpdateLoginRequest(
	ctx context.Context, flow string, csrfToken, identifier string, cookies []*http.Cookie,
) (*http.Response, error) {
	req, err := c.newIdentifierFirstUpdateLoginRequest(ctx, flow, csrfToken, identifier, cookies)
	if err != nil {
		return nil, err
	}

	client := *c.httpClient
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		// Kratos returns 303 for identifier first flows. We disable these automatic redirects
		// in order to let the handler process the BrowserLocationChangeRequired and change to 200
		return http.ErrUseLastResponse
	}
	return client.Do(req)
}

func NewClient(baseURL string, debug bool) *Client {
	configuration := client.NewConfiguration()
	configuration.Debug = debug
	configuration.Servers = []client.ServerConfiguration{
		{
			URL: baseURL,
		},
	}

	httpClient := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	configuration.HTTPClient = httpClient

	loginURL, err := url.Parse(baseURL + "/self-service/login")
	if err != nil {
		panic(fmt.Sprintf("invalid kratos login url: %v", err))
	}

	return &Client{
		c:          client.NewAPIClient(configuration),
		httpClient: httpClient,
		loginURL:   loginURL,
	}
}

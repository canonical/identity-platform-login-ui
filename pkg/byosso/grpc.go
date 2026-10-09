// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// connectParams reconnect sooner than grpc-go does by default (1s growing to
// 120s): a backend that restarted is reached again within seconds, where the
// default can leave login-ui refusing sign-ins for two minutes.
var connectParams = grpc.ConnectParams{
	Backoff: backoff.Config{
		BaseDelay:  reconnectBaseDelay,
		Multiplier: 1.6,
		Jitter:     0.2,
		MaxDelay:   reconnectMaxDelay,
	},
	MinConnectTimeout: connectTimeout,
}

// retryPolicy tries a call again, twice at most, when it ends UNAVAILABLE:
// the service could not be reached, or answered that something it depends on
// could not. The call may have run all the same, so only calls that are safe
// to repeat get it.
const retryPolicy = `{"maxAttempts": 3, "initialBackoff": "0.1s", "maxBackoff": "1s", "backoffMultiplier": 2, "retryableStatusCodes": ["UNAVAILABLE"]}`

// dialOptions make a channel reconnect sooner, retry the given methods and
// send the service token with every call.
func dialOptions(tokens oauth2.TokenSource, retried []string) []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithConnectParams(connectParams),
		grpc.WithDefaultServiceConfig(retryServiceConfig(retried)),
		grpc.WithPerRPCCredentials(bearerCredentials{tokens: tokens}),
	}
}

// retryServiceConfig returns the gRPC service config applying retryPolicy to
// the given methods, named as the generated clients name them
// ("/package.Service/Method").
func retryServiceConfig(fullMethodNames []string) string {
	names := make([]string, 0, len(fullMethodNames))
	for _, name := range fullMethodNames {
		service, method, _ := strings.Cut(strings.TrimPrefix(name, "/"), "/")
		names = append(names, fmt.Sprintf(`{"service": %q, "method": %q}`, service, method))
	}
	return fmt.Sprintf(`{"methodConfig": [{"name": [%s], "retryPolicy": %s}]}`, strings.Join(names, ", "), retryPolicy)
}

// NewServiceTokenSource returns a cached client-credentials token source for
// the calls to tenant-service and sso-service.
func NewServiceTokenSource(tokenURL, clientID, clientSecret string, scopes []string) oauth2.TokenSource {
	return newTokenSource(tokenTimeout, tokenURL, clientID, clientSecret, scopes)
}

// newTokenSource fetches tokens with an HTTP client of its own: the default
// one of oauth2 has no timeout, and every call to tenant-service and
// sso-service waits for the token.
func newTokenSource(timeout time.Duration, tokenURL, clientID, clientSecret string, scopes []string) oauth2.TokenSource {
	cfg := clientcredentials.Config{ClientID: clientID, ClientSecret: clientSecret, TokenURL: tokenURL, Scopes: scopes}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{Timeout: timeout})
	return oauth2.ReuseTokenSource(nil, cfg.TokenSource(ctx))
}

// bearerCredentials sends `authorization: Bearer <token>` with every call of
// a channel, on a plaintext connection too: the gRPC listeners of the
// services are not TLS in every deployment.
type bearerCredentials struct {
	tokens oauth2.TokenSource
}

// GetRequestMetadata fails the call with the status of the token request: a
// token endpoint that cannot be reached, or answers with a server error, is
// an outage (UNAVAILABLE, so the call is retried and reported like one); one
// that refuses the client is not.
func (b bearerCredentials) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	t, err := b.tokens.Token()
	if err != nil {
		code := codes.Unavailable
		var refused *oauth2.RetrieveError
		if errors.As(err, &refused) && refused.Response != nil && refused.Response.StatusCode < http.StatusInternalServerError {
			code = codes.Unauthenticated
		}
		return nil, status.Errorf(code, "cannot get a service token: %v", err)
	}
	return map[string]string{"authorization": "Bearer " + strings.TrimSpace(t.AccessToken)}, nil
}

func (bearerCredentials) RequireTransportSecurity() bool { return false }

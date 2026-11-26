// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
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

// dialOptions make a channel reconnect sooner and retry the given methods.
func dialOptions(retried []string) []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithConnectParams(connectParams),
		grpc.WithDefaultServiceConfig(retryServiceConfig(retried)),
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

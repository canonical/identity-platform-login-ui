// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package extra

import (
	"context"

	hClient "github.com/ory/hydra-client-go/v26"
	kClient "github.com/ory/kratos-client-go/v25"
)

// Option configures optional API behaviour.
type Option func(*API)

// WithExtension plugs an extension into the consent handler.
func WithExtension(ext ExtensionInterface) Option {
	return func(a *API) {
		a.ext = ext
	}
}

// NoOpExtension is used when no extension is configured. It never refuses a
// consent.
type NoOpExtension struct{}

func NewNoOpExtension() *NoOpExtension {
	return &NoOpExtension{}
}

func (n *NoOpExtension) GateConsent(_ context.Context, _ *kClient.Session, _ *hClient.OAuth2ConsentRequest) (string, error) {
	return "", nil
}

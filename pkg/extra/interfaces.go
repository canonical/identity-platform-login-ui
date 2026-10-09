// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package extra

import (
	"context"

	hClient "github.com/ory/hydra-client-go/v26"
	kClient "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/internal/hydra"
	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

type HydraClientInterface interface {
	OAuth2API() hydra.OAuth2API
}

type ServiceInterface interface {
	GetConsent(context.Context, string) (*hClient.OAuth2ConsentRequest, error)
	AcceptConsent(context.Context, kClient.Identity, *hClient.OAuth2ConsentRequest, string) (*hClient.OAuth2RedirectTo, error)
}

type SecondFactorPolicyInterface interface {
	// For returns what the given sign-in still needs before it may complete.
	For(kratos.SignIn) kratos.Requirement
}

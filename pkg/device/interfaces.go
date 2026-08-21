// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package device

import (
	"context"
	"net/http"

	hClient "github.com/ory/hydra-client-go/v26"
	kClient "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/internal/hydra"
)

type KratosClientInterface interface {
	FrontendApi() kClient.FrontendAPI
}

type HydraClientInterface interface {
	OAuth2API() hydra.OAuth2API
}

type ServiceInterface interface {
	AcceptUserCode(context.Context, string, *hClient.AcceptDeviceUserCodeRequest) (*hClient.OAuth2RedirectTo, error)
	ParseUserCodeBody(*http.Request) (*hClient.AcceptDeviceUserCodeRequest, error)
}

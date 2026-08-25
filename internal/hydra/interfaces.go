// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package hydra

import (
	hClient "github.com/ory/hydra-client-go/v26"
)

type OAuth2API interface {
	hClient.OAuth2API
}

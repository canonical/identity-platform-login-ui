// Copyright 2026 Canonical Ltd
// SPDX-License-Identifier: AGPL-3.0-only

package status

import (
	"context"
)

type ServiceInterface interface {
	KratosStatus(context.Context) bool
	HydraStatus(context.Context) bool
	BuildInfo(context.Context) *BuildInfo
}

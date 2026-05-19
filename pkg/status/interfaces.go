// Copyright 2026 Canonical Ltd
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"context"
)

type ServiceInterface interface {
	KratosStatus(context.Context) bool
	HydraStatus(context.Context) bool
	BuildInfo(context.Context) *BuildInfo
}

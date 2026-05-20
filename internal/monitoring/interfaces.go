// Copyright 2026 Canonical Ltd
// SPDX-License-Identifier: AGPL-3.0-only

package monitoring

type MonitorInterface interface {
	GetService() string
	SetResponseTimeMetric(map[string]string, float64) error
	SetDependencyAvailability(map[string]string, float64) error
}

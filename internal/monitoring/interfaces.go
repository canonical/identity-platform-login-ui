// Copyright 2026 Canonical Ltd
// SPDX-License-Identifier: Apache-2.0

package monitoring

type MonitorInterface interface {
	GetService() string
	SetResponseTimeMetric(map[string]string, float64) error
	SetDependencyAvailability(map[string]string, float64) error
}

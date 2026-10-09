// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package extra

import (
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

func TestWithExtension(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := NewMockLoggerInterface(ctrl)
	mockService := NewMockServiceInterface(ctrl)
	mockKratosService := kratos.NewMockServiceInterface(ctrl)
	mockTracer := NewMockTracingInterface(ctrl)
	mockExtension := NewMockExtensionInterface(ctrl)

	api := NewAPI(mockService, mockKratosService, BASE_URL, false, false, mockTracer, mockLogger, WithExtension(mockExtension))

	if api.ext != mockExtension {
		t.Fatalf("expected the extension to be set")
	}
}

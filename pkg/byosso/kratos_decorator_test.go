// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"testing"

	kClient "github.com/ory/kratos-client-go/v25"
	"go.uber.org/mock/gomock"

	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

// providerNodes returns the nodes of the company sign-in provider and of a
// public one, as Kratos lists them in a flow.
func providerNodes() []kClient.UiNode {
	return []kClient.UiNode{
		inputNode("default", "csrf_token", "csrf-1"),
		inputNode("oidc", "provider", providerID),
		inputNode("oidc", "provider", "google"),
		inputNode("oidc", "link", providerID),
		inputNode("oidc", "unlink", providerID),
	}
}

func TestWithoutProvider(t *testing.T) {
	nodes := withoutProvider(providerNodes())

	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if name := nodes[0].Attributes.UiNodeInputAttributes.Name; name != "csrf_token" {
		t.Fatalf("expected node csrf_token, got %s", name)
	}
	if value := nodes[1].Attributes.UiNodeInputAttributes.Value; value != "google" {
		t.Fatalf("expected provider google, got %v", value)
	}
}

func TestKratosServiceDecorator(t *testing.T) {
	ctx := context.Background()
	ui := func() kClient.UiContainer { return kClient.UiContainer{Nodes: providerNodes()} }

	// every method that returns a flow hides the company sign-in provider
	tests := []struct {
		name string
		call func(mockService *kratos.MockServiceInterface, decorator *KratosServiceDecorator) ([]kClient.UiNode, error)
	}{
		{name: "CreateBrowserLoginFlow", call: func(mockService *kratos.MockServiceInterface, decorator *KratosServiceDecorator) ([]kClient.UiNode, error) {
			mockService.EXPECT().CreateBrowserLoginFlow(ctx, "", "", "", false, nil).Times(1).Return(&kClient.LoginFlow{Ui: ui()}, nil, nil)
			flow, _, err := decorator.CreateBrowserLoginFlow(ctx, "", "", "", false, nil)
			return flow.Ui.Nodes, err
		}},
		{name: "GetLoginFlow", call: func(mockService *kratos.MockServiceInterface, decorator *KratosServiceDecorator) ([]kClient.UiNode, error) {
			mockService.EXPECT().GetLoginFlow(ctx, "f", nil).Times(1).Return(&kClient.LoginFlow{Ui: ui()}, nil, nil)
			flow, _, err := decorator.GetLoginFlow(ctx, "f", nil)
			return flow.Ui.Nodes, err
		}},
		{name: "FilterFlowProviderList", call: func(mockService *kratos.MockServiceInterface, decorator *KratosServiceDecorator) ([]kClient.UiNode, error) {
			mockService.EXPECT().FilterFlowProviderList(ctx, gomock.Any()).Times(1).Return(&kClient.LoginFlow{Ui: ui()}, nil)
			flow, err := decorator.FilterFlowProviderList(ctx, &kClient.LoginFlow{Ui: ui()})
			return flow.Ui.Nodes, err
		}},
		{name: "CreateBrowserRegistrationFlow", call: func(mockService *kratos.MockServiceInterface, decorator *KratosServiceDecorator) ([]kClient.UiNode, error) {
			mockService.EXPECT().CreateBrowserRegistrationFlow(ctx, "").Times(1).Return(&kClient.RegistrationFlow{Ui: ui()}, nil, nil)
			flow, _, err := decorator.CreateBrowserRegistrationFlow(ctx, "")
			return flow.Ui.Nodes, err
		}},
		{name: "GetRegistrationFlow", call: func(mockService *kratos.MockServiceInterface, decorator *KratosServiceDecorator) ([]kClient.UiNode, error) {
			mockService.EXPECT().GetRegistrationFlow(ctx, "f", nil).Times(1).Return(&kClient.RegistrationFlow{Ui: ui()}, nil, nil)
			flow, _, err := decorator.GetRegistrationFlow(ctx, "f", nil)
			return flow.Ui.Nodes, err
		}},
		{name: "CreateBrowserSettingsFlow", call: func(mockService *kratos.MockServiceInterface, decorator *KratosServiceDecorator) ([]kClient.UiNode, error) {
			mockService.EXPECT().CreateBrowserSettingsFlow(ctx, "", nil).Times(1).Return(&kClient.SettingsFlow{Ui: ui()}, nil, nil)
			flow, _, err := decorator.CreateBrowserSettingsFlow(ctx, "", nil)
			return flow.Ui.Nodes, err
		}},
		{name: "GetSettingsFlow", call: func(mockService *kratos.MockServiceInterface, decorator *KratosServiceDecorator) ([]kClient.UiNode, error) {
			mockService.EXPECT().GetSettingsFlow(ctx, "f", nil).Times(1).Return(&kClient.SettingsFlow{Ui: ui()}, nil, nil)
			flow, _, err := decorator.GetSettingsFlow(ctx, "f", nil)
			return flow.Ui.Nodes, err
		}},
		{name: "UpdateSettingsFlow", call: func(mockService *kratos.MockServiceInterface, decorator *KratosServiceDecorator) ([]kClient.UiNode, error) {
			mockService.EXPECT().UpdateSettingsFlow(ctx, "f", gomock.Any(), nil).Times(1).Return(&kClient.SettingsFlow{Ui: ui()}, nil, nil, nil)
			flow, _, _, err := decorator.UpdateSettingsFlow(ctx, "f", kClient.UpdateSettingsFlowBody{}, nil)
			return flow.Ui.Nodes, err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockService := kratos.NewMockServiceInterface(ctrl)

			nodes, err := tt.call(mockService, NewKratosServiceDecorator(mockService))

			expectNoError(t, err)
			if len(nodes) != 2 {
				t.Fatalf("expected 2 nodes, got %d", len(nodes))
			}
		})
	}
}

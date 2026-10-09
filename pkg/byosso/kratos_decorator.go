// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"context"
	"fmt"
	"net/http"

	kClient "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

// KratosServiceDecorator decorates the kratos service so that the nodes of
// the company sign-in provider never reach the frontend, in every login,
// registration and settings flow: only login-ui submits that provider (with a
// ticket), it never registers, and its links are managed through sso-service.
type KratosServiceDecorator struct {
	kratos.ServiceInterface
}

func NewKratosServiceDecorator(service kratos.ServiceInterface) *KratosServiceDecorator {
	return &KratosServiceDecorator{ServiceInterface: service}
}

func (s *KratosServiceDecorator) CreateBrowserLoginFlow(ctx context.Context, aal, returnTo, loginChallenge string, refresh bool, cookies []*http.Cookie) (*kClient.LoginFlow, []*http.Cookie, error) {
	flow, cookies, err := s.ServiceInterface.CreateBrowserLoginFlow(ctx, aal, returnTo, loginChallenge, refresh, cookies)
	if flow != nil {
		flow.Ui.Nodes = withoutProvider(flow.Ui.Nodes)
	}
	return flow, cookies, err
}

func (s *KratosServiceDecorator) GetLoginFlow(ctx context.Context, id string, cookies []*http.Cookie) (*kClient.LoginFlow, []*http.Cookie, error) {
	flow, cookies, err := s.ServiceInterface.GetLoginFlow(ctx, id, cookies)
	if flow != nil {
		flow.Ui.Nodes = withoutProvider(flow.Ui.Nodes)
	}
	return flow, cookies, err
}

func (s *KratosServiceDecorator) FilterFlowProviderList(ctx context.Context, flow *kClient.LoginFlow) (*kClient.LoginFlow, error) {
	flow, err := s.ServiceInterface.FilterFlowProviderList(ctx, flow)
	if flow != nil {
		flow.Ui.Nodes = withoutProvider(flow.Ui.Nodes)
	}
	return flow, err
}

func (s *KratosServiceDecorator) CreateBrowserRegistrationFlow(ctx context.Context, returnTo string) (*kClient.RegistrationFlow, []*http.Cookie, error) {
	flow, cookies, err := s.ServiceInterface.CreateBrowserRegistrationFlow(ctx, returnTo)
	if flow != nil {
		flow.Ui.Nodes = withoutProvider(flow.Ui.Nodes)
	}
	return flow, cookies, err
}

func (s *KratosServiceDecorator) GetRegistrationFlow(ctx context.Context, id string, cookies []*http.Cookie) (*kClient.RegistrationFlow, []*http.Cookie, error) {
	flow, cookies, err := s.ServiceInterface.GetRegistrationFlow(ctx, id, cookies)
	if flow != nil {
		flow.Ui.Nodes = withoutProvider(flow.Ui.Nodes)
	}
	return flow, cookies, err
}

func (s *KratosServiceDecorator) UpdateRegistrationFlow(ctx context.Context, id string, body kClient.UpdateRegistrationFlowBody, cookies []*http.Cookie) (*kratos.RegistrationFlowResponse, []*http.Cookie, error) {
	resp, cookies, err := s.ServiceInterface.UpdateRegistrationFlow(ctx, id, body, cookies)
	if resp != nil {
		if flow, _ := resp.GetFlowAndStatus(); flow != nil {
			if rf, ok := flow.(*kClient.RegistrationFlow); ok && rf != nil {
				rf.Ui.Nodes = withoutProvider(rf.Ui.Nodes)
			}
		}
	}
	return resp, cookies, err
}

func (s *KratosServiceDecorator) CreateBrowserSettingsFlow(ctx context.Context, returnTo string, cookies []*http.Cookie) (*kClient.SettingsFlow, *kratos.BrowserLocationChangeRequired, error) {
	flow, redirect, err := s.ServiceInterface.CreateBrowserSettingsFlow(ctx, returnTo, cookies)
	if flow != nil {
		flow.Ui.Nodes = withoutProvider(flow.Ui.Nodes)
	}
	return flow, redirect, err
}

func (s *KratosServiceDecorator) GetSettingsFlow(ctx context.Context, id string, cookies []*http.Cookie) (*kClient.SettingsFlow, *kratos.BrowserLocationChangeRequired, error) {
	flow, redirect, err := s.ServiceInterface.GetSettingsFlow(ctx, id, cookies)
	if flow != nil {
		flow.Ui.Nodes = withoutProvider(flow.Ui.Nodes)
	}
	return flow, redirect, err
}

func (s *KratosServiceDecorator) UpdateSettingsFlow(ctx context.Context, id string, body kClient.UpdateSettingsFlowBody, cookies []*http.Cookie) (*kClient.SettingsFlow, *kratos.BrowserLocationChangeRequired, []*http.Cookie, error) {
	flow, redirect, cookies, err := s.ServiceInterface.UpdateSettingsFlow(ctx, id, body, cookies)
	if flow != nil {
		flow.Ui.Nodes = withoutProvider(flow.Ui.Nodes)
	}
	return flow, redirect, cookies, err
}

// withoutProvider drops the nodes of the company sign-in provider: the
// "provider" submit of a login or registration flow, and the "link" and
// "unlink" submits of a settings flow.
func withoutProvider(nodes []kClient.UiNode) []kClient.UiNode {
	kept := nodes[:0:0]
	for _, node := range nodes {
		attrs := node.Attributes.UiNodeInputAttributes
		if node.Group == methodOIDC && attrs != nil && fmt.Sprintf("%v", attrs.GetValue()) == providerID {
			continue
		}
		kept = append(kept, node)
	}
	return kept
}

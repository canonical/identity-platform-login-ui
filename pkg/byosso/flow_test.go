// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"testing"

	kClient "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

func TestHasOtherFirstFactor(t *testing.T) {
	tests := []struct {
		name     string
		nodes    []kClient.UiNode
		expected bool
	}{
		{name: "password", nodes: []kClient.UiNode{inputNode("password", "password", "")}, expected: true},
		{name: "public sign-in provider", nodes: []kClient.UiNode{inputNode("oidc", "provider", "google")}, expected: true},
		{name: "passkey", nodes: []kClient.UiNode{inputNode("passkey", "passkey_login", "")}, expected: true},
		{name: "passwordless webauthn", nodes: []kClient.UiNode{inputNode("webauthn", "webauthn_login", "")}, expected: true},
		{name: "code", nodes: []kClient.UiNode{inputNode("code", "code", "")}, expected: true},
		{name: "company sign-in provider", nodes: []kClient.UiNode{inputNode("oidc", "provider", providerID)}, expected: false},
		{name: "method of the password group", nodes: []kClient.UiNode{inputNode("password", "method", "password")}, expected: false},
		{name: "identifier", nodes: []kClient.UiNode{inputNode("default", "identifier", testEmail)}, expected: false},
		{name: "company sign-in", nodes: []kClient.UiNode{companySignInNode(ssoConnectionField, connA, "Acme")}, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flow := kClient.NewLoginFlowWithDefaults()
			flow.Ui.Nodes = tt.nodes

			if other := hasOtherFirstFactor(flow); other != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, other)
			}
		})
	}
}

func TestAccountNotFound(t *testing.T) {
	onNode := loginFlow("f", testLoginChallenge, testEmail, true)
	onNode.Ui.Nodes[1].Messages = []kClient.UiText{*kClient.NewUiText(kratos.IncorrectAccountIdentifier, "does not exist", "error")}

	tests := []struct {
		name     string
		flow     *kClient.LoginFlow
		expected bool
	}{
		{name: "message on the flow", flow: refusedLoginFlow(), expected: true},
		{name: "message on a node", flow: onNode, expected: true},
		{name: "no message", flow: loginFlow("f", testLoginChallenge, testEmail, true), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if notFound := accountNotFound(tt.flow); notFound != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, notFound)
			}
		})
	}
}

func TestIsRefresh(t *testing.T) {
	withLoginChallenge := refreshFlow()
	withLoginChallenge.SetOauth2LoginChallenge(testLoginChallenge)

	tests := []struct {
		name     string
		flow     *kClient.LoginFlow
		expected bool
	}{
		{name: "refresh flow", flow: refreshFlow(), expected: true},
		{name: "refresh flow of a login challenge", flow: withLoginChallenge, expected: false},
		{name: "login flow", flow: loginFlow("f", "", testEmail, false), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if refresh := isRefresh(tt.flow); refresh != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, refresh)
			}
		})
	}
}

func TestWithoutIdentifierStep(t *testing.T) {
	flow := refusedLoginFlow()
	flow.Ui.Nodes[1].Messages = []kClient.UiText{*kClient.NewUiText(kratos.IncorrectAccountIdentifier, "does not exist", "error")}

	flow = withoutIdentifierStep(flow)

	if hasIdentifierFirst(flow) {
		t.Fatalf("expected the identifier step to be removed")
	}
	if len(flow.Ui.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(flow.Ui.Nodes))
	}

	identifier := flow.Ui.Nodes[1]
	// the address is kept for the company sign-in to read
	if value := flowInputValue(flow.Ui.Nodes, "identifier"); value != testEmail {
		t.Fatalf("expected identifier %s, got %s", testEmail, value)
	}
	if identifier.Attributes.UiNodeInputAttributes.Type != "hidden" {
		t.Fatalf("expected the identifier to be hidden, got %s", identifier.Attributes.UiNodeInputAttributes.Type)
	}
	if len(identifier.Messages) != 0 {
		t.Fatalf("expected no message on the identifier, got %+v", identifier.Messages)
	}
}

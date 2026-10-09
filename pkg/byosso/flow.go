// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0-only

package byosso

import (
	"fmt"

	kClient "github.com/ory/kratos-client-go/v25"

	"github.com/canonical/identity-platform-login-ui/pkg/kratos"
)

// firstFactorGroups are the node groups of the sign-ins Kratos offers the
// account itself, and of the identifier step.
var firstFactorGroups = map[string]bool{
	methodPassword:        true,
	methodOIDC:            true,
	methodPasskey:         true,
	methodWebAuthn:        true,
	methodCode:            true,
	methodIdentifierFirst: true,
}

// submitNode returns a submit button of group that posts name=value back to
// login-ui.
func submitNode(group, name, value, label string, labelID int64) kClient.UiNode {
	attrs := kClient.NewUiNodeInputAttributesWithDefaults()
	attrs.Name = name
	attrs.Type = "submit"
	attrs.Value = value
	attrs.NodeType = "input"

	text := kClient.NewUiTextWithDefaults()
	text.Text = label
	text.Id = labelID

	node := kClient.NewUiNodeWithDefaults()
	node.Type = "input"
	node.Group = group
	node.Attributes = kClient.UiNodeAttributes{UiNodeInputAttributes: attrs}
	node.Meta = *kClient.NewUiNodeMeta()
	node.Meta.Label = text
	return *node
}

// companySignInNode returns the button of a company sign-in, posting its
// connection ID as field.
func companySignInNode(field, connectionID, label string) kClient.UiNode {
	return submitNode(ssoNodeGroup, field, connectionID, fmt.Sprintf("Continue with %s", label), signInWith)
}

// linkNode returns the text node of one company sign-in of the account.
func linkNode(l Link) kClient.UiNode {
	text := kClient.NewUiTextWithDefaults()
	text.Id = nodeLabel
	text.Type = "info"
	text.Text = fmt.Sprintf("Signed in with %s", l.Label)

	node := kClient.NewUiNodeWithDefaults()
	node.Type = "text"
	node.Group = ssoNodeGroup
	node.Attributes = kClient.UiNodeAttributes{UiNodeTextAttributes: kClient.NewUiNodeTextAttributes(ssoLinkNodePrefix+l.ConnectionID, "text", *text)}
	node.Meta = *kClient.NewUiNodeMeta()
	return *node
}

// flowInputValue returns the value of the named input node as a string, or
// "".
func flowInputValue(nodes []kClient.UiNode, name string) string {
	for _, node := range nodes {
		attrs := node.Attributes.UiNodeInputAttributes
		if node.Type != "input" || attrs == nil || attrs.Name != name {
			continue
		}
		if s, ok := attrs.Value.(string); ok {
			return s
		}
		if attrs.Value == nil {
			return ""
		}
		return fmt.Sprintf("%v", attrs.Value)
	}
	return ""
}

// hasIdentifierFirst reports whether the flow is at the identifier step.
func hasIdentifierFirst(flow *kClient.LoginFlow) bool {
	if flow == nil {
		return false
	}
	for _, node := range flow.Ui.Nodes {
		attrs := node.Attributes.UiNodeInputAttributes
		if node.Type == "input" && attrs != nil && attrs.Name == "method" && fmt.Sprintf("%v", attrs.Value) == methodIdentifierFirst {
			return true
		}
	}
	return false
}

// hasOtherFirstFactor reports whether the flow offers a first factor other
// than a company sign-in: the portal password, a public sign-in provider, a
// passkey, or a passwordless WebAuthn or code login. After the identifier
// step Kratos lists only the methods the account holds, and the passwordless
// ones the deployment enables.
func hasOtherFirstFactor(flow *kClient.LoginFlow) bool {
	for _, n := range flow.Ui.Nodes {
		attrs := n.Attributes.UiNodeInputAttributes
		if n.Type != "input" || attrs == nil {
			continue
		}
		switch n.Group {
		case methodPassword:
			if attrs.Name == "password" {
				return true
			}
		case methodOIDC:
			if attrs.Name == "provider" && attrs.Value != providerID {
				return true
			}
		case methodPasskey, methodWebAuthn, methodCode:
			return true
		}
	}
	return false
}

// accountNotFound reports whether Kratos refused the address at the
// identifier step: the account does not exist, or has no sign-in method.
func accountNotFound(flow *kClient.LoginFlow) bool {
	for _, m := range flow.Ui.Messages {
		if m.Id == kratos.IncorrectAccountIdentifier {
			return true
		}
	}
	for _, n := range flow.Ui.Nodes {
		for _, m := range n.Messages {
			if m.Id == kratos.IncorrectAccountIdentifier {
				return true
			}
		}
	}
	return false
}

// isAccountLinking reports whether flow is the account-linking login flow of
// Kratos: a company sign-in with no link, whose address belongs to an
// account, became a registration, which stopped at the unique address of the
// account and turned into this login flow. Signing in to the account on it
// makes Kratos add the company sign-in.
func isAccountLinking(flow *kClient.LoginFlow) bool {
	if flow == nil {
		return false
	}
	for _, m := range flow.Ui.Messages {
		if m.Id == signInAndLink {
			return true
		}
	}
	return false
}

// isRefresh reports whether flow is the refresh login of Kratos for the
// session of the browser, with no Hydra login request: what a privileged
// settings change asks for once the session is older than the privileged
// session age.
func isRefresh(flow *kClient.LoginFlow) bool {
	return flow.GetRefresh() && flow.GetOauth2LoginChallenge() == ""
}

func withMessage(flow *kClient.LoginFlow, id int64, text, kind string) *kClient.LoginFlow {
	flow.Ui.Messages = append(flow.Ui.Messages, *kClient.NewUiText(id, text, kind))
	return flow
}

// withTenants appends the tenant list.
func withTenants(flow *kClient.LoginFlow, tenants []SignInTenant) *kClient.LoginFlow {
	for _, t := range tenants {
		label := t.Name
		if t.Invited {
			label += invitationLabel
		}
		flow.Ui.Nodes = append(flow.Ui.Nodes, submitNode(tenantNodeGroup, tenantField, t.ID, label, signInWith))
	}
	return flow
}

// withoutFirstFactors keeps only the nodes that are not a sign-in of the
// account: the CSRF token, the address and the nodes login-ui adds.
func withoutFirstFactors(flow *kClient.LoginFlow) *kClient.LoginFlow {
	nodes := flow.Ui.Nodes[:0:0]
	for _, n := range flow.Ui.Nodes {
		if !firstFactorGroups[n.Group] {
			nodes = append(nodes, n)
		}
	}
	flow.Ui.Nodes = nodes
	return flow
}

// withoutIdentifierStep turns a flow Kratos refused at the identifier step
// into one past it: the refusal and the submit of the step are removed, and
// the address is kept as a hidden field, which a company sign-in reads.
func withoutIdentifierStep(flow *kClient.LoginFlow) *kClient.LoginFlow {
	nodes := flow.Ui.Nodes[:0:0]
	for _, n := range flow.Ui.Nodes {
		attrs := n.Attributes.UiNodeInputAttributes
		if attrs != nil && attrs.Name == "method" && fmt.Sprintf("%v", attrs.Value) == methodIdentifierFirst {
			continue
		}
		if attrs != nil && attrs.Name == "identifier" {
			attrs.Type = "hidden"
		}
		kept := n.Messages[:0:0]
		for _, m := range n.Messages {
			if m.Id != kratos.IncorrectAccountIdentifier {
				kept = append(kept, m)
			}
		}
		n.Messages = kept
		nodes = append(nodes, n)
	}
	flow.Ui.Nodes = nodes
	return flow
}

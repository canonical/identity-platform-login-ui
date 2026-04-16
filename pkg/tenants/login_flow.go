// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

package tenants

import client "github.com/ory/kratos-client-go/v25"

// InjectTenantPayload sets tenant_id in the transient_payload of every login
// flow method type that supports it. LookupSecret and Passkey are omitted
// because Kratos does not include transient_payload in their login schemas
// (upstream omission — both use the same WebAuthn protocol as webauthn which
// does support it).
func InjectTenantPayload(body *client.UpdateLoginFlowBody, tenantID string) {
	payload := map[string]interface{}{"tenant_id": tenantID}
	switch {
	case body.UpdateLoginFlowWithPasswordMethod != nil:
		body.UpdateLoginFlowWithPasswordMethod.SetTransientPayload(payload)
	case body.UpdateLoginFlowWithTotpMethod != nil:
		body.UpdateLoginFlowWithTotpMethod.SetTransientPayload(payload)
	case body.UpdateLoginFlowWithOidcMethod != nil:
		body.UpdateLoginFlowWithOidcMethod.SetTransientPayload(payload)
	case body.UpdateLoginFlowWithWebAuthnMethod != nil:
		body.UpdateLoginFlowWithWebAuthnMethod.SetTransientPayload(payload)
	case body.UpdateLoginFlowWithCodeMethod != nil:
		body.UpdateLoginFlowWithCodeMethod.SetTransientPayload(payload)
	case body.UpdateLoginFlowWithSamlMethod != nil:
		body.UpdateLoginFlowWithSamlMethod.SetTransientPayload(payload)
	case body.UpdateLoginFlowWithIdentifierFirstMethod != nil:
		body.UpdateLoginFlowWithIdentifierFirstMethod.SetTransientPayload(payload)
	}
}

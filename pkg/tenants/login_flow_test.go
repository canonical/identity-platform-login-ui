// Copyright 2026 Canonical Ltd.
// SPDX-License-Identifier: AGPL-3.0

package tenants

import (
	"testing"

	client "github.com/ory/kratos-client-go/v25"
)

func TestInjectTenantPayload(t *testing.T) {
	const tenantID = "tenant-abc"

	testCases := []struct {
		name     string
		body     func() *client.UpdateLoginFlowBody
		expected bool // true if transient_payload should be set
	}{
		{
			name: "password method",
			body: func() *client.UpdateLoginFlowBody {
				b := client.UpdateLoginFlowWithPasswordMethodAsUpdateLoginFlowBody(
					client.NewUpdateLoginFlowWithPasswordMethod("", "password", ""),
				)
				return &b
			},
			expected: true,
		},
		{
			name: "TOTP method",
			body: func() *client.UpdateLoginFlowBody {
				b := client.UpdateLoginFlowWithTotpMethodAsUpdateLoginFlowBody(
					client.NewUpdateLoginFlowWithTotpMethod("", "totp"),
				)
				return &b
			},
			expected: true,
		},
		{
			name: "WebAuthn method",
			body: func() *client.UpdateLoginFlowBody {
				b := client.UpdateLoginFlowWithWebAuthnMethodAsUpdateLoginFlowBody(
					client.NewUpdateLoginFlowWithWebAuthnMethod("", "webauthn"),
				)
				return &b
			},
			expected: true,
		},
		{
			name: "lookup secret method - not supported",
			body: func() *client.UpdateLoginFlowBody {
				b := client.UpdateLoginFlowWithLookupSecretMethodAsUpdateLoginFlowBody(
					client.NewUpdateLoginFlowWithLookupSecretMethod("", "lookup_secret"),
				)
				return &b
			},
			expected: false,
		},
		{
			name: "passkey method - not supported",
			body: func() *client.UpdateLoginFlowBody {
				b := client.UpdateLoginFlowWithPasskeyMethodAsUpdateLoginFlowBody(
					client.NewUpdateLoginFlowWithPasskeyMethod("passkey"),
				)
				return &b
			},
			expected: false,
		},
		{
			name: "code method",
			body: func() *client.UpdateLoginFlowBody {
				b := client.UpdateLoginFlowWithCodeMethodAsUpdateLoginFlowBody(
					client.NewUpdateLoginFlowWithCodeMethod("", "code"),
				)
				return &b
			},
			expected: true,
		},
		{
			name: "OIDC/social method",
			body: func() *client.UpdateLoginFlowBody {
				b := client.UpdateLoginFlowWithOidcMethodAsUpdateLoginFlowBody(
					client.NewUpdateLoginFlowWithOidcMethod("oidc", "google"),
				)
				return &b
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body := tc.body()
			InjectTenantPayload(body, tenantID)

			got := getTransientPayload(body)
			if tc.expected {
				if got == nil {
					t.Fatal("expected transient_payload to be set, but it was nil")
				}
				tid, ok := got["tenant_id"].(string)
				if !ok || tid != tenantID {
					t.Fatalf("expected tenant_id=%q, got %v", tenantID, got["tenant_id"])
				}
			} else {
				if got != nil {
					t.Fatalf("expected transient_payload to be nil for unsupported method, got %v", got)
				}
			}
		})
	}
}

func TestInjectTenantPayload_EmptyTenantID(t *testing.T) {
	body := client.UpdateLoginFlowWithPasswordMethodAsUpdateLoginFlowBody(
		client.NewUpdateLoginFlowWithPasswordMethod("", "password", ""),
	)

	InjectTenantPayload(&body, "")

	got := getTransientPayload(&body)
	if got == nil {
		t.Fatal("expected transient_payload to be set (empty tenant_id is still injected)")
	}
	tid, ok := got["tenant_id"].(string)
	if !ok || tid != "" {
		t.Fatalf("expected tenant_id=\"\", got %v", got["tenant_id"])
	}
}

func TestInjectTenantPayload_NoneSentinel(t *testing.T) {
	body := client.UpdateLoginFlowWithPasswordMethodAsUpdateLoginFlowBody(
		client.NewUpdateLoginFlowWithPasswordMethod("", "password", ""),
	)

	InjectTenantPayload(&body, "_none")

	got := getTransientPayload(&body)
	if got == nil {
		t.Fatal("expected transient_payload to be set (_none is still injected at this layer)")
	}
	tid, ok := got["tenant_id"].(string)
	if !ok || tid != "_none" {
		t.Fatalf("expected tenant_id=\"_none\", got %v", got["tenant_id"])
	}
}

// getTransientPayload extracts the transient_payload from whichever method
// variant is set in the body. Returns nil if none is set or the method doesn't
// support it.
func getTransientPayload(body *client.UpdateLoginFlowBody) map[string]interface{} {
	switch {
	case body.UpdateLoginFlowWithPasswordMethod != nil:
		p, ok := body.UpdateLoginFlowWithPasswordMethod.GetTransientPayloadOk()
		if !ok || p == nil {
			return nil
		}
		return p
	case body.UpdateLoginFlowWithTotpMethod != nil:
		p, ok := body.UpdateLoginFlowWithTotpMethod.GetTransientPayloadOk()
		if !ok || p == nil {
			return nil
		}
		return p
	case body.UpdateLoginFlowWithOidcMethod != nil:
		p, ok := body.UpdateLoginFlowWithOidcMethod.GetTransientPayloadOk()
		if !ok || p == nil {
			return nil
		}
		return p
	case body.UpdateLoginFlowWithWebAuthnMethod != nil:
		p, ok := body.UpdateLoginFlowWithWebAuthnMethod.GetTransientPayloadOk()
		if !ok || p == nil {
			return nil
		}
		return p
	case body.UpdateLoginFlowWithCodeMethod != nil:
		p, ok := body.UpdateLoginFlowWithCodeMethod.GetTransientPayloadOk()
		if !ok || p == nil {
			return nil
		}
		return p
	default:
		return nil
	}
}

# 1. Withhold `login_challenge` from Kratos When Multi-Tenancy Is Enabled

Date: 2026-04-09

## Status

Accepted

## Context

When the Login UI creates a Kratos browser login flow and passes the Hydra `login_challenge`,
Kratos may auto-accept the login request via Hydra internally during OIDC authentication. In this
case, Kratos calls `AcceptOAuth2LoginRequest` itself and 302-redirects the browser directly to the
Hydra callback — the Login UI never gets a chance to intercept the flow.

This is a problem for multi-tenancy because the `tenant_id` selected by the user must be injected
into the Hydra login context (via `AcceptLoginRequest`'s `context` field) so it propagates to the
consent step and ultimately into the OAuth2 token. When Kratos auto-accepts, it passes no
`tenant_id` in the context, and the Login UI cannot inject it.

A secondary issue was that the consent handler's fallback to resolve `tenant_id` from the
`login_ui_state` cookie was broken: the consent handler receives a Hydra `consent_challenge`,
not the original `login_challenge`, so the hash-based cookie lookup never matched.

### Existing precedent

The `OIDC_WEBAUTHN_SEQUENCING_ENABLED` feature already withholds `login_challenge` from Kratos
for the same structural reason — it needs the Login UI to intercept the post-authentication
redirect to enforce a WebAuthn registration step (see `docs/sequencing.md`).

## Decision

When `MULTI_TENANCY_ENABLED=true`, the Login UI backend withholds `login_challenge` from Kratos
when creating browser login flows (`CreateBrowserLoginFlow`). The implementation reuses the same
conditional gate as the OIDC-WebAuthn sequencing feature:

```go
if !s.oidcWebAuthnSequencingEnabled && !s.multiTenancyEnabled {
    if loginChallenge != "" {
        request = request.LoginChallenge(loginChallenge)
    }
}
```

When `login_challenge` is withheld:

1. Kratos creates the login flow as a standalone self-service flow (no Hydra integration).
2. After authentication (password, OIDC, WebAuthn, etc.), Kratos redirects to the `return_to` URL
   — which points back to the Login UI.
3. The Login UI retrieves the authenticated session, resolves the `tenant_id` from the
   `login_ui_state` cookie, and calls `AcceptLoginRequest` on Hydra with the `tenant_id`
   in the `context` field.
4. Hydra stores  the context and proceeds to the consent step.
5. The consent handler reads `tenant_id` from `consent.GetContext()["tenant_id"]` and passes it
   in `AcceptConsentRequest`.

The consent handler's `resolveTenantID` method was simplified to read exclusively from the consent
context (set by `AcceptLoginRequest`). The broken cookie-based fallback was removed.

### Alternatives considered

| Option | Description | Why rejected |
|---|---|---|
| A — Patch Kratos to forward `tenant_id` during auto-accept | Modify Kratos to pass custom context when it auto-accepts via Hydra | Requires maintaining a Kratos fork; fragile across upgrades |
| B — Cookie fallback at consent | Read `tenant_id` from the `login_ui_state` cookie during consent | Broken by design — consent handler receives `consent_challenge`, not `login_challenge`, so the hash-keyed cookie lookup never matches |
| C — Withhold `login_challenge` (chosen) | Prevent Kratos from auto-accepting by not giving it the challenge | Reuses existing pattern; no Kratos changes; Login UI retains full control |

## Consequences

### Positive

- The Login UI retains full control over the Hydra login acceptance, ensuring `tenant_id` is
  always present in the context when multi-tenancy is enabled.
- The consent handler is simplified — a single code path reads `tenant_id` from the consent context
  rather than attempting cookie-based fallbacks.
- The pattern is consistent with the existing OIDC-WebAuthn sequencing feature.

### Negative

- The login flow has one additional redirect hop (Kratos → Login UI → Hydra) compared to when
  Kratos auto-accepts. This adds a small amount of latency.
- Both `OIDC_WEBAUTHN_SEQUENCING_ENABLED` and `MULTI_TENANCY_ENABLED` independently cause
  `login_challenge` withholding. If new features need the same, they should contribute to the
  same conditional rather than adding a separate flag.

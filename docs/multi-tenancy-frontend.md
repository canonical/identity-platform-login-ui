# Multi-Tenancy Frontend Implementation Guide

This document is a reference for the developer who will harden the frontend tenant
selection flow. It describes the backend API contracts, the expected decision tree, and
the error-handling requirements. No large-scale UI refactor is needed — the structure in
`ui/pages/login.tsx` and `ui/pages/select_tenant.tsx` is already correct. The work
described here is targeted hardening within those files.

---

## Feature flag

Multi-tenancy is optional. It is enabled by the backend environment variable
`MULTI_TENANCY_ENABLED=true`. When disabled, tenant-related endpoints still exist but
the resolver always returns empty results, so the frontend should behave identically to a
non-tenant deployment.

Check `ui/config/useAppConfig.tsx` (which fetches `/api/v0/app-config`) for the
`multiTenancyEnabled` flag (`multi_tenancy_enabled` in the JSON response) before making
any tenant API calls. Do not call the lookup API if the flag is `false`.

### Backend consequence: `login_challenge` withholding

When `MULTI_TENANCY_ENABLED=true`, the backend withholds the Hydra `login_challenge` from
Kratos when creating browser login flows. This is the same pattern used by
`OIDC_WEBAUTHN_SEQUENCING_ENABLED` (see [docs/sequencing.md](docs/sequencing.md)). Without
this, Kratos auto-accepts the Hydra login request internally during OIDC authentication,
bypassing the Login UI and losing the `tenant_id` context. Withholding the challenge forces
Kratos to redirect back to the Login UI after authentication, allowing it to call
`AcceptLoginRequest` with `tenant_id` in the context.

See ADR 0001 (`docs/adr/0001-withhold-login-challenge-for-multitenancy.md`) for the full
rationale.

---

## API contracts

### Tenant lookup
```
GET /api/v0/tenants
GET /api/v0/tenants?flow=<flow_id>
```
- When `flow` is provided the email is extracted from the Kratos flow; otherwise the
  identity ID is read from the active Kratos session. Email is never accepted as a URL
  parameter to prevent unauthenticated tenant enumeration.
- **200 OK** — JSON object with a `tenants` array (may be empty):
  ```json
  {"tenants": [{"id": "t1", "name": "Acme Corp"}, ...]}
  ```
- **401 Unauthorized** — no `?flow=` provided and no active Kratos session.
- **500 Internal Server Error** — upstream tenant-service call failed.

### Tenant selection (store selection in cookie)
```
POST /api/v0/auth/tenant
Content-Type: application/json
{
  "login_challenge": "<challenge>",
  "tenant_id": "<id>",  // empty string when the user has no tenants
  "flow": "<flow_id>"   // optional
}
```
- **200 OK** — `{"redirect_to": "<url>"}` — frontend must follow this redirect.
- **400 Bad Request** — `login_challenge` is missing.
- **500 Internal Server Error** — failed to persist tenant cookie.

---

## Decision tree (post identifier-first success)

After `updateIdentifierFirstFlow` succeeds and returns an email, the login page must
evaluate tenant membership:

```
multiTenancyEnabled == false
  → followData()   // continue normal flow

multiTenancyEnabled == true  →  backend handles tenant resolution during identifier-first submit:

  0 tenants  → backend stores no-tenant sentinel; redirects to login flow
  1 tenant   → backend auto-stores the tenant; redirects to login flow
  2+ tenants → backend redirects to /ui/select_tenant?flow=<flowId>&login_challenge=<challenge>

When the user lands on /ui/select_tenant:

  - On mount: calls GET /api/v0/tenants?flow=<flowId> to get the tenant list.
  - 0 or 1 result: submitTenantSelection is called automatically.
  - 2+ results: user picks a tenant; POST /api/v0/auth/tenant is called on click.
  - On POST success: frontend follows the redirect_to URL.
```

The `ui/pages/select_tenant.tsx` implementation already reflects this logic.

---

## Error handling requirements

| Condition | Current behaviour | Required behaviour |
|---|---|---|
| Tenant lookup fails (network / 500) | `setError(...)` shown on page | Correct — no change needed |
| Lookup returns 0 tenants | Auto-submits with empty `tenant_id` | Correct — no change needed |
| Lookup returns 1 tenant | Auto-submits with `tenant_id` | Correct — no change needed |
| Lookup returns >1 tenants | Shows tenant selection list | Correct — no change needed |
| `POST /api/v0/auth/tenant` fails | `setError(...)` shown on page | Correct — no change needed |
| Missing `login_challenge` in URL | `setError(...)` shown on page | Correct — no change needed |

---

## `ui/api/tenants.ts`

Exports `fetchTenantsByFlow(flowId)` and `fetchTenantsBySession()`, both returning
`Promise<Tenant[]>`. Non-ok responses throw a generic `Error` with the HTTP status,
which the caller in `select_tenant.tsx` catches and surfaces via `setError`.

---

## `ui/pages/select_tenant.tsx`

Handles the multi-tenant selection page. On mount it calls `fetchTenantsByFlow` (when
`?flow=` is present) or `fetchTenantsBySession`. Auto-submits when 0 or 1 tenant is
returned. Surfaces all errors (lookup failure, missing `login_challenge`, POST failure)
via `setError`.

---

## Testing guidance

- **Unit**: mock `fetchTenantsByFlow` / `fetchTenantsBySession` to reject and assert
  the `setError` message is rendered on the page.
- **E2E (Playwright)**: The `ui/tests/` directory has existing login flow tests. Add a
  test fixture that registers a user under a tenant via the tenant-service API, then
  walks the full OIDC flow through `http://localhost:4446/` verifying auto-select
  (1 tenant) and manual selection (>1 tenants).

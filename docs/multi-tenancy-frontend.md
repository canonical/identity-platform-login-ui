# Multi-Tenancy Frontend Implementation Guide

This document is a reference for the developer who will harden the frontend tenant
selection flow. It describes the backend API contracts, the expected decision tree, and
the error-handling requirements. No large-scale UI refactor is needed — the structure in
`ui/pages/login.tsx` and `ui/pages/select_tenant.tsx` is already correct. The work
described here is targeted hardening within those files.

---

## Feature flag

Multi-tenancy is optional. It is enabled by the backend environment variable
`MULTI_TENANCY_ENABLED=true`. When disabled, the `/api/v0/login/challenge/tenant-state`
and `/api/v0/tenants/lookup` endpoints still exist but the resolver always returns
`tenant_selected: false`, so the frontend should behave identically to a non-tenant
deployment.

Check `ui/pages/status.tsx` (the `/api/v0/status` response) for the `tenantSelectionEnabled`
flag before making any tenant API calls. Do not call the lookup API if the flag is `false`.

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
GET /api/v0/tenants/lookup?email=<url-encoded-email>
```
- **200 OK** — JSON array of tenant objects (may be empty):
  ```json
  [{"id": "t1", "name": "Acme Corp", "enabled": true}, ...]
  ```
- **400 Bad Request** — `email` query parameter is missing.
- **500 Internal Server Error** — upstream tenant-service call failed.

> **Note**: The response is a bare array, **not** a `{"tenants": [...]}` wrapper.
> `ui/api/tenants.ts` already expects this correctly.

### Tenant redirect (store selection in cookie)
```
GET /api/v0/login/challenge/tenant-redirect
    ?login_challenge=<challenge>
    &tenant_id=<id>
    [&flow=<flow_id>]
```
- **302 Found** — redirects to `/ui/login?flow=<flow_id>` when `flow` is provided,
  otherwise redirects to `/self-service/login/browser?login_challenge=<challenge>`.
- **400 Bad Request** — `login_challenge` or `tenant_id` is missing.
- **500 Internal Server Error** — failed to persist tenant cookie.

### Tenant state
```
GET /api/v0/login/challenge/tenant-state?login_challenge=<challenge>
```
- **200 OK** — `{"tenant_selected": true|false}`.
- **400 Bad Request** — `login_challenge` is missing.
- **500 Internal Server Error** — failed to read state cookie.

---

## Decision tree (post identifier-first success)

After `updateIdentifierFirstFlow` succeeds and returns an email, the login page must
evaluate tenant membership:

```
multiTenancyEnabled == false
  → followData()   // continue normal flow

tenantSelection enabled  →  await fetchTenantsByEmail(email)

  LOOKUP FAILS (network error / 500)
    → fail closed: show a user-readable error; do NOT silently fall through.
      The backend is configured fail-closed; the frontend must match this behaviour.

  tenants.length == 0
    → followData()   // user has no tenants — allow login without one

  tenants.length == 1
    → redirect to tenant-redirect immediately (auto-select)
      Use: /api/v0/login/challenge/tenant-redirect
             ?login_challenge=<challenge>
             &tenant_id=<tenants[0].id>
             [&flow=<flowId>]

  tenants.length > 1
    → push('/select_tenant?...')   // mandatory selection
```

The current `ui/pages/login.tsx` implementation already reflects this logic. The key
change required is **replacing the silent `.catch(() => { followData() })` with an
error-surface call** so lookup failures fail closed.

---

## Error handling requirements

| Condition | Current behaviour | Required behaviour |
|---|---|---|
| Lookup 500 / network error | `.catch()` → `followData()` (silent fallback) | Show user-facing error: "Unable to determine your tenants. Please try again." |
| Lookup returns 0 tenants | `followData()` | No change — correct |
| Lookup returns 1 tenant | Redirect | No change — correct |
| Lookup returns >1 tenants | Push to select_tenant | No change — correct |
| tenant-redirect 500 | Browser lands on error page | Catch redirect failure; show error to user |

---

## `ui/api/tenants.ts` changes

Minimal changes needed:

1. Export a typed error class so callers can distinguish lookup failures from empty results:
   ```ts
   export class TenantLookupError extends Error {}
   ```
2. In `fetchTenantsByEmail`, throw `TenantLookupError` (not a generic `Error`) on non-ok
   responses so callers can narrow the type.

---

## `ui/pages/login.tsx` changes

Replace the silent catch in the identifier-first flow branch with an explicit error handler:

```ts
// Before (existing — do not ship this to production)
fetchTenantsByEmail(email).then(handleTenants).catch(() => followData())

// After
fetchTenantsByEmail(email).then(handleTenants).catch((err) => {
  if (err instanceof TenantLookupError) {
    setError("Unable to determine your tenants. Please try again.")
    return
  }
  throw err   // unexpected — rethrow
})
```

The `setError` call should use the same error state that renders beneath the identifier
field (consistent with how other flow errors are displayed using `@canonical/react-components`).

Do **not** change any other part of the multi-step identifier-first flow.

---

## `ui/pages/select_tenant.tsx` — no changes required

The page already handles loading state, re-fetches on mount, and redirects to
`tenant-redirect` on selection. No changes are required — only verify it inherits the
error surfacing improvements in `ui/api/tenants.ts`.

---

## Testing guidance

- **Unit**: mock `fetchTenantsByEmail` to reject with `TenantLookupError` and assert the
  error message is rendered (not `followData()` called).
- **E2E (Playwright)**: The `ui/tests/` directory has existing login flow tests. Add a
  test fixture that registers a user under a tenant via the tenant-service API, then
  walks the full OIDC flow through `http://localhost:4446/` verifying auto-select
  (1 tenant) and manual selection (>1 tenants).

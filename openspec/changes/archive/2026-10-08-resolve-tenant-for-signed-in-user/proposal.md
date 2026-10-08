## Why

With `MULTI_TENANCY_ENABLED=true`, the Login UI records the tenant of a login in its state cookie, for the login challenge, and passes it to Hydra when the login is accepted. The record follows the login request, not the user who signs in: a second email entered for the same request keeps the tenant recorded for the first one.

### Problem statement

- The tenant is recorded at the email step (`checkTenantSelectionByEmail` in `pkg/kratos/handlers.go`) or by the tenant selection endpoint (`POST /api/v0/auth/tenant`), before anyone has signed in.
- `NeedsTenantSelectionByEmail` and `needsTenantSelectionByIdentityID` (`pkg/tenants/resolver.go`) skip the lookup whenever a tenant is already recorded for the challenge, so the record is never compared with the tenants of the user who signs in.
- A user with no tenant who signs in after the email of a user with one tenant is accepted with that tenant. A user with a tenant who signs in after an email with none is accepted with no tenant. An id sent to the selection endpoint is used in the same way.
- The Tenant Service's hooks can check membership after the Login UI. Where they do, the second user is refused for a tenant they never asked for; where they do not, the recorded tenant is what the tokens carry.

### Scope

- Resolve the tenant of a login from the tenants of its user every time it is resolved, at the email step and before the login is accepted.
- Use the recorded tenant only to choose among several tenants of that user.
- Start an email submission from an empty record.

### Non-goals

- Binding the record to a user. The cookie stays bound to a login challenge only; the spec states what that leaves open.
- Checking the tenant id that is sent to Kratos with the credentials for the Tenant Service's login hook.
- Membership checks in the tenant selection endpoint.
- Any change with multi-tenancy disabled, and any frontend (`ui/`) change.

### Success criteria

- Two users who sign in one after the other for the same login request each get their own tenant, in both orders.
- A user with several tenants still selects once per login and keeps the selection across the steps of that login.
- A tenant id that is not one of the user's never reaches Hydra's accept.

## What Changes

- `pkg/tenants/resolver.go`: both lookups always run and go through one `resolve`: no tenant, the only tenant, or a selection among several where the recorded tenant counts only if it is one of them. A record that is not the user's is dropped.
- `pkg/kratos/handlers.go`: `checkTenantSelectionByEmail` starts from a new cookie for the challenge. `handleUpdateFlow` writes the resolved cookie when it sends the user to the tenant selection; it wrote the one from before the lookup.
- User-visible besides the fix, and accepted:
  - A user who selected a tenant and submits their email again selects again.
  - A selected tenant that the user has lost since, with one tenant left, becomes that one.
  - One more lookup at the Tenant Service per login, and one each time the login page is reached with a session. When it fails the login fails there, where a recorded tenant used to carry it through.

## Capabilities

### New Capabilities

- `login-tenant-resolution`: how the Login UI decides the tenant of a client's login with multi-tenancy enabled, and what the tenant recorded in the state cookie may and may not decide.

### Modified Capabilities

<!-- None: `tenant-grpc-client` covers the lookup client, not the use of its result. -->

## Impact

- **Backend Go**: `pkg/tenants`, `pkg/kratos`, with their unit tests.
- **Frontend**: unaffected.
- **Kratos**: no configuration change. The tenant id sent to Kratos in the transient payload is unchanged.
- **Hydra**: no configuration change. The tenant in the context of the accepted login is now always one of the user's, or absent.
- **Cookies**: no new field. The email step no longer reads the state cookie.
- **Tenant Service**: more `LookupTenants` calls, as stated above.
- **OpenFGA**: unaffected.

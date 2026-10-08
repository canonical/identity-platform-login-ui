## 1. The user's tenants decide

- [x] 1.1 In `pkg/tenants/resolver.go`, remove the early return for a recorded tenant from `NeedsTenantSelectionByEmail` and `needsTenantSelectionByIdentityID`, and move their common ending into `resolve(tenants, cookie, loginChallenge)`: none, the only one, or the recorded one if it is among several, otherwise a cleared record and a selection.
- [x] 1.2 `pkg/tenants/resolver_test.go`: `TestNeedsTenantSelectionDropsTenantOfAnotherUser` (no tenant, one, several) and `TestNeedsTenantSelectionByEmailLooksUpDespiteRecordedTenant`. Completion: `go test ./pkg/tenants/`.

## 2. The handlers

- [x] 2.1 In `pkg/kratos/handlers.go` `checkTenantSelectionByEmail`, start from `cookies.FlowStateCookie{LoginChallengeHash: …}` and drop the read of the state cookie.
- [x] 2.2 In `handleUpdateFlow`, write `updatedCookie` when the user is sent to the tenant selection.
- [x] 2.3 `pkg/kratos/handlers_test.go`: `TestHandleUpdateFlowPersistsResolvedCookieForTenantSelection`; it fails with 2.2 reverted. Update the comments in `pkg/tenants/handlers.go` and `pkg/kratos/interfaces.go`. Completion: `go vet ./pkg/... ./internal/...` and `go test ./pkg/... ./internal/...`.

## 3. Verification in a browser

- [x] 3.1 On the identity platform's test plane (`canonical/canonical-identity-platform-testing`) with `MULTI_TENANCY_ENABLED=true` and no Tenant Service hooks: on v0.28.0 and on `main` the second user's tokens carry the tenant of the email entered first, or none; with an image built from the branch each user gets their own, in both orders.
- [x] 3.2 Same plane, same image: the tenant selection scenarios pass as before, and the rest of that suite gives the same results as on `main`.
- [x] 3.3 Read, not run: an account other than the email entered, and a tenant lost during a login. The outcomes are in `design.md` and in the spec.

## 4. Specification

- [x] 4.1 `npx @fission-ai/openspec validate resolve-tenant-for-signed-in-user --strict` passes.
- [x] 4.2 Archive the change so that `openspec/specs/login-tenant-resolution/spec.md` holds the requirements and no active change is left.

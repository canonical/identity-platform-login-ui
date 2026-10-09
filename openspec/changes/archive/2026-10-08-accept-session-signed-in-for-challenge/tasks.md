## 1. The state cookie records the start of a login

- [x] 1.1 In `internal/cookies/cookies.go`, add `LoginStartedAt time.Time` (`json:"ls,omitzero"`) to `FlowStateCookie`, carry it in `RenewForChallenge` for the same challenge only, and add `StartLogin(loginChallenge, flowIssuedAt)` and `SignedInFor(loginChallenge, authenticatedAt)`.
- [x] 1.2 `internal/cookies/cookies_test.go`: `TestSignedInFor` (after, before and at the start; no start; another challenge; no authentication time; through the JSON form) and `TestStartLoginAndRenewForChallenge`. Completion: `go test ./internal/cookies/`.

## 2. A session that did not sign in is a reused session

- [x] 2.1 In `pkg/tenants/resolver.go` `InterceptLogin`, return `DeferMFAChecks` for a session that fails `SignedInFor`, after renewing the cookie for the challenge. In `StoreTenant`, start from `RenewForChallenge`.
- [x] 2.2 In `pkg/kratos/handlers.go`, call `StartLogin` with the flow's `issued_at` in `handleUpdateFlow`, and pass `intercept.Cookie` to `MustReAuthenticate` in `handleCreateFlow`.
- [x] 2.3 Tests: `TestInterceptLoginDefersForSessionThatDidNotSignInForChallenge` and the `StoreTenant` case in `pkg/tenants/resolver_test.go`; `TestHandleCreateFlowForcesLoginForSessionThatDidNotSignInForChallenge` and `TestHandleUpdateFlow` in `pkg/kratos/handlers_test.go`. Completion: `go vet ./pkg/... ./internal/...` and `go test ./pkg/... ./internal/...`.

## 3. Verification in a browser

- [x] 3.1 On the identity platform's test plane (`canonical/canonical-identity-platform-testing`) with `MULTI_TENANCY_ENABLED=true`: the steps of issue #988 give the client a code on v0.28.0 and on `main`, and show the login again with an image built from the branch.
- [x] 3.2 Same plane, same image: a `max_age=0` login completes with one sign-in for a user with one tenant, for a user with several, and through an external provider. The rest of that suite gives the same results as on `main`.
- [x] 3.3 Read, not run: the password reset started from a login, and a login in progress across the deploy. The outcomes are in `design.md` (Risks) and in the spec.

## 4. Specification

- [x] 4.1 `npx @fission-ai/openspec validate accept-session-signed-in-for-challenge --strict` passes.
- [x] 4.2 Archive the change so that `openspec/specs/login-session-reuse/spec.md` holds the requirements and no active change is left.

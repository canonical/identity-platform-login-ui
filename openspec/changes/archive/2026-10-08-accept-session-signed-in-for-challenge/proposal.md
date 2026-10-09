## Why

Issue #988: with `MULTI_TENANCY_ENABLED=true`, a login that demands re-authentication (`prompt=login`, `max_age`) can be accepted on the session the browser already has. With that session, entering an email on the login page and opening the login request again is enough. No password and no second factor is asked, the client gets its authorization code, and Hydra stamps the accept as a new authentication, so the tokens carry a fresh `auth_time`.

### Problem statement

- `handleCreateFlow` (`pkg/kratos/handlers.go`) accepts a login without asking Hydra when the tenant resolver finds the state cookie bound to the login challenge and a Kratos session. It takes the binding as proof that the user signed in for that challenge.
- The cookie is bound before any credential is given: at the email step, by the tenant selection endpoint, before the browser leaves for an external provider, and by the verification redirect of a reused session.
- A setup flag recorded in the cookie for a challenge (authenticator, passkey, backup code) makes `MustReAuthenticate` answer "no" without asking Hydra, whoever the session belongs to.

### Scope

- State when a session may be accepted for a client's login without asking Hydra, and what the Login UI does with every other session.
- Record in the state cookie when the login for a challenge started, and compare it with when the session authenticated.
- Stop state recorded for one challenge, or for someone else's sign-in, from vouching for a session.

### Non-goals

- Proving that a sign-in went through the login it is accepted for. The evidence is a time; the spec states what that leaves open.
- Binding the tenant recorded for a challenge to a user.
- Any change with multi-tenancy disabled, where the handler already goes through `MustReAuthenticate` for every session.
- Any frontend (`ui/`) change.

### Success criteria

- A login request that demands re-authentication, opened again after the email step with a session from before, shows the login again.
- A sign-in that completes for a challenge is still accepted with one sign-in: with a password and a second factor, through an external provider, after a first-time authenticator setup, and for a user who selects a tenant.
- A session reused for another client's login is still accepted without credentials when Hydra allows it.

## What Changes

- `internal/cookies/cookies.go`: `FlowStateCookie` gains `LoginStartedAt`. `StartLogin` records it once per challenge; `SignedInFor` says whether a session authenticated after it; `RenewForChallenge` carries it only for the same challenge.
- `pkg/kratos/handlers.go`: `handleUpdateFlow` records the start from the issue time of the login flow it submitted. `handleCreateFlow` gives `MustReAuthenticate` the resolver's cookie.
- `pkg/tenants/resolver.go`: `InterceptLogin` treats a session that did not sign in for the challenge as a reused session, and renews the cookie for it. `StoreTenant` renews the cookie instead of rebinding it.
- User-visible besides the fix, and accepted:
  - A password reset started from a login ends on the sign-in form when the browser returns to that login. Before, the session of the recovery was accepted for it.
  - A login in progress across the deploy is asked to sign in once more if it returns from a detour, because its cookie has no start.
  - A reused session of a user with several tenants costs one more request to Hydra after the tenant page, and is no longer checked for a missing authenticator or passkey there.

## Capabilities

### New Capabilities

- `login-session-reuse`: when the Login UI may accept a client's login on a session the browser already has, with multi-tenancy enabled, and what it records in the state cookie to decide it.

### Modified Capabilities

<!-- None: no existing spec covers the acceptance of a login. -->

## Impact

- **Backend Go**: `internal/cookies`, `pkg/kratos`, `pkg/tenants`, with their unit tests.
- **Frontend**: unaffected.
- **Kratos**: no configuration change. The Login UI reads two times Kratos already returns: a login flow's `issued_at` and a session's `authenticated_at`.
- **Hydra**: no configuration change. Hydra is asked for the login request in more cases; its answer (`skip`) decides them.
- **Cookies**: the encrypted state cookie has one more field (`ls`). A cookie written before the change has no start and is treated as not signed in.
- **OpenFGA**: unaffected.

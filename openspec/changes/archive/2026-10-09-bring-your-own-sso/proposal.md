## Why

### Problem statement

A tenant cannot make its users sign in through its own identity provider. Login UI offers every user the same sign-ins (the portal password, the public providers, a passkey) whatever tenant they sign in to, and asks for MFA by one platform-wide rule. A company that manages its users in its own OIDC identity provider needs the opposite: its users sign in there ("company sign-in"), the portal password is not a way into its tenant, and the tenant decides whether MFA is asked on top.

Kratos has no notion of a tenant. It cannot route a user to the sign-ins of one tenant, refuse a session at another, or require MFA per tenant. Login UI is Hydra's login and consent provider for every app, and the only component that knows the tenant chosen for an app sign-in. The decisions therefore belong here.

## What Changes

### Scope

"Bring your own SSO" (BYO-SSO) lets a tenant offer or require company sign-in. In this repository it is an extension of the login, registration, settings and consent flows, switched on by `BYOSSO_ENABLED`. Enabling it also puts each tenant's MFA policy in place of the platform-wide rule of `MFA_ENABLED`.

- **Extension hooks** in `pkg/kratos` and `pkg/extra`: narrow call sites in the existing handlers, which do nothing unless an extension is plugged in.
- **The extension**, `pkg/byosso`, which uses those hooks for:
  - the sign-in screen after the email: the tenants the address may use, then what the chosen tenant offers according to its enforcement (`off`, `optional`, `required`);
  - the checks before every Hydra login is accepted: membership or admission, a tenant that requires company sign-in, which company sign-in a session came from, the tenant's MFA, and re-authentication for `prompt=login` and `max_age`;
  - joining a tenant at sign-in, for a pending invitation or auto-join;
  - company sign-ins through the SSO service: a ticket per attempt, submitted to the one Kratos provider `byo-sso`, confirmed with the receipt the SSO service leaves in the browser, and the new endpoint `GET /api/v0/sso/complete`;
  - Kratos's account-linking page; registration of an address a tenant admits; Connected accounts in settings;
  - a consent check that refuses a login Login UI did not accept;
  - deadlines and retries on the calls to the two backends, and defined answers when one is down.
- **Wiring**: `pkg/web/byosso.go`, `cmd/serve.go`, `internal/config`, `internal/grpc` (the gRPC dial helper, moved out of `pkg/tenants`), two options in `pkg/tenants`.
- **Frontend** (`ui/`): the tenant list and company sign-in buttons on the login page, "Redirecting to …", errors shown in place, company sign-ins on the Connected accounts page, invitations marked in the tenant list.

### Non-goals

- Managing company sign-ins or a tenant's policy. Login UI only reads them: connections live in the SSO service; the policy (enforcement, domains, auto-join, MFA) and the invitations live in the tenant service.
- Talking to a customer's identity provider. The SSO service does that; Login UI never sees a token of it.
- Writing links between accounts and company sign-ins. Kratos writes every link, through its own account linking and registration. Login UI only asks the SSO service to remove one.
- Revoking sessions. A removed member, a raised policy or a deleted connection takes effect at the next sign-in.
- A way into a tenant that requires company sign-in while its identity provider is down.
- `prompt=none`: Login UI does not read it, as before.
- Rate limiting of the email step and the tenant lookup.
- An MFA policy other than `none` or `required`.
- Metrics and security-log entries for the extension: it traces and logs only.
- Playwright cases in `ui/tests`. The Go code is covered by Go unit tests; nothing in this repository covers the changes in `ui/`, which a browser suite kept outside it exercises.
- Any change to OpenFGA, or to Kratos or Hydra code.

### Success criteria

- With `BYOSSO_ENABLED` off the backend behaves as before this change. It was compared with upstream `main` in a browser, step by step, and the tests that existed pass unmodified.
- With it on, no Hydra login is accepted for a session that does not pass the checks of the chosen tenant, on any path: the login page with a session, a first factor or MFA just submitted, a tenant just picked.
- A tenant that requires company sign-in never falls back to the portal password: not when the SSO service is down, not when no company sign-in applies to the address, not when the feature is half configured (the process then does not start).
- `go vet ./...` and `go test ./...` pass after `make mocks`; ESLint reports no error on the changed frontend files.
- `openspec validate bring-your-own-sso --strict` passes.

## Capabilities

### New Capabilities

- `login-extension-hooks`: the hooks in `pkg/kratos` and `pkg/extra`, and where each is called.
- `sign-in-screen`: the tenant list after the email, then the chosen tenant's options.
- `tenant-sign-in-checks`: the checks before every Hydra login accept, and what the accept carries.
- `tenant-join-at-sign-in`: a pending invitation or auto-join becomes a membership at sign-in.
- `company-sign-in`: tickets and receipts, the `byo-sso` submission, provenance, `/api/v0/sso/complete`, the cookies.
- `account-linking`: Kratos's account-linking page.
- `registration-routing`: "Create account" with an address a tenant admits.
- `connected-accounts`: company sign-ins in settings.
- `tenant-mfa`: MFA per tenant, decided in Login UI.
- `re-authentication`: `prompt=login`, `max_age`, another user remembered by Hydra.
- `consent-check`: consent refused for a login Login UI did not accept.
- `backend-resilience`: deadlines, retries, service tokens, outages.
- `byosso-configuration`: the settings and the startup checks.
- `byosso-disabled`: what is unchanged with the flag off, in the backend and in the frontend.

### Modified Capabilities

None. The accepted specs (`artifact-security-gate`, `resource-indicator-audience`, `tenant-grpc-client`) stay as they are: the client interface of `pkg/tenants` is unchanged and, with the flag off, every tenant lookup goes through its `LookupTenants`; the consent handler's audience rule is untouched. With the flag on, `pkg/tenants` gets a second client interface by option, and its lookups by email go through `ListSignInTenants` (`tenant-sign-in-checks`).

## Impact

### Backend (Go)

- New: `pkg/byosso`, `pkg/kratos/extension.go`, `pkg/extra/extension.go`, `pkg/web/byosso.go`, `internal/grpc/client.go`; the endpoint `GET /api/v0/sso/complete`, registered only when the extension is enabled.
- Changed: `pkg/kratos/handlers.go` and `interfaces.go` (hook calls; the session checks moved into `enforceSessionChecks`; `RedirectResponse` exported), `pkg/extra/handlers.go` and `interfaces.go` (consent hook), `pkg/tenants` (two options, the client interface one of them takes, the `invited` flag), `pkg/web/router.go` (`NewRouter` returns an error), `cmd/serve.go`, `internal/config/specs.go`, `internal/cookies/cookies.go` (one field). Removed: `pkg/tenants/grpc.go`.
- Dependencies: `github.com/canonical/identity-platform-api` with the `v0/sso` package (`SSOSignInService`) and, in `v0/tenant`, `TenantSignInService` (`ListSignInTenants`, `GetSignInContext`, `JoinTenant`, `CreatePersonalTenant`). **`go.mod` points at a local checkout of that module through a `replace` directive: it has to name a published version before this change can merge.**

### Frontend (Next.js)

- `ui/pages/login.tsx`: the tenant list and company sign-in buttons (nodes of groups `tenant` and `sso` the backend adds to the flow); "Redirecting to <label>…"; the errors `sso_unavailable`, `sso_not_applicable` and `tenant_not_a_member` shown on the page.
- `ui/pages/manage_connected_accounts.tsx`: a "Company sign-ins" section. `ui/pages/register.tsx`: follows the redirect to a company sign-in. `ui/pages/select_tenant.tsx`: a pending invitation is marked.
- New components `FlowMessages` and `RedirectingNotice`; helpers in `ui/util` and `ui/api`.

### Settings added

`BYOSSO_ENABLED` (default off; needs `MULTI_TENANCY_ENABLED`). It has no separate switch for MFA. Enabling it puts the tenants' MFA rule in place of `MFA_ENABLED`'s, which then has no effect; needs Kratos to run with `session.whoami.required_aal: aal1` (below); and cannot be combined with `OIDC_WEBAUTHN_SEQUENCING_ENABLED` (the process does not start). For the channel to the SSO service: `SSO_SERVICE_GRPC_ADDRESS`, `SSO_SERVICE_GRPC_TIMEOUT` (3 s), `SSO_SERVICE_TLS_ENABLED`. For Login UI's own service token (client credentials): `SERVICE_TOKEN_URL`, `SERVICE_CLIENT_ID`, `SERVICE_CLIENT_SECRET`, `SERVICE_TOKEN_SCOPES`. And `KRATOS_PRIVILEGED_SESSION_MAX_AGE` (1 h), which has to equal Kratos's `selfservice.flows.settings.privileged_session_max_age`. The `byosso-configuration` spec has the table and the startup checks.

### Kratos

No Kratos code changes. Two uses of its APIs are new: the backend starts a verification flow itself (to verify the address of a company-sign-in session), and it looks an account up by its address through the admin API (`ListIdentities` by credentials identifier). The extension relies on this configuration:

- An OIDC provider with the id `byo-sso` (a constant in `pkg/byosso`). Login UI submits it with `upstream_parameters.login_hint` set to the ticket of the SSO service. Nobody else may use that provider: Login UI removes its nodes from every flow, and refuses every submission of it that it did not make itself.
- Identifier-first login, as multi-tenancy uses already.
- `selfservice.flows.registration.login_hints: true`, so that Kratos's account-linking page lists only the providers the account has.
- Allowed return URLs that include Login UI's `/ui/login`, `/api/v0/sso/complete` and `/ui/manage_connected_accounts`.
- Verification with the code method, and recovery.
- `session.whoami.required_aal: aal1`, so that Login UI sees the `aal1` session of an account that has a second factor and can apply the tenant's policy to it. `selfservice.flows.settings.required_aal` stays `highest_available`.

### Hydra

No Hydra code or configuration changes. The login accept is unchanged: subject, `remember`, `amr`, and `context.tenant_id`. The login request is now also read for `prompt`, `max_age`, the remembered subject and its `sid`. Two calls to the admin API are new: `RevokeOAuth2LoginSessions` by `sid` (another user remembered in this browser) and `RejectOAuth2ConsentRequest` (`login_required`). Login UI needs an OAuth2 client with the client-credentials grant at `SERVICE_TOKEN_URL`; its token authenticates the calls to the tenant service and the SSO service.

### Cookies

Four new cookies, all encrypted with the existing cookie cipher (`COOKIES_ENCRYPTION_KEY`), each sealed together with its own name so that the value of one is not accepted as another, `HttpOnly`, `Secure`, `SameSite=Lax`: `login_ui_sso_signin`, `login_ui_sso_fresh` and `login_ui_sso_registration` (30 minutes each) and `login_ui_sso_provenance` (until the Kratos session expires). The `company-sign-in` spec lists what each holds. The existing state cookie gains one field, `tc`. Login UI expires the Kratos session cookie in the browser when it starts a company sign-in, so an abandoned company sign-in leaves the browser signed out.

Login UI also handles one cookie that is not its own: the receipt the SSO service sets in the browser that completes a company sign-in, `__Host-sso_receipt_<digest of the ticket>`. Login UI sends its value to the SSO service with the ticket and clears it with the sign-in cookie; it cannot read it. For the browser to send it to Login UI, the SSO service's browser pages have to be served on Login UI's host.

### OpenFGA

None. The provider filter and the provider check of the handlers run as before for what the browser submits; the `byo-sso` provider is removed after the filter, and refused before the check. The per-app provider allow-list (`CheckAllowedProvider`) does not apply to company sign-ins: Login UI submits `byo-sso` itself, past the handler's check, and which company sign-ins a user is offered follows the tenant's policy, not the app's list.

### What changes with `BYOSSO_ENABLED` off

The backend behaves as before: the hooks do nothing, `/api/v0/sso/complete` is not registered, the new settings are not validated, the channel to the tenant service is dialled with the same options, and tenant lookups go through `LookupTenants`, which lists no invitations or auto-join candidates. The frontend is one bundle for every deployment: what it gained appears only in answer to what the extension sends (requirements in the `byosso-disabled` spec). It relies on the frontend showing Kratos's flow-level messages and following flow errors (the `frontend-flow-feedback` capability), which is not part of this change.

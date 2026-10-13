## Why

### Problem statement

A tenant cannot make its users sign in through its own identity provider. Login UI offers every user the same sign-ins, whatever tenant the user signs in to. These are the portal password, the public providers and a passkey. Login UI also asks for MFA by one rule for the whole platform.

A company that manages its users in its own OIDC identity provider needs the opposite. Its users sign in there ("company sign-in"). The portal password is not a way into its tenant. The tenant decides if the system asks for MFA on top.

Kratos has no notion of a tenant. It cannot send a user to the sign-ins of one tenant. It cannot refuse a session at another tenant. It cannot require MFA for each tenant. Login UI is Hydra's login and consent provider for every app. It is also the only component that knows the tenant chosen for an app sign-in. For these reasons the decisions belong here.

## What Changes

### Scope

"Bring your own SSO" (BYO-SSO) lets a tenant offer or require company sign-in. In this repository it is an extension of the login, registration, settings and consent flows. `BYOSSO_ENABLED` switches it on. When the flag is on, each tenant's MFA policy replaces the platform-wide rule of `MFA_ENABLED`.

- **Extension hooks** in `pkg/kratos` and `pkg/extra`. They are narrow call sites in the existing handlers. They do nothing unless an extension is plugged in.
- **The extension**, `pkg/byosso`, which uses those hooks for these functions:
  - The sign-in screen after the email. It shows the tenants that the address can use. Then it shows what the chosen tenant offers, according to its enforcement (`off`, `optional`, `required`).
  - The checks before every Hydra login is accepted. They cover these points:
    - membership or admission
    - a tenant that requires company sign-in
    - the company sign-in that a session came from
    - the tenant's MFA
    - re-authentication for `prompt=login` and `max_age`
  - The join of a tenant at sign-in, for a pending invitation or auto-join.
  - Company sign-ins through the SSO service. Each attempt has a ticket. Login UI submits the ticket to the one Kratos provider `byo-sso`. The receipt that the SSO service leaves in the browser confirms the attempt. The new endpoint is `GET /api/v0/sso/complete`.
  - Kratos's account-linking page, the registration of an address that a tenant admits, and Connected accounts in settings.
  - A consent check that refuses a login that Login UI did not accept.
  - Deadlines and retries on the calls to the two backends, and defined answers when one of them is down.
- **Wiring**: `pkg/web/byosso.go`, `cmd/serve.go`, `internal/config` and `internal/grpc` (the gRPC dial helper, moved out of `pkg/tenants`). Also two options in `pkg/tenants`.
- **Frontend** (`ui/`) has these changes:
  - the tenant list and the company sign-in buttons on the login page
  - "Redirecting to …"
  - errors shown in place
  - company sign-ins on the Connected accounts page
  - invitations marked in the tenant list

### Non-goals

- The management of company sign-ins and of a tenant's policy. Login UI only reads them. Connections live in the SSO service. The policy (enforcement, domains, auto-join, MFA) and the invitations live in the tenant service.
- Contact with a customer's identity provider. The SSO service does that. Login UI never sees a token of that provider.
- Writes of links between accounts and company sign-ins. Kratos writes every link, through its own account linking and registration. Login UI only asks the SSO service to remove a link.
- Revocation of sessions. A removed member, a raised policy or a deleted connection takes effect at the next sign-in.
- A way into a tenant that requires company sign-in while its identity provider is down.
- `prompt=none`. Login UI does not read it, as before.
- Rate limits for the email step and the tenant lookup.
- An MFA policy other than `none` or `required`.
- Metrics and security-log entries for the extension. It only writes traces and logs.
- Playwright cases in `ui/tests`. Go unit tests cover the Go code. Nothing in this repository covers the changes in `ui/`. A browser suite outside this repository tests them.
- A change to OpenFGA, or a change to the code of Kratos or Hydra.

### Success criteria

- With `BYOSSO_ENABLED` off, the backend behaves as before this change. We compared it with upstream `main` in a browser, step by step. The tests that existed pass without a change.
- With the flag on, the system accepts no Hydra login for a session that does not pass the chosen tenant's checks. This is true on every path. The paths are: the login page with a session, a first factor or MFA just submitted, and a tenant just picked.
- A tenant that requires company sign-in never falls back to the portal password. This is true when the SSO service is down. It is true when no company sign-in applies to the address. It is true when the configuration of the feature is half done, because the process then does not start.
- `go vet ./...` and `go test ./...` pass after `make mocks`. ESLint reports no error on the changed frontend files.
- `openspec validate bring-your-own-sso --strict` passes.

## Capabilities

### New Capabilities

- `login-extension-hooks`: the hooks in `pkg/kratos` and `pkg/extra`, and where each hook is called.
- `sign-in-screen`: the tenant list after the email, then the options of the chosen tenant.
- `tenant-sign-in-checks`: the checks before every accept of a Hydra login, and what the accept carries.
- `tenant-join-at-sign-in`: a pending invitation or auto-join becomes a membership at sign-in.
- `company-sign-in`: tickets and receipts, the `byo-sso` submission, provenance, `/api/v0/sso/complete` and the cookies.
- `account-linking`: Kratos's account-linking page.
- `registration-routing`: "Create account" with an address that a tenant admits.
- `connected-accounts`: company sign-ins in settings.
- `tenant-mfa`: MFA for each tenant, decided in Login UI.
- `re-authentication`: `prompt=login`, `max_age`, and another user that Hydra remembers.
- `consent-check`: consent refused for a login that Login UI did not accept.
- `backend-resilience`: deadlines, retries and outages.
- `byosso-configuration`: the settings and the startup checks.
- `byosso-disabled`: what stays the same with the flag off, in the backend and in the frontend.

### Modified Capabilities

None. The accepted specs (`artifact-security-gate`, `resource-indicator-audience`, `tenant-grpc-client`) stay as they are. The client interface of `pkg/tenants` does not change. With the flag off, every tenant lookup goes through its `LookupTenants`. The audience rule of the consent handler does not change.

With the flag on, `pkg/tenants` gets a second client interface by option. Its lookups by email then go through `ListSignInTenants` (`tenant-sign-in-checks`).

## Impact

### Backend (Go)

- New: `pkg/byosso`, `pkg/kratos/extension.go`, `pkg/extra/extension.go`, `pkg/web/byosso.go` and `internal/grpc/client.go`. Also the endpoint `GET /api/v0/sso/complete`, which the system registers only when the extension is enabled.
- Changed: `pkg/kratos/handlers.go` and `interfaces.go`. They have the hook calls. The session checks moved into `enforceSessionChecks`, and `RedirectResponse` is exported.
- Changed: `pkg/extra/handlers.go` and `interfaces.go` (the consent hook).
- Changed: `pkg/tenants` (two options, the client interface that one of them takes, and the `invited` flag).
- Changed: `pkg/web/router.go` (`NewRouter` returns an error), `cmd/serve.go`, `internal/config/specs.go` and `internal/cookies/cookies.go` (one field).
- Removed: `pkg/tenants/grpc.go`.
- Dependencies: `github.com/canonical/identity-platform-api` with the `v0/sso` package (`SSOSignInService`) and, in `v0/tenant`, `TenantSignInService` (`ListSignInTenants`, `GetSignInContext`, `JoinTenant`, `CreatePersonalTenant`). **`go.mod` requires the commit of the pull request that adds them to that module. It must name a released version before this change can merge.**

### Frontend (Next.js)

- `ui/pages/login.tsx`: the tenant list and the company sign-in buttons. These are nodes of the groups `tenant` and `sso` that the backend adds to the flow. Also "Redirecting to <label>…", and the errors `sso_unavailable`, `sso_not_applicable` and `tenant_not_a_member` on the page.
- `ui/pages/manage_connected_accounts.tsx`: a "Company sign-ins" section.
- `ui/pages/register.tsx`: it follows the redirect to a company sign-in.
- `ui/pages/select_tenant.tsx`: it marks a pending invitation.
- New components `FlowMessages` and `RedirectingNotice`, and new helpers in `ui/util` and `ui/api`.

### Settings added

`BYOSSO_ENABLED` is off by default. It needs `MULTI_TENANCY_ENABLED`. It has no separate switch for MFA. When the flag is on, the MFA rule of the tenants replaces the rule of `MFA_ENABLED`, and `MFA_ENABLED` has no effect. Kratos must run with `session.whoami.required_aal: aal1` (see below). The flag cannot be on together with `OIDC_WEBAUTHN_SEQUENCING_ENABLED`, because the process does not start.

For the channel to the SSO service the settings are `SSO_SERVICE_GRPC_ADDRESS`, `SSO_SERVICE_GRPC_TIMEOUT` (3 s) and `SSO_SERVICE_TLS_ENABLED`. Login UI calls both services without a token. They serve these RPCs to the internal network.

`KRATOS_PRIVILEGED_SESSION_MAX_AGE` (1 h) must equal Kratos's `selfservice.flows.settings.privileged_session_max_age`. The `byosso-configuration` spec has the table and the startup checks.

### Kratos

The code of Kratos does not change. The backend uses its APIs in two new ways. It starts a verification flow itself, to verify the address of a company-sign-in session. It also looks up an account by its address through the admin API (`ListIdentities` by credentials identifier).

The extension relies on this configuration:

- An OIDC provider with the id `byo-sso` (a constant in `pkg/byosso`). Login UI submits it with `upstream_parameters.login_hint` set to the ticket of the SSO service. Nobody else can use that provider. Login UI removes its nodes from every flow. Login UI refuses every submission of it that Login UI did not make itself.
- Identifier-first login, as multi-tenancy already uses.
- `selfservice.flows.registration.login_hints: true`. Then the account-linking page of Kratos lists only the providers that the account has.
- Allowed return URLs that include the `/ui/login`, `/api/v0/sso/complete` and `/ui/manage_connected_accounts` of Login UI.
- Verification with the code method, and recovery.
- `session.whoami.required_aal: aal1`. Login UI then sees the `aal1` session of an account that has a second factor. It can apply the policy of the tenant to that session. `selfservice.flows.settings.required_aal` stays `highest_available`.

### Hydra

The code and the configuration of Hydra do not change. The login accept stays the same: subject, `remember`, `amr` and `context.tenant_id`. Login UI now also reads `prompt`, `max_age`, the remembered subject and its `sid` from the login request.

Two calls to the admin API are new. `RevokeOAuth2LoginSessions` by `sid` ends the Hydra login session when another user is remembered in this browser. `RejectOAuth2ConsentRequest` rejects a consent with `login_required`.

Login UI needs no OAuth2 client of its own. Its calls to the tenant service and the SSO service carry no token.

### Cookies

There are four new cookies. Login UI encrypts all of them with the existing cookie cipher (`COOKIES_ENCRYPTION_KEY`). Each cookie is sealed together with its own name, so the system does not accept the value of one cookie as another. All are `HttpOnly`, `Secure` and `SameSite=Lax`.

- `login_ui_sso_signin`, `login_ui_sso_fresh` and `login_ui_sso_registration` last 30 minutes each.
- `login_ui_sso_provenance` lasts until the Kratos session expires.

The `company-sign-in` spec lists what each cookie holds. The existing state cookie gets one new field, `tc`. Login UI expires the Kratos session cookie in the browser when it starts a company sign-in. For this reason an abandoned company sign-in leaves the browser signed out.

Login UI also handles one cookie that is not its own. The SSO service sets this receipt in the browser that completes a company sign-in. Its name is `__Host-sso_receipt_<digest of the ticket>`. Login UI sends its value to the SSO service with the ticket and clears it with the sign-in cookie. Login UI cannot read it. The browser must send the receipt to Login UI. For this reason the SSO service's browser pages must be on Login UI's host.

### OpenFGA

None. The provider filter and the provider check of the handlers run as before for what the browser submits. The system removes the `byo-sso` provider after the filter. It refuses a submission of the provider before the check.

The allow-list of providers for each app (`CheckAllowedProvider`) does not apply to company sign-ins. Login UI submits `byo-sso` itself, past the check of the handler. The tenant's policy, not the app's list, decides which company sign-ins a user gets.

### What changes with `BYOSSO_ENABLED` off

The backend behaves as before:

- The hooks do nothing.
- `/api/v0/sso/complete` is not registered.
- The system does not validate the new settings.
- The system dials the channel to the Tenant Service with the same options.
- Tenant lookups go through `LookupTenants`. It lists no invitations or auto-join candidates.

The frontend is one bundle for every deployment. What it gained appears only in answer to what the extension sends (the requirements are in the `byosso-disabled` spec). It relies on two behaviors of the frontend: it shows Kratos's flow-level messages, and it follows flow errors. This is the `frontend-flow-feedback` capability. It is not part of this change.

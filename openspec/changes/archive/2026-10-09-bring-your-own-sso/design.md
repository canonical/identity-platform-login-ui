## Context

Login UI is Hydra's login and consent provider and the user interface of Kratos's self-service flows. With multi-tenancy it already asks the tenant service for a user's tenants, lets the user pick one, and puts the pick into the Hydra login accept as `context.tenant_id`, which the consent handler passes on to the token. What it does not have is any rule per tenant: every tenant gets the same sign-ins, and one platform-wide flag decides about MFA.

This change adds those rules. Three other parties are involved, and none of them changes in this repository:

- the **tenant service** owns memberships, personal tenants, pending invitations and each tenant's policy (enforcement, domains, auto-join, MFA). Login UI asks its `TenantSignInService` four things: `ListSignInTenants`, `GetSignInContext`, `CreatePersonalTenant`, `JoinTenant`;
- the **SSO service** owns the company sign-ins (connections to customers' identity providers) and runs them. Login UI asks it five things: `ListOptions`, `StartAttempt`, `CompleteAttempt`, `ListLinks`, `DeleteLink`;
- **Kratos** keeps one OIDC provider, `byo-sso`, for all company sign-ins, and writes every link between an account and a company sign-in itself.

The code is laid out as follows.

| path | role |
|---|---|
| `pkg/kratos/extension.go`, `pkg/extra/extension.go` | the hook interfaces' default implementations and the `WithExtension` options; exported wrappers of the handlers' answers to a Kratos error (`WriteGetFlowError`, `WriteUpdateFlowError`), for the extension's own calls to Kratos |
| `pkg/byosso/handlers.go` | the `API` type: every hook, and `GET /api/v0/sso/complete` |
| `pkg/byosso/signin.go` | the sign-in screen: tenant list, a tenant's options, picks |
| `pkg/byosso/accept.go`, `decide.go` | the procedure before an accept, and the decision as a function with no I/O |
| `pkg/byosso/attempt.go` | starting and confirming a company sign-in; the fresh mark |
| `pkg/byosso/linking.go`, `registration.go`, `settings.go`, `mfa.go`, `consent.go` | one file per flow the extension takes part in |
| `pkg/byosso/directory.go`, `sso.go`, `hydra.go`, `kratos.go` | thin clients of the tenant service, the SSO service, Hydra and Kratos |
| `pkg/byosso/kratos_decorator.go` | removes the `byo-sso` nodes from every flow |
| `pkg/byosso/cookies.go`, `grpc.go` | the extension's cookies; dial options and the service token |
| `pkg/web/byosso.go`, `cmd/serve.go`, `internal/config/specs.go`, `internal/grpc/client.go` | wiring and configuration |
| `ui/pages/login.tsx`, `register.tsx`, `manage_connected_accounts.tsx`, `select_tenant.tsx`; `ui/components/FlowMessages.tsx`, `RedirectingNotice.tsx`; `ui/util/*` | the frontend |

## Goals / Non-Goals

**Goals:**

- A tenant's rules are applied to every app sign-in, whatever path the browser takes to the accept.
- With the flag off, the backend is the one before this change.
- Kratos and Hydra are used as they are: their own account linking, registration, verification, recovery, MFA and consent rejection, with configuration only.
- A slow or absent backend produces an answer the user can act on, and never a weaker check.

**Non-Goals:** those of the proposal. In particular Login UI keeps no state of its own on the server: what it must remember between requests is in encrypted cookies.

## Decisions

### 1. Hooks and an extension package, not conditions inside the handlers

**Decision**: `pkg/kratos/handlers.go` and `pkg/extra/handlers.go` call an `ExtensionInterface` at thirteen call sites, in nine handlers. `NoOpExtension` is the default; `pkg/byosso.API` implements both interfaces and is plugged in by `pkg/web/byosso.go` when `BYOSSO_ENABLED` is set.

**Rationale**: the handlers serve every deployment. With hooks, "off means unchanged" is a property a reviewer can check at the call sites, each of which either passes a flow through or goes on when the hook returns false. The clients of the tenant service and the SSO service, the cookies and the rules stay out of `pkg/kratos`. The only restructuring of the handler is the extraction of `enforceSessionChecks` from `handleCreateFlow`, with the same checks in the same order, so that the extension's path can run them too.

**Alternative turned down**: `if byossoEnabled` branches in the handlers. The branches would have had to be read in every review of those handlers from then on, and the dependencies would have landed in `pkg/kratos`. A middleware in front of the routes was not enough either: the extension needs what the handler has already fetched (the flow, the session, the state cookie) and must act in the middle of a handler, right before the accept.

### 2. The extension answers every login page request that has a session

**Decision**: when the extension reports `HandlesSessionLogin`, `handleCreateFlow` hands every request with a Kratos session and a login challenge to `HandleSessionLogin`, after its own session checks. The handler's tenant resolver (`InterceptLogin`) and its `MustReAuthenticate` are not consulted. For the same reason `handleUpdateFlow` does not ask the resolver to select a tenant after a submission: the extension resolves it in `BeforeAcceptLogin`.

**Rationale**: the handler's own path decides three things the extension must decide differently. Which tenant: the resolver looks tenants up by identity id, which does not list the tenants an address is only admitted to (the extension asks for the sign-in tenants of the address), and it binds a single membership on its own, which would replace an invitation or auto-join tenant the user picked once the state cookie has expired. Whether the session may be reused: `MustReAuthenticate` follows Hydra's `skip`, which says nothing about how the session relates to the tenant, and cannot express `max_age` against the first factor of the Kratos session (decision 9). And when to accept: the handler accepts as soon as a tenant is resolved. Every company sign-in, every public sign-in and every reused session arrives at this one request, so this is where the tenant's checks have to be.

**Alternative turned down**: keep the handler's path and rely on `BeforeAcceptLogin` alone. The handler would then have chosen the tenant and decided about re-authentication before the hook is reached; two places would decide the same request, with different inputs.

### 3. Which company sign-in a session came from is kept in an encrypted cookie

**Decision**: when Login UI starts a company sign-in it writes `login_ui_sso_signin` (the ticket, and the id of the session the browser has, if any). When a session with another id that includes a `byo-sso` method comes back, it asks the SSO service to confirm the attempt (`CompleteAttempt`), sending the ticket with the receipt the SSO service left in the browser that completed the sign-in (its cookie `__Host-sso_receipt_<digest of the ticket>`), and writes `login_ui_sso_provenance` (`session id`, `connection id`). At later accepts the provenance counts only while the cookie names the current session.

**Rationale**: all company sign-ins share one Kratos provider, so the session says "byo-sso" and nothing about the connection, and a tenant must accept only sessions from its own connections. Login UI has no store. A cookie bound to the session id needs none, the SSO service (not the browser) vouches for what goes into it, and its loss fails closed: a company-sign-in session with no provenance goes through the company sign-in again. Two conditions make "a sign-in has just happened" checkable without comparing clocks: every first factor is started without the Kratos session cookie, and Login UI expires that cookie before sending the browser to the identity provider, so Kratos always issues a new session and the id differs from the recorded one.

The ticket alone proves too little: it is valid from the moment the sign-in starts, and one account can be linked to the company sign-ins of two tenants. A user who started a sign-in at tenant A, stopped at its identity provider and then signed in through B could present A's ticket with B's session, and the session would count as A's. The SSO service has to know that A's sign-in was completed, and in this browser. So it hands the browser whose sign-in it accepts a receipt that only it can make or check, and Login UI passes the receipt on with the ticket, reading nothing in it.

**Alternative turned down**: sessions bound to a tenant (an identity or a session per tenant). A user has one account and one session; a session from one company sign-in must also serve a second app at the same tenant, and a password session several tenants. A server-side record keyed by session would need a store Login UI does not have.

A record of completed sign-ins kept by the SSO service, in place of the receipt, was turned down too. It says that the identity provider answered for a ticket, not in which browser: whoever started the attempt could have another user's browser finish it (that user opens the link and is signed in silently at the identity provider), and then present the ticket with a session made another way. The proof has to end up in the browser that signed in.

### 4. The tenant's MFA is decided in Login UI, with Kratos at `aal1`

**Decision**: with the extension enabled, Kratos runs with `session.whoami.required_aal: aal1`, the handlers' platform-wide MFA flag is passed as off (`pkg/web/byosso.go`), and `decide` asks for MFA per tenant: always for a password, for a company or public sign-in only where the tenant's policy is `required`. MFA counts when the session lists a second factor at `aal2`. This has no flag of its own: `BYOSSO_ENABLED` brings it, and `MFA_ENABLED` has no effect while it is on.

**Rationale**: Kratos's required assurance level is one value for the deployment. Left at `highest_available`, Kratos refuses the `aal1` session of any account that has a second factor, so a company sign-in at a tenant that asks for no MFA would still be sent to enter a code. Login UI is the only component that knows the tenant. The step-up itself is still Kratos's (`aal=aal2`, the settings flow for the first set-up, and `selfservice.flows.settings.required_aal` stays `highest_available` for the account pages).

**Alternative turned down**: keep `highest_available` and `MFA_ENABLED`. It cannot express "only where the tenant requires it". The cost of the decision is that a Kratos setting and a Login UI flag must change together (Rollout), and that an `aal1` session is now something consent can meet (decision 6).

### 5. Login UI creates the membership, just before the accept

**Decision**: an account that a pending invitation or auto-join admits is checked as a member. When its session passes every check, `checkAndAccept` calls `JoinTenant` and then accepts. The tenant service checks admission again and may answer `NOT_ADMITTED`.

**Rationale**: the membership is the consequence of a sign-in that satisfied the tenant, and only Login UI knows that it did: the SSO service sees the identity provider's answer but not the MFA or the address verification that follow, and an invitation to a tenant without company sign-in is accepted with a password, which the SSO service never sees. `JoinTenant` has the same effect when repeated, so it can be retried.

**Alternative turned down**: creating the membership earlier, at invitation time or when the identity provider returns. A user would then be a member of a tenant that requires company sign-in, or MFA, before having passed either. The price of doing it last: if admission is withdrawn between Kratos registering a new account and the accept, the user is refused and the account stays, with no tenant.

### 6. Consent refuses a login that carries no tenant

**Decision**: `GateConsent` rejects (`login_required`) a consent whose subject is not the session's account, or whose login context has no `tenant_id`.

**Rationale**: Kratos accepts a Hydra login request itself when it is given the login challenge. Login UI does not give it one when multi-tenancy is on (`pkg/kratos/service.go`, `CreateBrowserLoginFlow`), but a browser can. With decision 4 such a login could be at `aal1` and would have passed no tenant check. Every login `checkAndAccept` accepts names a tenant (the personal tenant at least), so the absence of `tenant_id` identifies a login accepted elsewhere, with no call to a backend.

**Alternative turned down**: marking accepted logins with something new (an `acr`, a signed marker). The accept stays exactly what it was, which keeps "off means unchanged" true for Hydra and for the apps' tokens. Running the tenant's checks again at consent was turned down too: consent would then depend on the tenant service, for a question it does not need to ask.

### 7. A wiring fault stops the process

**Decision**: `web.NewRouter` returns an error when the extension is enabled and cannot be built, and `serve` exits. `cmd/serve.go` refuses incomplete settings before that.

**Alternative turned down**: what multi-tenancy does when its client is missing, a warning and a fallback to an implementation that does nothing. For an optional convenience that is reasonable. Here the fallback is a Login UI that serves a tenant that requires company sign-in with the portal password, and looks healthy.

### 8. Retries only where a repeat is harmless, and one budget per request

**Decision**: the gRPC service config (`pkg/byosso/grpc.go`) retries `UNAVAILABLE` up to three attempts for the reads and for `JoinTenant`, `CreatePersonalTenant`, `StartAttempt` and `CompleteAttempt`. `DeleteLink` is not retried. Each call has its own timeout, each browser request 12 seconds for everything, and the channels reconnect within 5 seconds.

**Rationale**: an `UNAVAILABLE` call may have run. The listed writes leave the same state a second time; a second `DeleteLink` would answer "no such link" for a removal that worked, and the user would be shown a refusal. A timed-out call is not retried: the budget is better spent answering. The reconnect backoff is shortened because gRPC's default grows to two minutes, during which Login UI would refuse every sign-in after a backend restart. The 12 seconds sit under the server's 15-second write timeout, so the browser gets the extension's answer and not a closed connection.

**Alternative turned down**: retry loops in the callers. The channel policy states once, next to the method list, what is safe to repeat.

### 9. Re-authentication is judged on the Kratos session

**Decision**: `prompt=login` and `max_age` are read from the login request's `request_url` and compared with the completion time of the session's first factor (`pkg/byosso/hydra.go`). A cookie, `login_ui_sso_fresh`, marks a first factor started for a login challenge so that its result is not asked to re-authenticate again. The mark records the session the browser had and the time Kratos issued the login flow the first factor began on, and only another session whose first factor completed at or after that time counts. Kratos timed both, so no allowance for a difference between clocks is needed. When Hydra remembers another subject for the browser, Login UI revokes that Hydra login session and restarts the app's request.

**Rationale**: Hydra stamps the authentication time when a login is accepted, not when the user authenticated, so its own view of `max_age` would call an old Kratos session fresh. And when the accepted subject differs from the remembered one, Hydra restarts the request with `prompt=login` added; Login UI could not tell that from the app's own `prompt=login` and would ask for a first factor the user has just given, which a company sign-in at an identity provider with a live session may not be able to give again.

The session id alone does not prove a fresh first factor: a mark made while the browser sent no session cookie names no session, and any session shown later would differ from it. Hence the time.

**Alternative turned down**: following Hydra's `skip`, as the handler does. See decision 2.

### 10. Kratos writes the links; Login UI only removes them, through the SSO service

**Decision**: a first company sign-in for an existing account goes through Kratos's account-linking flow, for a new address through Kratos's registration. Login UI adds the account's other company sign-ins to the account-linking page (`linking.go`) and to Kratos's refresh login (`settings.go`), and removes a link with `DeleteLink`.

**Rationale**: account linking already makes the owner prove the account before a link is written, which is the protection against an identity provider asserting someone else's address. Kratos's settings unlink cannot be used: it removes every link of the provider, that is, all company sign-ins at once.

## Data and control flow

A company sign-in from the email to the app's code, for a user with one tenant that requires its one company sign-in, and an account already linked to it:

1. **Login page.** The app sends the browser to Hydra, Hydra to `/ui/login?login_challenge=C`. The frontend calls `GET /api/kratos/self-service/login/browser`; Login UI creates a Kratos login flow that returns to that page (the challenge is not given to Kratos).
2. **Email.** `POST …/login/id-first` → Kratos takes the identifier step → `BeforeTenantSelection`: tenant service `ListSignInTenants` (one tenant T) and `GetSignInContext` (`required`, connection K); SSO service `ListOptions` (one), then `StartAttempt` (ticket). Login UI submits the flow to Kratos (`oidc`, `byo-sso`, `login_hint` = ticket), writes the sign-in cookie, binds T to C in the state cookie, writes the fresh cookie, expires the Kratos session cookie, and answers with Kratos's redirect and the label. The page shows "Redirecting to …".
3. **Outside Login UI.** The browser goes to the OpenID Provider behind `byo-sso`, which the SSO service fronts, to the customer's identity provider and back to Kratos's callback. On the way back the SSO service accepts the sign-in and sets its receipt cookie for the ticket. Kratos finds the link, issues a new session and returns to the login page of C. (With no link: account linking, or registration; then the same return.)
4. **Return.** The frontend calls `GET …/login/browser?login_challenge=C` again, now with a session → `HandleSessionLogin` → `checkAndAccept`: the address is verified; SSO service `CompleteAttempt` with the ticket and its receipt → provenance cookie for (session, K), the sign-in cookie and the receipt cookie cleared; Hydra `GetOAuth2LoginRequest`; tenant T from the state cookie; tenant service `GetSignInContext` for the account; `decide` (MFA, if owed, sends the browser to `/ui/login?aal=aal2` and back to this step); `JoinTenant` if the account was only admitted; Hydra `AcceptOAuth2LoginRequest` with `context.tenant_id` = T.
5. **Consent.** Hydra sends the browser to consent; `GET /api/consent` → `GateConsent` (subject and `tenant_id` present) → the consent is accepted as before, and Hydra returns the code to the app.

A later sign-in to another app with the same session repeats only step 4, without `CompleteAttempt`: one `ListSignInTenants` or none, one `GetSignInContext`, no call to the SSO service.

## Rollout

- `BYOSSO_ENABLED` defaults to off, and off is the previous backend. The frontend bundle is the same for everyone, and shows nothing of the feature until the extension sends it (`byosso-disabled`).
- Before `BYOSSO_ENABLED`: a tenant service and an SSO service that serve the RPCs above (otherwise every app sign-in answers `503`), Login UI's OAuth2 client for the service token, and the Kratos provider `byo-sso` with the configuration listed in the proposal. `OIDC_WEBAUTHN_SEQUENCING_ENABLED` has to be off: the process does not start with both.
- The SSO service's browser pages (where its identity providers return to) are served on Login UI's host: the very same host, whatever the port or the path. The browser then sends the SSO service's receipt cookie to Login UI, as the cookies of Kratos already have to reach it. Login UI cannot check this at startup; where it is not met, no company sign-in is confirmed.
- `BYOSSO_ENABLED` goes on before Kratos is set to `session.whoami.required_aal: aal1`, and Kratos goes back to `highest_available` before the flag goes off. With Kratos at `aal1` and the flag off, nothing asks a password sign-in for its second factor, and with `MFA_ENABLED` the consent handler refuses its `aal1` session.
- From the moment the flag is on, the tenants' MFA rule applies in place of `MFA_ENABLED`'s: a portal-password sign-in to an app needs MFA at every tenant, also on a deployment that ran with `MFA_ENABLED=false`, and a company or public sign-in only where the tenant requires it.
- Rollback is the Kratos setting, then the flag: no data in Login UI to migrate. Memberships created at sign-in, links written by Kratos and personal tenants stay where they are, in the other services.
- Turning the flag off again is not neutral for tenants that already rely on it, because nothing outside `pkg/byosso` enforces their policy:
  - a tenant that requires company sign-in is no longer enforced: its members sign in to it with the portal password or a public sign-in, as to any tenant;
  - the tenants' MFA rule is gone, and `MFA_ENABLED` decides again for everyone;
  - an account whose only way in is a company sign-in cannot sign in while the flag is off: nothing obtains a ticket any more, and the SSO service refuses a submission of the `byo-sso` provider without one. While Kratos still has that provider configured, the login page shows it as Kratos lists it, and it leads to that refusal.
- Before merge, the `replace` directive for `github.com/canonical/identity-platform-api` in `go.mod` has to give way to a published version.

## Observability

- **Traces**: a span per backend call (`byosso.Directory.*`, `byosso.SSO.*`, `byosso.Verification.Start`, `hydra.OAuth2API.GetOAuth2LoginRequest`, `…RevokeOAuth2LoginSessions`, `…RejectOAuth2ConsentRequest`, `kratos.IdentityAPI.*`) and around the steps that combine several (`byosso.API.completePending`, `accountLinks`, `admittingTenants`, `hasSecondFactor`), each with the error recorded.
- **Logs**: failures at error level, with the ids of the identity, tenant, connection or flow; decisions and ordinary refusals (not a member, a company sign-in is required) at debug level ("the session of identity … at tenant … needs: company_sign_in"). The refusals that only a request made by hand gets are warnings: a submission of the `byo-sso` provider to a login, registration or settings flow, a submission that names a field twice, a disconnect from another site, and a consent for a login that Login UI did not accept. The extension's own lines carry no address, no ticket and no receipt. At debug level the full gRPC status or HTTP response of a failed backend call is logged, as elsewhere in this repository. One info line at startup says that the extension is on.
- **Not there**: the extension records no metric and writes nothing to the security log.

## Failure handling

| what fails | effect |
|---|---|
| tenant service unreachable or slow | every app sign-in answers `503` "signing in is temporarily unavailable; try again in a moment"; nothing is accepted. A registration answers `503` too, whatever the failure, and nothing reaches Kratos |
| tenant service answers something unreadable (an unspecified or unknown policy value, an empty answer) | `503` "single sign-on is temporarily unavailable"; never read as the weakest policy |
| SSO service unreachable | new company sign-ins fail (`503`), a tenant with enforcement `optional` offers the rest with a notice, settings render without company sign-ins; tenants with enforcement `off` and sessions with provenance are untouched |
| token endpoint unreachable | as an outage of both services |
| Hydra admin API fails | reading the login request or accepting: `500`; rejecting a consent: `403`, consent not accepted |
| Kratos fails to create, fetch or submit a flow for the extension | an error of Kratos that carries an id is passed on to the frontend as the handlers pass their own: `400` for a submission, the status Kratos gave it for a flow fetched or created. The pages act on an id they know (a redirect Kratos names is followed, a flow that expired is started again); for any other the login and registration pages go to the error page, and the Connected accounts page reports that the disconnect failed. Any other failure is `500` |
| a cookie of the extension is lost, does not decrypt, or holds the value of another cookie | treated as absent: no provenance (one more company sign-in), no fresh mark (asked again), no registration context |
| the receipt cookie of a company sign-in does not reach Login UI (the browser dropped it, it expired with the ticket, or the SSO service's pages are on another host) | the SSO service does not confirm the attempt: no provenance, and the session goes through the tenant's company sign-in again. When the deployment is the cause, this repeats after every return |
| the state cookie expires during a sign-in (`COOKIE_TTL`, 5 minutes by default) | the tenant chosen for the login challenge is forgotten: a company sign-in picked then goes back to the tenant list, and a first factor submitted then has its tenant resolved again from the tenants of the address |
| the user abandons a company sign-in | the browser has no Kratos session; the sign-in cookie expires after 30 minutes |

## Risks / Trade-offs

- **One pending company sign-in per browser.** There is one sign-in cookie: two company sign-ins started in parallel (two tabs) keep the later one, and the earlier one returns with no provenance and goes through the company sign-in again.
- **The receipt is not single use.** The same browser can present it again until the ticket expires, 30 minutes after the sign-in started. That gives nothing beyond the session the sign-in already produced.
- **The receipt proves the browser, not the session.** It shows that this browser completed the sign-in of the ticket, as a company sign-in the account is linked to. It does not show that the Kratos session presented with it is the one that sign-in produced: a user who has just completed tenant A's company sign-in can have another session of theirs, made in the same browser within the ticket's lifetime, counted as A's. They hold a session from A's sign-in anyway.
- **The SSO service's pages share Login UI's host.** The receipt is a `__Host-` cookie, which a browser returns only to the host that set it. A deployment that serves the two on different hosts confirms no company sign-in.
- **A session is replaced, not upgraded.** Every first factor makes a new Kratos session, so moving between tenants with different company sign-ins repeats the sign-in, and the MFA where required.
- **A registration waits while the tenant service cannot answer.** `InterceptRegistrationSubmission` cannot tell whether a tenant admits the address when the lookup or a tenant's sign-in context fails, for whatever reason, and an admitted address must not get a password account, so it answers `503` and nothing reaches Kratos. The one exception is an address the tenant service refuses as malformed: no tenant admits it, and Kratos tells the user what is wrong with it. Alternative turned down: letting the submission through, which was the behaviour at first; it made the protection depend on the tenant service being up.
- **`/api/v0/sso/complete` joins only with a verified address.** The accept path verifies the address of a company-sign-in session before anything else; the endpoint for sign-ins with no Hydra login request applies the same condition to the join. An account whose address is not verified yet joins at its first sign-in to the tenant from an app.
- **The backup code prompt has a switch of its own.** The handlers' MFA flag also guarded the prompt to regenerate backup codes, so passing the flag as off for the tenants' MFA would turn the prompt off with it. `pkg/kratos` takes an option that keeps it (`WithBackupCodesRegeneration`), which `pkg/web` sets whenever the extension is enabled; without the option nothing changes.
- **The service token travels in clear on a plaintext channel.** `TENANT_SERVICE_TLS_ENABLED` and `SSO_SERVICE_TLS_ENABLED` default to false; the deployment decides.
- **Lookup by address.** The tenant list after the email is not authenticated, as before; it now also shows tenants that invited the address or admit its domain.

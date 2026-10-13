## Context

Login UI is Hydra's login and consent provider. It is also the user interface of Kratos's self-service flows. With multi-tenancy, Login UI already asks the tenant service for a user's tenants and lets the user pick one. It puts the pick into the Hydra login accept as `context.tenant_id`. The consent handler passes this value on to the token.

Login UI has no rule for each tenant. Every tenant gets the same sign-ins. One flag for the whole platform decides about MFA.

This change adds those rules. Three other parties are involved. None of them changes in this repository:

- The **tenant service** owns memberships, personal tenants, pending invitations and each tenant's policy (enforcement, domains, auto-join, MFA). Login UI asks its `TenantSignInService` four things: `ListSignInTenants`, `GetSignInContext`, `CreatePersonalTenant` and `JoinTenant`.
- The **SSO service** owns the company sign-ins (connections to customers' identity providers) and runs them. Login UI asks it five things: `ListOptions`, `StartAttempt`, `CompleteAttempt`, `ListLinks` and `DeleteLink`.
- **Kratos** keeps one OIDC provider, `byo-sso`, for all company sign-ins. Kratos itself writes every link between an account and a company sign-in.

The table below shows the layout of the code.

| path | role |
|---|---|
| `pkg/kratos/extension.go`, `pkg/extra/extension.go` | the hook interfaces' default implementations and the `WithExtension` options. Also exported wrappers of the handlers' answers to a Kratos error (`WriteGetFlowError`, `WriteUpdateFlowError`), for the extension's own calls to Kratos |
| `pkg/byosso/handlers.go` | the `API` type with every hook, and `GET /api/v0/sso/complete` |
| `pkg/byosso/signin.go` | the sign-in screen: tenant list, a tenant's options, picks |
| `pkg/byosso/accept.go`, `decide.go` | the procedure before an accept, and the decision as a function with no I/O |
| `pkg/byosso/attempt.go` | the start and the confirmation of a company sign-in, and the fresh mark |
| `pkg/byosso/linking.go`, `registration.go`, `settings.go`, `mfa.go`, `consent.go` | one file for each flow that the extension takes part in |
| `pkg/byosso/directory.go`, `sso.go`, `hydra.go`, `kratos.go` | thin clients of the tenant service, the SSO service, Hydra and Kratos |
| `pkg/byosso/kratos_decorator.go` | removes the `byo-sso` nodes from every flow |
| `pkg/byosso/cookies.go`, `grpc.go` | the extension's cookies, and the dial options |
| `pkg/web/byosso.go`, `cmd/serve.go`, `internal/config/specs.go`, `internal/grpc/client.go` | wiring and configuration |
| `ui/pages/login.tsx`, `register.tsx`, `manage_connected_accounts.tsx`, `select_tenant.tsx`, `ui/components/FlowMessages.tsx`, `RedirectingNotice.tsx`, `ui/util/*` | the frontend |

## Goals / Non-Goals

**Goals:**

- The system applies a tenant's rules to every app sign-in, whatever path the browser takes to the accept.
- With the flag off, the backend is the one before this change.
- Kratos and Hydra work as they are. The change uses their own account linking, registration, verification, recovery, MFA and consent rejection, with configuration only.
- A slow or absent backend gives an answer that the user can act on. It never gives a weaker check.

**Non-Goals:** those of the proposal. In particular, Login UI keeps no state of its own on the server. What it must remember between requests is in encrypted cookies.

## Decisions

### 1. Hooks and an extension package, not conditions inside the handlers

**Decision**: `pkg/kratos/handlers.go` and `pkg/extra/handlers.go` call an `ExtensionInterface` at thirteen call sites, in nine handlers. `NoOpExtension` is the default. `pkg/byosso.API` implements both interfaces. `pkg/web/byosso.go` plugs it in when `BYOSSO_ENABLED` is set.

**Rationale**: the handlers serve every deployment. With hooks, a reviewer can check at the call sites that "off means unchanged". Each call site either passes a flow through, or goes on when the hook returns false. The clients of the tenant service and the SSO service, the cookies and the rules stay out of `pkg/kratos`.

The only restructuring of the handler is the extraction of `enforceSessionChecks` from `handleCreateFlow`. It has the same checks in the same order. The path of the extension can then run them too.

**Alternative turned down**: `if byossoEnabled` branches in the handlers. With such branches, a reviewer must read them in every later review of those handlers, and the dependencies land in `pkg/kratos`. A middleware in front of the routes was not enough either. The extension needs what the handler has already fetched (the flow, the session, the state cookie). It must also act in the middle of a handler, right before the accept.

### 2. The extension answers every login page request that has a session

**Decision**: when the extension reports `HandlesSessionLogin`, `handleCreateFlow` gives every request with a Kratos session and a login challenge to `HandleSessionLogin`, after its own session checks. The handler does not use its tenant resolver (`InterceptLogin`) or its `MustReAuthenticate`. For the same reason, `handleUpdateFlow` does not ask the resolver to select a tenant after a submission. The extension resolves the tenant in `BeforeAcceptLogin`.

**Rationale**: the own path of the handler decides 3 things that the extension must decide in another way.

- Which tenant. The resolver looks up tenants by identity id. This does not list the tenants that only admit an address. The extension asks for the sign-in tenants of the address. The resolver also binds a single membership on its own. When the state cookie has expired, this replaces the tenant of an invitation or auto-join that the user picked.
- If the session can be reused. `MustReAuthenticate` follows Hydra's `skip`. This says nothing about how the session relates to the tenant. It cannot express `max_age` against the first factor of the Kratos session (decision 9).
- When to accept. The handler accepts as soon as it resolves a tenant.

Every company sign-in, every public sign-in and every reused session arrives at this one request. So the tenant's checks must be here.

**Alternative turned down**: keep the path of the handler and rely on `BeforeAcceptLogin` alone. The handler then chooses the tenant and decides about re-authentication before the request reaches the hook. Two places decide the same request, with different inputs.

### 3. Which company sign-in a session came from is kept in an encrypted cookie

**Decision**: when Login UI starts a company sign-in, it writes `login_ui_sso_signin`. The cookie holds the ticket and the id of the session that the browser has, if any.

A session with another id that includes a `byo-sso` method can come back. Then Login UI asks the SSO service to confirm the attempt (`CompleteAttempt`). It sends the ticket with the receipt. The SSO service left the receipt in the browser that completed the sign-in, in its cookie `__Host-sso_receipt_<digest of the ticket>`. Login UI then writes `login_ui_sso_provenance` (`session id`, `connection id`). At later accepts, the provenance counts only while the cookie names the current session.

**Rationale**: all company sign-ins share one Kratos provider. So the session says "byo-sso" and nothing about the connection. A tenant must accept only sessions from its own connections. Login UI has no store. A cookie bound to the session id needs none. The SSO service, not the browser, vouches for what goes into the cookie. If the cookie is lost, the check fails closed: a company-sign-in session with no provenance goes through the company sign-in again.

Two conditions make "a sign-in has just happened" checkable without a comparison of clocks:

- Every first factor starts without the Kratos session cookie.
- Login UI expires that cookie before it sends the browser to the identity provider.

So Kratos always issues a new session, and its id differs from the recorded one.

The ticket alone proves too little. It is valid from the moment the sign-in starts. One account can be linked to the company sign-ins of 2 tenants. A user can start a sign-in at tenant A and stop at its identity provider. The user can then sign in through B and present A's ticket with B's session. The session then counts as A's.

The SSO service must know that the sign-in at A was completed, and in this browser. So the SSO service gives a receipt to the browser whose sign-in it accepts. Only the SSO service can make or check the receipt. Login UI passes the receipt on with the ticket and reads nothing in it.

**Alternative turned down**: sessions bound to a tenant (an identity or a session for each tenant). A user has one account and one session. A session from one company sign-in must also serve a second app at the same tenant. A password session must serve several tenants. A server-side record keyed by session needs a store, and Login UI has none.

We also turned down a record of completed sign-ins that the SSO service keeps in place of the receipt. The record says that the identity provider answered for a ticket. It does not say in which browser. Whoever started the attempt can have another user's browser finish it. That other user opens the link and the identity provider signs them in silently. The person who started the attempt can then present the ticket with a session made in another way. The proof must end up in the browser that signed in.

### 4. The tenant's MFA is decided in Login UI, with Kratos at `aal1`

**Decision**: with the extension enabled, Kratos runs with `session.whoami.required_aal: aal1`. `pkg/web/byosso.go` passes the platform-wide MFA flag of the handlers as off. `decide` asks for MFA for each tenant. It always asks for MFA for a password sign-in. For a company or public sign-in, it asks only where the tenant's policy is `required`. MFA counts when the session lists a second factor at `aal2`. This decision has no flag of its own. `BYOSSO_ENABLED` brings it, and `MFA_ENABLED` has no effect while the flag is on.

**Rationale**: Kratos's required assurance level is one value for the deployment. If it stays at `highest_available`, Kratos refuses the `aal1` session of any account that has a second factor. A company sign-in at a tenant that asks for no MFA then still sends the user to enter a code. Login UI is the only component that knows the tenant.

The step-up itself stays with Kratos. It uses `aal=aal2` and the settings flow for the first set-up. `selfservice.flows.settings.required_aal` stays `highest_available` for the account pages.

**Alternative turned down**: keep `highest_available` and `MFA_ENABLED`. It cannot express "only where the tenant requires it". The decision has two costs. A Kratos setting and Login UI's flag must change together (Rollout). An `aal1` session is now something that consent can meet (decision 6).

### 5. Login UI creates the membership, just before the accept

**Decision**: a pending invitation or auto-join can admit an account. Login UI checks that account as a member. When its session passes every check, `checkAndAccept` calls `JoinTenant` and then accepts. The tenant service checks the admission again. It can answer `NOT_ADMITTED`.

**Rationale**: the membership is the result of a sign-in that satisfied the tenant. Only Login UI knows that the sign-in did this. The SSO service sees the answer of the identity provider. It does not see the MFA or the address verification that follow. A user accepts an invitation to a tenant without company sign-in with a password, and the SSO service never sees that. `JoinTenant` has the same effect when repeated, so a retry is safe.

**Alternative turned down**: create the membership earlier, at invitation time or when the identity provider returns. Then a user is a member of a tenant that requires company sign-in, or MFA, before the user passed either.

The price of the last step: the tenant service can withdraw the admission between the registration of a new account by Kratos and the accept. Then the system refuses the user, and the account stays, with no tenant.

### 6. Consent refuses a login that carries no tenant

**Decision**: `GateConsent` rejects (`login_required`) a consent in two cases. The subject of the consent is not the account of the session. Or the login context has no `tenant_id`.

**Rationale**: Kratos accepts a Hydra login request itself when it gets the login challenge. When multi-tenancy is on, Login UI does not give it the challenge (`pkg/kratos/service.go`, `CreateBrowserLoginFlow`). But a browser can give it. With decision 4, such a login can be at `aal1`, and it passes no tenant check. Every login that `checkAndAccept` accepts names a tenant, at least the personal tenant. So the absence of `tenant_id` identifies a login accepted elsewhere. This needs no call to a backend.

**Alternative turned down**: mark accepted logins with something new (an `acr`, a signed marker). The accept stays exactly as it was. This keeps "off means unchanged" true for Hydra and for the tokens of the apps. We also turned down a second run of the checks of the tenant at consent. Then consent depends on the tenant service, for a question that it does not need to ask.

### 7. A wiring fault stops the process

**Decision**: `web.NewRouter` returns an error when the extension is enabled and cannot be built, and `serve` exits. `cmd/serve.go` refuses incomplete settings before that.

**Alternative turned down**: what multi-tenancy does when its client is missing. It writes a warning and falls back to an implementation that does nothing. For an optional convenience this is reasonable. Here the fallback is a Login UI that serves a tenant that requires company sign-in with the portal password, and it looks healthy.

### 8. Retries only where a repeat is harmless, and one budget per request

**Decision**: the gRPC service config (`pkg/byosso/grpc.go`) retries `UNAVAILABLE` up to 3 attempts. This applies to the reads and to `JoinTenant`, `CreatePersonalTenant`, `StartAttempt` and `CompleteAttempt`. It does not retry `DeleteLink`. Each call has its own timeout. Each browser request has 12 seconds for everything. The channels reconnect within 5 seconds.

**Rationale**: a call that answers `UNAVAILABLE` can have run. The listed writes leave the same state when they run a second time. A second `DeleteLink` answers "no such link" for a removal that worked, and the user sees a refusal. The system does not retry a call that timed out. The budget is better spent on an answer.

The reconnect backoff is shorter than the default. gRPC's default backoff grows to 2 minutes. In that time Login UI refuses every sign-in after a restart of a backend. The 12 seconds are under the server's 15-second write timeout. So the browser gets the answer of the extension and not a closed connection.

**Alternative turned down**: retry loops in the callers. The channel policy states once, next to the list of methods, what is safe to repeat.

### 9. Re-authentication is judged on the Kratos session

**Decision**: Login UI reads `prompt=login` and `max_age` from the login request's `request_url`. It compares them with the completion time of the session's first factor (`pkg/byosso/hydra.go`).

A cookie, `login_ui_sso_fresh`, marks a first factor that started for a login challenge. This prevents a second request to re-authenticate for its result. The mark records two things:

- the session that the browser had
- the time when Kratos issued the login flow that the first factor began on

Only another session counts, and only if its first factor completed at or after that time. Kratos timed both, so the system needs no allowance for a difference between clocks.

When Hydra remembers another subject for the browser, Login UI revokes that Hydra login session. It then restarts the app's request.

**Rationale**: Hydra stamps the authentication time when it accepts a login. It does not stamp the time when the user authenticated. So its own view of `max_age` calls an old Kratos session fresh.

Also, the accepted subject can differ from the remembered subject. Then Hydra restarts the request with `prompt=login` added. Login UI cannot tell this from the app's own `prompt=login`. It then asks for a first factor that the user has just given. For a company sign-in, an identity provider with a live session can be unable to give it again.

The session id alone does not prove a fresh first factor. A mark made while the browser sent no session cookie names no session. Any session shown later differs from it. For this reason the mark holds the time.

**Alternative turned down**: follow Hydra's `skip`, as the handler does. See decision 2.

### 10. Kratos writes the links; Login UI only removes them, through the SSO service

**Decision**: a first company sign-in for an existing account goes through Kratos's account-linking flow. A first company sign-in for a new address goes through Kratos's registration. Login UI adds the account's other company sign-ins to the account-linking page (`linking.go`) and to Kratos's refresh login (`settings.go`). Login UI removes a link with `DeleteLink`.

**Rationale**: account linking already makes the owner prove the account before Kratos writes a link. This protects against an identity provider that asserts someone else's address. Kratos's settings unlink cannot be used. It removes every link of the provider, that is, all company sign-ins at once.

## Data and control flow

This is a company sign-in from the email to the code of the app. The user has one tenant that requires its one company sign-in. The account is already linked to it.

1. **Login page.** The app sends the browser to Hydra, and Hydra sends it to `/ui/login?login_challenge=C`. The frontend calls `GET /api/kratos/self-service/login/browser`. Login UI creates a Kratos login flow that returns to that page. Login UI does not give the challenge to Kratos.
2. **Email.** `POST …/login/id-first` goes to Kratos, which takes the identifier step. Then `BeforeTenantSelection` runs:
   1. The tenant service gets `ListSignInTenants` (one tenant T) and `GetSignInContext` (`required`, connection K).
   2. The SSO service gets `ListOptions` (one option), then `StartAttempt` (ticket).
   3. Login UI submits the flow to Kratos (`oidc`, `byo-sso`, `login_hint` = ticket).
   4. Login UI writes the sign-in cookie, binds T to C in the state cookie, and writes the fresh cookie.
   5. Login UI expires the Kratos session cookie.
   6. Login UI answers with Kratos's redirect and the label.

   The page shows "Redirecting to …".
3. **Outside Login UI.** The browser goes to the OpenID Provider behind `byo-sso`. The SSO service fronts this provider. The browser goes on to the customer's identity provider and back to Kratos's callback. On the way back, the SSO service accepts the sign-in and sets its receipt cookie for the ticket. Kratos finds the link, issues a new session and returns to the login page of C. (With no link, account linking or registration runs. Then the return is the same.)
4. **Return.** The frontend calls `GET …/login/browser?login_challenge=C` again, now with a session. `HandleSessionLogin` calls `checkAndAccept`:
   1. It verifies the address.
   2. The SSO service gets `CompleteAttempt` with the ticket and its receipt. Login UI writes the provenance cookie for (session, K). It clears the sign-in cookie and the receipt cookie.
   3. It calls Hydra `GetOAuth2LoginRequest`.
   4. It takes tenant T from the state cookie.
   5. The tenant service gets `GetSignInContext` for the account.
   6. `decide` runs. If MFA is owed, it sends the browser to `/ui/login?aal=aal2` and back to this step.
   7. `JoinTenant` runs if the account was only admitted.
   8. Hydra gets `AcceptOAuth2LoginRequest` with `context.tenant_id` = T.
5. **Consent.** Hydra sends the browser to consent. `GET /api/consent` runs `GateConsent` (subject and `tenant_id` present). The consent is accepted as before, and Hydra returns the code to the app.

A later sign-in to another app with the same session repeats only step 4, without `CompleteAttempt`. It makes one `ListSignInTenants` or none, and one `GetSignInContext`. It makes no call to the SSO service.

## Rollout

- `BYOSSO_ENABLED` is off by default, and off is the previous backend. The frontend bundle is the same for everyone. It shows nothing of the feature until the extension sends it (`byosso-disabled`).
- Before `BYOSSO_ENABLED` goes on, these conditions must be true:
  - A tenant service and an SSO service serve the RPCs above. If not, every app sign-in answers `503`.
  - Login UI can reach both on the internal network.
  - The Kratos provider `byo-sso` has the configuration in the proposal.
  - `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is off. The process does not start with both.
- The SSO service's browser pages are on Login UI's host. The identity providers return to these pages. The host must be the very same host, whatever the port or the path. The browser then sends the SSO service's receipt cookie to Login UI, as Kratos's cookies already must reach it. Login UI cannot check this at startup. Where the condition is not met, no company sign-in is confirmed.
- `BYOSSO_ENABLED` goes on before Kratos is set to `session.whoami.required_aal: aal1`. Kratos goes back to `highest_available` before the flag goes off. If Kratos is at `aal1` and the flag is off, nothing asks a password sign-in for its second factor. If `MFA_ENABLED` is also on, the consent handler refuses its `aal1` session.
- From the moment the flag is on, the tenants' MFA rule applies in place of `MFA_ENABLED`'s. A portal-password sign-in to an app needs MFA at every tenant. This is also true on a deployment that ran with `MFA_ENABLED=false`. A company or public sign-in needs MFA only where the tenant requires it.
- Rollback is the Kratos setting, then the flag. Login UI has no data to migrate. Memberships created at sign-in, links written by Kratos and personal tenants stay where they are, in the other services.
- To turn the flag off again is not neutral for tenants that already rely on it, because nothing outside `pkg/byosso` enforces their policy:
  - A tenant that requires company sign-in is no longer enforced. Its members sign in to it with the portal password or a public sign-in, as to any tenant.
  - The tenants' MFA rule is gone, and `MFA_ENABLED` decides again for everyone.
  - An account whose only way in is a company sign-in cannot sign in while the flag is off. Nothing gets a ticket any more, and the SSO service refuses a submission of the `byo-sso` provider without one. While Kratos still has that provider configured, the login page shows it as Kratos lists it. It leads to that refusal.
- Before merge, `go.mod` must require a released version of `github.com/canonical/identity-platform-api`. Until then it requires the commit of the pull request that adds the API.

## Observability

- **Traces**: there is a span for each backend call. The names are `byosso.Directory.*`, `byosso.SSO.*`, `byosso.Verification.Start`, `hydra.OAuth2API.GetOAuth2LoginRequest`, `…RevokeOAuth2LoginSessions`, `…RejectOAuth2ConsentRequest` and `kratos.IdentityAPI.*`. There is also a span around the steps that combine several calls (`byosso.API.completePending`, `accountLinks`, `admittingTenants`, `hasSecondFactor`). Each span records the error.
- **Logs**: failures are at error level, with the ids of the identity, tenant, connection or flow. Decisions and ordinary refusals are at debug level. Ordinary refusals are "not a member" and "a company sign-in is required". An example of a decision line is "the session of identity … at tenant … needs: company_sign_in".
- Some refusals are warnings, because only a request made by hand gets them:
  - a submission of the `byo-sso` provider to a login, registration or settings flow
  - a submission that names a field twice
  - a disconnect from another site
  - a consent for a login that Login UI did not accept
- The extension's own lines carry no address, no ticket and no receipt. At debug level, the log has the full gRPC status or HTTP response of a failed backend call, as elsewhere in this repository. One info line at startup says that the extension is on.
- **Not there**: the extension records no metric and writes nothing to the security log.

## Failure handling

| what fails | effect |
|---|---|
| tenant service unreachable or slow | Every app sign-in answers `503` "signing in is temporarily unavailable; try again in a moment". Nothing is accepted. A registration also answers `503`, whatever the failure, and nothing reaches Kratos |
| tenant service answers something unreadable (an unspecified or unknown policy value, an empty answer) | The answer is `503` "single sign-on is temporarily unavailable". The system never reads this as the weakest policy |
| SSO service unreachable | New company sign-ins fail (`503`). A tenant with enforcement `optional` offers the rest with a notice. Settings render without company sign-ins. Tenants with enforcement `off` and sessions with provenance are not affected |
| Hydra admin API fails | When it reads the login request or accepts, the answer is `500`. When it rejects a consent, the answer is `403` and the consent is not accepted |
| Kratos fails to create, fetch or submit a flow for the extension | If the Kratos error carries an id, the system passes it on to the frontend as the handlers pass their own. The status is `400` for a submission. For a flow that the system fetched or created, the status is the status that Kratos gave. The pages act on an id they know: they follow a redirect that Kratos names, and they start a flow again if it expired. For any other id, the login and registration pages go to the error page. The Connected accounts page reports that the disconnect failed. Any other failure is `500` |
| a cookie of the extension is lost, does not decrypt, or holds the value of another cookie | The system treats it as absent. There is no provenance (one more company sign-in), no fresh mark (the user is asked again) and no registration context |
| the receipt cookie of a company sign-in does not reach Login UI (the browser dropped it, it expired with the ticket, or the SSO service's pages are on another host) | The SSO service does not confirm the attempt. There is no provenance, and the session goes through the tenant's company sign-in again. If the deployment is the cause, this repeats after every return |
| the state cookie expires during a sign-in (`COOKIE_TTL`, 5 minutes by default) | The system forgets the tenant chosen for the login challenge. A company sign-in picked then goes back to the tenant list. For a first factor submitted then, the system resolves the tenant again from the tenants of the address |
| the user abandons a company sign-in | The browser has no Kratos session. The sign-in cookie expires after 30 minutes |

## Risks / Trade-offs

- **One pending company sign-in for each browser.** There is one sign-in cookie. If a user starts 2 company sign-ins in parallel (2 tabs), the cookie keeps the later one. The earlier one returns with no provenance and goes through the company sign-in again.
- **The receipt is not single use.** The same browser can present it again until the ticket expires, 30 minutes after the sign-in started. That gives nothing beyond the session that the sign-in already produced.
- **The receipt proves the browser, not the session.** It shows that this browser completed the sign-in of the ticket. The sign-in was a company sign-in that the account is linked to. It does not show that the Kratos session presented with it is the session that this sign-in produced. A user can complete tenant A's company sign-in. Another session of the same user, made in the same browser within the ticket's lifetime, can then count as A's. The user holds a session from the sign-in at A anyway.
- **The SSO service's pages share Login UI's host.** The receipt is a `__Host-` cookie. A browser returns it only to the host that set it. A deployment that serves the two on different hosts confirms no company sign-in.
- **A session is replaced, not upgraded.** Every first factor makes a new Kratos session. If a user moves between tenants with different company sign-ins, the sign-in repeats, and the MFA where it is required.
- **A registration waits while the tenant service cannot answer.** `InterceptRegistrationSubmission` cannot tell if a tenant admits the address when the lookup fails. It also cannot tell this when a tenant's sign-in context fails. The reason does not matter. An admitted address must not get a password account. So it answers `503` and nothing reaches Kratos. There is one exception: an address that the tenant service refuses as malformed. No tenant admits it, and Kratos tells the user what is wrong with it. We turned down the alternative to let the submission through. This was the behavior at first. It made the protection depend on the tenant service being up.
- **`/api/v0/sso/complete` joins only with a verified address.** The accept path verifies the address of a company-sign-in session before anything else. The endpoint for sign-ins with no Hydra login request applies the same condition to the join. An account whose address is not verified yet joins at its first sign-in to the tenant from an app.
- **The backup code prompt has a switch of its own.** The handlers' MFA flag also guarded the prompt to regenerate backup codes. If the system passes the flag as off for the tenants' MFA, the prompt goes off with it. `pkg/kratos` takes an option that keeps the prompt (`WithBackupCodesRegeneration`). `pkg/web` sets it whenever the extension is enabled. Without the option, nothing changes.
- **Login UI calls both services without a token.** The system serves the RPCs that Login UI calls to the internal network. It already serves the tenant lookup and Kratos's admin API in the same way. That network confines the RPCs. We turned down a client-credentials token for Login UI. It needs a client, a token endpoint and the certificate authority of that endpoint to manage. The calls never leave the network. `TENANT_SERVICE_TLS_ENABLED` and `SSO_SERVICE_TLS_ENABLED` are false by default. The deployment decides.
- **Lookup by address.** The tenant list after the email is not authenticated, as before. It now also shows tenants that invited the address or admit its domain.

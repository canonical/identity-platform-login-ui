# tenant-sign-in-checks Specification

## Purpose

A Kratos session is not bound to a tenant. The tenant is chosen for each app sign-in (each Hydra login request) and reaches the app's token through the `tenant_id` of the login accept, so one session can be offered to several tenants, each with its own rules. Login UI checks the session against the chosen tenant before every accept, and asks only for what is missing.

## Requirements
### Requirement: Every Hydra login accept goes through the tenant's checks
With the extension enabled, Login UI SHALL accept a Hydra login request only at the end of one procedure (`checkAndAccept`, `pkg/byosso/accept.go`), reached from the login page with a session (`HandleSessionLogin`) and from every accept the handlers are about to make after a submission (`BeforeAcceptLogin`). A session with no identity is answered `401`. The procedure stops at the first step that answers:

1. a session that includes a company sign-in and whose address is unverified is sent to verify it;
2. a company sign-in that has just come back is confirmed with the SSO service (`company-sign-in`);
3. the Hydra login request is read, and another user remembered by Hydra is handled (`re-authentication`);
4. the tenant is resolved;
5. the tenant service is asked about the tenant and the account (`GetSignInContext` with `tenant_id` and `identity_id`);
6. the decision below is taken;
7. an account that is admitted but not a member joins the tenant (`tenant-join-at-sign-in`);
8. the login is accepted.

#### Scenario: A tenant id stored for a session
- **WHEN** `POST /api/v0/auth/tenant` stores a tenant the account does not belong to and the browser returns to the login page
- **THEN** step 6 refuses the sign-in: the stored tenant is never trusted on its own

### Requirement: The tenant of a sign-in with a session
The tenant SHALL be the one bound to the login challenge in the state cookie. With none bound, the extension SHALL look up the tenants of the session's address (`ListSignInTenants` with `email`); the email is not asked again:

- one: it is bound and used;
- several: the answer is `{"error": {"id": "tenant_selection_required"}, "redirect_to": "/ui/select_tenant?login_challenge=…"}`. `GET /api/v0/tenants` SHALL list a session's tenants by its address, with the same `ListSignInTenants` (an option of `pkg/tenants` that `pkg/web` sets with the extension), so that page shows invitations and auto-join candidates too;
- none: the account gets its personal tenant (`CreatePersonalTenant` with `identity_id`, which returns the existing one or creates it). When the tenant service answers `HAS_TENANT` instead (the account belongs to a tenant, or invitations wait for it), the tenants are looked up again.

A session that includes a company sign-in MUST NOT be accepted for an account with no tenant: no personal tenant is created, and a fresh first factor is asked for.

The handler MUST NOT select the tenant in the extension's place after a submission (`login-extension-hooks`): its resolver knows memberships only, and would bind a single membership where the user had picked a tenant that invited the address or that it could auto-join.

#### Scenario: The state cookie expired before the first factor was submitted
- **WHEN** a first factor is submitted for a login challenge whose tenant the state cookie no longer holds, and the address has one membership and one pending invitation
- **THEN** the answer sends the browser to `/ui/select_tenant`, which lists both; the membership is not bound on its own

#### Scenario: Account with no tenant signs in with its password
- **WHEN** the lookup returns no tenant
- **THEN** the login is accepted with the personal tenant, created if it did not exist, once MFA is done

### Requirement: The decision
The extension SHALL decide from the tenant service's answer, the session, and the provenance of the session (the connection of the company sign-in it came from, known only while the provenance cookie names this session), in this order:

| # | condition | outcome |
|---|---|---|
| 1 | the account is neither a member nor admitted by a pending invitation or auto-join | refused: `403`, error id `tenant_not_a_member`, "this account is not a member of that tenant" |
| 2 | the app asks for re-authentication and the first factor was not made for this login request | a fresh first factor |
| 3 | the session includes a company sign-in, its provenance is unknown, and the tenant has an active binding that applies to the address | a company sign-in of the tenant |
| 4 | the session includes a company sign-in, and its connection, known or not, is not such a binding | at `required`: a company sign-in of the tenant; otherwise a fresh first factor |
| 5 | the session includes no company sign-in and the enforcement is `required` | a company sign-in of the tenant |
| 6 | the session owes MFA (`tenant-mfa`) | MFA |
| 7 | none of the above | accepted |

A session includes a company sign-in when any of its authentication methods is `oidc` with provider `byo-sso`: its first factor, or one Kratos added by account linking.

#### Scenario: Password session at a tenant that requires company sign-in
- **WHEN** a session from a password is checked against a tenant with enforcement `required`
- **THEN** the login is not accepted and the tenant's company sign-in is started

#### Scenario: Company-sign-in session at a tenant with enforcement off
- **WHEN** the tenant's enforcement is `off`, or it is a personal tenant
- **THEN** the login is not accepted, and a fresh first factor is asked for

#### Scenario: Company-sign-in session at the tenant it was made for
- **WHEN** the provenance cookie names this session and a connection that is an active, applying binding of the tenant
- **THEN** the first factor is taken as it is, with no call to the SSO service

#### Scenario: Company-sign-in session with no provenance
- **WHEN** the provenance cookie is missing, does not decrypt, or names another session, and the tenant has company sign-ins
- **THEN** the user goes through the tenant's company sign-in again

### Requirement: Asking for what is missing
A **company sign-in** SHALL be started from what the SSO service lists for the tenant (`ListOptions`): with one, straight to it; with several, on a login flow that shows them. When the list is empty or cannot be read, a tenant that does not require company sign-in gets a fresh first factor instead; one that does answers `403` `sso_not_applicable` (empty) or `503` `sso_unavailable` (failure).

A **fresh first factor** SHALL run on a new Kratos login flow that returns to the login page of the login challenge, with the tenant bound and the identifier step already submitted for the address of the session: the email is not asked again, and the page shows what the tenant offers. At a tenant that requires company sign-in, a company sign-in is started instead.

Every flow created here MUST be created without the Kratos session cookie, so that Kratos issues a new session.

#### Scenario: Session of another tenant's company sign-in at a tenant with enforcement optional
- **WHEN** the session's connection is not a binding of the tenant
- **THEN** the browser lands on a login flow for the same address, showing that tenant's company sign-ins and the account's own first factors

### Requirement: What the accept carries
The accept SHALL be the `AcceptLoginRequest` of `pkg/kratos`, unchanged by this change: subject the identity id, `remember` for the lifetime of the Kratos session, `amr` the methods of the session, `identity_provider_session_id` the session id, and `context` `{"tenant_id": <tenant>}`. Nothing is added; no `acr` is set. With the extension enabled the context MUST always name a tenant: the chosen one, or the personal tenant. After the accept the state cookie and the fresh cookie are cleared, and the answer is `{"redirect_to": <where Hydra sends the browser>}`.

#### Scenario: Accepted login
- **WHEN** a session passes the checks of tenant T
- **THEN** the accept Hydra receives has `context.tenant_id` T

### Requirement: The address of a company-sign-in session is verified first
Tenants admit users by their address, and Kratos registers an account from a company sign-in with the address unverified unless the identity provider confirmed it. The extension MUST NOT go on with a session that includes a company sign-in while the email of the account is one of its verifiable addresses and is not verified. It SHALL start a Kratos verification flow that returns to the login page of the login challenge, submit the address to it (so the code goes to that address only), and answer `{"error": {"id": "verification_required"}, "redirect_to": "/ui/verification?flow=<id>"}`. This does not depend on `VERIFICATION_ENABLED`.

#### Scenario: Identity provider that says nothing about the address
- **WHEN** the session of an account just registered by a company sign-in returns with the address unverified
- **THEN** neither the tenant service nor the SSO service is called, no login is accepted, and the browser is sent to enter the code


## Purpose

A Kratos session is not bound to a tenant. The tenant is chosen for each app sign-in (each Hydra login request). The tenant reaches the token of the app through the `tenant_id` of the login accept. So one session can go to several tenants, and each tenant has its own rules.

Login UI checks the session against the chosen tenant before every accept. Login UI asks only for what is missing.

## ADDED Requirements

### Requirement: Every Hydra login accept goes through the tenant's checks
With the extension enabled, Login UI SHALL accept a Hydra login request only at the end of one procedure (`checkAndAccept`, `pkg/byosso/accept.go`). The procedure starts from the login page with a session (`HandleSessionLogin`). It also starts from every accept that the handlers are about to make after a submission (`BeforeAcceptLogin`). Login UI answers a session with no identity with `401`.

The procedure stops at the first step that gives an answer. These are the steps in order:

1. If the session includes a company sign-in and its address is not verified, Login UI sends the user to verify the address.
2. If a company sign-in has just come back, Login UI confirms it with the SSO service (`company-sign-in`).
3. Login UI reads the Hydra login request, and it handles another user that Hydra remembers (`re-authentication`).
4. Login UI resolves the tenant.
5. Login UI asks the Tenant Service about the tenant and the account (`GetSignInContext` with `tenant_id` and `identity_id`).
6. Login UI takes the decision below.
7. If the account is admitted but is not a member, the account joins the tenant (`tenant-join-at-sign-in`).
8. Login UI accepts the login.

#### Scenario: A tenant id stored for a session
- **WHEN** `POST /api/v0/auth/tenant` stores a tenant that the account does not belong to
- **AND** the browser returns to the login page
- **THEN** step 6 refuses the sign-in
- **AND** Login UI never trusts the stored tenant on its own

### Requirement: The tenant of a sign-in with a session
The tenant SHALL be the tenant that the state cookie binds to the login challenge. If the cookie binds none, the extension SHALL look up the tenants of the session's address (`ListSignInTenants` with `email`). The extension does not ask for the email again.

The result of the look-up decides what happens:

- One tenant: the extension binds it and uses it.
- Several tenants: the answer is `{"error": {"id": "tenant_selection_required"}, "redirect_to": "/ui/select_tenant?login_challenge=…"}`.
- No tenant: the account gets its personal tenant. `CreatePersonalTenant` with `identity_id` returns the existing personal tenant or creates it.

If the Tenant Service answers `HAS_TENANT` to `CreatePersonalTenant`, the account belongs to a tenant, or invitations wait for it. Then the extension looks up the tenants again.

`GET /api/v0/tenants` SHALL list a session's tenants by its address, with the same `ListSignInTenants`. This is an option of `pkg/tenants` that `pkg/web` sets with the extension. So the page that lists the tenants shows invitations and auto-join candidates also.

Login UI MUST NOT accept a session that includes a company sign-in for an account with no tenant. Login UI creates no personal tenant for it, and it asks for a fresh first factor.

After a submission, the handler MUST NOT select the tenant in the place of the extension (`login-extension-hooks`). The handler's resolver knows memberships only. If the handler selects the tenant, its resolver binds a single membership where the user picked a tenant that invited the address. It does the same where the user picked a tenant that the address can auto-join.

#### Scenario: The state cookie expired before the first factor was submitted
- **WHEN** a user submits a first factor for a login challenge whose tenant the state cookie no longer holds
- **AND** the address has one membership and one pending invitation
- **THEN** the answer sends the browser to `/ui/select_tenant`, which lists both
- **AND** the extension does not bind the membership on its own

#### Scenario: Account with no tenant signs in with its password
- **WHEN** the look-up returns no tenant
- **THEN** the login is accepted with the personal tenant, after MFA is done
- **AND** Login UI creates the personal tenant if it did not exist

### Requirement: The decision
The extension SHALL decide from the Tenant Service's answer, the session, and the provenance of the session, in the order of the table below. The provenance is the connection of the company sign-in that the session came from. Login UI knows it only while the provenance cookie names this session.

| # | condition | outcome |
|---|---|---|
| 1 | the account is neither a member nor admitted by a pending invitation or auto-join | refused: `403`, error id `tenant_not_a_member`, "this account is not a member of that tenant" |
| 2 | the app asks for re-authentication and the first factor was not made for this login request | a fresh first factor |
| 3 | the session includes a company sign-in, its provenance is unknown, and the tenant has an active binding that applies to the address | a company sign-in of the tenant |
| 4 | the session includes a company sign-in, and its connection, known or not, is not such a binding | at `required`, a company sign-in of the tenant. Otherwise, a fresh first factor |
| 5 | the session includes no company sign-in and the enforcement is `required` | a company sign-in of the tenant |
| 6 | the session owes MFA (`tenant-mfa`) | MFA |
| 7 | none of the above | accepted |

A session includes a company sign-in when any of its authentication methods is `oidc` with provider `byo-sso`. This can be its first factor, or a method that Kratos added by account linking.

#### Scenario: Password session at a tenant that requires company sign-in
- **WHEN** Login UI checks a session from a password against a tenant with enforcement `required`
- **THEN** Login UI does not accept the login
- **AND** Login UI starts the company sign-in of the tenant

#### Scenario: Company-sign-in session at a tenant with enforcement off
- **WHEN** the enforcement of the tenant is `off`, or the tenant is a personal tenant
- **THEN** Login UI does not accept the login
- **AND** Login UI asks for a fresh first factor

#### Scenario: Company-sign-in session at the tenant it was made for
- **WHEN** the provenance cookie names this session
- **AND** it names a connection that is an active, applying binding of the tenant
- **THEN** Login UI takes the first factor as it is
- **AND** Login UI makes no call to the SSO service

#### Scenario: Company-sign-in session with no provenance
- **WHEN** the provenance cookie is missing, does not decrypt, or names another session
- **AND** the tenant has company sign-ins
- **THEN** the user goes through the company sign-in of the tenant again

### Requirement: Asking for what is missing
Login UI SHALL start a **company sign-in** from the list that the SSO service gives for the tenant (`ListOptions`). With one company sign-in, Login UI SHALL go straight to it. With several, Login UI SHALL start the company sign-in on a login flow that shows them.

If the list is empty or Login UI cannot read it, the result depends on the tenant:

- A tenant that does not require company sign-in gets a fresh first factor instead.
- A tenant that requires company sign-in gets `403` `sso_not_applicable` for an empty list.
- A tenant that requires company sign-in gets `503` `sso_unavailable` if Login UI cannot read the list.

A **fresh first factor** SHALL run on a new Kratos login flow that returns to the login page of the login challenge. The flow SHALL have the tenant bound, and the identifier step already submitted for the address of the session. So the email is not asked again, and the page shows what the tenant offers. At a tenant that requires company sign-in, Login UI starts a company sign-in instead.

Login UI MUST create every flow of this requirement without the Kratos session cookie. This makes Kratos issue a new session.

#### Scenario: Session of another tenant's company sign-in at a tenant with enforcement optional
- **WHEN** the connection of the session is not a binding of the tenant
- **THEN** the browser lands on a login flow for the same address
- **AND** the flow shows the company sign-ins of that tenant and the first factors of the account

### Requirement: What the accept carries
The accept SHALL be the `AcceptLoginRequest` of `pkg/kratos`, and this change does not change it. It has these fields:

- subject: the identity id
- `remember`: for the lifetime of the Kratos session
- `amr`: the methods of the session
- `identity_provider_session_id`: the session id
- `context`: `{"tenant_id": <tenant>}`

Nothing is added to the accept, and no `acr` is set. With the extension enabled, the context MUST always name a tenant: the chosen tenant or the personal tenant. After the accept, Login UI clears the state cookie and the fresh cookie. The answer is `{"redirect_to": <where Hydra sends the browser>}`.

#### Scenario: Accepted login
- **WHEN** a session passes the checks of tenant T
- **THEN** the accept that Hydra receives has `context.tenant_id` T

### Requirement: The address of a company-sign-in session is verified first
Tenants admit users by their address. Kratos registers an account from a company sign-in with the address not verified, unless the identity provider confirmed it. The extension MUST NOT go on with a session that includes a company sign-in while both of these conditions are true:

- the email of the account is one of its verifiable addresses
- that email is not verified

In that case the extension SHALL do these steps in order:

1. Start a Kratos verification flow that returns to the login page of the login challenge.
2. Submit the address to the flow, so that the code goes to that address only.
3. Answer `{"error": {"id": "verification_required"}, "redirect_to": "/ui/verification?flow=<id>"}`.

This does not depend on `VERIFICATION_ENABLED`.

#### Scenario: Identity provider that says nothing about the address
- **WHEN** the session of an account that a company sign-in just registered returns with the address not verified
- **THEN** Login UI calls neither the Tenant Service nor the SSO service
- **AND** Login UI accepts no login
- **AND** Login UI sends the browser to enter the code

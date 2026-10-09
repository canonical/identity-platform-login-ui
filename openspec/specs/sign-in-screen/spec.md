# sign-in-screen Specification

## Purpose

Which sign-ins a user may use depends on the tenant, so the user chooses the tenant before the first factor and sees only what that tenant accepts. What the browser picks from is a hint: every pick is checked again on the server.

## Requirements
### Requirement: Tenants offered after the email
After Kratos takes the identifier step of a login flow that has a login challenge, the extension SHALL ask the tenant service for the tenants the address may use (`ListSignInTenants` with `email`: memberships, pending invitations, tenants whose auto-join admits it) and answer `POST /api/kratos/self-service/login/id-first` itself:

- **no tenant**: the "no tenant" value is bound to the login challenge in the state cookie; the login page shows the first factors Kratos lists for the account;
- **one**: it is bound to the login challenge; the browser goes to the login page or, when the tenant requires company sign-in and exactly one applies to the address, straight to that company sign-in;
- **several**: nothing is bound, the state cookie records that there is a choice (`tc`), and the login page lists the tenants and nothing else.

#### Scenario: One tenant that requires its only company sign-in
- **WHEN** the address has one tenant, its enforcement is `required` and one company sign-in applies
- **THEN** the response is `{"redirect_to": <where Kratos sends the browser>, "redirect_label": <label>}`, and the page shows "Redirecting to <label>…" before it leaves

#### Scenario: Another address for the same login request
- **WHEN** a tenant was bound to the login challenge for one address, and another address is entered for the same login request
- **THEN** the tenant of the first address is not kept: the binding is written again from the tenants of the second address, and the token issued for the second address names a tenant of that address

#### Scenario: Several tenants
- **WHEN** the address has several tenants
- **THEN** the flow the page renders has no first-factor node (groups `password`, `oidc`, `passkey`, `webauthn`, `code`, `identifier_first`) and one submit node per tenant (group `tenant`, name `sso_tenant`, value the tenant id) in the tenant service's order, under "Choose a tenant to sign in to"

### Requirement: An unknown address keeps the answer of Kratos
When Kratos refuses the address at the identifier step (message `4000037`), the extension MUST NOT reveal more than Kratos does about an address nobody holds:

- a tenant admits the address (pending invitation or auto-join): the tenants are offered as for an account, with the refusal and the identifier step removed from the flow;
- no tenant, and no account holds the address (Kratos admin API, `ListIdentities` by credentials identifier): the flow is returned unchanged;
- no tenant, and an account holds the address: it has no sign-in method, and the recovery prompt below is shown.

#### Scenario: Address with no account and no tenant
- **WHEN** Kratos refused the address, the lookup returns nothing and no account holds it
- **THEN** the page shows the flow and the message of Kratos, unchanged

### Requirement: What a chosen tenant offers
For a flow whose login challenge has a tenant bound, `GET /api/kratos/self-service/login/flows` SHALL lay the flow out from the tenant service's answer for the tenant and the address (`GetSignInContext` with `tenant_id` and `email`) and from the company sign-ins the SSO service lists for the tenant's active bindings that apply to the address (`ListOptions` with those `connection_ids`; no call when there are none):

| enforcement | the flow shows |
|---|---|
| `off` (also reported for a personal tenant and a tenant with no active binding) | the first factors of Kratos; no company sign-in |
| `optional` | the first factors of Kratos, and one button per company sign-in |
| `required` | company sign-ins only: every first-factor node of Kratos is removed |

A company sign-in button is a submit node of group `sso`, name `sso_connection`, value the connection id, labelled "Continue with <label>". A tenant picked from several also gets "Choose another tenant" (`sso_tenant_reset`). A tenant the address is neither a member of nor admitted to MUST be treated as not chosen. Where the account is left nothing to sign in with (at a tenant that does not require company sign-in, or with no tenant), the flow SHALL carry the info message `1990001` "You have no way to sign in with this account any more. Recover it to set a portal password." and a link "Recover your account" to `/ui/reset_email?return_to=<login page of the login challenge>`.

#### Scenario: Required, and no company sign-in applies
- **WHEN** the enforcement is `required` and none is listed, for example because the address is outside the tenant's domains
- **THEN** the response is `403` with error id `sso_not_applicable`, and the page shows "Company sign-in is not available for this address; contact your tenant's administrator"; no password is offered

#### Scenario: Enforcement off, account with no password
- **WHEN** Kratos lists no first factor for the account
- **THEN** the page shows the recovery prompt and its link

### Requirement: Picks are checked on the server
A submission to `POST /api/kratos/self-service/login` that carries `sso_tenant`, `sso_tenant_reset` or `sso_connection` SHALL be answered by the extension, not submitted to Kratos, and checked against the tenant service again:

- `sso_tenant`: the flow must have a login challenge (`400`), and the address must be a member of the tenant or admitted to it (`403` "tenant not allowed"); the tenant is then bound, or its only company sign-in started, as after the email;
- `sso_connection`: a tenant must be chosen for the flow, and the connection must be one of its active bindings that apply to the address (`403` "Provider not allowed"). The chosen tenant is kept in the state cookie, which lives `COOKIE_TTL`: when a flow with a login challenge and an address has no tenant bound any more, the answer is `{"redirect_to": <login page of the flow>}`, and that page lists the tenants again. A flow bound to "no tenant", or with no address, is answered `400`;
- `sso_tenant_reset`: the binding is removed and the list shown again.

The login page SHALL post a pick with its own field and the CSRF token, leaving out what the user typed into the form.

#### Scenario: A connection of another tenant
- **WHEN** `sso_connection` names a connection that is not an active, applying binding of the chosen tenant
- **THEN** the response is `403` and no ticket is asked for

#### Scenario: The state cookie expired on the tenant's page
- **WHEN** a user clicks a company sign-in after the state cookie has expired
- **THEN** the browser goes to the login page of the flow, which lists the tenants of the address again


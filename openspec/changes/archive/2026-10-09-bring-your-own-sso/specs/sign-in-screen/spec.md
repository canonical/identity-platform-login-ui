## Purpose

The sign-ins that a user can use depend on the tenant. So the user chooses the tenant before the first factor, and the user sees only what that tenant accepts. What the browser picks from is a hint. The server checks every pick again.

## ADDED Requirements

### Requirement: Tenants offered after the email
The following applies after Kratos takes the identifier step of a login flow that has a login challenge. The extension SHALL ask the Tenant Service for the tenants that the address can use. The call is `ListSignInTenants` with `email`. The tenants are memberships, pending invitations, and tenants whose auto-join admits the address. The extension SHALL then answer `POST /api/kratos/self-service/login/id-first` itself.

The answer depends on the number of tenants:

- **no tenant**: The extension binds the "no tenant" value to the login challenge in the state cookie. The login page shows the first factors that Kratos lists for the account.
- **one**: The extension binds the tenant to the login challenge. The browser goes to the login page. But if the tenant requires company sign-in and exactly one company sign-in applies to the address, the browser goes straight to that company sign-in instead.
- **several**: The extension binds nothing. The state cookie records that there is a choice (`tc`). The login page lists the tenants and nothing else.

#### Scenario: One tenant that requires its only company sign-in
- **WHEN** the address has one tenant, its enforcement is `required` and one company sign-in applies
- **THEN** the response is `{"redirect_to": <where Kratos sends the browser>, "redirect_label": <label>}`
- **AND** the page shows "Redirecting to <label>…" before it leaves

#### Scenario: Another address for the same login request
- **WHEN** a tenant was bound to the login challenge for one address
- **AND** a user enters another address for the same login request
- **THEN** the extension does not keep the tenant of the first address
- **AND** the extension writes the binding again from the tenants of the second address
- **AND** the token that is issued for the second address names a tenant of that address

#### Scenario: Several tenants
- **WHEN** the address has several tenants
- **THEN** the flow that the page renders has no first-factor node (groups `password`, `oidc`, `passkey`, `webauthn`, `code`, `identifier_first`)
- **AND** the flow has one submit node per tenant (group `tenant`, name `sso_tenant`, value the tenant id) in the Tenant Service's order
- **AND** the nodes are under "Choose a tenant to sign in to"

### Requirement: An unknown address keeps the answer of Kratos
When Kratos refuses the address at the identifier step (message `4000037`), the extension MUST NOT reveal more than Kratos does about an address that nobody holds. Three cases are possible:

- A tenant admits the address (pending invitation or auto-join): the extension offers the tenants as for an account. It removes the refusal and the identifier step from the flow.
- No tenant, and no account holds the address: the extension returns the flow unchanged. It finds out if an account holds the address with the Kratos admin API (`ListIdentities` by credentials identifier).
- No tenant, and an account holds the address: the account has no sign-in method. The page shows the recovery prompt below.

#### Scenario: Address with no account and no tenant
- **WHEN** Kratos refused the address, the look-up returns nothing and no account holds the address
- **THEN** the page shows the flow and the message of Kratos, unchanged

### Requirement: What a chosen tenant offers
For a flow whose login challenge has a tenant bound, `GET /api/kratos/self-service/login/flows` SHALL lay out the flow from two sources:

- the answer of the Tenant Service for the tenant and the address (`GetSignInContext` with `tenant_id` and `email`)
- the company sign-ins that the SSO service lists for the tenant's active bindings that apply to the address (`ListOptions` with those `connection_ids`)

The extension makes no `ListOptions` call when there are no such bindings.

| enforcement | the flow shows |
|---|---|
| `off` (also reported for a personal tenant and a tenant with no active binding) | the first factors of Kratos, and no company sign-in |
| `optional` | the first factors of Kratos, and one button per company sign-in |
| `required` | company sign-ins only: every first-factor node of Kratos is removed |

A company sign-in button is a submit node of group `sso`, name `sso_connection`, value the connection id, labelled "Continue with <label>". If the user picked the tenant from several, the flow also shows "Choose another tenant" (`sso_tenant_reset`). The extension MUST treat a tenant as not chosen if the address is neither a member of the tenant nor admitted to it.

The following applies at a tenant that does not require company sign-in, or with no tenant. When the account is left nothing to sign in with, the flow SHALL carry the info message `1990001` "You have no way to sign in with this account any more. Recover it to set a portal password." The flow SHALL also carry a link "Recover your account" to `/ui/reset_email?return_to=<login page of the login challenge>`.

#### Scenario: Required, and no company sign-in applies
- **WHEN** the enforcement is `required` and no company sign-in is listed (for example because the address is outside the tenant's domains)
- **THEN** the response is `403` with error id `sso_not_applicable`
- **AND** the page shows "Company sign-in is not available for this address; contact your tenant's administrator"
- **AND** the page offers no password

#### Scenario: Enforcement off, account with no password
- **WHEN** Kratos lists no first factor for the account
- **THEN** the page shows the recovery prompt and its link

### Requirement: Picks are checked on the server
The extension SHALL answer a submission to `POST /api/kratos/self-service/login` that carries `sso_tenant`, `sso_tenant_reset` or `sso_connection`. The extension SHALL NOT submit it to Kratos. The extension SHALL check the submission against the Tenant Service again.

For `sso_tenant`, these rules apply:

- The flow must have a login challenge. If not, the answer is `400`.
- The address must be a member of the tenant or admitted to it. If not, the answer is `403` "tenant not allowed".
- Then the extension binds the tenant, or it starts the only company sign-in of the tenant, as after the email.

For `sso_connection`, these rules apply:

- A tenant must be chosen for the flow.
- The connection must be one of the chosen tenant's active bindings that apply to the address. If not, the answer is `403` "Provider not allowed".
- The state cookie keeps the chosen tenant, and the cookie lives `COOKIE_TTL`. When a flow with a login challenge and an address has no tenant bound any more, the answer is `{"redirect_to": <login page of the flow>}`. That page lists the tenants again.
- If a flow is bound to "no tenant", or the flow has no address, the answer is `400`.

For `sso_tenant_reset`, the extension removes the binding and shows the list again.

The login page SHALL post a pick with its own field and the CSRF token. It SHALL leave out what the user typed into the form.

#### Scenario: A connection of another tenant
- **WHEN** `sso_connection` names a connection that is not an active, applying binding of the chosen tenant
- **THEN** the response is `403`
- **AND** the extension asks for no ticket

#### Scenario: The state cookie expired on the tenant's page
- **WHEN** a user clicks a company sign-in after the state cookie expired
- **THEN** the browser goes to the login page of the flow
- **AND** that page lists the tenants of the address again

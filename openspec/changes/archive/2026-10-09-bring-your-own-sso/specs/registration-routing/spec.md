## Purpose

"Create account" normally ends with a portal password and a personal tenant. For an address that a tenant admits through its company sign-in, this is the wrong account. Such an account has a password that the tenant does not accept, and whoever types the address first creates it. So the extension sends such an address to the company sign-in of the tenant, and Kratos registers the account from that sign-in.

## ADDED Requirements

### Requirement: Tenants that admit a registration
The following applies to a submission to `POST /api/kratos/self-service/registration` that carries an email trait (`traits.email`, nested or flat). The extension reads the submission as `company-sign-in` says it reads submissions. Before the submission reaches Kratos, the extension SHALL work out the tenants whose company sign-in the address goes to. It SHALL do this in two steps:

1. Look up the address (`ListSignInTenants`). Keep the tenants that are marked as auto-join candidate or pending invitation.
2. For each of these tenants, ask `GetSignInContext` with `tenant_id` and `email`.

A tenant admits the registration when the address is not a member and one of these conditions is true:

- auto-join admits the address
- a pending invitation admits the address, no account holds the address, and the enforcement of the tenant is `required`

If no tenant admits the registration, the hook MUST NOT answer. Kratos handles the registration as before.

If the look-up fails, or the sign-in context of one of these tenants fails, the hook MUST answer `503`. The failure can be an outage or have any other reason. The answer MUST have a plain message that creating an account is temporarily unavailable. The hook MUST NOT let the submission reach Kratos, because the address can be one that a tenant admits. Such an address must not get a password account. The hook MUST NOT leave out a tenant whose sign-in context failed as a tenant that does not admit the address.

There is one exception: an address that the Tenant Service refuses as malformed (`INVALID_ARGUMENT` from the look-up). No tenant admits such an address. The hook MUST NOT answer, and Kratos answers with its own validation message.

The extension MUST refuse with `400` a submission that carries the email trait both nested and flat, with different values. Such a submission MUST NOT reach Kratos. The handler reads the flat trait for the `profile` method and the nested trait for other methods. Without the refusal, the address that the extension looks up can differ from the address that Kratos registers.

#### Scenario: The tenant service is down
- **WHEN** a user submits a registration while the Tenant Service cannot be reached
- **THEN** the answer is `503`
- **AND** the registration page shows that creating an account is temporarily unavailable
- **AND** no account is created

#### Scenario: The tenant service fails for one tenant
- **WHEN** the look-up lists a tenant as an auto-join candidate
- **AND** the sign-in context of the tenant ends in an error that is not an outage
- **THEN** the answer is `503`
- **AND** the submission does not reach Kratos as a password registration

#### Scenario: Malformed address
- **WHEN** the submission carries an email trait that is not an address
- **THEN** the submission goes on to Kratos
- **AND** the registration page shows the validation message of Kratos

#### Scenario: Two addresses in one submission
- **WHEN** the browser posts `{"method": "profile", "traits": {"email": "a@example.com"}, "traits.email": "b@example.com"}`
- **THEN** the response is `400`
- **AND** the extension does not pass the submission on to Kratos

#### Scenario: Ordinary address
- **WHEN** no tenant invited the address and no auto-join admits it
- **THEN** the registration goes on as before, with a password or a public provider

#### Scenario: Invited to a tenant that does not require company sign-in
- **WHEN** the only pending invitation is to a tenant with enforcement `off` or `optional`
- **THEN** the registration goes on as before

### Requirement: An admitted address goes to the company sign-in
With one admitting tenant, the extension SHALL answer the submission itself. It lists the company sign-ins of the tenant for the address (`ListOptions`). If the call fails, the answer is `503` `sso_unavailable`. If the list is empty, the answer is `403` `sso_not_applicable`.

The extension then creates a login flow without the Kratos session cookie. The flow returns to one of two places:

- the login page of the registration's login challenge
- if there is no login challenge, `/api/v0/sso/complete?return_to=<return_to of the registration flow>`

The number of company sign-ins decides what happens next:

- One company sign-in: The extension starts an attempt on that flow with the join mark. The answer is `200` with `redirect_to`, `redirect_label` and a `continue_with` entry `redirect_browser_to`. The registration page shows "Redirecting to <label>…".
- Several company sign-ins: The registration cookie records the login flow, the tenant, the address and the `return_to`. The browser goes to the login page of that flow. That page shows the company sign-ins of the tenant and nothing else.

With several admitting tenants, the login page of such a flow SHALL first list those tenants and nothing else. A pick MUST name a tenant that admits the address at that moment. Otherwise the answer is `403` "tenant not allowed".

Suppose none of the tenants admits the address any more when the page is laid out. Then the flow SHALL carry the error message `1990004` "No tenant accepts this address any more. Start creating your account again." in place of the list.

The extension never asks the user for a password. Kratos registers the account when the company sign-in returns. Login UI creates the membership once the sign-in passes the checks of the tenant (`tenant-join-at-sign-in`). The time depends on the registration:

- If the registration had a login challenge, Login UI creates the membership at the accept of the Hydra login.
- Otherwise, Login UI creates the membership at `/api/v0/sso/complete`.

#### Scenario: Address in an auto-join domain
- **WHEN** a user with no account submits an address that the auto-join of one tenant admits
- **AND** the tenant has one company sign-in
- **THEN** the extension sends the browser to that company sign-in
- **AND** the extension does not submit the registration to Kratos

#### Scenario: No company sign-in applies
- **WHEN** the admitting tenant has no company sign-in for the address
- **THEN** the response is `403` `sso_not_applicable`
- **AND** the extension offers no password registration in its place

#### Scenario: The tenants stopped admitting the address
- **WHEN** the login page of a registration that several tenants admitted is laid out
- **AND** no tenant admits the address any more
- **THEN** the page shows "No tenant accepts this address any more. Start creating your account again."
- **AND** the page lists no tenant

#### Scenario: Registration started from an app
- **WHEN** the registration flow belongs to a Hydra login request
- **THEN** the company sign-in returns to the login page of that request
- **AND** the checks run, the account joins the tenant, and Login UI accepts the login for it

# registration-routing Specification

## Purpose

"Create account" normally ends with a portal password and a personal tenant. For an address a tenant admits through its company sign-in, that would be the wrong account: a password the tenant does not accept, created by whoever typed the address first. Such an address is sent to the tenant's company sign-in, and Kratos registers the account from it.

## Requirements
### Requirement: Tenants that admit a registration
For a submission to `POST /api/kratos/self-service/registration` that carries an email trait (`traits.email`, nested or flat, read as `company-sign-in` says submissions are read), the extension SHALL work out, before the submission reaches Kratos, the tenants whose company sign-in the address is sent to:

1. look the address up (`ListSignInTenants`) and keep the tenants marked as auto-join candidate or pending invitation;
2. for each, ask `GetSignInContext` with `tenant_id` and `email`: the tenant admits the registration when the address is not a member and either auto-join admits it, or a pending invitation admits it while no account holds the address and the tenant's enforcement is `required`.

With no such tenant the hook MUST NOT answer, and Kratos handles the registration as before. When the lookup or the sign-in context of any of those tenants fails, for an outage or for any other reason, the hook MUST answer `503` with a plain message that creating an account is temporarily unavailable, and MUST NOT let the submission reach Kratos: the address may be one that a tenant admits, which must not get a password account. A tenant whose sign-in context failed MUST NOT be left out as one that does not admit the address. The one exception is an address the tenant service refuses as malformed (`INVALID_ARGUMENT` from the lookup): no tenant admits it, the hook MUST NOT answer, and Kratos answers with its own validation message.

A submission that carries the email trait both nested and flat, with different values, MUST be refused with `400` and MUST NOT reach Kratos: the handler reads the flat one for the `profile` method and the nested one otherwise, so the address looked up could differ from the one registered.

#### Scenario: The tenant service is down
- **WHEN** a registration is submitted while the tenant service cannot be reached
- **THEN** the answer is `503`, the registration page shows that creating an account is temporarily unavailable, and no account is created

#### Scenario: The tenant service fails for one tenant
- **WHEN** the lookup lists a tenant as an auto-join candidate and its sign-in context ends in an error that is not an outage
- **THEN** the answer is `503` and the submission does not reach Kratos as a password registration

#### Scenario: Malformed address
- **WHEN** the submission carries an email trait that is not an address
- **THEN** the submission goes on to Kratos, and the registration page shows the validation message of Kratos

#### Scenario: Two addresses in one submission
- **WHEN** the browser posts `{"method": "profile", "traits": {"email": "a@example.com"}, "traits.email": "b@example.com"}`
- **THEN** the response is `400` and the submission is not passed on to Kratos

#### Scenario: Ordinary address
- **WHEN** no tenant invited the address and no auto-join admits it
- **THEN** the registration goes on as before, with a password or a public provider

#### Scenario: Invited to a tenant that does not require company sign-in
- **WHEN** the only pending invitation is to a tenant with enforcement `off` or `optional`
- **THEN** the registration goes on as before

### Requirement: An admitted address goes to the company sign-in
With one admitting tenant the extension SHALL answer the submission itself. It lists the tenant's company sign-ins for the address (`ListOptions`; a failure is `503` `sso_unavailable`, an empty list `403` `sso_not_applicable`) and creates a login flow, without the Kratos session cookie, that returns to the login page of the registration's login challenge or, with none, to `/api/v0/sso/complete?return_to=<return_to of the registration flow>`.

- One company sign-in: an attempt is started on that flow with the join mark; the answer is `200` with `redirect_to`, `redirect_label` and a `continue_with` entry `redirect_browser_to`, and the registration page shows "Redirecting to <label>…".
- Several: the registration cookie records the login flow, the tenant, the address and the `return_to`, and the browser goes to that flow's login page, which shows the tenant's company sign-ins and nothing else.

With several admitting tenants, the login page of such a flow SHALL first list those tenants and nothing else; a pick MUST name a tenant that admits the address at that moment (`403` "tenant not allowed" otherwise). When none of them admits the address any more by the time the page is laid out, the flow SHALL carry the error message `1990004` "No tenant accepts this address any more. Start creating your account again." in place of the list. The user is never asked for a password. Kratos registers the account when the company sign-in returns, and Login UI creates the membership once the sign-in passes the tenant's checks: at the accept of the Hydra login when the registration had a login challenge, at `/api/v0/sso/complete` otherwise (`tenant-join-at-sign-in`).

#### Scenario: Address in an auto-join domain
- **WHEN** a user with no account submits an address one tenant's auto-join admits, and the tenant has one company sign-in
- **THEN** the browser is sent to that company sign-in, and the registration is not submitted to Kratos

#### Scenario: No company sign-in applies
- **WHEN** the admitting tenant has no company sign-in for the address
- **THEN** the response is `403` `sso_not_applicable`; no password registration is offered in its place

#### Scenario: The tenants stopped admitting the address
- **WHEN** the login page of a registration several tenants admitted is laid out, and no tenant admits the address any more
- **THEN** the page shows "No tenant accepts this address any more. Start creating your account again." and lists no tenant

#### Scenario: Registration started from an app
- **WHEN** the registration flow belongs to a Hydra login request
- **THEN** the company sign-in returns to the login page of that request, the checks run, the account joins the tenant, and the login is accepted for it


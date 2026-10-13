# backend-resilience Specification

## Purpose

With the extension, a sign-in depends on 2 more services. When one of them is slow or down, the browser must get an answer. The answer must tell an outage ("try again in a moment") from a refusal ("you are not a member"). The outage must not weaken a rule. When Login UI cannot ask the SSO service, a tenant that requires company sign-in does not fall back to the password.

## Requirements
### Requirement: Every call has a deadline
Everything that the extension does for one browser request SHALL fit in 12 seconds. The write timeout of the server is 15 seconds. Within the 12 seconds, one call to the tenant service SHALL take at most `TENANT_SERVICE_GRPC_TIMEOUT` (default 5 s). One call to the SSO service SHALL take at most `SSO_SERVICE_GRPC_TIMEOUT` (default 3 s).

#### Scenario: A backend that does not answer
- **WHEN** the tenant service accepts a call and never answers
- **THEN** the call ends after its timeout
- **AND** the browser receives the outage answer below

### Requirement: Only repeatable calls are retried
The gRPC channels SHALL retry a call that ends `UNAVAILABLE`, up to 3 attempts in all (backoff from 0.1 s to 1 s). The channels SHALL retry only these methods, each of which leaves the same state when repeated:

- `ListSignInTenants`, `GetSignInContext`, `JoinTenant` and `CreatePersonalTenant` of the tenant service
- the `LookupTenants` that the handlers call on the same channel
- `ListOptions`, `StartAttempt`, `CompleteAttempt` and `ListLinks` of the SSO service

The channels MUST NOT retry `DeleteLink`. This is because, when the answer to the first call is lost, a second call reports a removed link as a link that never existed. The channels retry no other status. The channels SHALL reconnect with a backoff from 0.2 s to at most 5 s. This lets Login UI reach a restarted service again within seconds.

#### Scenario: Removing a link
- **WHEN** `DeleteLink` ends `UNAVAILABLE`
- **THEN** the channel does not repeat the call
- **AND** the settings submission answers `500`

### Requirement: Calls carry no token
Login UI SHALL call both services without credentials of its own. The services serve the RPCs that Login UI calls to the internal network without a token. Those RPCs are `LookupTenants` and `TenantSignInService` of the tenant service, and `SSOSignInService` of the SSO service. The channels do not need to be reachable from anywhere else.

#### Scenario: A call to the SSO service
- **WHEN** the extension calls `StartAttempt`
- **THEN** the call carries no `authorization` metadata

### Requirement: The tenant service is needed for every sign-in to an app
Login UI SHALL NOT accept a Hydra login without the answer of the tenant service for the tenant and the account. Login UI answers a failed call to the tenant service with `503`, error id `sso_unavailable` and one of 2 messages. After the email, Login UI answers only an outage in this way, and leaves another failure of the lookup to the handler. The 2 messages are:

- Login UI cannot reach the tenant service, or the tenant service is too slow (`UNAVAILABLE`, `DEADLINE_EXCEEDED`, or the 12 seconds ran out): "signing in is temporarily unavailable; try again in a moment".
- Any other failure, also an answer that Login UI cannot interpret: "single sign-on is temporarily unavailable".

A refusal keeps its own answer (`403` `tenant_not_a_member`, `403` `sso_not_applicable`). An enforcement or MFA policy value that the answer leaves unspecified, or that this version does not know, MUST be an error. Login UI MUST NOT read such a value as `off` or `none`.

When the lookup fails while Login UI lays out the tenant list, the flow SHALL carry an error message in place of the tenant list. The error message is `1990002` "Your tenants could not be loaded. Try again in a moment.". Login UI returns a flow that still shows the first factors of the account as Kratos made it, and the failure appears at the accept. Login UI also answers a registration with `503`, whatever the failure, and nothing reaches Kratos (`registration-routing`).

#### Scenario: Tenant service down after the email
- **WHEN** the lookup after the identifier step ends `UNAVAILABLE`
- **THEN** the response is `503`
- **AND** the login page shows "Signing in is temporarily unavailable; try again in a moment"

#### Scenario: A policy value from a newer tenant service
- **WHEN** the answer names an enforcement value that this version does not know, or names none
- **THEN** the sign-in fails with `503`
- **AND** Login UI does not treat the tenant as a tenant with no enforcement

### Requirement: An SSO service outage affects only new company sign-ins
When the extension cannot ask the SSO service, the extension SHALL confine the effect to company sign-ins:

- A tenant with enforcement `off`, a personal tenant, and a session whose provenance cookie is already written need no call. The outage does not affect them.
- At a tenant with enforcement `optional`, the page shows the other sign-ins with the info message `1990003` "Company sign-in is unavailable right now. You can sign in another way.". The account-linking page and the refresh login do the same.
- At a tenant with enforcement `required`, the answer MUST be `503`, error id `sso_unavailable`, "single sign-on is temporarily unavailable". The answer MUST be the same wherever the extension must start or confirm a company sign-in. The extension MUST NOT offer the password instead.
- Settings pages render without the company sign-ins.

#### Scenario: SSO service down, tenant that requires company sign-in
- **WHEN** a member of such a tenant enters their email
- **THEN** the login page shows "Single sign-on is temporarily unavailable" and offers no password

#### Scenario: SSO service down on the return
- **WHEN** `CompleteAttempt` fails for a session that just came back from a company sign-in
- **THEN** the response is `503`
- **AND** Login UI records no provenance and does not accept the login

### Requirement: Outages and refusals are shown on the login page
The login page SHALL keep the user on the page for an answer whose error id is `sso_unavailable`, `sso_not_applicable` or `tenant_not_a_member`. This applies whichever request produced the answer. The login page SHALL show the message of the answer as an error notification. When a pick fails for a reason that the page does not recognise, the page SHALL show "Something went wrong. Try again." and leave the buttons usable. Any other failure goes to the error page, as before.

#### Scenario: Not a member
- **WHEN** the checks refuse the account at the chosen tenant
- **THEN** the login page shows "This account is not a member of that tenant"


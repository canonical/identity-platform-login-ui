# backend-resilience Specification

## Purpose

With the extension, a sign-in depends on two more services. When one is slow or down, the browser must get an answer, the answer must tell an outage ("try again in a moment") from a refusal ("you are not a member"), and the outage must not weaken a rule: a tenant that requires company sign-in does not fall back to the password because the SSO service cannot be asked.

## Requirements
### Requirement: Every call has a deadline
Everything the extension does for one browser request SHALL fit in 12 seconds (the server's write timeout is 15). Within it, one call to the tenant service SHALL take at most `TENANT_SERVICE_GRPC_TIMEOUT` (default 5 s), one to the SSO service at most `SSO_SERVICE_GRPC_TIMEOUT` (default 3 s), and one request for a service token at most 5 seconds.

#### Scenario: A backend that does not answer
- **WHEN** the tenant service accepts a call and never answers
- **THEN** the call ends after its timeout, and the browser receives the outage answer below

### Requirement: Only repeatable calls are retried
The gRPC channels SHALL retry a call that ends `UNAVAILABLE`, up to three attempts in all (backoff from 0.1 s to 1 s), and only these methods, each of which leaves the same state when repeated: `ListSignInTenants`, `GetSignInContext`, `JoinTenant` and `CreatePersonalTenant` of the tenant service, and the `LookupTenants` the handlers call on the same channel; `ListOptions`, `StartAttempt`, `CompleteAttempt` and `ListLinks` of the SSO service. `DeleteLink` MUST NOT be retried: a second call, after the answer to the first was lost, would report a removed link as one that never existed. No other status is retried. The channels SHALL reconnect with a backoff from 0.2 s to at most 5 s, so that a restarted service is reached again within seconds.

#### Scenario: Removing a link
- **WHEN** `DeleteLink` ends `UNAVAILABLE`
- **THEN** it is not repeated, and the settings submission answers `500`

### Requirement: Calls carry Login UI's service token
Every call on both channels SHALL carry `authorization: Bearer <token>`, the tenant lookups multi-tenancy already made included (they share the channel). The token is obtained with the client-credentials grant from `SERVICE_TOKEN_URL` and reused until it expires; it is sent on a plaintext channel too. When no token can be obtained the call MUST fail without being sent: as `UNAVAILABLE` when the token endpoint cannot be reached or answers with a server error, as `UNAUTHENTICATED` when it refuses the client.

#### Scenario: Token endpoint down
- **WHEN** the token endpoint cannot be reached
- **THEN** calls to both services are answered as outages

### Requirement: The tenant service is needed for every sign-in to an app
No Hydra login SHALL be accepted without the tenant service's answer for the tenant and the account. A failed call to it is answered `503` with error id `sso_unavailable` and one of two messages (after the email, only an outage is answered this way; another failure of the lookup is left to the handler):

- it could not be reached or was too slow (`UNAVAILABLE`, `DEADLINE_EXCEEDED`, or the 12 seconds ran out): "signing in is temporarily unavailable; try again in a moment";
- any other failure, including an answer Login UI cannot interpret: "single sign-on is temporarily unavailable".

A refusal keeps its own answer (`403` `tenant_not_a_member`, `403` `sso_not_applicable`). An enforcement or MFA policy value that the answer leaves unspecified, or that this version does not know, MUST be an error, never read as `off` or `none`. When the lookup fails while the tenant list is laid out, the flow SHALL carry the error message `1990002` "Your tenants could not be loaded. Try again in a moment." in its place; a flow that still shows the account's first factors is returned as Kratos made it, and the failure surfaces at the accept. A registration is answered `503` as well, whatever the failure, and nothing reaches Kratos (`registration-routing`).

#### Scenario: Tenant service down after the email
- **WHEN** the lookup after the identifier step ends `UNAVAILABLE`
- **THEN** the response is `503`, and the login page shows "Signing in is temporarily unavailable; try again in a moment"

#### Scenario: A policy value from a newer tenant service
- **WHEN** the answer names an enforcement value this version does not know, or names none
- **THEN** the sign-in fails with `503`; the tenant is not treated as having no enforcement

### Requirement: An SSO service outage affects only new company sign-ins
When the SSO service cannot be asked, the extension SHALL confine the effect to company sign-ins:

- a tenant with enforcement `off`, a personal tenant, and a session whose provenance cookie is already written need no call, and are not affected;
- at a tenant with enforcement `optional`, the page shows the other sign-ins with the info message `1990003` "Company sign-in is unavailable right now. You can sign in another way."; so do the account-linking page and the refresh login;
- at a tenant with enforcement `required`, and wherever a company sign-in has to be started or confirmed, the answer MUST be `503`, error id `sso_unavailable`, "single sign-on is temporarily unavailable"; the password MUST NOT be offered instead;
- settings pages render without the company sign-ins.

#### Scenario: SSO service down, tenant that requires company sign-in
- **WHEN** a member of such a tenant enters their email
- **THEN** the login page shows "Single sign-on is temporarily unavailable" and offers no password

#### Scenario: SSO service down on the return
- **WHEN** `CompleteAttempt` fails for a session that has just come back from a company sign-in
- **THEN** the response is `503`, no provenance is recorded and the login is not accepted

### Requirement: Outages and refusals are shown on the login page
The login page SHALL keep the user on the page for an answer whose error id is `sso_unavailable`, `sso_not_applicable` or `tenant_not_a_member`, whichever request produced it, and show the message of the answer as an error notification. A pick that fails for a reason the page does not recognise SHALL show "Something went wrong. Try again." and leave the buttons usable. Any other failure goes to the error page, as before.

#### Scenario: Not a member
- **WHEN** the checks refuse the account at the chosen tenant
- **THEN** the login page shows "This account is not a member of that tenant"


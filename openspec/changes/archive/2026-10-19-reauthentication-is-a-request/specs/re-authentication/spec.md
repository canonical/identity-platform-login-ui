## MODIFIED Requirements

### Requirement: A company sign-in passes the request on
A fresh first factor SHALL be what the tenant offers. At a tenant that requires company sign-in, it is the company sign-in of the tenant.

This rule applies to a company sign-in that starts from the sign-in screen, or because the checks before an accept asked for one. The `StartAttempt` of such a company sign-in SHALL carry `reauthenticate`. The value SHALL be true in any of these cases:

- The request has `prompt=login`.
- The request has a `max_age` that the current session exceeds.
- There is no session, and the request has `max_age=0`.

The SSO service then asks the identity provider for a fresh login. The identity provider can ignore that request. No service checks that the user signed in again at the identity provider. When a user signs in again for a settings change (`connected-accounts`), `reauthenticate` is always set.

#### Scenario: `prompt=login` at a tenant that requires company sign-in
- **WHEN** the user has a session from the company sign-in of that tenant, and the app sends `prompt=login`
- **THEN** Login UI starts a new attempt with `reauthenticate: true`
- **AND** Login UI accepts the session that comes back

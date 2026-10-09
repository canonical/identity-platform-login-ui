# login-session-reuse Specification

## Purpose

A client can demand that the user signs in again for a login (`prompt=login`, `max_age`), and Hydra then answers that the login must not be skipped. With multi-tenancy enabled the Login UI keeps per-login state in an encrypted cookie bound to the login challenge, and uses it to tell a user who has just signed in for a login from one who arrives with a session from before. If that state can be created without signing in, a session from before is accepted for a login that demanded a new sign-in, and Hydra records the accept as a new authentication. This capability defines the evidence on which a session may be accepted without asking Hydra, what the Login UI does with every other session, and what that evidence leaves open.

## Requirements

### Requirement: Hydra decides for a session that did not sign in for the login
With multi-tenancy enabled, when the login page is opened for a login challenge with a Kratos session that did not sign in for that challenge, the Login UI SHALL ask Hydra whether the login may be skipped. It SHALL accept the login on that session, or send the user to the tenant selection, only if Hydra answers that the login may be skipped; otherwise it SHALL show the login. It SHALL NOT send such a session to the authenticator or passkey setup.

#### Scenario: Request that demands re-authentication, opened again after the email
- **WHEN** a user with a session starts a login with `max_age=0` or `prompt=login`, enters their email, and opens the address of the login request again
- **THEN** the login is shown again and the client receives no authorization code

#### Scenario: Tenant selected without signing in
- **WHEN** a tenant is stored for a login challenge through the tenant selection endpoint by a browser whose session is from before the login, and Hydra answers that the login must not be skipped
- **THEN** the login is shown and the session is not accepted

#### Scenario: Session reused for another client
- **WHEN** a user with a session and one tenant starts a login that Hydra allows to be skipped
- **THEN** the login is accepted on that session without credentials

#### Scenario: Session reused by a user with several tenants
- **WHEN** a user with a session and several tenants starts a login that Hydra allows to be skipped
- **THEN** the user selects a tenant and the login is accepted on that session without credentials

### Requirement: A session signs in for a login by authenticating after the login started
The Login UI SHALL treat a session as signed in for a login challenge only when the state cookie is bound to that challenge, records when the login started, and the session's `authenticated_at` is later than that start. The start SHALL be the `issued_at` of the first Kratos login flow for the challenge whose submission of a credential or of a provider choice Kratos did not refuse; a later flow of the same login SHALL NOT replace it, and the email step SHALL NOT record one. The Login UI SHALL take both times from Kratos and SHALL NOT use its own clock or a value sent by the browser.

#### Scenario: Sign-in with a password and a second factor
- **WHEN** a user enters email, password and an authentication code for a client's login
- **THEN** the session is signed in for the challenge and the login is accepted without asking Hydra whether it may be skipped

#### Scenario: Sign-in through an external provider
- **WHEN** a user chooses an external provider for a client's login and returns from it with a session
- **THEN** the session is signed in for the challenge, because the start was recorded before the browser left

#### Scenario: Re-authentication completed
- **WHEN** a user with a session starts a login with `max_age=0` and signs in again for it
- **THEN** the login is accepted after that one sign-in

#### Scenario: Cookie with no start
- **WHEN** the state cookie is bound to the challenge and records no start, as a cookie written before this capability existed does
- **THEN** the session is not signed in for the challenge and Hydra decides

#### Scenario: Password reset started from the login
- **WHEN** a user enters their email for a client's login, resets their password from the password step, and the browser returns to the login with the session of the recovery
- **THEN** the session is not signed in for the challenge, and the sign-in form is shown unless Hydra allows the login to be skipped

#### Scenario: Sign-in made elsewhere after the login started
- **WHEN** a credential was submitted for a client's login, and the user then signs in on another page of the Login UI in the same browser before returning to that login
- **THEN** the session is treated as signed in for the challenge; this is accepted, the evidence being a time and not a link between the sign-in and the login

### Requirement: State recorded for a login vouches only for the session that signed in for it
When a session did not sign in for the login challenge, the Login UI SHALL renew the state cookie for that challenge before it decides, keeping only a tenant recorded for the same challenge. A record of an authenticator setup, a passkey setup or a used backup code SHALL NOT exempt such a session from Hydra's decision. The tenant selection endpoint SHALL renew the cookie for the challenge it stores a tenant for, and SHALL NOT carry over anything recorded for another challenge.

#### Scenario: Setup recorded by another sign-in
- **WHEN** the state cookie records an authenticator setup for a login challenge, and the login page is opened for that challenge with a session that did not sign in for it
- **THEN** Hydra is asked whether the login may be skipped, as if nothing had been recorded

#### Scenario: Tenant selected for a second login
- **WHEN** a tenant is selected for a login challenge and the browser's state cookie was written for another challenge
- **THEN** the cookie stored for the new challenge holds the selected tenant and nothing else from the other login

### Requirement: Logins without multi-tenancy are unchanged
With multi-tenancy disabled the Login UI SHALL decide about a login opened with a session as it did before this capability, and SHALL NOT read the recorded start.

#### Scenario: Request that demands re-authentication without multi-tenancy
- **WHEN** a user with a session starts a login with `max_age=0`, enters their email, and opens the address of the login request again
- **THEN** the login is shown again


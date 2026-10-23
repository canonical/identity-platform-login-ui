# re-authentication Specification

## Purpose

An app can ask for a sign-in made now (`prompt=login`) or not older than a given age (`max_age`). Hydra cannot answer this for Login UI. Hydra sets the authentication time of a login to the moment when Login UI accepts it. So Hydra sees a week-old Kratos session that Login UI accepted a minute ago as fresh. The extension judges freshness itself, against the first factor of the Kratos session.

## Requirements
### Requirement: The app's request is read from the authorization URL
The extension SHALL read `prompt` and `max_age` from the `request_url` of the Hydra login request. It uses the copy that the login flow carries, and otherwise it gets the value from Hydra. These rules apply:

- `prompt` asks for re-authentication when its space-separated values include `login`.
- `max_age` counts when it is a non-negative integer of seconds.
- The extension MUST NOT use the `skip` of Hydra.
- The extension does not read `prompt=none`.

#### Scenario: Reusing a session
- **WHEN** an app sends neither `prompt=login` nor `max_age`
- **AND** the browser has a session that passes the checks of the tenant
- **THEN** the login is accepted with no prompt, in both cases: Hydra remembers the user, or Hydra does not

### Requirement: When a fresh first factor is owed
A session SHALL owe a fresh first factor in one of two cases:

- The request has `prompt=login`.
- The request has a `max_age` that is shorter than the time since the first authentication method of the session completed.

A session SHALL NOT owe a fresh first factor when its first factor started for this very login request. The fresh mark tells this. The mark names three things:

- the hash of the login challenge
- the id of the session that the browser had when the first factor started (none if the browser sent no session)
- the time when Kratos issued the login flow that the first factor began on

A session is fresh only when all of these conditions are true:

- The mark is for this login challenge.
- The session has another id than the id that the mark names.
- Its first factor completed at or after the time that the mark names.

The time of the first factor is the completion time of the first authentication method of the session. If the method carries no time, the time is the `authenticated_at` of the session. A mark with no time MUST NOT make a session fresh.

The extension SHALL write the fresh mark (cookie `login_ui_sso_fresh`) when a first factor starts for a login challenge. This happens in two cases:

- A submission on an `aal1` flow has the method `password`, `oidc`, `passkey`, `webauthn` or `code`, or it names a provider.
- Login UI starts any company sign-in.

The mark of the request itself also counts, because a password completes in the same request. The extension clears the mark at the accept.

#### Scenario: `prompt=login` with a password session
- **WHEN** an app sends `prompt=login` and the browser has a session from a portal password
- **THEN** the extension does not accept the session
- **AND** the browser lands on a new login flow for the same address, and the email is not asked again
- **AND** after the password, and MFA again, Login UI accepts the login without a further prompt

#### Scenario: `max_age`
- **WHEN** the first factor of the session completed 10 minutes ago
- **THEN** `max_age=3600` asks nothing
- **AND** `max_age=60` asks for a fresh first factor

#### Scenario: The login request opened again
- **WHEN** an app sends `prompt=login` or `max_age=0`, and the browser has a session
- **AND** the user opens the address of the login request again before a first factor is given
- **THEN** the extension still does not accept the session
- **AND** the extension asks for the first factor again, each time that the user opens the address
- **AND** a state cookie that is bound to the login challenge does not make a session fresh
- **AND** a session is fresh only if it differs from the session that the browser had when the first factor started
- **AND** such a session is fresh only if its first factor completed after that start

#### Scenario: An older session shown after the first factor started
- **WHEN** an app sends `prompt=login`
- **AND** a browser that sends no session cookie starts a first factor
- **AND** a session whose first factor completed before that start is then presented for the login request
- **THEN** the extension does not accept the session
- **AND** the extension asks for the first factor again

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

### Requirement: Another user remembered by Hydra
The extension MUST NOT accept when both of these conditions are true:

- the Hydra login request names a subject that Hydra remembers for this browser
- that subject is not the account of the Kratos session

If the extension accepts, Hydra starts the request of the app again with `prompt=login` added. Then the user is asked for the first factor that the user has just given.

The extension SHALL do two things instead:

- End that Hydra login session by its `sid` (`RevokeOAuth2LoginSessions`, with back-channel logout to the apps of the other user).
- Answer `{"redirect_to": <request_url of the login request>}`, so that the authorization of the app starts again for the user who is here.

If the revoke fails, the answer is `500`.

#### Scenario: A colleague signs in on the same browser
- **WHEN** user A signed in to an app earlier
- **AND** user B now signs in through their company in the same browser
- **THEN** the return of B from the company sign-in ends the Hydra login session of A
- **AND** the request of the app starts again
- **AND** B is signed in after one company sign-in


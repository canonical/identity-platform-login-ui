## Purpose

An app can ask for a sign-in made now (`prompt=login`) or not older than a given age (`max_age`). Hydra cannot answer that for Login UI: it sets the authentication time of a login to the moment Login UI accepts it, so a week-old Kratos session accepted a minute ago looks fresh to it. The extension judges freshness itself, against the first factor of the Kratos session.

## ADDED Requirements

### Requirement: The app's request is read from the authorization URL
The extension SHALL read `prompt` and `max_age` from the `request_url` of the Hydra login request (the copy the login flow carries, otherwise from Hydra): `prompt` asks for re-authentication when its space-separated values include `login`; `max_age` counts when it is a non-negative integer of seconds. Hydra's `skip` MUST NOT be used. `prompt=none` is not read.

#### Scenario: Reusing a session
- **WHEN** an app sends neither `prompt=login` nor `max_age`, and the browser has a session that passes the tenant's checks
- **THEN** the login is accepted with no prompt, whether or not Hydra remembers the user

### Requirement: When a fresh first factor is owed
A session SHALL owe a fresh first factor when the request has `prompt=login`, or a `max_age` shorter than the time since the first authentication method of the session completed. It SHALL NOT owe one when its first factor was started for this very login request, which the fresh mark tells. The mark names the hash of the login challenge, the id of the session the browser had when the first factor started (none when it sent no session), and the time Kratos issued the login flow the first factor began on. A session is fresh only when all of these hold:

- the mark is for this login challenge;
- the session has another id than the one the mark names;
- its first factor completed at or after the time the mark names. The time is the completion time of the first authentication method of the session, or the session's `authenticated_at` when the method carries none.

A mark with no time MUST NOT make a session fresh.

The fresh mark (cookie `login_ui_sso_fresh`) SHALL be written when a first factor starts for a login challenge: a submission on an `aal1` flow whose method is `password`, `oidc`, `passkey`, `webauthn` or `code`, or that names a provider; and every company sign-in Login UI starts. The mark of the request itself counts too (a password completes in the same request). It is cleared at the accept.

#### Scenario: `prompt=login` with a password session
- **WHEN** an app sends `prompt=login` and the browser has a session from a portal password
- **THEN** the session is not accepted; the browser lands on a new login flow for the same address (the email is not asked again), and after the password, and MFA again, the login is accepted without a further prompt

#### Scenario: `max_age`
- **WHEN** the first factor of the session completed ten minutes ago
- **THEN** `max_age=3600` asks nothing, and `max_age=60` asks for a fresh first factor

#### Scenario: The login request opened again
- **WHEN** an app sends `prompt=login` or `max_age=0`, the browser has a session, and the address of the login request is opened again before a first factor is given
- **THEN** the session is still not accepted and the first factor is asked again, however often the address is opened: a state cookie bound to the login challenge does not make a session fresh, only a session other than the one the browser had when the first factor started, whose first factor completed after that start, does

#### Scenario: An older session shown after the first factor started
- **WHEN** an app sends `prompt=login`, a first factor is started by a browser that sends no session cookie, and a session whose first factor completed before that start is then presented for the login request
- **THEN** the session is not accepted, and the first factor is asked again

### Requirement: A company sign-in passes the request on
A fresh first factor SHALL be what the tenant offers: at a tenant that requires company sign-in, its company sign-in. A company sign-in started from the sign-in screen, or because the checks before an accept asked for one, SHALL carry `reauthenticate` in its `StartAttempt`: true when the request has `prompt=login`, or a `max_age` the current session exceeds, or, with no session, `max_age=0`. The SSO service then asks the identity provider for a fresh login. Signing in again for a settings change (`connected-accounts`) always sets it.

#### Scenario: `prompt=login` at a tenant that requires company sign-in
- **WHEN** the user has a session from that tenant's company sign-in and the app sends `prompt=login`
- **THEN** a new attempt is started with `reauthenticate: true`, and the session that comes back is accepted

### Requirement: Another user remembered by Hydra
When the Hydra login request names a subject Hydra remembers for this browser, and it is not the account of the Kratos session, the extension MUST NOT accept: Hydra would start the app's request again with `prompt=login` added, and the user would be asked for the first factor they have just given. The extension SHALL instead end that Hydra login session by its `sid` (`RevokeOAuth2LoginSessions`, with back-channel logout to the other user's apps) and answer `{"redirect_to": <request_url of the login request>}`, so that the app's authorization starts again for the user who is here. A failure to revoke is answered `500`.

#### Scenario: A colleague signs in on the same browser
- **WHEN** user A signed in to an app earlier, and user B now signs in through their company in the same browser
- **THEN** B's return from the company sign-in ends A's Hydra login session, the app's request starts again, and B is signed in after one company sign-in

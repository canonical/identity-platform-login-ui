## Purpose

A login started by an OAuth 2.0 client is identified by Hydra's login challenge. The Login UI walks the user through several steps for it (email, tenant selection, password, second factor), each on its own URL, and starts a new Kratos login flow whenever the flow of a step can no longer be used, for example after the browser's Back button. If the new flow is not tied to the same login challenge, the user signs in successfully and the client never receives its authorization code. This capability defines how the login challenge is kept across the steps and across a restart, and how a restart ends when the client's request can no longer be completed, so that a restart never silently turns the client's login into a login with no client.

## ADDED Requirements

### Requirement: Login steps carry the client's login challenge in their URL
When the login page loads a login flow by its id, and the flow carries an `oauth2_login_challenge`, the Login UI SHALL add that challenge to the page URL as the `login_challenge` query parameter if the URL has none. It SHALL do so by replacing the current history entry, without navigating and without adding a history entry, and SHALL leave the other query parameters of the URL unchanged. The Login UI SHALL add the challenge only to the URL of its own login page.

#### Scenario: Step after the email
- **WHEN** the user submits their email for a client's login and the browser is sent to `/ui/login?flow=<id>`
- **THEN** the URL of that step becomes `/ui/login?flow=<id>&login_challenge=<challenge>` and the browser history has no additional entry

#### Scenario: Password step reached from the tenant selection
- **WHEN** a user with several tenants picks one and the browser is sent to `/ui/login?flow=<id>`
- **THEN** the URL of that step gains the `login_challenge` of the flow

#### Scenario: Second-factor step
- **WHEN** the browser is redirected to the second factor of a client's login at `/ui/login?flow=<id>`
- **THEN** the URL of that step gains the `login_challenge` of the flow

#### Scenario: Login with no client
- **WHEN** the login page loads a flow that carries no `oauth2_login_challenge`
- **THEN** the URL is left unchanged

### Requirement: A restarted login stays the client's login
When the Login UI starts a new login flow in place of one that can no longer be used, and the URL of the page names both a `flow` and a `login_challenge`, it SHALL start the new login for the same `login_challenge`. The restart SHALL go to the login page of the Login UI itself with the URL-encoded `login_challenge` as its only query parameter.

#### Scenario: Back on the second-factor page
- **WHEN** a user has entered email and password for a client's login, presses the browser's Back button on the second-factor page, and signs in again on the login that is shown
- **THEN** the user is returned to the client with an authorization code

#### Scenario: Back on the authenticator setup of a first sign-in
- **WHEN** a user without a second factor has entered email and password for a client's login, presses the browser's Back button on the authenticator setup page, signs in again and completes the setup
- **THEN** the user is returned to the client with an authorization code

#### Scenario: Back after a tenant selection
- **WHEN** a user with several tenants has picked a tenant and entered the password for a client's login, presses the browser's Back button on the second-factor page, and signs in again
- **THEN** the user is returned to the client with an authorization code, and the token carries the tenant that was picked

### Requirement: A login that could not be created is not restarted with its challenge
When starting a login flow fails and the URL of the page carries a `login_challenge` but names no `flow`, a restart SHALL NOT keep the `login_challenge`. The Login UI SHALL start a login without it, so that a challenge for which no flow can be created does not make the page reload itself repeatedly.

#### Scenario: Flow creation refused
- **WHEN** creating the flow for `/ui/login?login_challenge=<challenge>` is answered with `self_service_flow_return_to_forbidden`
- **THEN** the browser goes to the login page without a `login_challenge` once, and does not load `/ui/login?login_challenge=<challenge>` again

### Requirement: Logins with no client and other flows restart as before
A restarted login whose URL carries no `login_challenge` SHALL start at the login page with no query parameters. The restart of registration, recovery, settings and verification flows SHALL NOT be changed by this capability.

#### Scenario: Login with no client
- **WHEN** the flow of a login with no client can no longer be used
- **THEN** the new login starts at `/ui/login` with no query parameters

#### Scenario: Settings flow
- **WHEN** a settings flow can no longer be used
- **THEN** the page it was on is loaded again with its query parameters except the flow id

### Requirement: A restart is a login for the client's request whatever its state
The Login UI SHALL treat a restarted login as a login for the client's request and SHALL NOT replace it with a login that has no client, also when Hydra no longer completes that request. The Login UI does not record which login challenges it has accepted. When the request can no longer be completed, the login SHALL NOT complete for the client and the user SHALL NOT end on the account page as if it had.

#### Scenario: The login had already completed
- **WHEN** the login for the challenge has completed, the user goes back to one of its steps, and signs in again on the login that is shown
- **THEN** Hydra refuses the replay and the client is called back with `error=access_denied`, "The consent verifier has already been used"

#### Scenario: The challenge has expired and Kratos is given the challenge
- **WHEN** the login challenge is older than Hydra's login request lifetime, the login is restarted, and the Login UI passes the challenge to Kratos (`OIDC_WEBAUTHN_SEQUENCING_ENABLED` and `MULTI_TENANCY_ENABLED` both off)
- **THEN** no flow is created and the error page is shown

#### Scenario: The challenge has expired and Kratos is not given the challenge
- **WHEN** the login challenge is older than Hydra's login request lifetime, the login is restarted, and the Login UI does not pass the challenge to Kratos (`OIDC_WEBAUTHN_SEQUENCING_ENABLED` or `MULTI_TENANCY_ENABLED` on)
- **THEN** the user can enter their credentials, the login request is not accepted, and an error is shown on the login page

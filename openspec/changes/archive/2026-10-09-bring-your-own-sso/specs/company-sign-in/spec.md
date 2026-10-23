## Purpose

Every company sign-in of every tenant goes through one Kratos OIDC provider, `byo-sso`. Login UI gets a ticket for one tenant, one address and one connection. From that ticket, the SSO service decides which identity provider the user meets. So only Login UI is allowed to submit that provider.

A Kratos session records only the provider. Thus Login UI must learn and remember which connection a session came from. Login UI learns the connection from the SSO service, which answers only for the browser that completed the sign-in.

## ADDED Requirements

### Requirement: Only Login UI submits the `byo-sso` provider
Login UI MUST remove the nodes of the provider from every login, registration and settings flow that it returns. The nodes of the provider have group `oidc` and value `byo-sso`:

- the `provider` submit of a login or registration flow
- the `link` and `unlink` submits of a settings flow

Login UI MUST refuse a browser submission that names the provider with `403` "Provider not allowed", and MUST NOT pass that submission on to Kratos. These submissions name the provider:

- `provider=byo-sso` on a login or registration flow
- `link=byo-sso` or `unlink=byo-sso` on a settings flow

The extension MUST read a submission in the same way as the handler reads it afterwards. The handler decodes the JSON object that the body starts with into a struct. This step matches a field name whatever its case. Of 2 fields that differ only in case, it takes the later field. Thus the extension reads the top-level fields of that object whatever their case, but the keys inside `traits` stay exact.

The extension MUST refuse with `400` a login, registration or settings submission in which 2 top-level fields are equal when case is ignored. This rule applies to the fields in the body of the submission and to the fields in its query.

#### Scenario: Crafted login submission
- **WHEN** the browser posts `{"method": "oidc", "provider": "byo-sso"}` to `POST /api/kratos/self-service/login`
- **THEN** the response is `403`
- **AND** Login UI does not pass the submission on to Kratos

#### Scenario: The field in another case
- **WHEN** the browser posts `{"method": "oidc", "Provider": "byo-sso"}`
- **THEN** the response is `403`

#### Scenario: A field named twice
- **WHEN** the browser posts `{"provider": "google", "Provider": "byo-sso"}`
- **THEN** the response is `400`
- **AND** Login UI does not pass the submission on to Kratos

### Requirement: Starting a company sign-in
To start a company sign-in, Login UI SHALL do these steps in this order:

1. Ask the SSO service for a ticket with `StartAttempt`, which carries `tenant_id`, `email`, `connection_id` and `reauthenticate`. Login UI answers the reason `NOT_APPLICABLE` (for example a connection that is not tested) with `403` `sso_not_applicable`. Login UI answers any other failure with `503` `sso_unavailable`.
2. Submit the login flow to Kratos itself. The submission has method `oidc`, provider `byo-sso`, the CSRF token of the flow, and `upstream_parameters.login_hint` set to the ticket. The submission carries the cookies of the request except the Kratos session cookie. Login UI passes an error of Kratos that carries an id on to the frontend, as the handlers pass the errors of their own submissions (`400` with the error as JSON). Any other failure, or an answer that names no redirect, is `500`. The login and registration pages act on an id that they know. They follow a redirect that Kratos names, and start an expired flow again. For any other id, the pages go to the error page.
3. Write the sign-in cookie. For a login challenge, also bind the tenant to the login challenge in the state cookie and write the fresh cookie.
4. Expire the Kratos session cookie in the browser. The identity provider returns to Kratos directly, and this return must not carry a session.
5. Answer `{"redirect_to": <the URL Kratos returned>}`. Add `redirect_label` (the label of the company sign-in), unless the user clicked the `sso_connection` button of that same sign-in. With a label, the page shows "Redirecting to <label>…" before it leaves.

The login flow MUST return to a URL of Login UI. This URL is the login page of the login challenge, or `/api/v0/sso/complete` when there is no login challenge.

#### Scenario: Ticket refused
- **WHEN** `StartAttempt` answers `NOT_APPLICABLE`
- **THEN** the response is `403` `sso_not_applicable`
- **AND** Login UI does not submit the provider
- **AND** Login UI writes no sign-in cookie

### Requirement: Confirming the return and recording provenance
This requirement applies when a session reaches the checks before an accept, or reaches `/api/v0/sso/complete`. Login UI SHALL treat the session as the return of a company sign-in only if all of these conditions are true:

- the sign-in cookie holds a ticket
- the id of the session differs from the session id that the cookie recorded
- the session includes a `byo-sso` authentication method

Login UI SHALL then ask the SSO service to confirm that the attempt ended at this account, in this browser. The request is `CompleteAttempt` with `ticket`, `identity_id` and `receipt`. The `receipt` is the receipt of the ticket, which the requirement below describes. The SSO service answers with `connection_id` and `tenant_id`. Login UI SHALL then write the provenance cookie for this session and that connection. Login UI SHALL also clear the sign-in cookie and the receipt cookies of its tickets.

These rules also apply:

- The SSO service answers `NOT_APPLICABLE` when the attempt ended at another account, or when the ticket expired. It also answers `NOT_APPLICABLE` when the browser holds no receipt that the SSO service made for this ticket and a company sign-in of this account. Login UI then records no provenance, and clears the sign-in cookie and the receipt cookies of its tickets.
- Login UI answers any other failure with `503` `sso_unavailable`. Login UI accepts nothing, and the cookies stay.
- Provenance MUST count only while the provenance cookie names the id of the current session.

#### Scenario: Return to the login page
- **WHEN** a new session with a `byo-sso` method arrives with the sign-in cookie of its attempt
- **AND** the session arrives with the receipt of its ticket
- **THEN** `CompleteAttempt` confirms the attempt
- **AND** Login UI sets the provenance cookie
- **AND** the checks continue with that connection as the provenance

#### Scenario: The session the browser already had
- **WHEN** the session id equals the session id in the sign-in cookie
- **THEN** Login UI does not call `CompleteAttempt`

### Requirement: The receipt of a company sign-in
With each ticket that it asks the SSO service about, Login UI SHALL send the receipt of that ticket. The receipt is the unchanged value of a cookie that the SSO service sets in the browser whose company sign-in it accepts. A ticket says which sign-in was started. A ticket does not say that the sign-in was completed, and does not say in which browser. The SSO service confirms an attempt only when the receipt comes with the ticket.

The cookie belongs to the SSO service. The name of the cookie is `__Host-sso_receipt_` followed by the first 16 lower-case hexadecimal digits of the SHA-256 of the ticket. Thus each attempt has its own cookie. The value of the cookie is an authentication tag that only the SSO service can make or check. The value holds nothing about the user.

When the request has no such cookie, Login UI MUST still ask, with an empty receipt. The SSO service refuses an empty receipt, so one place makes and counts every refusal. Login UI MUST NOT decode the value, log the value, or store the value anywhere else. Login UI reads the value for this purpose only.

Wherever Login UI clears the sign-in cookie of an attempt, it SHALL also clear the receipt cookie of every ticket that the sign-in cookie held. This applies after a confirmation and after a refusal. The clearing MUST carry the same name, `Path=/` and `Secure`, because a browser rejects a change to a `__Host-` cookie without them.

The browser sends the cookie to Login UI only when the same host serves Login UI and the browser pages of the SSO service. The identity providers of the SSO service return to those browser pages. The host must be exactly the same, because a `__Host-` cookie cannot name a `Domain`. Ports and paths do not matter.

This is a condition of the deployment, in the same way as the existing condition that the cookies of Kratos reach Login UI. Where the deployment does not meet this condition, no company sign-in is confirmed. Login UI then asks the user for the company sign-in of the tenant again after every return.

The receipt does not do 2 things:

- The receipt is not single use. The same browser can present the receipt again until the ticket expires, 30 minutes after the sign-in started. That gives nothing more than the session that the sign-in already produced.
- The receipt proves that this browser completed the sign-in of this ticket, as a company sign-in that the account is linked to. The receipt does not prove that the Kratos session presented with it is the session that this sign-in produced.

#### Scenario: The receipt goes with its ticket
- **WHEN** a new session with a `byo-sso` method arrives with the sign-in cookie of ticket T
- **AND** the session arrives with the cookie `__Host-sso_receipt_<digest of T>`
- **THEN** `CompleteAttempt` carries T and the value of that cookie
- **AND** the response clears both cookies when `CompleteAttempt` confirms, and also when it answers `NOT_APPLICABLE`

#### Scenario: No receipt in the browser
- **WHEN** the browser sends the sign-in cookie of a ticket and no receipt cookie for that ticket
- **THEN** Login UI calls `CompleteAttempt` with an empty receipt
- **AND** the SSO service answers `NOT_APPLICABLE`
- **AND** Login UI records no provenance

#### Scenario: A ticket presented with a session made through another connection
- **WHEN** a user starts a company sign-in of tenant A and stops at the identity provider of that sign-in
- **AND** the user signs in through another connection that the account is linked to
- **AND** the user then presents the sign-in cookie of the first attempt with the new session
- **THEN** the browser holds no receipt for that ticket
- **AND** the SSO service does not confirm the attempt
- **AND** Login UI does not record that the session came from the connection of A
- **AND** at tenant A the user goes through the company sign-in of A again

#### Scenario: Someone else's browser finished the sign-in
- **WHEN** a browser presents the same ticket after another person's browser completed the sign-in of that ticket at the identity provider
- **THEN** the receipt is in that other browser and not in the browser that presents the ticket
- **AND** the SSO service does not confirm the attempt

### Requirement: `GET /api/v0/sso/complete`
A company sign-in with no Hydra login request SHALL return to `GET /api/v0/sso/complete?return_to=…`. Such a sign-in is a registration with no app, or a sign-in that a user does again for a settings change. With no Kratos session, the endpoint redirects (`303`) to `/ui/login`. With a Kratos session, the endpoint SHALL do these steps:

1. confirm the attempt as above (`503` when the SSO service fails)
2. join the tenant when the attempt carries the join mark (`tenant-join-at-sign-in`)
3. redirect (`303`) to `return_to` only when it equals the `return_to` stored in the sign-in cookie of the attempt just confirmed

The stored `return_to` came from a Kratos flow, so it is a return URL that Kratos allows. In every other case the endpoint redirects to `/ui/manage_details`.

#### Scenario: Another `return_to`
- **WHEN** the query names a `return_to` that is not the `return_to` in the sign-in cookie
- **THEN** the endpoint redirects the browser to `/ui/manage_details`

### Requirement: Cookies of the extension
The extension SHALL keep its state between requests in these cookies only. The extension MUST encrypt and authenticate each cookie with the cookie cipher of `internal/cookies` (AES-GCM under `COOKIES_ENCRYPTION_KEY`). The extension MUST set each cookie with path `/`, `HttpOnly`, `Secure` and `SameSite=Lax`, because the cookies must arrive on the redirect chain that starts at the identity provider.

The state cookie uses the same cipher, and the cipher binds a value to no purpose. Thus the extension seals each cookie together with its own name, and a value MUST open only as the cookie that it was sealed for. The extension MUST treat a cookie as absent when the cookie does not decrypt, or when it holds a value sealed for another cookie. None of these cookies uses `COOKIE_TTL`.

| cookie | holds | lifetime |
|---|---|---|
| `login_ui_sso_signin` | the ticket, the id of the session that the browser had at the start, the hash of the login challenge, `return_to` for `/api/v0/sso/complete`, tenant id, connection id, the ticket being linked (`account-linking`), the join mark | 30 minutes. Cleared, with the receipt cookies of its tickets, when the attempt is confirmed, or when it is found that the attempt did not end at this account |
| `login_ui_sso_provenance` | session id, connection id | until the Kratos session expires (1 hour when it names no expiry) |
| `login_ui_sso_fresh` | the hash of the login challenge, the id of the session that the browser had when the first factor started, the time when the mark was made (`re-authentication`) | 30 minutes. Cleared at the accept |
| `login_ui_sso_registration` | login flow id, tenant id, address, `return_to` (`registration-routing`) | 30 minutes. Cleared when the user picks a company sign-in on that flow |

The state cookie of `internal/cookies` gets the field `tc`. This field means that the user has several tenants to choose from.

The receipt cookie of the SSO service (above) is not one of these cookies. Login UI does not write the receipt cookie and cannot read what it holds. Login UI passes the value on and clears the cookie.

#### Scenario: The value of one cookie sent as another
- **WHEN** the browser sends the value of the sign-in cookie, or of the state cookie, as the fresh cookie
- **THEN** there is no fresh mark

#### Scenario: Tampered provenance cookie
- **WHEN** Login UI cannot decrypt the provenance cookie
- **THEN** the provenance is unknown
- **AND** the session goes through the company sign-in again where the tenant has a company sign-in

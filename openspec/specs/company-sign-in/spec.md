# company-sign-in Specification

## Purpose

Every company sign-in, of every tenant, goes through one Kratos OIDC provider, `byo-sso`. The SSO service decides which identity provider the user meets, from a ticket Login UI obtains for one tenant, one address and one connection. So only Login UI may submit that provider. And since a Kratos session records only the provider, Login UI must learn and remember which connection a session came from. It learns it from the SSO service, which answers only for the browser that completed the sign-in.

## Requirements
### Requirement: Only Login UI submits the `byo-sso` provider
The nodes of the provider (group `oidc`, value `byo-sso`: the `provider` submit of a login or registration flow, the `link` and `unlink` submits of a settings flow) MUST be removed from every login, registration and settings flow Login UI returns. A browser submission that names the provider MUST be refused with `403` "Provider not allowed" and not passed on to Kratos: `provider=byo-sso` on a login or registration flow, `link=byo-sso` or `unlink=byo-sso` on a settings flow.

The extension MUST read a submission as the handler goes on to read it. The handler decodes the JSON object the body starts with into a struct, which matches a field name whatever its case and, of two fields that differ only in case, takes the later. The extension therefore reads the top-level fields of that object whatever their case (the keys inside `traits` stay exact), and MUST refuse with `400` a login, registration or settings submission in which two top-level fields are equal ignoring case, in its body or in its query.

#### Scenario: Crafted login submission
- **WHEN** the browser posts `{"method": "oidc", "provider": "byo-sso"}` to `POST /api/kratos/self-service/login`
- **THEN** the response is `403` and the submission is not passed on to Kratos

#### Scenario: The field in another case
- **WHEN** the browser posts `{"method": "oidc", "Provider": "byo-sso"}`
- **THEN** the response is `403`

#### Scenario: A field named twice
- **WHEN** the browser posts `{"provider": "google", "Provider": "byo-sso"}`
- **THEN** the response is `400` and the submission is not passed on to Kratos

### Requirement: Starting a company sign-in
To start a company sign-in Login UI SHALL, in this order:

1. ask the SSO service for a ticket: `StartAttempt` with `tenant_id`, `email`, `connection_id` and `reauthenticate`. The reason `NOT_APPLICABLE` (for example a connection that is not tested) is answered `403` `sso_not_applicable`, any other failure `503` `sso_unavailable`;
2. submit the login flow to Kratos itself: method `oidc`, provider `byo-sso`, the CSRF token of the flow, and `upstream_parameters.login_hint` set to the ticket, with the cookies of the request except the Kratos session cookie. An error of Kratos that carries an id is passed on to the frontend as the handlers pass the errors of their own submissions (`400` with the error as JSON); any other failure, or an answer that names no redirect, is `500`. The login and registration pages act on an id they know (a redirect Kratos names is followed, a flow that expired is started again) and go to the error page for any other;
3. write the sign-in cookie and, for a login challenge, bind the tenant to it in the state cookie and write the fresh cookie;
4. expire the Kratos session cookie in the browser: the identity provider returns to Kratos directly, and must not carry a session there;
5. answer `{"redirect_to": <the URL Kratos returned>}`, with `redirect_label` (the label of the company sign-in) unless the user clicked that sign-in's own `sso_connection` button. With a label, the page shows "Redirecting to <label>…" before it leaves.

The login flow MUST return to a Login UI URL: the login page of the login challenge, or `/api/v0/sso/complete` when there is none.

#### Scenario: Ticket refused
- **WHEN** `StartAttempt` answers `NOT_APPLICABLE`
- **THEN** the response is `403` `sso_not_applicable`, the provider is not submitted, and no sign-in cookie is written

### Requirement: Confirming the return and recording provenance
When a session reaches the checks before an accept, or `/api/v0/sso/complete`, Login UI SHALL treat it as the return of a company sign-in only if the sign-in cookie holds a ticket, the id of the session differs from the one the cookie recorded, and the session includes a `byo-sso` authentication method. It SHALL then ask the SSO service to confirm that the attempt ended at this account, in this browser (`CompleteAttempt` with `ticket`, `identity_id` and `receipt`, the receipt of the ticket described below; answered with `connection_id` and `tenant_id`), write the provenance cookie for this session and that connection, and clear the sign-in cookie and the receipt cookies of its tickets.

- `NOT_APPLICABLE` (the attempt ended at another account, the browser holds no receipt that the SSO service made for this ticket and a company sign-in of this account, or the ticket expired) records no provenance, and clears the sign-in cookie and the receipt cookies of its tickets.
- Any other failure is answered `503` `sso_unavailable`; nothing is accepted, and the cookies stay.
- Provenance MUST count only while the provenance cookie names the id of the current session.

#### Scenario: Return to the login page
- **WHEN** a new session with a `byo-sso` method arrives with the sign-in cookie of its attempt and the receipt of its ticket
- **THEN** `CompleteAttempt` confirms the attempt, the provenance cookie is set, and the checks go on with that connection as the provenance

#### Scenario: The session the browser already had
- **WHEN** the session id equals the one in the sign-in cookie
- **THEN** `CompleteAttempt` is not called

### Requirement: The receipt of a company sign-in
With each ticket it asks the SSO service about, Login UI SHALL send the receipt of that ticket: the value, unchanged, of a cookie the SSO service sets in the browser whose company sign-in it accepts. A ticket says which sign-in was started. It does not say that the sign-in was completed, or in which browser, and the SSO service confirms an attempt only when the receipt comes with the ticket.

The cookie is the SSO service's own. Its name is `__Host-sso_receipt_` followed by the first 16 lower-case hexadecimal digits of the SHA-256 of the ticket, so each attempt has its own. Its value is an authentication tag that only the SSO service can make or check, and holds nothing about the user. With no such cookie in the request Login UI MUST still ask, with an empty receipt: the SSO service refuses it, so every refusal is made, and counted, in one place. Login UI MUST NOT decode the value, log it, or store it anywhere else, and reads it for nothing but this.

Wherever Login UI clears the sign-in cookie of an attempt, after a confirmation and after a refusal alike, it SHALL also clear the receipt cookie of every ticket the sign-in cookie held. The clearing MUST carry the same name, `Path=/` and `Secure`: a browser rejects a change to a `__Host-` cookie without them.

The browser sends the cookie to Login UI only when the SSO service's browser pages (where its identity providers return to) and Login UI are served on the same host. It has to be the very same host, since a `__Host-` cookie cannot name a `Domain`; ports and paths do not matter. This is a condition of the deployment, as it already is that the cookies of Kratos reach Login UI. Where it is not met no company sign-in is confirmed, and the user is asked for the tenant's company sign-in again after every return.

The receipt does not do two things:

- it is not single use. The same browser can present it again until the ticket expires, 30 minutes after the sign-in started. That gives nothing beyond the session the sign-in already produced;
- it proves that this browser completed the sign-in of this ticket, as a company sign-in the account is linked to. It does not prove that the Kratos session presented with it is the one that sign-in produced.

#### Scenario: The receipt goes with its ticket
- **WHEN** a new session with a `byo-sso` method arrives with the sign-in cookie of ticket T and the cookie `__Host-sso_receipt_<digest of T>`
- **THEN** `CompleteAttempt` carries T and the value of that cookie, and whether it confirms or answers `NOT_APPLICABLE`, the response clears both cookies

#### Scenario: No receipt in the browser
- **WHEN** the browser sends the sign-in cookie of a ticket and no receipt cookie for it
- **THEN** `CompleteAttempt` is called with an empty receipt, the SSO service answers `NOT_APPLICABLE`, and no provenance is recorded

#### Scenario: A ticket presented with a session made through another connection
- **WHEN** a user starts a company sign-in of tenant A and stops at its identity provider, signs in through another connection the account is linked to, and then presents the sign-in cookie of the first attempt with the new session
- **THEN** the browser holds no receipt for that ticket, the attempt is not confirmed, and the session is not recorded as coming from A's connection: at tenant A the user goes through A's company sign-in again

#### Scenario: Someone else's browser finished the sign-in
- **WHEN** the same ticket is presented after its sign-in was completed at the identity provider in another person's browser
- **THEN** the receipt is in that browser and not in the one presenting the ticket, and the attempt is not confirmed

### Requirement: `GET /api/v0/sso/complete`
A company sign-in with no Hydra login request (a registration with no app, or signing in again for a settings change) SHALL return to `GET /api/v0/sso/complete?return_to=…`. With no Kratos session the endpoint redirects (`303`) to `/ui/login`. Otherwise it SHALL confirm the attempt as above (`503` when the SSO service fails), join the tenant when the attempt carries the join mark (`tenant-join-at-sign-in`), and redirect (`303`) to `return_to` only when it equals the `return_to` stored in the sign-in cookie of the attempt just confirmed, which came from a Kratos flow and is therefore a return URL Kratos allows. In every other case it redirects to `/ui/manage_details`.

#### Scenario: Another `return_to`
- **WHEN** the query names a `return_to` that is not the one in the sign-in cookie
- **THEN** the browser is redirected to `/ui/manage_details`

### Requirement: Cookies of the extension
The extension SHALL keep its state between requests in these cookies only. Each MUST be encrypted and authenticated with the cookie cipher of `internal/cookies` (AES-GCM under `COOKIES_ENCRYPTION_KEY`) and set with path `/`, `HttpOnly`, `Secure` and `SameSite=Lax` (they must arrive on the redirect chain that starts at the identity provider). The cipher is shared with the state cookie and binds a value to no purpose, so each cookie is sealed together with its own name, and a value MUST open only as the cookie it was sealed for. A cookie that does not decrypt, or holds a value sealed for another cookie, MUST be treated as absent. None uses `COOKIE_TTL`.

| cookie | holds | lifetime |
|---|---|---|
| `login_ui_sso_signin` | the ticket; the id of the session the browser had at the start; the hash of the login challenge; `return_to` for `/api/v0/sso/complete`; tenant id; connection id; the ticket being linked (`account-linking`); the join mark | 30 minutes; cleared, with the receipt cookies of its tickets, when the attempt is confirmed, or found not to have ended at this account |
| `login_ui_sso_provenance` | session id, connection id | until the Kratos session expires (1 hour when it names no expiry) |
| `login_ui_sso_fresh` | the hash of the login challenge; the id of the session the browser had when the first factor started; the time the mark was made (`re-authentication`) | 30 minutes; cleared at the accept |
| `login_ui_sso_registration` | login flow id, tenant id, address, `return_to` (`registration-routing`) | 30 minutes; cleared when a company sign-in is picked on that flow |

The state cookie of `internal/cookies` gains the field `tc`: the user has several tenants to choose from.

The receipt cookie of the SSO service (above) is not one of them. Login UI does not write it and cannot read what it holds: it passes the value on and clears the cookie.

#### Scenario: The value of one cookie sent as another
- **WHEN** the browser sends the value of the sign-in cookie, or of the state cookie, as the fresh cookie
- **THEN** there is no fresh mark

#### Scenario: Tampered provenance cookie
- **WHEN** the provenance cookie cannot be decrypted
- **THEN** the provenance is unknown, and the session goes through the company sign-in again where the tenant has one


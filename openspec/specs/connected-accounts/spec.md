# connected-accounts Specification

## Purpose

A user can see which company sign-ins are attached to their account, and remove one. Kratos cannot do this: all company sign-ins share one Kratos provider, so its "unlink" would remove every one of them, and its "link" would start a sign-in with no tenant behind it. Login UI hides those controls, lists the company sign-ins from the SSO service, and removes one through it, under the conditions Kratos sets for changing credentials.

## Requirements
### Requirement: Company sign-ins are listed in settings
Every settings flow returned to the browser SHALL carry, for each company sign-in the SSO service lists for the account (`ListLinks` with `identity_id`), a text node (group `sso`, id `sso_link_<connection id>`, "Signed in with <label>") and a submit node (group `sso`, name `sso_unlink`, value the connection id, "Unlink <label>"). The Connected accounts page SHALL show them in a section "Company sign-ins", each with a Disconnect button behind a confirmation, and no button to connect one. When the SSO service cannot be reached, the settings flow MUST be returned without them rather than fail.

#### Scenario: Account with a company sign-in and a public provider
- **WHEN** the user opens Connected accounts
- **THEN** the page lists the public provider with Kratos's controls, and the company sign-in with a Disconnect button

### Requirement: Disconnecting a company sign-in
A settings submission that carries `sso_unlink` SHALL be answered by the extension and not sent to Kratos. Since Kratos then never checks its CSRF token, the request MUST come from Login UI's own pages: its `Origin` header equals the origin of `BASE_URL` or, with no `Origin`, its `Sec-Fetch-Site` is `same-origin` or `none`; otherwise `403`. A request with neither header is refused: a browser sends at least one of them with this request. The extension SHALL then require, as Kratos does for its own unlink:

1. a Kratos session (`401` otherwise);
2. that the settings flow loads for this session: a redirect Kratos answers with, for example to the second factor, is passed on to the browser, and so is an error of Kratos (for example an expired flow), with its id and the status Kratos gave it, as `GET /settings/flows` passes it. The page acts on an id it knows (an expired flow is started again), and reports any other as a failed disconnect;
3. that the session authenticated within `KRATOS_PRIVILEGED_SESSION_MAX_AGE`; otherwise the answer is `{"error": {"id": "session_refresh_required"}, "redirect_to": "/ui/login?refresh=true&return_to=<…/ui/manage_connected_accounts>"}`.

It then calls `DeleteLink` with `connection_id` and `identity_id`, and answers `200` with the settings flow, listing what is left. The reason `LAST_CREDENTIAL` removes nothing and adds the error message "You cannot remove your only way to sign in. Set a password first."; a link that does not exist adds "This account is not linked to that sign-in method."; any other failure is `500`. The page SHALL report success only when the entry is gone from the returned flow and the flow carries no error message.

#### Scenario: The only way in
- **WHEN** the company sign-in is the account's only way to sign in
- **THEN** the page shows "You cannot remove your only way to sign in. Set a password first.", and the entry stays

#### Scenario: Cross-site request
- **WHEN** the submission arrives with an `Origin` that is not Login UI's
- **THEN** the response is `403` and `DeleteLink` is not called

#### Scenario: Request with neither header
- **WHEN** the submission carries neither `Origin` nor `Sec-Fetch-Site`
- **THEN** the response is `403` and `DeleteLink` is not called

#### Scenario: Session older than the privileged session age
- **WHEN** the session authenticated longer ago than `KRATOS_PRIVILEGED_SESSION_MAX_AGE`
- **THEN** nothing is removed, and the browser is sent to sign in again and back to Connected accounts

### Requirement: Signing in again with a company sign-in
Kratos's refresh login (a login flow with `refresh` and no login challenge, which a privileged settings change asks for) SHALL offer the company sign-ins of the account of the session: a submit node per link, group `sso`, name `sso_reauthenticate`, "Continue with <label>". Without them, a user whose only sign-ins are company sign-ins could not sign in again. A pick MUST be checked: the flow is a refresh flow (`400`), the browser has a session (`401`), and the connection is one the account is linked to (`403` "Provider not allowed").

The company sign-in SHALL run on a new login flow, created without the Kratos session cookie, that returns to `/api/v0/sso/complete?return_to=<return_to of the refresh flow>`: for the tenant of the link and the address of the session, with `reauthenticate` set, so that the identity provider asks the user to log in again. Kratos issues a new session, authenticated then; `/api/v0/sso/complete` confirms the attempt, records provenance, and returns to the page that asked.

#### Scenario: User with company sign-ins only, session older than an hour
- **WHEN** the refresh login is shown
- **THEN** it offers "Continue with <label>" for each company sign-in of the account, and picking one leads back to the settings page with a session authenticated now


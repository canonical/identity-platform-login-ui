## Purpose

A user can see which company sign-ins are attached to their account, and can remove one. Kratos cannot do this. All company sign-ins share one Kratos provider. So the "unlink" of Kratos removes every company sign-in. Its "link" starts a sign-in with no tenant behind it.

Login UI hides those controls. It lists the company sign-ins from the SSO service. It removes one through the SSO service, under the conditions that Kratos sets for a change of credentials.

## ADDED Requirements

### Requirement: Company sign-ins are listed in settings
Every settings flow returned to the browser SHALL carry these nodes for each company sign-in that the SSO service lists for the account (`ListLinks` with `identity_id`):

- a text node (group `sso`, id `sso_link_<connection id>`, "Signed in with <label>")
- a submit node (group `sso`, name `sso_unlink`, value the connection id, "Unlink <label>")

The Connected accounts page SHALL show them in a section "Company sign-ins". Each SHALL have a Disconnect button behind a confirmation. The page SHALL NOT have a button to connect a company sign-in. If the SSO service is not reachable, the extension MUST return the settings flow without these nodes and MUST NOT fail.

#### Scenario: Account with a company sign-in and a public provider
- **WHEN** the user opens Connected accounts
- **THEN** the page lists the public provider with the controls of Kratos
- **AND** the page lists the company sign-in with a Disconnect button

### Requirement: Disconnecting a company sign-in
The extension SHALL itself answer a settings submission that carries `sso_unlink`. It SHALL NOT send the submission to Kratos. Kratos then never checks its CSRF token. So the request MUST come from Login UI's own pages. This means that one of these conditions is true:

- The `Origin` header equals the origin of `BASE_URL`.
- The request has no `Origin` header, and its `Sec-Fetch-Site` is `same-origin` or `none`.

Otherwise the answer is `403`. A request with neither header is refused, because a browser sends at least one of them with this request.

The extension SHALL then require the same as Kratos requires for its own unlink:

1. A Kratos session. If there is none, the answer is `401`.
2. The settings flow loads for this session. This case has these rules:
   - When Kratos answers with a redirect, for example to the second factor, the extension passes it on to the browser.
   - When Kratos answers with an error, for example an expired flow, the extension passes it on with its id and the status that Kratos gave. It does this as `GET /settings/flows` does.
   - The page acts on an id that it knows. For an expired flow it starts the flow again. It reports an error with any other id as a failed disconnect.
3. The session authenticated within `KRATOS_PRIVILEGED_SESSION_MAX_AGE`. If not, the answer is `{"error": {"id": "session_refresh_required"}, "redirect_to": "/ui/login?refresh=true&return_to=<…/ui/manage_connected_accounts>"}`.

Then the extension calls `DeleteLink` with `connection_id` and `identity_id`. It answers `200` with the settings flow, which lists what is left. These cases can occur:

- The reason `LAST_CREDENTIAL` removes nothing. The extension adds the error message "You cannot remove your only way to sign in. Set a password first."
- A link that does not exist adds the error message "This account is not linked to that sign-in method."
- Any other failure gives `500`.

The page SHALL report success only when the entry is gone from the returned flow and the flow carries no error message.

#### Scenario: The only way in
- **WHEN** the company sign-in is the only way to sign in for the account
- **THEN** the page shows "You cannot remove your only way to sign in. Set a password first."
- **AND** the entry stays

#### Scenario: Cross-site request
- **WHEN** the submission arrives with an `Origin` that is not the origin of Login UI
- **THEN** the response is `403`
- **AND** the extension does not call `DeleteLink`

#### Scenario: Request with neither header
- **WHEN** the submission carries neither `Origin` nor `Sec-Fetch-Site`
- **THEN** the response is `403`
- **AND** the extension does not call `DeleteLink`

#### Scenario: Session older than the privileged session age
- **WHEN** the session authenticated longer ago than `KRATOS_PRIVILEGED_SESSION_MAX_AGE`
- **THEN** the extension removes nothing
- **AND** the browser goes to sign in again and then back to Connected accounts

### Requirement: Signing in again with a company sign-in
The refresh login of Kratos is a login flow with `refresh` and no login challenge. A privileged settings change asks for it. The refresh login SHALL offer the company sign-ins of the account of the session. It SHALL offer them as a submit node for each link, with group `sso`, name `sso_reauthenticate` and the label "Continue with <label>". Without these nodes, a user whose only sign-ins are company sign-ins cannot sign in again.

The extension MUST check each pick:

- The flow is a refresh flow. If not, the answer is `400`.
- The browser has a session. If not, the answer is `401`.
- The account is linked to the connection. If not, the answer is `403` "Provider not allowed".

The company sign-in SHALL run on a new login flow. The extension SHALL create this flow without the Kratos session cookie. The flow SHALL return to `/api/v0/sso/complete?return_to=<return_to of the refresh flow>`. The company sign-in SHALL be for the tenant of the link and the address of the session. It SHALL have `reauthenticate` set, so that the identity provider asks the user to log in again.

Kratos issues a new session that authenticated just now. `/api/v0/sso/complete` confirms the attempt and records provenance. Then it returns to the page that asked.

#### Scenario: User with company sign-ins only, session older than an hour
- **WHEN** the refresh login is shown
- **THEN** the refresh login offers "Continue with <label>" for each company sign-in of the account
- **AND** a pick leads back to the settings page with a session that authenticated now

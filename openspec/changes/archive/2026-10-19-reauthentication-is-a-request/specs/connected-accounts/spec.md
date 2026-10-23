## MODIFIED Requirements

### Requirement: Signing in again with a company sign-in
The refresh login of Kratos is a login flow with `refresh` and no login challenge. A privileged settings change asks for it. The refresh login SHALL offer the company sign-ins of the account of the session. It SHALL offer them as a submit node for each link, with group `sso`, name `sso_reauthenticate` and the label "Continue with <label>". Without these nodes, a user whose only sign-ins are company sign-ins cannot sign in again.

The extension MUST check each pick:

- The flow is a refresh flow. If not, the answer is `400`.
- The browser has a session. If not, the answer is `401`.
- The account is linked to the connection. If not, the answer is `403` "Provider not allowed".

The company sign-in SHALL run on a new login flow. The extension SHALL create this flow without the Kratos session cookie. The flow SHALL return to `/api/v0/sso/complete?return_to=<return_to of the refresh flow>`. The company sign-in SHALL be for the tenant of the link and the address of the session. It SHALL have `reauthenticate` set, so that the SSO service asks the identity provider for a fresh login (see `re-authentication`).

Kratos issues a new session that authenticated just now. `/api/v0/sso/complete` confirms the attempt and records provenance. Then it returns to the page that asked.

#### Scenario: User with company sign-ins only, session older than an hour
- **WHEN** the refresh login is shown
- **THEN** the refresh login offers "Continue with <label>" for each company sign-in of the account
- **AND** a pick leads back to the settings page with a session that authenticated now

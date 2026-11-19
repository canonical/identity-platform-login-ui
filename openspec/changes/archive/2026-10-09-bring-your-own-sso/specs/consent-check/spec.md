## Purpose

Login UI runs the checks of the tenant where it accepts a Hydra login. These checks have no value if something else can accept a login. Kratos accepts a Hydra login request itself when it gets the login challenge. When multi-tenancy is on, Login UI never gives the challenge to Kratos. But a crafted browser request can give it. Such a login has no tenant and no MFA of the tenant. Every login goes through consent in Login UI as the last step, so Login UI stops there a login that it did not accept.

## ADDED Requirements

### Requirement: Consent is refused for a login Login UI did not accept
With the extension enabled, `GET /api/consent` SHALL refuse a consent request in either of these cases:

- The subject of the consent request is not the account of the Kratos session. The description is "this sign-in belongs to another account".
- The login context of the consent request has no `tenant_id`. The description is "this sign-in did not go through the portal's sign-in checks".

Every login that the extension accepts names a tenant in its context. A login that anything else accepted names no tenant.

The check SHALL run after the existing checks of the handler for the session and the assurance level. It SHALL run before Login UI accepts the consent. The extension SHALL refuse through Hydra. It calls `RejectOAuth2ConsentRequest` with the error `login_required` and the description. It SHALL answer `{"redirect_to": <where Hydra sends the browser>}`. So the app gets an OAuth2 error and no token.

If the rejection fails, the answer MUST be `403` and Login UI MUST NOT accept the consent. The check MUST NOT call the tenant service or the SSO service. It does not run the checks of the tenant again. It only recognises a login that went through them.

#### Scenario: Login accepted by Login UI
- **WHEN** the consent request has the subject of the session
- **AND** `context.tenant_id` is T
- **THEN** Login UI accepts the consent as before, with T as the tenant of the token

#### Scenario: Login accepted by Kratos from a crafted request
- **WHEN** the login context of the consent request has no `tenant_id`
- **THEN** Hydra gets a rejection with `login_required`
- **AND** Login UI does not accept the consent

#### Scenario: Another account's login
- **WHEN** the subject of the consent request differs from the identity of the Kratos session
- **THEN** the extension rejects the consent with `login_required`

#### Scenario: The rejection fails
- **WHEN** `RejectOAuth2ConsentRequest` fails
- **THEN** the response status is `403`
- **AND** `AcceptOAuth2ConsentRequest` is not called

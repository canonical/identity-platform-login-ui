## Purpose

The tenant's checks run where Login UI accepts a Hydra login, and are worth nothing if a login can be accepted elsewhere. Kratos accepts a Hydra login request itself when it is handed the login challenge. Login UI never hands it over when multi-tenancy is on, but a crafted browser request could, and such a login would carry neither a tenant nor the tenant's MFA. Consent is the last step every login goes through in Login UI, so a login it did not accept is stopped there.

## ADDED Requirements

### Requirement: Consent is refused for a login Login UI did not accept
With the extension enabled, `GET /api/consent` SHALL refuse a consent request, after the handler's existing session and assurance-level checks and before the consent is accepted, when either holds:

- the subject of the consent request is not the account of the Kratos session: description "this sign-in belongs to another account";
- the login context of the consent request has no `tenant_id`: description "this sign-in did not go through the portal's sign-in checks". Every login the extension accepts names a tenant in its context; a login accepted by anything else names none.

It SHALL refuse through Hydra (`RejectOAuth2ConsentRequest` with error `login_required` and the description) and answer `{"redirect_to": <where Hydra sends the browser>}`, so the app receives an OAuth2 error and no token. When the rejection itself fails, the answer MUST be `403` and the consent MUST NOT be accepted. The check MUST NOT call the tenant service or the SSO service: it does not run the tenant's checks again, it only recognises a login that went through them.

#### Scenario: Login accepted by Login UI
- **WHEN** the consent request has the subject of the session and `context.tenant_id` is T
- **THEN** the consent is accepted as before, with T as the tenant of the token

#### Scenario: Login accepted by Kratos from a crafted request
- **WHEN** the login context of the consent request has no `tenant_id`
- **THEN** Hydra receives a rejection with `login_required`, and the consent is not accepted

#### Scenario: Another account's login
- **WHEN** the subject of the consent request differs from the identity of the Kratos session
- **THEN** the consent is rejected with `login_required`

#### Scenario: The rejection fails
- **WHEN** `RejectOAuth2ConsentRequest` fails
- **THEN** the response status is `403` and `AcceptOAuth2ConsentRequest` is not called

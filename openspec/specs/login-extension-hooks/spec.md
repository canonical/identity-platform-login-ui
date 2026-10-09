# login-extension-hooks Specification

## Purpose

The login, registration, settings and consent handlers serve every deployment. An optional feature must not change what they do when it is off. The handlers therefore call a fixed set of hooks at fixed points, and by default every hook does nothing.

## Requirements
### Requirement: Extension hooks with a default that does nothing
`pkg/kratos` SHALL define `ExtensionInterface` with the hooks listed below, and `pkg/extra` one with the hook `GateConsent`. `kratos.NewAPI` and `extra.NewAPI` SHALL use a `NoOpExtension` unless the option `WithExtension` is passed. The `NoOpExtension` MUST return flows unchanged, never answer a request, report that it does not handle session logins, and never refuse a consent. When a hook reports that it answered, the handler MUST write nothing more and return.

#### Scenario: No extension is configured
- **WHEN** the APIs are built without `WithExtension`
- **THEN** every handler writes the response it wrote before the hooks existed

#### Scenario: A hook answers before an accept
- **WHEN** `BeforeAcceptLogin` returns `true`
- **THEN** the handler returns without calling `AcceptLoginRequest`

### Requirement: Hook call sites
The handlers SHALL call the hooks at these points and nowhere else:

| hook | endpoint (under `/api/kratos/self-service` unless given in full) | point |
|---|---|---|
| `HandlesSessionLogin`, `HandleSessionLogin` | `GET /login/browser` | the request has a Kratos session and a `login_challenge`: after the handler's session checks (email verification, MFA set-up, WebAuthn set-up), instead of its tenant selection and its check of Hydra's `skip` |
| `HydrateLoginFlow` | `GET /login/flows`; `GET /login/browser` | before a fetched flow is written; for a flow just created, only when it has no login challenge and is returned as JSON |
| `BeforeTenantSelection` | `POST /login/id-first` | after Kratos took the identifier step, with multi-tenancy on, before the handler selects a tenant from the email |
| `InterceptLoginSubmission` | `POST /login` | the flow is fetched, the body not yet parsed; the hook returns the request the handler goes on with |
| `HandlesSessionLogin` | `POST /login` | after the submission, where the handler would ask its tenant resolver whether the session has to select a tenant: when the extension handles session logins the resolver is not asked, and the state cookie reaches `BeforeAcceptLogin` as the handler read it |
| `BeforeAcceptLogin` | wherever the handlers call `AcceptLoginRequest` | right before it, with the session, the login challenge and the state cookie |
| `InterceptRegistrationSubmission` | `POST /registration` | before the body is parsed |
| `InterceptSettingsSubmission` | `POST /settings` | before the body is parsed |
| `HydrateSettingsFlow` | `GET /settings/browser`, `GET /settings/flows`, `POST /settings` | every settings flow returned to the browser |
| `GateConsent` (`pkg/extra`) | `GET /api/consent` | after the session and the consent request are read, before the consent is accepted |

When `GateConsent` returns a URL the consent handler MUST answer `{"redirect_to": <URL>}`; when it returns an error, `403`. In both cases the consent MUST NOT be accepted.

#### Scenario: Login page with a session and a login challenge
- **WHEN** the extension reports `HandlesSessionLogin` and none of the handler's session checks redirects
- **THEN** the handler calls `HandleSessionLogin` and returns, without consulting the tenant resolver or Hydra's `skip`

#### Scenario: A first factor submitted with no tenant in the state cookie
- **WHEN** the extension reports `HandlesSessionLogin`, and a login submission produces a session for a login challenge the state cookie holds no tenant for
- **THEN** the handler does not call the tenant resolver's `NeedsTenantSelection`, and calls `BeforeAcceptLogin` with a state cookie that names no tenant

#### Scenario: A session check redirects first
- **WHEN** the session's email is unverified and `VERIFICATION_ENABLED` is on
- **THEN** the handler answers with its verification redirect, and `HandleSessionLogin` is not called

#### Scenario: The consent hook rejects
- **WHEN** `GateConsent` returns a URL
- **THEN** the response is `{"redirect_to": <URL>}` and `AcceptOAuth2ConsentRequest` is not called


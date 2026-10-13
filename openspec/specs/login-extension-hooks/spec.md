# login-extension-hooks Specification

## Purpose

The login, registration, settings and consent handlers serve every deployment. An optional feature must not change what the handlers do when the feature is off. Thus the handlers call a fixed set of hooks at fixed points, and by default every hook does nothing.

## Requirements
### Requirement: Extension hooks with a default that does nothing
`pkg/kratos` SHALL define `ExtensionInterface` with the hooks listed below. `pkg/extra` SHALL define an `ExtensionInterface` with the hook `GateConsent`. `kratos.NewAPI` and `extra.NewAPI` SHALL use a `NoOpExtension` unless the option `WithExtension` is passed. The `NoOpExtension` MUST return flows unchanged and MUST NOT answer a request. The `NoOpExtension` MUST report that it does not handle session logins, and MUST NOT refuse a consent. When a hook reports that it answered, the handler MUST write nothing more and MUST return.

#### Scenario: No extension is configured
- **WHEN** the APIs are built without `WithExtension`
- **THEN** every handler writes the response that it wrote before the hooks existed

#### Scenario: A hook answers before an accept
- **WHEN** `BeforeAcceptLogin` returns `true`
- **THEN** the handler returns and does not call `AcceptLoginRequest`

### Requirement: Hook call sites
The handlers SHALL call the hooks at these points and nowhere else:

| hook | endpoint (under `/api/kratos/self-service` unless given in full) | point |
|---|---|---|
| `HandlesSessionLogin`, `HandleSessionLogin` | `GET /login/browser` | the request has a Kratos session and a `login_challenge`. The handler calls the hooks after its session checks (email verification, MFA set-up, WebAuthn set-up), in place of its tenant selection and its check of Hydra's `skip` |
| `HydrateLoginFlow` | `GET /login/flows`, `GET /login/browser` | before the handler writes a fetched flow. For a flow that the handler just created, only when the flow has no login challenge and the handler returns it as JSON |
| `BeforeTenantSelection` | `POST /login/id-first` | after Kratos took the identifier step, with multi-tenancy on, before the handler selects a tenant from the email |
| `InterceptLoginSubmission` | `POST /login` | after the handler fetched the flow and before it parses the body. The hook returns the request that the handler continues with |
| `HandlesSessionLogin` | `POST /login` | after the submission, at the point where the handler otherwise asks its tenant resolver if the session must select a tenant. When the extension handles session logins, the handler does not ask the resolver, and the state cookie reaches `BeforeAcceptLogin` as the handler read it |
| `BeforeAcceptLogin` | wherever the handlers call `AcceptLoginRequest` | right before that call, with the session, the login challenge and the state cookie |
| `InterceptRegistrationSubmission` | `POST /registration` | before the handler parses the body |
| `InterceptSettingsSubmission` | `POST /settings` | before the handler parses the body |
| `HydrateSettingsFlow` | `GET /settings/browser`, `GET /settings/flows`, `POST /settings` | every settings flow that the handler returns to the browser |
| `GateConsent` (`pkg/extra`) | `GET /api/consent` | after the handler read the session and the consent request, before it accepts the consent |

When `GateConsent` returns a URL, the consent handler MUST answer `{"redirect_to": <URL>}`. When `GateConsent` returns an error, the consent handler MUST answer `403`. In both cases the consent handler MUST NOT accept the consent.

#### Scenario: Login page with a session and a login challenge
- **WHEN** the extension reports `HandlesSessionLogin` and none of the handler's session checks redirects
- **THEN** the handler calls `HandleSessionLogin` and returns
- **AND** the handler does not consult the tenant resolver or Hydra's `skip`

#### Scenario: A first factor submitted with no tenant in the state cookie
- **WHEN** the extension reports `HandlesSessionLogin`
- **AND** a login submission produces a session for a login challenge
- **AND** the state cookie holds no tenant for that login challenge
- **THEN** the handler does not call the tenant resolver's `NeedsTenantSelection`
- **AND** the handler calls `BeforeAcceptLogin` with a state cookie that names no tenant

#### Scenario: A session check redirects first
- **WHEN** the session's email is unverified and `VERIFICATION_ENABLED` is on
- **THEN** the handler answers with its verification redirect
- **AND** the handler does not call `HandleSessionLogin`

#### Scenario: The consent hook rejects
- **WHEN** `GateConsent` returns a URL
- **THEN** the response is `{"redirect_to": <URL>}`
- **AND** the handler does not call `AcceptOAuth2ConsentRequest`


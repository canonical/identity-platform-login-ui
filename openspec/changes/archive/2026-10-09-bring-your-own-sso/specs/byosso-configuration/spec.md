## Purpose

The extension is optional, and it is a security control. It is off by default and asks for nothing while off. Once it is on, a deployment that is only half configured must not start: a Login UI that came up without the extension would serve a tenant that requires company sign-in with the portal password and no checks.

## ADDED Requirements

### Requirement: Settings of the extension
Login UI SHALL read these environment variables (`internal/config/specs.go`), all optional while `BYOSSO_ENABLED` is false:

| variable | default | meaning |
|---|---|---|
| `BYOSSO_ENABLED` | `false` | switches the extension on, and with it MFA per tenant (`tenant-mfa`) |
| `SSO_SERVICE_GRPC_ADDRESS` | empty | gRPC target of the SSO service |
| `SSO_SERVICE_GRPC_TIMEOUT` | `3s` | deadline of one call to the SSO service |
| `SSO_SERVICE_TLS_ENABLED` | `false` | TLS (1.2 or later) on the channel to the SSO service |
| `SERVICE_TOKEN_URL` | empty | OAuth2 token endpoint for Login UI's service token |
| `SERVICE_CLIENT_ID`, `SERVICE_CLIENT_SECRET` | empty | Login UI's client credentials |
| `SERVICE_TOKEN_SCOPES` | empty | comma-separated scopes asked for the token |
| `KRATOS_PRIVILEGED_SESSION_MAX_AGE` | `1h` | the age up to which a session may remove a company sign-in; it has to equal Kratos's `selfservice.flows.settings.privileged_session_max_age` |

The id of the Kratos provider, `byo-sso`, is a constant, not a setting. The deadline of a call to the tenant service is the existing `TENANT_SERVICE_GRPC_TIMEOUT`. The channel to the SSO service is dialled only when the extension is on, and closed at shutdown.

The tenants' MFA rule has no setting of its own. `BYOSSO_ENABLED` brings with it:

- MFA decided per tenant (`tenant-mfa`) in place of the platform-wide rule: `MFA_ENABLED` has no effect while the extension is on;
- a Kratos that has to run with `session.whoami.required_aal: aal1` (`tenant-mfa`), which Login UI cannot check;
- `OIDC_WEBAUTHN_SEQUENCING_ENABLED` off: startup refuses the two together (below).

#### Scenario: Defaults
- **WHEN** none of the variables is set
- **THEN** the extension is off and Login UI starts as before

### Requirement: Startup refuses an incomplete configuration
`serve` SHALL return an error, so that the process exits, when:

- `BYOSSO_ENABLED` is true and `MULTI_TENANCY_ENABLED` is not;
- `BYOSSO_ENABLED` is true and any of `SSO_SERVICE_GRPC_ADDRESS`, `SERVICE_TOKEN_URL`, `SERVICE_CLIENT_ID`, `SERVICE_CLIENT_SECRET` is empty;
- `BYOSSO_ENABLED` and `OIDC_WEBAUTHN_SEQUENCING_ENABLED` are both true: that setting asks a WebAuthn key of every sign-in through an external provider, and with the extension each tenant decides MFA, so with both two rules would answer the same question.

#### Scenario: Enabled without a service client
- **WHEN** `BYOSSO_ENABLED=true` and `SERVICE_CLIENT_SECRET` is unset
- **THEN** the process exits with "cannot enable BYO-SSO without SSO_SERVICE_GRPC_ADDRESS, SERVICE_TOKEN_URL, SERVICE_CLIENT_ID and SERVICE_CLIENT_SECRET"

#### Scenario: Enabled together with OIDC-WebAuthn sequencing
- **WHEN** `BYOSSO_ENABLED=true` and `OIDC_WEBAUTHN_SEQUENCING_ENABLED=true`
- **THEN** the process exits with "cannot enable BYO-SSO with OIDC_WEBAUTHN_SEQUENCING_ENABLED"

### Requirement: A wiring fault stops the process
`web.NewRouter` SHALL return an error, and `serve` SHALL exit on it, when the extension is enabled and cannot be built: multi-tenancy has no working tenant resolver, the gRPC client of the tenant service or of the SSO service is missing, or `BASE_URL` does not parse. It MUST NOT fall back to a router without the extension. (Multi-tenancy on its own keeps its existing fallback, with a warning, to a resolver that does nothing when its client is missing.)

#### Scenario: Enabled, but the client of the tenant service was not built
- **WHEN** the router is built with the extension enabled and no client of the tenant service
- **THEN** `NewRouter` returns an error and no router, while the same options with the extension disabled build one

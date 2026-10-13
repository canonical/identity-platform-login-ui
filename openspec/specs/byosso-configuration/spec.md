# byosso-configuration Specification

## Purpose

The extension is optional, and it is a security control. It is off by default and asks for nothing while it is off. When it is on, a deployment that is only half configured must not start. This is because a Login UI that starts without the extension serves a tenant that requires company sign-in with the portal password and no checks.

## Requirements
### Requirement: Settings of the extension
Login UI SHALL read the environment variables in the table below (`internal/config/specs.go`). Each variable is optional while `BYOSSO_ENABLED` is false.

| variable | default | meaning |
|---|---|---|
| `BYOSSO_ENABLED` | `false` | switches on the extension and, with it, MFA per tenant (`tenant-mfa`) |
| `SSO_SERVICE_GRPC_ADDRESS` | empty | gRPC target of the SSO service |
| `SSO_SERVICE_GRPC_TIMEOUT` | `3s` | deadline of one call to the SSO service |
| `SSO_SERVICE_TLS_ENABLED` | `false` | TLS (1.2 or later) on the channel to the SSO service |
| `KRATOS_PRIVILEGED_SESSION_MAX_AGE` | `1h` | the age up to which a session can remove a company sign-in. It must equal Kratos's `selfservice.flows.settings.privileged_session_max_age` |

The id of the Kratos provider, `byo-sso`, is a constant and not a setting. The deadline of a call to the tenant service is the existing `TENANT_SERVICE_GRPC_TIMEOUT`. Login UI dials the channel to the SSO service only when the extension is on, and closes it at shutdown.

The MFA rule of the tenants has no setting of its own. `BYOSSO_ENABLED` brings these things with it:

- MFA per tenant (`tenant-mfa`) replaces the platform-wide rule. `MFA_ENABLED` has no effect while the extension is on.
- Kratos must run with `session.whoami.required_aal: aal1` (`tenant-mfa`). Login UI cannot check that.
- `OIDC_WEBAUTHN_SEQUENCING_ENABLED` must be off. Startup refuses the 2 settings together (below).

#### Scenario: Defaults
- **WHEN** none of the variables is set
- **THEN** the extension is off and Login UI starts as before

### Requirement: Startup refuses an incomplete configuration
`serve` SHALL return an error, so that the process exits, in each of these cases:

- `BYOSSO_ENABLED` is true and `MULTI_TENANCY_ENABLED` is not true.
- `BYOSSO_ENABLED` is true and `SSO_SERVICE_GRPC_ADDRESS` is empty.
- `BYOSSO_ENABLED` and `OIDC_WEBAUTHN_SEQUENCING_ENABLED` are both true.

`serve` refuses the last case because, with both settings on, 2 rules answer the same question. `OIDC_WEBAUTHN_SEQUENCING_ENABLED` asks every sign-in through an external provider for a WebAuthn key. With the extension, each tenant decides MFA.

#### Scenario: Enabled without the SSO service
- **WHEN** `BYOSSO_ENABLED=true` and `SSO_SERVICE_GRPC_ADDRESS` is unset
- **THEN** the process exits with "cannot enable BYO-SSO without SSO_SERVICE_GRPC_ADDRESS"

#### Scenario: Enabled together with OIDC-WebAuthn sequencing
- **WHEN** `BYOSSO_ENABLED=true` and `OIDC_WEBAUTHN_SEQUENCING_ENABLED=true`
- **THEN** the process exits with "cannot enable BYO-SSO with OIDC_WEBAUTHN_SEQUENCING_ENABLED"

### Requirement: A wiring fault stops the process
`web.NewRouter` SHALL return an error when the extension is enabled and cannot be built. `serve` SHALL exit on that error. The extension cannot be built in each of these cases:

- Multi-tenancy has no working tenant resolver.
- The gRPC client of the tenant service or of the SSO service is missing.
- `BASE_URL` does not parse.

`web.NewRouter` MUST NOT fall back to a router without the extension. Multi-tenancy on its own keeps its existing fallback. When its client is missing, multi-tenancy falls back, with a warning, to a resolver that does nothing.

#### Scenario: Enabled, but the client of the tenant service was not built
- **WHEN** the router is built with the extension enabled and no client of the tenant service
- **THEN** `NewRouter` returns an error and no router
- **AND** the same options with the extension disabled build a router


## MODIFIED Requirements

### Requirement: The backend is unchanged with the flag off
With `BYOSSO_ENABLED` false, the backend SHALL behave as before this change:

- The handlers of `pkg/kratos` and `pkg/extra` run with the extension that does nothing. No hook answers a request or changes a flow. The tenant selection of the handler and Hydra's `skip` decide the login page with a session.
- Login UI does not register `GET /api/v0/sso/complete`.
- Login UI does not decorate the Kratos service. The handlers use the second factor policy of `MFA_ENABLED`.
- Every tenant lookup goes through `LookupTenants` of the tenant service, as before. That lookup lists no invitations and no auto-join candidates. Login UI looks up the tenants of a session by identity id, and marks no tenant as an invitation.
- Login UI dials the channel to the tenant service with the same options as before: no retry policy, and the default reconnect backoff. The dial helper only moved to `internal/grpc`.
- Login UI opens no channel to the SSO service, and does not validate the settings of the extension.
- The Hydra login accept and the consent are what they were.

#### Scenario: Sign-in to an app with the flag off
- **WHEN** a user signs in to an app on a deployment with multi-tenancy and `BYOSSO_ENABLED=false`
- **THEN** every request and response of the backend is what it was before this change

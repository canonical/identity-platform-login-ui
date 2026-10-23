# byosso-disabled Specification

## Purpose

Most deployments will run this code with the extension off. For those deployments, this change must make no difference. The frontend is one bundle for every deployment, so what the frontend gained must stay out of sight there. This spec states both guarantees. A difference that a user of such a deployment can notice is a defect, unless it is the one named here.

## Requirements
### Requirement: The backend is unchanged with the flag off
With `BYOSSO_ENABLED` false, the backend SHALL behave as before this change:

- The handlers of `pkg/kratos` and `pkg/extra` run with the extension that does nothing. No hook answers a request or changes a flow. The tenant selection of the handler and Hydra's `skip` decide the login page with a session.
- Login UI does not register `GET /api/v0/sso/complete`.
- Login UI does not decorate the Kratos service, and the MFA flag of the handlers is `MFA_ENABLED`.
- Every tenant lookup goes through `LookupTenants` of the tenant service, as before. That lookup lists no invitations and no auto-join candidates. Login UI looks up the tenants of a session by identity id, and marks no tenant as an invitation.
- Login UI dials the channel to the tenant service with the same options as before: no retry policy, and the default reconnect backoff. The dial helper only moved to `internal/grpc`.
- Login UI opens no channel to the SSO service, and does not validate the settings of the extension.
- The Hydra login accept and the consent are what they were.

#### Scenario: Sign-in to an app with the flag off
- **WHEN** a user signs in to an app on a deployment with multi-tenancy and `BYOSSO_ENABLED=false`
- **THEN** every request and response of the backend is what it was before this change

### Requirement: The frontend shows nothing of the feature with the flag off
Everything that the frontend gained SHALL appear only in answer to what the extension sends, and thus never with the flag off. The frontend gained these things:

- the tenant list
- the company sign-in buttons
- "Redirecting to …"
- errors shown in place
- the error page for an error of Kratos that the extension passes on
- the "Company sign-ins" section
- the invitation mark

One handler is hardened for every deployment. A submission on the login page SHALL go to the error page when the backend answers it with both of these:

- status 400
- a body that is not a JSON object

Before this change, the handler threw an exception on such a body.

#### Scenario: Login page with the flag off
- **WHEN** a user opens the login page of a deployment with `BYOSSO_ENABLED=false`
- **THEN** the page shows no tenant list and no company sign-in button
- **AND** the page follows at once a redirect that the backend answers with, as before

#### Scenario: Connected accounts with the flag off
- **WHEN** a user opens the Connected accounts page of a deployment with `BYOSSO_ENABLED=false`
- **THEN** the page has no "Company sign-ins" section

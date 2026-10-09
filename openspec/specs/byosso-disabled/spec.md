# byosso-disabled Specification

## Purpose

Most deployments will run this code with the extension off, and for them the change must be a non-event. The frontend is one bundle for every deployment, so what it gained has to stay out of sight there. This spec states both guarantees: a difference a user of such a deployment can notice, other than the one named here, is a defect.

## Requirements
### Requirement: The backend is unchanged with the flag off
With `BYOSSO_ENABLED` false, the backend SHALL behave as before this change:

- the handlers of `pkg/kratos` and `pkg/extra` run with the extension that does nothing: no hook answers a request or changes a flow, and the login page with a session is decided by the tenant selection of the handler and Hydra's `skip`;
- `GET /api/v0/sso/complete` is not registered;
- the Kratos service is not decorated, and the MFA flag of the handlers is `MFA_ENABLED`;
- every tenant lookup goes through `LookupTenants` of the tenant service, as before: it lists no invitations or auto-join candidates, a session's tenants are looked up by identity id, and no tenant is marked as an invitation;
- the channel to the tenant service is dialled with the same options as before (the dial helper only moved to `internal/grpc`): no service token, no retry policy, the default reconnect backoff;
- no channel to the SSO service is opened, and the settings of the extension are not validated;
- the Hydra login accept and the consent are what they were.

#### Scenario: Sign-in to an app with the flag off
- **WHEN** a user signs in to an app on a deployment with multi-tenancy and `BYOSSO_ENABLED=false`
- **THEN** every request and response of the backend is what it was before this change

### Requirement: The frontend shows nothing of the feature with the flag off
Everything the frontend gained (the tenant list, company sign-in buttons, "Redirecting to …", errors shown in place, the error page for an error of Kratos the extension passes on, the "Company sign-ins" section, the invitation mark) SHALL appear only in answer to what the extension sends, and so never with the flag off.

One handler is hardened for every deployment: a submission on the login page that the backend answers with status 400 and a body that is not a JSON object SHALL go to the error page. Before, the handler threw on such a body.

#### Scenario: Login page with the flag off
- **WHEN** a user opens the login page of a deployment with `BYOSSO_ENABLED=false`
- **THEN** the page shows no tenant list and no company sign-in button, and a redirect the backend answers with is followed at once, as before

#### Scenario: Connected accounts with the flag off
- **WHEN** a user opens the Connected accounts page of a deployment with `BYOSSO_ENABLED=false`
- **THEN** the page has no "Company sign-ins" section

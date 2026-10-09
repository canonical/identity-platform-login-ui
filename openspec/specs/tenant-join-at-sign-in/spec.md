# tenant-join-at-sign-in Specification

## Purpose

A pending invitation, or a tenant's auto-join for the domain of an address, gives a user the right to become a member; neither makes them one. The membership is created by the sign-in that passes the tenant's checks, so nobody is a member of a tenant that requires company sign-in without having gone through it.

## Requirements
### Requirement: The membership is created just before the accept
The tenant service's answer for a tenant and an address or account (`GetSignInContext`) says whether it is a member, and whether a pending invitation (`invitation_admits`) or auto-join (`auto_join_admits`) admits it. The extension SHALL treat an account that is admitted but not a member exactly as a member, on the sign-in screen and in the decision before the accept: the same enforcement, the same provenance rule, the same MFA.

When its session has passed every check, the extension SHALL call `JoinTenant` on the tenant service with `tenant_id` and the `identity_id` of the session's account, and accept the Hydra login only after it succeeds. `JoinTenant` MUST NOT be called before the checks pass, nor for a member. The tenant service decides again whether the address of that account is still admitted:

- reason `NOT_ADMITTED` (the invitation expired, or auto-join was switched off, since the check): `403`, error id `tenant_not_a_member`; nothing is accepted;
- the tenant service cannot be reached: `503`; nothing is accepted.

A registration's company sign-in with no app waiting has no accept to join at: its sign-in cookie carries a join mark, and `GET /api/v0/sso/complete` SHALL call `JoinTenant` once the attempt is confirmed and only when the session's address is verified (`403` for `NOT_ADMITTED`, `503` for any other failure). With the address not verified yet it joins nothing: the account joins at its first sign-in to the tenant from an app, where the address is verified first.

#### Scenario: Invited to a tenant that requires company sign-in, signed in with a password
- **WHEN** an account with a pending invitation to such a tenant reaches the checks with a password session
- **THEN** it is sent to the tenant's company sign-in, and no membership is created

#### Scenario: Invitation accepted by signing in
- **WHEN** an account with a pending invitation to a tenant with enforcement `off` signs in to it with its password and MFA
- **THEN** `JoinTenant` is called, and the login is accepted for that tenant

#### Scenario: Auto-join switched off in between
- **WHEN** the check found the address admitted and `JoinTenant` answers `NOT_ADMITTED`
- **THEN** the page shows "This account is not a member of that tenant", and no login is accepted

### Requirement: Invitations in the tenant list
A tenant the lookup marks as a pending invitation (`invited`) SHALL be labelled "<name> — invitation" wherever the user picks a tenant: in the list on the login page, and on `/ui/select_tenant` (where `GET /api/v0/tenants` returns `"invited": true` for it). Picking it and signing in is how the invitation is accepted. An account with no membership whose invitations are pending MUST NOT get a personal tenant at sign-in: the inviting tenants are offered instead.

#### Scenario: Existing account with a pending invitation
- **WHEN** a user with a personal tenant and a pending invitation to tenant T enters their email
- **THEN** the list shows the personal tenant and "T — invitation"


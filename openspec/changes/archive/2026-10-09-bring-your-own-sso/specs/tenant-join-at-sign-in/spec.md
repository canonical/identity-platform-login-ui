## Purpose

A pending invitation gives a user the right to become a member. A tenant's auto-join for the domain of an address gives the same right. Neither makes the user a member. The sign-in that passes the checks of the tenant creates the membership. So nobody is a member of a tenant that requires company sign-in without a company sign-in.

## ADDED Requirements

### Requirement: The membership is created just before the accept
The tenant service answers for a tenant and an address or an account (`GetSignInContext`). The answer says if the address or account is a member. It also says if a pending invitation (`invitation_admits`) or auto-join (`auto_join_admits`) admits it. The extension SHALL treat an account that is admitted but not a member exactly as a member. This applies on the sign-in screen and in the decision before the accept. The extension SHALL apply the same enforcement, the same provenance rule and the same MFA.

When the session of such an account has passed every check, the extension SHALL call `JoinTenant` on the tenant service. The call SHALL carry `tenant_id` and the `identity_id` of the session's account. The extension SHALL accept the Hydra login only after `JoinTenant` succeeds. The extension MUST NOT call `JoinTenant` before the checks pass. It MUST NOT call `JoinTenant` for a member.

The tenant service decides again if the address of that account is still admitted. These cases can occur:

- The reason is `NOT_ADMITTED`. The invitation expired, or auto-join was switched off, after the check. The answer is `403` with the error id `tenant_not_a_member`. The extension accepts nothing.
- The extension cannot reach the tenant service. The answer is `503`. The extension accepts nothing.

A registration's company sign-in with no app that waits has no accept at which to join. The sign-in cookie of such a sign-in carries a join mark. For such a sign-in, `GET /api/v0/sso/complete` SHALL call `JoinTenant` after the attempt is confirmed and only when the session's address is verified. The answer is `403` for `NOT_ADMITTED` and `503` for any other failure.

If the address is not verified yet, the extension joins nothing. The account joins at its first sign-in to the tenant from an app. At that sign-in the address is verified first.

#### Scenario: Invited to a tenant that requires company sign-in, signed in with a password
- **WHEN** an account with a pending invitation to such a tenant reaches the checks with a password session
- **THEN** the extension sends the account to the company sign-in of the tenant
- **AND** the extension creates no membership

#### Scenario: Invitation accepted by signing in
- **WHEN** an account with a pending invitation to a tenant with enforcement `off` signs in to that tenant
- **AND** the account uses its password and MFA
- **THEN** the extension calls `JoinTenant`
- **AND** the extension accepts the login for that tenant

#### Scenario: Auto-join switched off in between
- **WHEN** the check found the address admitted and `JoinTenant` answers `NOT_ADMITTED`
- **THEN** the page shows "This account is not a member of that tenant"
- **AND** the extension accepts no login

### Requirement: Invitations in the tenant list
A tenant that the lookup marks as a pending invitation (`invited`) SHALL be labelled "<name> — invitation" wherever the user picks a tenant. This applies to the list on the login page. It also applies to `/ui/select_tenant`, where `GET /api/v0/tenants` returns `"invited": true` for the tenant. The user accepts the invitation when they pick the tenant and sign in.

An account with no membership whose invitations are pending MUST NOT get a personal tenant at sign-in. Login UI offers the tenants that invited the account instead.

#### Scenario: Existing account with a pending invitation
- **WHEN** a user with a personal tenant and a pending invitation to tenant T enters their email
- **THEN** the list shows the personal tenant and "T — invitation"

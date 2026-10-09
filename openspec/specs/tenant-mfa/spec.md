# tenant-mfa Specification

## Purpose

Whether a sign-in needs MFA depends on the tenant, and Kratos can only require MFA for the whole deployment. With the extension enabled, Kratos is configured to hand out `aal1` sessions, and Login UI decides for the tenant of each app sign-in whether the session has done enough. A portal password alone is never enough, at any tenant. This has no setting of its own: it comes with `BYOSSO_ENABLED`, and replaces the rule of `MFA_ENABLED`.

## Requirements
### Requirement: Who owes MFA
The decision before a Hydra login accept (`tenant-sign-in-checks`) SHALL require MFA of a session according to the tenant's MFA policy from the tenant service (`mfa_requirement`: `none` or `required`):

- a session whose first factor is `oidc` (a company sign-in or a public provider), or to which a company sign-in was added by account linking: only when the policy is `required`;
- every other session (portal password, passkey, code): always.

MFA done at the customer's identity provider MUST NOT count: Login UI cannot see it.

#### Scenario: Company sign-in, policy none
- **WHEN** an account that has an authenticator app signs in through the company sign-in of a tenant whose policy is `none`
- **THEN** no code is asked

#### Scenario: Company sign-in, policy required
- **WHEN** the same account signs in through the company sign-in of a tenant whose policy is `required`
- **THEN** the login is not accepted until MFA is done in this session

#### Scenario: Password, policy none
- **WHEN** a session from a portal password is checked against a tenant whose policy is `none`
- **THEN** MFA is still required

### Requirement: What counts as MFA done
MFA SHALL count as done only when the session itself lists an authentication method after its first one, at `aal2`, whose method is `totp`, `webauthn` or `lookup_secret`. Holding a second factor is not enough: it has to have been used, or set up, in this session (Kratos adds a factor to the session at `aal2` when the user sets it up). MFA done once in a session SHALL count for every tenant the session is offered to.

#### Scenario: MFA done earlier in the same session
- **WHEN** a session that entered an authenticator code for one app is checked for another app at a tenant that requires MFA
- **THEN** no code is asked again

### Requirement: Asking for MFA
When MFA is owed, the extension SHALL look for a second factor of the account (a WebAuthn credential, an authenticator app, or recovery codes; `500` when the look-up fails) and answer:

- with one: `{"error": {"id": "session_aal2_required"}, "redirect_to": "/ui/login?aal=aal2&return_to=<login page of the login challenge>"}`, with `refresh=true` added for a session that is already at `aal2` without a counting method;
- with none: `{"error": {"id": "totp_registration_required"}, "redirect_to": "/ui/setup_secure?return_to=<login page of the login challenge>"}`. The first set-up trusts the first factor, as the platform-wide rule does.

Either way the browser comes back to the login page of the login challenge, where the checks run again; only the second factor is asked, not the first. The extension MUST NOT amend, or answer picks on, a login flow whose requested level is `aal2`.

#### Scenario: First sign-in to a tenant that requires MFA
- **WHEN** the account has no second factor
- **THEN** the browser is sent to set up an authenticator app, and on return the login is accepted without another code

#### Scenario: Lost authenticator, recovery codes kept
- **WHEN** the account holds recovery codes
- **THEN** the user is sent to the second-factor step, where a recovery code is accepted, and not to set up a new authenticator on the first factor alone

### Requirement: The tenant's policy replaces the platform-wide rule
With the extension enabled, `pkg/web` SHALL build the `pkg/kratos` and `pkg/extra` APIs with their MFA flag off, whatever `MFA_ENABLED` says. In the handlers this switches off what the tenant checks decide instead: the redirect to set up an authenticator after a password sign-in, and the `aal2` the consent handler requires of a password session. The redirect to regenerate backup codes SHALL stay: `pkg/web` passes `pkg/kratos` the option `WithBackupCodesRegeneration`, so that a user who signs in with a backup code and is left with three or fewer is asked whether to make new ones before the login is accepted, as with `MFA_ENABLED`. Without the option the redirect follows `MFA_ENABLED` as before.

The tenant checks run only where a Hydra login is accepted: after a sign-in with no login challenge (the portal's own pages), an account with no second factor is not sent to set one up, whatever `MFA_ENABLED` says.

Kratos MUST be configured with `session.whoami.required_aal: aal1`: left at `highest_available`, it refuses the `aal1` session of an account that has a second factor before Login UI can apply the tenant's policy to it. `OIDC_WEBAUTHN_SEQUENCING_ENABLED` cannot be on with the extension (`byosso-configuration`).

#### Scenario: `MFA_ENABLED` with the extension on
- **WHEN** `BYOSSO_ENABLED` is true, whatever `MFA_ENABLED` says
- **THEN** whether a sign-in to an app needs MFA is decided by the tenant checks alone

#### Scenario: One of the last backup codes
- **WHEN** a user answers the tenant's MFA with a backup code that leaves them three unused
- **THEN** the browser goes to the page that offers new backup codes, and from there, with or without new codes, back to the login, which is accepted


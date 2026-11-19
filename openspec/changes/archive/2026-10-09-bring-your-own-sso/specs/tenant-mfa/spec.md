## Purpose

The need for MFA at a sign-in depends on the tenant. Kratos can require MFA only for the whole deployment. With the extension enabled, Kratos is configured to hand out `aal1` sessions. Login UI decides, for the tenant of each app sign-in, if the session has done enough.

A portal password alone is never enough, at any tenant. This has no setting of its own. It comes with `BYOSSO_ENABLED`, and it replaces the rule of `MFA_ENABLED`.

## ADDED Requirements

### Requirement: Who owes MFA
The decision before a Hydra login accept (`tenant-sign-in-checks`) SHALL require MFA of a session according to the MFA policy of the tenant. The policy comes from the Tenant Service (`mfa_requirement`: `none` or `required`). The decision SHALL require MFA as follows:

- For a session whose first factor is `oidc` (a company sign-in or a public provider), or to which account linking added a company sign-in: only when the policy is `required`.
- For every other session (portal password, passkey, code): always.

MFA done at the identity provider of the customer MUST NOT count, because Login UI cannot see it.

#### Scenario: Company sign-in, policy none
- **WHEN** an account that has an authenticator app signs in through the company sign-in of a tenant whose policy is `none`
- **THEN** Login UI asks for no code

#### Scenario: Company sign-in, policy required
- **WHEN** the same account signs in through the company sign-in of a tenant whose policy is `required`
- **THEN** Login UI does not accept the login until MFA is done in this session

#### Scenario: Password, policy none
- **WHEN** Login UI checks a session from a portal password against a tenant whose policy is `none`
- **THEN** MFA is still required

### Requirement: What counts as MFA done
MFA SHALL count as done only when the session itself lists an authentication method that meets all of these conditions:

- the method comes after the first method of the session
- the method is at `aal2`
- the method is `totp`, `webauthn` or `lookup_secret`

It is not enough to hold a second factor. The user must use it, or set it up, in this session. Kratos adds a factor to the session at `aal2` when the user sets it up.

MFA that the user did once in a session SHALL count for every tenant that the session goes to.

#### Scenario: MFA done earlier in the same session
- **WHEN** a session that entered an authenticator code for one app is checked for another app
- **AND** this check is at a tenant that requires MFA
- **THEN** Login UI asks for no code again

### Requirement: Asking for MFA
When MFA is owed, the extension SHALL look for a second factor of the account. A second factor is a WebAuthn credential, an authenticator app, or recovery codes. If the look-up fails, the answer is `500`. The extension SHALL then answer according to the result.

If the account has a second factor, the answer is `{"error": {"id": "session_aal2_required"}, "redirect_to": "/ui/login?aal=aal2&return_to=<login page of the login challenge>"}`. The extension adds `refresh=true` for a session that is already at `aal2` without a method that counts.

If the account has no second factor, the answer is `{"error": {"id": "totp_registration_required"}, "redirect_to": "/ui/setup_secure?return_to=<login page of the login challenge>"}`. The first set-up trusts the first factor, as the platform-wide rule does.

In both cases the browser comes back to the login page of the login challenge, and the checks run again. Login UI asks only for the second factor, not for the first factor. The extension MUST NOT amend a login flow whose requested level is `aal2`. The extension MUST NOT answer picks on such a flow.

#### Scenario: First sign-in to a tenant that requires MFA
- **WHEN** the account has no second factor
- **THEN** Login UI sends the browser to set up an authenticator app
- **AND** on return Login UI accepts the login without another code

#### Scenario: Lost authenticator, recovery codes kept
- **WHEN** the account holds recovery codes
- **THEN** Login UI sends the user to the second-factor step, where Login UI accepts a recovery code
- **AND** Login UI does not send the user to set up a new authenticator on the first factor alone

### Requirement: The tenant's policy replaces the platform-wide rule
With the extension enabled, `pkg/web` SHALL build the `pkg/kratos` and `pkg/extra` APIs with their MFA flag off, whatever `MFA_ENABLED` says. In the handlers this switches off two things. The tenant checks decide them instead:

- the redirect to set up an authenticator after a password sign-in
- the `aal2` that the consent handler requires of a password session

The redirect to regenerate backup codes SHALL stay. `pkg/web` passes `pkg/kratos` the option `WithBackupCodesRegeneration`. With this option, Login UI asks a user if the user wants new codes before it accepts the login, as with `MFA_ENABLED`. Login UI asks this when the user signs in with a backup code and has 3 or fewer codes left. Without the option, the redirect follows `MFA_ENABLED` as before.

The tenant checks run only where Login UI accepts a Hydra login. After a sign-in with no login challenge (the pages of the portal itself), Login UI does not send an account with no second factor to set one up. This is true whatever `MFA_ENABLED` says.

Kratos MUST be configured with `session.whoami.required_aal: aal1`. If it stays at `highest_available`, Kratos refuses the `aal1` session of an account that has a second factor. This happens before Login UI can apply the policy of the tenant to the session. `OIDC_WEBAUTHN_SEQUENCING_ENABLED` cannot be on with the extension (`byosso-configuration`).

#### Scenario: `MFA_ENABLED` with the extension on
- **WHEN** `BYOSSO_ENABLED` is true, whatever `MFA_ENABLED` says
- **THEN** the tenant checks alone decide if a sign-in to an app needs MFA

#### Scenario: One of the last backup codes
- **WHEN** a user answers the MFA of the tenant with a backup code that leaves 3 unused codes
- **THEN** the browser goes to the page that offers new backup codes
- **AND** from there, with or without new codes, the browser goes back to the login
- **AND** Login UI accepts the login

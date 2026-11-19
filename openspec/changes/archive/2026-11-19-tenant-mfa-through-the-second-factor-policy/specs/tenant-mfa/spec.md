## MODIFIED Requirements

### Requirement: The tenant's policy replaces the platform-wide rule
With the extension enabled, `pkg/web` SHALL give the `pkg/kratos` and `pkg/extra` APIs the second factor policy of the extension (`byosso.SecondFactorPolicy`). It uses the option `WithSecondFactorPolicy` of each API. This policy replaces the policy of `MFA_ENABLED`, whatever `MFA_ENABLED` says. For every sign-in, it asks for no second factor and for no setup of one. Thus the handlers do not do these two things. The tenant checks decide them instead:

- the redirect to set up an authenticator after a password sign-in
- the `aal2` that the consent handler requires of a password session

The redirect to regenerate backup codes SHALL stay. The policy keeps this redirect for every sign-in. Thus Login UI asks a user if the user wants new codes before it accepts the login, as with `MFA_ENABLED`. Login UI asks this when the user signs in with a backup code and has 3 or fewer codes left.

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

# second-factor-policy Specification

## Purpose

Before a sign-in completes, the Login UI may ask more of the user than Kratos did: an authenticator app set up, a WebAuthn key set up after signing in through an external provider, backup codes regenerated when few are left, and a second factor before consent is given. Which of these applies is the platform's second factor policy, set by `MFA_ENABLED` and `OIDC_WEBAUTHN_SEQUENCING_ENABLED`.

This capability puts that policy in one object which the login and consent handlers ask, so that it can be read and tested in one place instead of in each handler. The requirements below describe the policy and how the handlers apply it. They are the behaviour the Login UI had before the policy object existed, and the policy MUST preserve it.

## Requirements
### Requirement: One policy decides what a sign-in needs
The Login UI SHALL obtain what a sign-in still needs from one policy, `SecondFactorPolicyInterface.For`. The policy SHALL receive the authentication methods the session completed, in order, and SHALL return whether a second factor is required, which second factor method the user must have set up, if any, and whether backup codes are to be regenerated. `handleCreateFlow`, `handleUpdateFlow` and `handleConsent` SHALL NOT read `MFA_ENABLED` or `OIDC_WEBAUTHN_SEQUENCING_ENABLED`. `kratos.NewAPI` and `extra.NewAPI` SHALL build the platform policy from those two settings, and SHALL use the policy given with `WithSecondFactorPolicy` in its place when one is given.

#### Scenario: Platform policy by default
- **WHEN** an API is built without `WithSecondFactorPolicy`
- **THEN** its handlers follow the platform policy for the `MFA_ENABLED` and `OIDC_WEBAUTHN_SEQUENCING_ENABLED` values given to the constructor

#### Scenario: Another policy is given
- **WHEN** an API is built with `WithSecondFactorPolicy` and both settings are false
- **THEN** its handlers follow what the given policy asks, and send a user without an authenticator app to set one up when that policy asks for `totp`

#### Scenario: Asked once for a login request
- **WHEN** a login flow is created or updated for a Kratos session
- **THEN** the policy is asked once, after the email verification check and before any lookup of the user's credentials

#### Scenario: No session after a login flow update
- **WHEN** a login flow is updated and Kratos returns no session
- **THEN** none of the user's credentials is looked up, whatever the policy asks

#### Scenario: The tenant resolver defers the MFA checks
- **WHEN** a login flow is created for a Kratos session and the tenant resolver defers the MFA checks
- **THEN** the policy is not asked and none of the user's credentials is looked up

### Requirement: The platform policy asks for a second factor by the first method
The platform policy SHALL ask for a second factor when the session's first method is `oidc` and `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is true, and when the session's first method is `password` or `webauthn` and `MFA_ENABLED` is true. It MUST NOT ask for a second factor for any other first method, whatever the settings are and whatever the session completed afterwards.

#### Scenario: Password with MFA
- **WHEN** `MFA_ENABLED` is true and the first method is `password` or `webauthn`
- **THEN** a second factor is required

#### Scenario: External provider with MFA only
- **WHEN** `MFA_ENABLED` is true, `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is false and the first method is `oidc`
- **THEN** no second factor is required

#### Scenario: External provider with sequencing
- **WHEN** `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is true and the first method is `oidc`
- **THEN** a second factor is required

#### Scenario: Password with sequencing only
- **WHEN** `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is true, `MFA_ENABLED` is false and the first method is `password` or `webauthn`
- **THEN** no second factor is required

#### Scenario: Another first method
- **WHEN** both settings are true and the first method is `passkey`, `code_recovery` or missing
- **THEN** no second factor is required

#### Scenario: The first method decides
- **WHEN** `MFA_ENABLED` is true, `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is false and the session completed `password` and then `oidc`
- **THEN** a second factor is required

### Requirement: The platform policy asks for the second factor method to set up
The platform policy SHALL ask for a WebAuthn key (`webauthn`) to be set up when one of the methods the session completed is `oidc` and `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is true. It SHALL ask for an authenticator app (`totp`) to be set up when none of the methods the session completed is `oidc` and `MFA_ENABLED` is true. It MUST NOT ask for anything to be set up otherwise.

#### Scenario: Password with MFA
- **WHEN** `MFA_ENABLED` is true and the session completed `password`
- **THEN** `totp` is asked for, whatever `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is

#### Scenario: External provider with sequencing
- **WHEN** `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is true and the session completed `oidc`
- **THEN** `webauthn` is asked for, whatever `MFA_ENABLED` is

#### Scenario: External provider with MFA only
- **WHEN** `MFA_ENABLED` is true, `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is false and the session completed `oidc`
- **THEN** nothing is asked to be set up

#### Scenario: Password with sequencing only
- **WHEN** `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is true, `MFA_ENABLED` is false and the session completed `password`
- **THEN** nothing is asked to be set up

#### Scenario: External provider after another method
- **WHEN** both settings are true and the session completed `password` and then `oidc`
- **THEN** `webauthn` is asked for

#### Scenario: A method that is neither a password nor an external provider
- **WHEN** `MFA_ENABLED` is true and the session completed `passkey` or `code_recovery`, or lists no method
- **THEN** `totp` is asked for

### Requirement: The platform policy asks for backup codes to be regenerated
The platform policy SHALL ask for backup codes to be regenerated when `MFA_ENABLED` is true, for every sign-in, and MUST NOT ask for it when `MFA_ENABLED` is false.

#### Scenario: MFA enabled
- **WHEN** `MFA_ENABLED` is true
- **THEN** backup codes are to be regenerated, whatever the first method is

#### Scenario: Sequencing only
- **WHEN** `MFA_ENABLED` is false and `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is true
- **THEN** backup codes are not to be regenerated

### Requirement: Creating a login flow for a session sets up what is missing
When `GET /api/kratos/self-service/login/browser` finds a Kratos session, the user's email does not have to be verified first and the tenant resolver does not defer the MFA checks, the Login UI SHALL check the method the policy asks to be set up before it continues with the Hydra login. When `totp` is asked for, it SHALL look up once whether the user has a TOTP credential, and SHALL answer a user without one with error id `totp_registration_required` and `redirect_to` `/ui/setup_secure?return_to=<return URL>`, and record the authenticator set-up in the state cookie. When `webauthn` is asked for, it SHALL look up once whether the user has a WebAuthn key that is not a passwordless one, and SHALL answer a user without one with error id `webauthn_registration_required` and `redirect_to` `/ui/setup_passkey?return_to=<return URL>`, and record the key set-up in the state cookie. The Login UI MUST NOT look up a method the policy does not ask for, and MUST NOT look up the user's backup codes in this handler. The return URL is the request's `return_to` or, without one, the login page URL that carries the login challenge.

#### Scenario: Authenticator app missing
- **WHEN** the policy asks for `totp` and the user has no TOTP credential
- **THEN** the response is 200 with error id `totp_registration_required` and `redirect_to` `/ui/setup_secure?return_to=<return URL>`, the state cookie records the authenticator set-up, and the Hydra login is not accepted

#### Scenario: WebAuthn key missing
- **WHEN** the policy asks for `webauthn` and the user has no WebAuthn key
- **THEN** the response is 200 with error id `webauthn_registration_required` and `redirect_to` `/ui/setup_passkey?return_to=<return URL>`, the state cookie records the key set-up, and the Hydra login is not accepted

#### Scenario: Nothing missing
- **WHEN** the user has the method the policy asks for, or the policy asks for none
- **THEN** the login flow continues as it does without MFA

#### Scenario: Email verification comes first
- **WHEN** email verification is enabled and the user's email is not verified
- **THEN** the user is sent to `/ui/verification` and none of the user's credentials is looked up

#### Scenario: A lookup fails
- **WHEN** the TOTP lookup or the WebAuthn lookup fails
- **THEN** the response is 500 with body `failed to check MFA status` or `failed to check WebAuthn status`, and the Hydra login is not accepted

### Requirement: Updating a login flow sets up an authenticator app and regenerates backup codes
After `POST /api/kratos/self-service/login` has updated the Kratos flow and Kratos returns a session, the Login UI SHALL look up once whether the user has a TOTP credential when the policy asks for `totp`. It SHALL then look up once how many unused backup codes the user has left when the policy asks for backup codes to be regenerated and the session's second method is `lookup_secret`. Both lookups SHALL be made before the response is chosen. A user who must verify their email SHALL be sent to `/ui/verification` first. Otherwise a user with too few backup codes left SHALL be answered with error id `regenerate_backup_codes` and `redirect_to` `/ui/backup_codes_regenerate?flow=<flow id>&return_to=<return URL>`, with the use of a backup code recorded in the state cookie. Otherwise a user without a TOTP credential SHALL be answered with error id `totp_registration_required` and `redirect_to` `/ui/setup_secure?return_to=<return URL>`, with the authenticator set-up recorded in the state cookie. The Login UI MUST NOT look up the user's WebAuthn key in this handler, whatever the policy asks for. The return URL is the login page URL that carries the login challenge for a Hydra login, and the Kratos flow's `return_to` otherwise.

#### Scenario: Authenticator app missing
- **WHEN** the policy asks for `totp` and the user has no TOTP credential
- **THEN** the response is 200 with error id `totp_registration_required` and `redirect_to` `/ui/setup_secure?return_to=<return URL>`, and the state cookie records the authenticator set-up

#### Scenario: Few backup codes left
- **WHEN** the policy asks for backup codes to be regenerated, the session's second method is `lookup_secret` and the user has too few unused codes left
- **THEN** the response is 200 with error id `regenerate_backup_codes` and `redirect_to` `/ui/backup_codes_regenerate?flow=<flow id>&return_to=<return URL>`, and the state cookie records the use of a backup code

#### Scenario: Backup codes before the authenticator app
- **WHEN** the user has too few backup codes left and no TOTP credential
- **THEN** both lookups are made and the user is sent to regenerate the backup codes

#### Scenario: No backup code was used
- **WHEN** the policy asks for backup codes to be regenerated and the session's second method is not `lookup_secret`
- **THEN** the backup codes are not looked up

#### Scenario: WebAuthn key asked for
- **WHEN** the policy asks for `webauthn`
- **THEN** the WebAuthn key is not looked up and the response follows Kratos

#### Scenario: Email verification comes first
- **WHEN** email verification is enabled, the user's email is not verified, and the user lacks an authenticator app and has too few backup codes left
- **THEN** both lookups are made and the user is sent to `/ui/verification`

#### Scenario: A lookup fails
- **WHEN** the TOTP lookup or the backup codes lookup fails
- **THEN** the response is 500 with body `internal server error`

### Requirement: Consent requires the assurance level the policy asks for
`GET /api/consent` SHALL require a session at `aal2` when the policy asks for a second factor, and at `aal1` when it does not. It SHALL answer a session below the required level with 403 and body `insufficient session aal`, before the consent request is fetched from Hydra.

#### Scenario: Second factor required and not done
- **WHEN** the policy asks for a second factor and the session is at `aal1`
- **THEN** the response is 403 with body `insufficient session aal` and Hydra is not called

#### Scenario: Second factor required and done
- **WHEN** the policy asks for a second factor and the session is at `aal2`
- **THEN** the consent request is fetched from Hydra and accepted

#### Scenario: No second factor required
- **WHEN** the policy asks for no second factor and the session is at `aal1`
- **THEN** the consent request is fetched from Hydra and accepted

#### Scenario: Session without an assurance level
- **WHEN** the session carries no assurance level
- **THEN** the response is 403 with body `insufficient session aal`, whatever the policy asks for

#### Scenario: No session
- **WHEN** the session check returns no session and no error
- **THEN** the response is 403 with body `insufficient session aal` and Hydra is not called


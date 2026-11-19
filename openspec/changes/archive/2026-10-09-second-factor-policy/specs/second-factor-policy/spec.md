## Purpose

Before a sign-in completes, the Login UI can ask more of the user than Kratos did. It can ask for these things:

- an authenticator app set up
- a WebAuthn key set up after the user signs in through an external provider
- backup codes regenerated when few are left
- a second factor before consent is given

The platform's second factor policy decides which of these applies. The settings `MFA_ENABLED` and `OIDC_WEBAUTHN_SEQUENCING_ENABLED` set this policy.

This capability puts the policy in one object. The login handlers and the consent handler ask this object. The policy can then be read and tested in one place and not in each handler. The requirements below describe the policy and how the handlers apply it. They are the behaviour that the Login UI had before the policy object existed. The policy MUST keep this behaviour.

## ADDED Requirements

### Requirement: One policy decides what a sign-in needs
The Login UI SHALL get what a sign-in still needs from one policy, `SecondFactorPolicyInterface.For`. The policy SHALL receive the authentication methods that the session completed, in order. The policy SHALL return these answers:

- if a second factor is required
- which second factor method the user must have set up, if any
- if backup codes are to be regenerated

`handleCreateFlow`, `handleUpdateFlow` and `handleConsent` SHALL NOT read `MFA_ENABLED` or `OIDC_WEBAUTHN_SEQUENCING_ENABLED`. `kratos.NewAPI` and `extra.NewAPI` SHALL build the platform policy from those settings. When a caller gives a policy with `WithSecondFactorPolicy`, they SHALL use that policy in its place.

#### Scenario: Platform policy by default
- **WHEN** a caller builds an API without `WithSecondFactorPolicy`
- **THEN** its handlers follow the platform policy
- **AND** the platform policy uses the `MFA_ENABLED` and `OIDC_WEBAUTHN_SEQUENCING_ENABLED` values that the caller gave to the constructor

#### Scenario: Another policy is given
- **WHEN** a caller builds an API with `WithSecondFactorPolicy` and both settings are false
- **THEN** its handlers follow what the given policy asks
- **AND** the handlers send a user without an authenticator app to set one up when that policy asks for `totp`

#### Scenario: Asked once for a login request
- **WHEN** the Login UI creates or updates a login flow for a Kratos session
- **THEN** the Login UI asks the policy once
- **AND** it asks after the email verification check and before any lookup of the user's credentials

#### Scenario: No session after a login flow update
- **WHEN** the Login UI updates a login flow and Kratos returns no session
- **THEN** the Login UI looks up none of the user's credentials, whatever the policy asks

#### Scenario: The tenant resolver defers the MFA checks
- **WHEN** the Login UI creates a login flow for a Kratos session and the tenant resolver defers the MFA checks
- **THEN** the Login UI does not ask the policy
- **AND** it looks up none of the user's credentials

### Requirement: The platform policy asks for a second factor by the first method
The platform policy SHALL ask for a second factor when `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is true and the first method of the session is `oidc`. It SHALL also ask for a second factor when `MFA_ENABLED` is true and the first method is `password` or `webauthn`. The policy MUST NOT ask for a second factor for any other first method. This is true for all settings and for all methods that the session completed afterwards.

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
The platform policy SHALL ask for a WebAuthn key (`webauthn`) to be set up in this case: `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is true and the session completed `oidc`. It SHALL ask for an authenticator app (`totp`) to be set up in this case: `MFA_ENABLED` is true and the session did not complete `oidc`. In all other cases the policy MUST NOT ask for anything to be set up.

#### Scenario: Password with MFA
- **WHEN** `MFA_ENABLED` is true and the session completed `password`
- **THEN** the policy asks for `totp`, whatever `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is

#### Scenario: External provider with sequencing
- **WHEN** `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is true and the session completed `oidc`
- **THEN** the policy asks for `webauthn`, whatever `MFA_ENABLED` is

#### Scenario: External provider with MFA only
- **WHEN** `MFA_ENABLED` is true, `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is false and the session completed `oidc`
- **THEN** the policy asks for nothing to be set up

#### Scenario: Password with sequencing only
- **WHEN** `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is true, `MFA_ENABLED` is false and the session completed `password`
- **THEN** the policy asks for nothing to be set up

#### Scenario: External provider after another method
- **WHEN** both settings are true and the session completed `password` and then `oidc`
- **THEN** the policy asks for `webauthn`

#### Scenario: A method that is neither a password nor an external provider
- **WHEN** `MFA_ENABLED` is true and the session completed `passkey` or `code_recovery`, or lists no method
- **THEN** the policy asks for `totp`

### Requirement: The platform policy asks for backup codes to be regenerated
The platform policy SHALL ask for backup codes to be regenerated for every sign-in when `MFA_ENABLED` is true. It MUST NOT ask for it when `MFA_ENABLED` is false.

#### Scenario: MFA enabled
- **WHEN** `MFA_ENABLED` is true
- **THEN** the policy asks for backup codes to be regenerated, whatever the first method is

#### Scenario: Sequencing only
- **WHEN** `MFA_ENABLED` is false and `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is true
- **THEN** the policy does not ask for backup codes to be regenerated

### Requirement: Creating a login flow for a session sets up what is missing
The Login UI SHALL check the method that the policy asks to be set up when all of these conditions are true:

- `GET /api/kratos/self-service/login/browser` finds a Kratos session
- the user's email does not have to be verified first
- the tenant resolver does not defer the MFA checks

The Login UI SHALL do this check before it continues with the Hydra login. When the policy asks for `totp`, the Login UI SHALL look up once if the user has a TOTP credential. It SHALL answer a user without one with error id `totp_registration_required` and `redirect_to` `/ui/setup_secure?return_to=<return URL>`. With that answer it SHALL record the authenticator set-up in the state cookie. When the policy asks for `webauthn`, it SHALL look up once if the user has a WebAuthn key that is not a passwordless one. It SHALL answer a user without such a key with error id `webauthn_registration_required` and `redirect_to` `/ui/setup_passkey?return_to=<return URL>`. With that answer it SHALL record the key set-up in the state cookie. The Login UI MUST NOT look up a method that the policy does not ask for. It MUST NOT look up the backup codes of the user in this handler. The return URL is the `return_to` of the request. If the request has no `return_to`, the return URL is the URL of the login page that carries the login challenge.

#### Scenario: Authenticator app missing
- **WHEN** the policy asks for `totp` and the user has no TOTP credential
- **THEN** the response is 200 with error id `totp_registration_required` and `redirect_to` `/ui/setup_secure?return_to=<return URL>`
- **AND** the state cookie records the authenticator set-up
- **AND** the Login UI does not accept the Hydra login

#### Scenario: WebAuthn key missing
- **WHEN** the policy asks for `webauthn` and the user has no WebAuthn key
- **THEN** the response is 200 with error id `webauthn_registration_required` and `redirect_to` `/ui/setup_passkey?return_to=<return URL>`
- **AND** the state cookie records the key set-up
- **AND** the Login UI does not accept the Hydra login

#### Scenario: Nothing missing
- **WHEN** the user has the method that the policy asks for, or the policy asks for none
- **THEN** the login flow continues as it does without MFA

#### Scenario: Email verification comes first
- **WHEN** email verification is enabled and the email of the user is not verified
- **THEN** the Login UI sends the user to `/ui/verification`
- **AND** it looks up none of the user's credentials

#### Scenario: A lookup fails
- **WHEN** the TOTP lookup or the WebAuthn lookup fails
- **THEN** the response is 500 with body `failed to check MFA status` or `failed to check WebAuthn status`
- **AND** the Login UI does not accept the Hydra login

### Requirement: Updating a login flow sets up an authenticator app and regenerates backup codes
After `POST /api/kratos/self-service/login` updates the Kratos flow and Kratos returns a session, the Login UI SHALL do these lookups in this order:

1. When the policy asks for `totp`, it SHALL look up once if the user has a TOTP credential.
2. It SHALL then look up once how many unused backup codes the user has left, when both of these conditions are true:
   - the policy asks for backup codes to be regenerated
   - the session's second method is `lookup_secret`

The Login UI SHALL make both lookups before it chooses the response. It SHALL send a user who must verify their email to `/ui/verification` first. Otherwise it SHALL answer a user with too few backup codes left with error id `regenerate_backup_codes` and `redirect_to` `/ui/backup_codes_regenerate?flow=<flow id>&return_to=<return URL>`. With that answer it SHALL record the use of a backup code in the state cookie. Otherwise it SHALL answer a user without a TOTP credential with error id `totp_registration_required` and `redirect_to` `/ui/setup_secure?return_to=<return URL>`. With that answer it SHALL record the authenticator set-up in the state cookie. The Login UI MUST NOT look up the WebAuthn key of the user in this handler, whatever the policy asks for. For a Hydra login, the return URL is the URL of the login page that carries the login challenge. In all other cases the return URL is the `return_to` of the Kratos flow.

#### Scenario: Authenticator app missing
- **WHEN** the policy asks for `totp` and the user has no TOTP credential
- **THEN** the response is 200 with error id `totp_registration_required` and `redirect_to` `/ui/setup_secure?return_to=<return URL>`
- **AND** the state cookie records the authenticator set-up

#### Scenario: Few backup codes left
- **WHEN** the policy asks for backup codes to be regenerated and the second method of the session is `lookup_secret`
- **AND** the user has too few unused codes left
- **THEN** the response is 200 with error id `regenerate_backup_codes` and `redirect_to` `/ui/backup_codes_regenerate?flow=<flow id>&return_to=<return URL>`
- **AND** the state cookie records the use of a backup code

#### Scenario: Backup codes before the authenticator app
- **WHEN** the user has too few backup codes left and no TOTP credential
- **THEN** the Login UI makes both lookups
- **AND** it sends the user to regenerate the backup codes

#### Scenario: No backup code was used
- **WHEN** the policy asks for backup codes to be regenerated and the second method of the session is not `lookup_secret`
- **THEN** the Login UI does not look up the backup codes

#### Scenario: WebAuthn key asked for
- **WHEN** the policy asks for `webauthn`
- **THEN** the Login UI does not look up the WebAuthn key
- **AND** the response follows Kratos

#### Scenario: Email verification comes first
- **WHEN** email verification is enabled and the email of the user is not verified
- **AND** the user has no authenticator app and has too few backup codes left
- **THEN** the Login UI makes both lookups
- **AND** it sends the user to `/ui/verification`

#### Scenario: A lookup fails
- **WHEN** the TOTP lookup or the backup codes lookup fails
- **THEN** the response is 500 with body `internal server error`

### Requirement: Consent requires the assurance level the policy asks for
`GET /api/consent` SHALL require a session at `aal2` when the policy asks for a second factor. It SHALL require a session at `aal1` when the policy does not ask for one. It SHALL answer a session below the required level with 403 and body `insufficient session aal`. It SHALL do this before it gets the consent request from Hydra.

#### Scenario: Second factor required and not done
- **WHEN** the policy asks for a second factor and the session is at `aal1`
- **THEN** the response is 403 with body `insufficient session aal`
- **AND** the Login UI does not call Hydra

#### Scenario: Second factor required and done
- **WHEN** the policy asks for a second factor and the session is at `aal2`
- **THEN** the Login UI gets the consent request from Hydra and accepts it

#### Scenario: No second factor required
- **WHEN** the policy asks for no second factor and the session is at `aal1`
- **THEN** the Login UI gets the consent request from Hydra and accepts it

#### Scenario: Session without an assurance level
- **WHEN** the session carries no assurance level
- **THEN** the response is 403 with body `insufficient session aal`, whatever the policy asks for

#### Scenario: No session
- **WHEN** the session check returns no session and no error
- **THEN** the response is 403 with body `insufficient session aal`
- **AND** the Login UI does not call Hydra

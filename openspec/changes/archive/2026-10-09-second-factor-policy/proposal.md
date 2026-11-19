## Why

Four functions in 2 packages decide what a sign-in still needs before it can complete. They read `MFA_ENABLED` and `OIDC_WEBAUTHN_SEQUENCING_ENABLED` inline. A reader cannot read the rule in one place. A developer cannot test the rule apart from the HTTP handlers.

This change moves the rule into one object. The two login handlers, the consent handler and their helpers answered the same question separately. They now ask this object. What the Login UI does does not change.

### Problem statement

- `pkg/kratos/handlers.go`: `shouldEnforceMFAWithSession`, `shouldEnforceWebAuthnWithSession` and `shouldRegenerateBackupCodesWithSession` each read a setting and the authentication methods of the session. Then they look up the credentials of the user. `handleCreateFlow` and `handleUpdateFlow` call them.
- `pkg/extra/handlers.go`: `sessionRequiredAAL` reads both settings and the first authentication method of the session. It decides the AAL that `handleConsent` requires.
- The two packages keep their own copies of the settings. Nothing in the code says that the four functions express one rule.
- `pkg/kratos/handlers.go` also has `shouldEnforceMFA`. It reads the session from the request cookies and then reads `mfaEnabled`. No handler calls it. Only its own test calls it.

### Scope

- One interface, `SecondFactorPolicyInterface`. It maps a sign-in to what the sign-in still needs. A sign-in is the authentication methods that the session completed, in order.
- One implementation, `PlatformSecondFactorPolicy`. It is the rule that the settings express today.
- `handleCreateFlow`, `handleUpdateFlow` and `handleConsent` ask the policy. They do not read the settings.
- `shouldEnforceMFA` is removed with its test. Then neither API keeps a copy of the settings.

### Non-goals

- Any change in behaviour. For every combination of the two settings, these stay as they are:
  - status codes
  - redirect targets and their query parameters
  - error ids and texts
  - cookies, log lines and tracing spans
  - the order and the number of calls to Kratos and Hydra
- One input gets a different answer. A session can have a list of authentication methods that is present and empty. This no longer makes `handleConsent` panic (see the risks in the design).
- New configuration.
- The other uses of `OIDC_WEBAUTHN_SEQUENCING_ENABLED`. They do not decide what a sign-in needs. They are the `pop` AMR value in `Service.AcceptLoginRequest`, the login challenge that `Service.CreateBrowserLoginFlow` does not give to Kratos, and the value that `pkg/status` reports.
- The email verification step (`VERIFICATION_ENABLED`). The handlers keep the decision about it as before.
- Any change to the frontend (`ui/`).

### Success criteria

- No existing test changes, except that `TestShouldEnforceMFA` is removed with `shouldEnforceMFA`, the unused function that it tested. Every other test of `pkg/kratos` and `pkg/extra` that existed before the change passes without an edit.
- A table test pins the platform policy. It has one row for each decision that the policy makes. The rows show what each setting asks of a password, a WebAuthn, an external-provider and a passkey first method.
- `pkg/kratos/handlers_mfa_test.go` and `pkg/extra/handlers_mfa_test.go` drive the handlers through the constructors' settings only. They use strict mocks for the service, the logger and the tracer. They pass unchanged on the code before this change and after it. They hold one row for each decision and each failure, not every combination.
- This repository does not test the combinations of the two settings with the methods of the session, before this change or after it. The Playwright tests in `ui/tests` run with both settings at their defaults.
- `go vet ./...` and `go test ./...` pass.

## What Changes

- New `pkg/kratos/second_factor_policy.go`. It holds these items:
  - the types `SignIn` and `Requirement`
  - `NewSignIn`, which builds a `SignIn` from a Kratos session
  - `PlatformSecondFactorPolicy` with `NewPlatformSecondFactorPolicy(mfaEnabled, oidcWebAuthnSequencingEnabled)`
- `SecondFactorPolicyInterface` is declared in `pkg/kratos/interfaces.go` and in `pkg/extra/interfaces.go`.
- `pkg/kratos/handlers.go`: `handleCreateFlow` and `handleUpdateFlow` ask the policy once for each request. `shouldEnforceMFAWithSession`, `shouldEnforceWebAuthnWithSession` and `shouldRegenerateBackupCodesWithSession` take the answer. They do not read a setting.
- `pkg/extra/handlers.go`: `sessionRequiredAAL` asks the policy. It does not read the settings.
- `pkg/kratos/handlers.go`: `shouldEnforceMFA` is removed, because no handler called it. The `mfaEnabled` field of `API` is removed with it. `TestShouldEnforceMFA` in `pkg/kratos/handlers_test.go` is removed too.
- `kratos.NewAPI` and `extra.NewAPI` keep their arguments. They build the platform policy from the two settings. Both accept a trailing `WithSecondFactorPolicy` option that replaces the platform policy. The tests use it to give the handlers a mock policy.
- Nothing is removed from the HTTP API. Nothing is **BREAKING**.

## Capabilities

### New Capabilities

- `second-factor-policy`: what the Login UI asks of a sign-in before it completes. It can ask for an authenticator app or a WebAuthn key set up, backup codes regenerated, and a second factor before consent. One policy decides this, and the login and consent handlers ask it.

### Modified Capabilities

<!-- None. No requirement of an existing spec changes. -->

## Impact

- **Backend Go**: `pkg/kratos/second_factor_policy.go` (new), `pkg/kratos/handlers.go`, `pkg/kratos/interfaces.go`, `pkg/extra/handlers.go`, `pkg/extra/interfaces.go`. `pkg/web/router.go` and `cmd/serve.go` do not change, because the constructors keep their arguments.
- **Mocks**: `make mocks` adds `MockSecondFactorPolicyInterface` to `pkg/kratos/mock_interfaces.go` and `pkg/extra/mock_extra.go`.
- **Tests**: new `pkg/kratos/second_factor_policy_test.go`, `pkg/kratos/handlers_mfa_test.go`, `pkg/extra/second_factor_policy_test.go` and `pkg/extra/handlers_mfa_test.go`. `pkg/kratos/handlers_test.go` loses `TestShouldEnforceMFA`. No other existing test file changes.
- **Kratos and Hydra**: the same calls in the same order. No configuration change.
- **Cookies**: no effect. The service sets and clears the state cookie as before.
- **OpenFGA**: no effect.
- **Frontend**: no effect.
- **Configuration**: none added. `MFA_ENABLED` and `OIDC_WEBAUTHN_SEQUENCING_ENABLED` keep their meaning and their defaults.

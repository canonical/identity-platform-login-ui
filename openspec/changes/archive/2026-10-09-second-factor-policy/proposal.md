## Why

What a sign-in still needs before it may complete is decided inline, in four functions of two packages, by reading `MFA_ENABLED` and `OIDC_WEBAUTHN_SEQUENCING_ENABLED`. The rule cannot be read in one place or tested apart from the HTTP handlers. This change moves it into one object that answers the question the two login handlers, the consent handler and their helpers answered separately, without changing what the Login UI does.

### Problem statement

- `pkg/kratos/handlers.go`: `shouldEnforceMFAWithSession`, `shouldEnforceWebAuthnWithSession` and `shouldRegenerateBackupCodesWithSession` each read a setting and the session's authentication methods, then look up the user's credentials. `handleCreateFlow` and `handleUpdateFlow` call them.
- `pkg/extra/handlers.go`: `sessionRequiredAAL` reads both settings and the session's first authentication method to decide the AAL that `handleConsent` requires.
- The two packages keep their own copies of the settings, and nothing in the code says that the four functions express one rule.
- `pkg/kratos/handlers.go` also has `shouldEnforceMFA`, a variant that reads the session from the request cookies and then reads `mfaEnabled`. No handler calls it; only its own test does.

### Scope

- One interface, `SecondFactorPolicyInterface`, that maps a sign-in (the authentication methods the session completed, in order) to what the sign-in still needs.
- One implementation, `PlatformSecondFactorPolicy`, which is the rule the settings express today.
- `handleCreateFlow`, `handleUpdateFlow` and `handleConsent` ask the policy instead of reading the settings.
- `shouldEnforceMFA` is removed with its test, so that neither API keeps a copy of the settings.

### Non-goals

- Any change in behaviour: status codes, redirect targets and their query parameters, error ids and texts, cookies, log lines, tracing spans, and the order and number of calls to Kratos and Hydra stay as they are for every combination of the two settings. One input is answered differently: a session whose list of authentication methods is present and empty no longer makes `handleConsent` panic (see the design's risks).
- New configuration.
- The other uses of `OIDC_WEBAUTHN_SEQUENCING_ENABLED`, which do not decide what a sign-in needs: the `pop` AMR value in `Service.AcceptLoginRequest`, the login challenge withheld from Kratos in `Service.CreateBrowserLoginFlow`, and the value reported by `pkg/status`.
- The email verification step (`VERIFICATION_ENABLED`), which the handlers keep deciding as before.
- Any frontend (`ui/`) change.

### Success criteria

- No existing test changes, except that `TestShouldEnforceMFA` is removed with `shouldEnforceMFA`, the unused function it tested. Every other test of `pkg/kratos` and `pkg/extra` that existed before the change passes without an edit.
- A table test pins the platform policy: one row for each decision it makes (what each setting asks of a password, a WebAuthn, an external-provider and a passkey first method).
- `pkg/kratos/handlers_mfa_test.go` and `pkg/extra/handlers_mfa_test.go` drive the handlers through the constructors' settings only, with strict mocks for the service, the logger and the tracer, and pass unchanged on the code before this change and after it. They hold one row per decision and per failure, not every combination. The combinations of the two settings with the session's methods are not tested in this repository, before this change or after it: the Playwright tests in `ui/tests` run with both settings at their defaults.
- `go vet ./...` and `go test ./...` pass.

## What Changes

- New `pkg/kratos/second_factor_policy.go`: the types `SignIn` and `Requirement`, `NewSignIn`, which builds a `SignIn` from a Kratos session, and `PlatformSecondFactorPolicy` with `NewPlatformSecondFactorPolicy(mfaEnabled, oidcWebAuthnSequencingEnabled)`.
- `SecondFactorPolicyInterface` is declared in `pkg/kratos/interfaces.go` and in `pkg/extra/interfaces.go`.
- `pkg/kratos/handlers.go`: `handleCreateFlow` and `handleUpdateFlow` ask the policy once per request, and `shouldEnforceMFAWithSession`, `shouldEnforceWebAuthnWithSession` and `shouldRegenerateBackupCodesWithSession` take the answer instead of reading a setting.
- `pkg/extra/handlers.go`: `sessionRequiredAAL` asks the policy instead of reading the settings.
- `pkg/kratos/handlers.go`: `shouldEnforceMFA`, which no handler called, is removed, and with it the `mfaEnabled` field of `API` and `TestShouldEnforceMFA` in `pkg/kratos/handlers_test.go`.
- `kratos.NewAPI` and `extra.NewAPI` keep their arguments and build the platform policy from the two settings. Both accept a trailing `WithSecondFactorPolicy` option that replaces it, which the tests use to give the handlers a mock policy.
- Nothing is removed from the HTTP API and nothing is **BREAKING**.

## Capabilities

### New Capabilities

- `second-factor-policy`: what the Login UI asks of a sign-in before it completes (an authenticator app or a WebAuthn key set up, backup codes regenerated, a second factor before consent), decided by one policy that the login and consent handlers ask.

### Modified Capabilities

<!-- None. No requirement of an existing spec changes. -->

## Impact

- **Backend Go**: `pkg/kratos/second_factor_policy.go` (new), `pkg/kratos/handlers.go`, `pkg/kratos/interfaces.go`, `pkg/extra/handlers.go`, `pkg/extra/interfaces.go`. `pkg/web/router.go` and `cmd/serve.go` are untouched, because the constructors keep their arguments.
- **Mocks**: `make mocks` adds `MockSecondFactorPolicyInterface` to `pkg/kratos/mock_interfaces.go` and `pkg/extra/mock_extra.go`.
- **Tests**: new `pkg/kratos/second_factor_policy_test.go`, `pkg/kratos/handlers_mfa_test.go`, `pkg/extra/second_factor_policy_test.go` and `pkg/extra/handlers_mfa_test.go`. `pkg/kratos/handlers_test.go` loses `TestShouldEnforceMFA` and is otherwise untouched; no other existing test file changes.
- **Kratos and Hydra**: the same calls in the same order. No configuration change.
- **Cookies**: unaffected. The state cookie is set and cleared as before.
- **OpenFGA**: unaffected.
- **Frontend**: unaffected.
- **Configuration**: none added. `MFA_ENABLED` and `OIDC_WEBAUTHN_SEQUENCING_ENABLED` keep their meaning and their defaults.

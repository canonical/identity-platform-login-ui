## 1. Policy

- [x] 1.1 Add `SignIn` and `Requirement` to a new `pkg/kratos/second_factor_policy.go`. A sign-in is the authentication methods that the session completed, in order. A requirement is `SecondFactor`, `SetUp` (`totp`, `webauthn` or empty) and `RegenerateBackupCodes`.
- [x] 1.2 Add `PlatformSecondFactorPolicy`, its `For` method, `NewPlatformSecondFactorPolicy(mfaEnabled, oidcWebAuthnSequencingEnabled)` and `NewSignIn` to `pkg/kratos/second_factor_policy.go`. `NewSignIn` builds a `SignIn` from a Kratos session. It takes a missing session as a sign-in with no methods. `For` makes no call and returns no error. It asks for a second factor by the first method: `oidc` with sequencing, `password` or `webauthn` with MFA. It asks for `webauthn` to set up when a method is `oidc` and sequencing is on. It asks for `totp` when no method is `oidc` and MFA is on. It asks for backup codes to be regenerated when MFA is on.
- [x] 1.3 Declare `SecondFactorPolicyInterface` in `pkg/kratos/interfaces.go` and in `pkg/extra/interfaces.go`. Run `make mocks` (or `go generate ./...`). Completion: `MockSecondFactorPolicyInterface` exists in `pkg/kratos/mock_interfaces.go` and in `pkg/extra/mock_extra.go`.
- [x] 1.4 Add the table test `TestPlatformSecondFactorPolicyFor` to a new `pkg/kratos/second_factor_policy_test.go`. Use one row for each decision of the rule. The rows are: no setting with `password`. MFA with `password`, `webauthn`, `oidc`, `passkey` and no method. Sequencing with `oidc` and with `oidc` after `password`. Both settings with `oidc`. Completion: `go test ./pkg/kratos/ -run TestPlatformSecondFactorPolicyFor` passes.

## 2. Login handlers

- [x] 2.1 In `pkg/kratos/handlers.go`, replace the fields `mfaEnabled` and `oidcWebAuthnSequencingEnabled` of `API` with `secondFactorPolicy`. Add `Option` and `WithSecondFactorPolicy`. Let `NewAPI` build the platform policy from its `mfaEnabled` and `oidcWebAuthnSequencingEnabled` arguments. Then let it apply a trailing `opts ...Option`.
- [x] 2.2 In `handleCreateFlow`, ask `secondFactorPolicy.For(NewSignIn(session))` once inside the `!intercept.DeferMFAChecks` block, before the 2 checks.
- [x] 2.3 In `handleUpdateFlow`, ask `secondFactorPolicy.For(NewSignIn(session))` once after the verification check.
- [x] 2.4 Give `shouldEnforceMFAWithSession`, `shouldEnforceWebAuthnWithSession` and `shouldRegenerateBackupCodesWithSession` the `Requirement` as an argument in place of the settings. The values are: `SetUp` is `totp`, `SetUp` is `webauthn`, `RegenerateBackupCodes`. Keep their span names, their `session == nil` checks, the lookups and the debug line.
- [x] 2.5 Remove `shouldEnforceMFA` from `pkg/kratos/handlers.go`, because no handler calls it. Remove its test `TestShouldEnforceMFA` from `pkg/kratos/handlers_test.go`. Completion: `grep -rn 'shouldEnforceMFA(' pkg/` and `grep -n 'a.mfaEnabled' pkg/kratos/handlers.go` print nothing, and `go vet ./pkg/kratos/` passes.
- [x] 2.6 Add `TestHandleCreateFlowAsksSecondFactorPolicy` and `TestHandleUpdateFlowAsksSecondFactorPolicy` to `pkg/kratos/second_factor_policy_test.go`. Use both settings off and a mock policy. The handlers follow the policy: they set up a missing authenticator or key and regenerate backup codes that run out. Completion: `go test ./pkg/kratos/ -run SecondFactorPolicy` passes.

## 3. Consent handler

- [x] 3.1 In `pkg/extra/handlers.go`, replace the fields `mfaEnabled` and `oidcWebAuthnSequencingEnabled` of `API` with `secondFactorPolicy`. Add `Option` and `WithSecondFactorPolicy`. Let `NewAPI` build the platform policy from its 2 boolean arguments. Then let it apply a trailing `opts ...Option`.
- [x] 3.2 Let `sessionRequiredAAL` ask the policy about `kratos.NewSignIn(session)`. It returns `aal2` when a second factor is required and `aal1` in all other cases. `handleConsent` does not change.
- [x] 3.3 Add `TestHandleConsentAsksSecondFactorPolicy` to a new `pkg/extra/second_factor_policy_test.go`. Use both settings off and a mock policy that requires a second factor. The handler refuses `aal1` and accepts `aal2`. Completion: `go test ./pkg/extra/ -run SecondFactorPolicy` passes.

## 4. Unchanged behaviour

- [x] 4.1 Add `pkg/kratos/handlers_mfa_test.go` (`TestHandleCreateFlowMFA`, `TestHandleUpdateFlowMFA`). The tests are tables that use only `NewAPI` and strict mocks. Use one row for each decision and each failed lookup. They pin the lookups made, their order, the response, the state cookie, the log lines and the spans.
- [x] 4.2 Add `pkg/extra/handlers_mfa_test.go` (`TestHandleConsentMFA`). It is a table over the methods and the AAL of the session, and a missing session. It uses only `NewAPI` and strict mocks, with one row for each decision.
- [x] 4.3 Run the 2 files of 4.1 and 4.2 unchanged against the parent commit. Copy them into a checkout of it, run `go generate ./...` and `go test ./pkg/kratos/ ./pkg/extra/`. Completion: every case passes there and on this change.
- [x] 4.4 Confirm that the only change to an existing test file is the removal of `TestShouldEnforceMFA`. Run `git diff --stat -- pkg/kratos/handlers_test.go pkg/kratos/service_test.go pkg/extra/handlers_test.go pkg/extra/service_test.go`. It lists `pkg/kratos/handlers_test.go` alone, with deletions only.
- [x] 4.5 Run `go vet ./...` and `go test ./...` (with `cmd/ui/dist` present, as `make vet` requires) and `openspec validate second-factor-policy --strict`. Completion: all 3 pass.

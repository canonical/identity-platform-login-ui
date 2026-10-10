## Context

`cmd/serve.go` passes `MFA_ENABLED` and `OIDC_WEBAUTHN_SEQUENCING_ENABLED` to `web.WithFlags`, and `pkg/web/router.go` hands them to `kratos.NewAPI` and `extra.NewAPI`. Before this change each API kept the two values and four functions read them:

| Function | Read | Decided |
| --- | --- | --- |
| `pkg/kratos` `shouldEnforceMFAWithSession` | `mfaEnabled`, whether any session method is `oidc` | whether to look up the user's TOTP credential and send a user without one to `/ui/setup_secure` |
| `pkg/kratos` `shouldEnforceWebAuthnWithSession` | `oidcWebAuthnSequencingEnabled`, whether any session method is `oidc` | whether to look up the user's WebAuthn key and send a user without one to `/ui/setup_passkey` |
| `pkg/kratos` `shouldRegenerateBackupCodesWithSession` | `mfaEnabled`, whether the session's second method is `lookup_secret` | whether to look up the unused backup codes and send a user with too few to `/ui/backup_codes_regenerate` |
| `pkg/extra` `sessionRequiredAAL` | both settings, the session's first method | the AAL `handleConsent` requires |

`handleCreateFlow` calls the first two, `handleUpdateFlow` the first and the third, `handleConsent` the fourth. A fifth function, `shouldEnforceMFA` in `pkg/kratos`, read the session from the request cookies, returned early on `mfaEnabled` and then called `shouldEnforceMFAWithSession`; no handler called it. Together the four are one rule, in three parts.

A second factor before consent, by the session's first method:

| First method | Second factor required |
| --- | --- |
| `oidc` | when `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is true |
| `password`, `webauthn` | when `MFA_ENABLED` is true |
| any other, or none | never |

The second factor method the user must have set up, by all the methods of the session:

| Session methods | To set up |
| --- | --- |
| one of them is `oidc` | `webauthn` when `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is true, otherwise nothing |
| none of them is `oidc` | `totp` when `MFA_ENABLED` is true, otherwise nothing |

Backup codes are regenerated when few are left, after one was used as the second method: when `MFA_ENABLED` is true.

Constraints:

- Behaviour must not change. Each lookup (`Service.HasTOTPAvailable`, `Service.HasWebAuthnAvailable`, `Service.HasNotEnoughLookupSecretsLeft`) is a call to the Kratos admin API, so the lookups must be made for the same sign-ins, in the same order, as before.
- The handler tests of `pkg/kratos` and `pkg/extra` use strict gomock mocks for the tracer and the logger. A new span name or log line on these paths fails them, and they must keep passing as they are.
- Repository rules: interfaces live in `interfaces.go` and each consuming package declares its own; tests use the standard library and gomock.

## Goals / Non-Goals

**Goals:**

- One object holds the rule; the handlers ask it.
- The rule is testable as a table, without HTTP and without mocks.
- Identical behaviour for every combination of the two settings.
- `kratos.NewAPI` and `extra.NewAPI` keep their arguments, so `pkg/web/router.go` and the calls in the existing tests do not change.
- Neither API keeps a copy of the two settings.

**Non-Goals:**

- New configuration, or any frontend change (see the proposal).
- Moving the other uses of `OIDC_WEBAUTHN_SEQUENCING_ENABLED` in `pkg/kratos/service.go` and `pkg/status`.
- Changing which parts of the rule each handler applies.

## Decisions

### 1. The policy is a pure function and the handlers gather the facts

`SecondFactorPolicyInterface.For(SignIn) Requirement` answers from the sign-in alone. `PlatformSecondFactorPolicy` holds the two settings and makes no call, so the method takes no context and returns no error. The handlers keep the lookups and make them only for what the policy asks.

**Rationale**: whether a user has a TOTP credential, a WebAuthn key or enough backup codes is a fact about the user, not part of the rule. The two login handlers also use these facts differently: `handleCreateFlow` looks up the TOTP credential and stops there when it is missing, otherwise the WebAuthn key, and answers a failed lookup with a text of its own for each; `handleUpdateFlow` looks up the TOTP credential and the backup codes, both, and then chooses. Keeping the lookups in the handlers keeps their order, their number and the error answers as they were, and lets the rule be tested as a table.

**Alternative considered**: a policy that declares an interface for the three lookups and makes them itself. Rejected: it would have to know which handler is asking and to report which lookup failed, which moves the handlers' control flow into the policy.

### 2. `Requirement` says three things

| Field | Meaning | Replaces |
| --- | --- | --- |
| `SecondFactor bool` | the session must be at AAL2 before consent is given | the `switch` on the first method in `sessionRequiredAAL` |
| `SetUp string` | the second factor method the user must have set up: `totp`, `webauthn`, or empty for none | the setting and the search for an `oidc` method in `shouldEnforceMFAWithSession` (`totp`) and `shouldEnforceWebAuthnWithSession` (`webauthn`) |
| `RegenerateBackupCodes bool` | a user who signs in with a backup code and is left with too few regenerates them | the `mfaEnabled` check in `shouldRegenerateBackupCodesWithSession` |

**Rationale**: these are the three things the handlers act on. The Login UI does not choose which second factor counts at sign-in: Kratos accepts any the user has, and `handleConsent` only compares the AAL. What the Login UI adds is which method a user must have set up, hence `SetUp`.

`SetUp` is one Kratos method name and not a list, because the rule never asks for two: a session either has an `oidc` method or has none. `handleCreateFlow` acts on `totp` and on `webauthn`; `handleUpdateFlow` acts on `totp` only, as before.

Whether a backup code was used (the session's second method is `lookup_secret`) and how many are left stay in `shouldRegenerateBackupCodesWithSession`. They are facts about the session and the user, and the debug line `User has not yet completed 2fa` stays where it was.

### 3. `SignIn` carries the methods the session completed

`SignIn{Methods}` lists every authentication method of the session, in order. `NewSignIn` in `pkg/kratos/second_factor_policy.go` builds it from a Kratos session, and both handler packages use it.

**Rationale**: the platform policy reads the list in two ways. A second factor is required by the first method alone, while what is set up depends on whether any method was `oidc`. A session that completed `password` and then `oidc` shows the difference, and the table test pins it. The policy takes the first method from the list itself, so a sign-in cannot name a first method that its list does not start with.

`NewSignIn` reads the session through the Kratos client's nil-safe getters, so a missing session is a sign-in with no methods. `handleConsent` relied on the same getters before: without a session it finds no assurance level and answers 403.

### 4. The handlers ask once per request and keep their helpers

`handleCreateFlow` and `handleUpdateFlow` call `secondFactorPolicy.For(NewSignIn(session))` once, where the first setting was read before: after the email verification check, and in `handleCreateFlow` only when the tenant resolver does not defer the MFA checks. `shouldEnforceMFAWithSession`, `shouldEnforceWebAuthnWithSession` and `shouldRegenerateBackupCodesWithSession` keep their names and their spans, and take the `Requirement` as an argument. `handleUpdateFlow` can reach its two helpers without a session, as before. They keep their `session == nil` checks and look nothing up then, whatever the requirement says.

Control flow of `handleCreateFlow` for a session, unchanged except for step 2:

1. Email verification check; a user who must verify is redirected.
2. `secondFactorPolicy.For`.
3. `SetUp` is `totp`: `HasTOTPAvailable`; a user without it is redirected to `/ui/setup_secure`.
4. `SetUp` is `webauthn`: `HasWebAuthnAvailable`; a user without it is redirected to `/ui/setup_passkey`.
5. The Hydra login continues (`MustReAuthenticate`, `AcceptLoginRequest`).

Control flow of `handleUpdateFlow` after `UpdateLoginFlow` and `CheckSession`, unchanged except for step 2:

1. Email verification check.
2. `secondFactorPolicy.For`.
3. `SetUp` is `totp` and there is a session: `HasTOTPAvailable`.
4. `RegenerateBackupCodes`, a session and a `lookup_secret` second method: `HasNotEnoughLookupSecretsLeft`.
5. The first that applies: verification redirect, backup codes redirect, authenticator redirect, tenant selection or the Hydra login, Kratos's own redirect.

`handleConsent`: `CheckSession`, `secondFactorPolicy.For`, the AAL comparison, then `GetConsent` and `AcceptConsent` as before.

**Alternative considered**: leave the handlers untouched and let each helper ask the policy itself. Rejected: the policy would be asked twice per request.

### 5. The constructors keep their arguments

`kratos.NewAPI` and `extra.NewAPI` build `NewPlatformSecondFactorPolicy(mfaEnabled, oidcWebAuthnSequencingEnabled)` from the arguments they already take. Both gain a trailing `opts ...Option` with one option, `WithSecondFactorPolicy`, in the style of the options of `pkg/web`. The tests use it to give the handlers a mock policy and check that they follow it. `pkg/web/router.go` gives no option, so each API builds its policy from the same two settings, as each kept its own copy of them before.

**Alternative considered**: build one policy in `pkg/web/router.go` and pass it to both constructors in place of the two booleans. Rejected: it changes two constructor signatures and every call in the existing tests, for no change in behaviour.

### 6. The handlers depend on an interface; the types live in `pkg/kratos`

The handlers hold a `SecondFactorPolicyInterface` and not the platform policy itself, as the repository asks of every dependency. It is what lets the handler tests give a mock policy and check that the handlers act on its answer instead of working it out again from the session.

The interface is declared in `pkg/kratos/interfaces.go` and again in `pkg/extra/interfaces.go`, following the rule that a package declares the interfaces it depends on. `SignIn`, `Requirement` and the platform policy live in `pkg/kratos/second_factor_policy.go`; `pkg/extra` already imports `pkg/kratos`. The existing `go:generate` lines produce the mocks.

### 7. The policy starts no span and logs nothing

The platform policy does no I/O, so there is nothing to trace. The spans of the three helpers and every log line stay as they were, which is also what lets the existing handler tests pass as they are.

### 8. `shouldEnforceMFA` is removed

`shouldEnforceMFA`, the variant that checks the session from the request cookies, was called by no handler. It was the last reader of `mfaEnabled` on `kratos.API`, and it had to read the setting before it knew of any session, which a policy asked about a sign-in cannot replace. It is removed together with its test, `TestShouldEnforceMFA`, and the `mfaEnabled` field. `kratos.NewAPI` still takes the argument and passes it to the platform policy.

`shouldEnforceVerification`, its counterpart for the email verification step, has no caller either. It reads `verificationEnabled`, which this change does not touch, and stays.

**Alternative considered**: keep the function and the field so that no existing test changes. Rejected: it leaves a copy of the setting on the API for code that no request reaches.

### 9. What is left as it is

- Each handler still applies part of the requirement: `handleCreateFlow` does not look at backup codes and `handleUpdateFlow` does not act on `webauthn`. The spec states both.
- The tenant resolver's `DeferMFAChecks` still decides whether `handleCreateFlow` runs the MFA checks at all.

## Risks / Trade-offs

- [A difference in behaviour slips in] → The existing tests pass as they are, apart from the removed `TestShouldEnforceMFA`. `handlers_mfa_test.go` in both packages uses only the constructors' settings and strict mocks, pins the lookups made, their order, the answers, the state cookie, the log lines and the spans for each decision of the rule and each failed lookup, and passes unchanged on the code before the change.
- [One input is answered differently] → A session whose `authentication_methods` is present and empty made `sessionRequiredAAL` read the first element of an empty list, and `handleConsent` panicked. `NewSignIn` makes it a sign-in with no methods, which needs no second factor, so `handleConsent` now compares the session's AAL with `aal1`. Kratos gives a session without methods the level `aal0`, which is refused with 403.
- [The unit tests do not walk every combination] → They keep one row per decision and per failure. Combinations of the two settings with the session's methods, and a returning session that has everything set up, are not tested in this repository, before this change or after it: the Playwright tests in `ui/tests` run with both settings at their defaults.
- [A handler ignores part of a requirement] → A requirement of `webauthn` has no effect in `handleUpdateFlow`, as the setting had none there. The row `sequencing: oidc then a backup code follows kratos without a lookup` of `TestHandleUpdateFlowMFA` pins it.
- [Removing `TestShouldEnforceMFA` loses coverage] → Its cases about the rule (MFA off, an `oidc` session, a TOTP credential present, missing, and a failed lookup) are rows of `TestHandleCreateFlowMFA`, `TestHandleUpdateFlowMFA` and `TestPlatformSecondFactorPolicyFor`. Its two cases about a failing session check tested only the removed function.

Rollout, observability and failure handling:

- No configuration, schema or migration. The change ships with any release and is rolled back by reverting it.
- Spans and log lines on the login and consent paths are the same. The policy cannot fail, so the change adds no failure path.
- Kratos and Hydra receive the same calls in the same order. OpenFGA is not involved.

## Migration Plan

1. Merge. No operator action and no configuration change.
2. Developers run `make mocks` to regenerate `mock_interfaces.go` and `mock_extra.go`.

## Open Questions

None.

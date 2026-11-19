## Context

`cmd/serve.go` passes `MFA_ENABLED` and `OIDC_WEBAUTHN_SEQUENCING_ENABLED` to `web.WithFlags`. `pkg/web/router.go` gives them to `kratos.NewAPI` and `extra.NewAPI`. Before this change each API kept the two values, and four functions read them:

| Function | Read | Decided |
| --- | --- | --- |
| `pkg/kratos` `shouldEnforceMFAWithSession` | `mfaEnabled`, if any session method is `oidc` | if the service looks up the user's TOTP credential and sends a user without one to `/ui/setup_secure` |
| `pkg/kratos` `shouldEnforceWebAuthnWithSession` | `oidcWebAuthnSequencingEnabled`, if any session method is `oidc` | if the service looks up the user's WebAuthn key and sends a user without one to `/ui/setup_passkey` |
| `pkg/kratos` `shouldRegenerateBackupCodesWithSession` | `mfaEnabled`, if the session's second method is `lookup_secret` | if the service looks up the unused backup codes and sends a user with too few to `/ui/backup_codes_regenerate` |
| `pkg/extra` `sessionRequiredAAL` | both settings, the session's first method | the AAL that `handleConsent` requires |

`handleCreateFlow` calls the first 2 functions. `handleUpdateFlow` calls the first and the third. `handleConsent` calls the fourth.

A fifth function, `shouldEnforceMFA` in `pkg/kratos`, read the session from the request cookies. It returned early on `mfaEnabled` and then called `shouldEnforceMFAWithSession`. No handler called it. The first 4 functions together are one rule, in 3 parts.

A second factor before consent depends on the session's first method:

| First method | Second factor required |
| --- | --- |
| `oidc` | when `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is true |
| `password`, `webauthn` | when `MFA_ENABLED` is true |
| any other, or none | never |

The second factor method that the user must have set up depends on all the methods of the session:

| Session methods | To set up |
| --- | --- |
| one of them is `oidc` | `webauthn` when `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is true, otherwise nothing |
| none of them is `oidc` | `totp` when `MFA_ENABLED` is true, otherwise nothing |

The service regenerates backup codes when few are left after the user used one as the second method. This happens when `MFA_ENABLED` is true.

Constraints:

- Behaviour must not change.
- Each lookup is a call to the Kratos admin API. The lookups are `Service.HasTOTPAvailable`, `Service.HasWebAuthnAvailable` and `Service.HasNotEnoughLookupSecretsLeft`. The service must make the lookups for the same sign-ins, in the same order, as before.
- The handler tests of `pkg/kratos` and `pkg/extra` use strict gomock mocks for the tracer and the logger. A new span name or log line on these paths makes them fail. They must keep passing as they are.
- Repository rules: interfaces live in `interfaces.go`, and each consuming package declares its own. Tests use the standard library and gomock.

## Goals / Non-Goals

**Goals:**

- One object holds the rule. The handlers ask it.
- A table can test the rule, without HTTP and without mocks.
- The behaviour is identical for every combination of the two settings.
- `kratos.NewAPI` and `extra.NewAPI` keep their arguments. Then `pkg/web/router.go` and the calls in the existing tests do not change.
- Neither API keeps a copy of the two settings.

**Non-Goals:**

- New configuration, or any change to the frontend (see the proposal).
- A move of the other uses of `OIDC_WEBAUTHN_SEQUENCING_ENABLED` in `pkg/kratos/service.go` and `pkg/status`.
- A change to the parts of the rule that each handler applies.

## Decisions

### 1. The policy is a pure function and the handlers gather the facts

`SecondFactorPolicyInterface.For(SignIn) Requirement` answers from the sign-in alone. `PlatformSecondFactorPolicy` holds the two settings and makes no call. So the method takes no context and returns no error. The handlers keep the lookups. They make a lookup only for what the policy asks.

**Rationale**: A fact about the user is not part of the rule. These facts are if a user has a TOTP credential, a WebAuthn key or enough backup codes. The two login handlers also use these facts in different ways.

- `handleCreateFlow` looks up the TOTP credential. It stops there when the credential is missing. Otherwise it looks up the WebAuthn key. It answers a failed lookup with a text of its own for each lookup.
- `handleUpdateFlow` looks up the TOTP credential and the backup codes. It makes both lookups and then chooses.

When the lookups stay in the handlers, their order, their number and the error answers stay as they were. The rule can also be tested as a table.

**Alternative considered**: a policy that declares an interface for the three lookups and makes them itself. We rejected it. A policy that makes the lookups has to know which handler asks. It also has to report which lookup failed. This moves the control flow of the handlers into the policy.

### 2. `Requirement` says three things

| Field | Meaning | Replaces |
| --- | --- | --- |
| `SecondFactor bool` | the session must be at AAL2 before the user gives consent | the `switch` on the first method in `sessionRequiredAAL` |
| `SetUp string` | the second factor method that the user must have set up: `totp`, `webauthn`, or empty for none | the setting and the search for an `oidc` method in `shouldEnforceMFAWithSession` (`totp`) and `shouldEnforceWebAuthnWithSession` (`webauthn`) |
| `RegenerateBackupCodes bool` | a user who signs in with a backup code and has too few left regenerates them | the `mfaEnabled` check in `shouldRegenerateBackupCodesWithSession` |

**Rationale**: The handlers act on these 3 things. The Login UI does not choose which second factor counts at sign-in. Kratos accepts any second factor that the user has. `handleConsent` only compares the AAL. The Login UI adds one thing: which method a user must have set up. This is the reason for `SetUp`.

`SetUp` is one Kratos method name and not a list. The rule never asks for two methods, because a session either has an `oidc` method or has none. `handleCreateFlow` acts on `totp` and on `webauthn`. `handleUpdateFlow` acts on `totp` only, as before.

Two facts stay in `shouldRegenerateBackupCodesWithSession`. The first fact is if the user used a backup code (the second method of the session is `lookup_secret`). The second fact is how many codes are left. They are facts about the session and the user. The debug line `User has not yet completed 2fa` stays where it was.

### 3. `SignIn` carries the methods the session completed

`SignIn{Methods}` lists every authentication method of the session, in order. `NewSignIn` in `pkg/kratos/second_factor_policy.go` builds it from a Kratos session. Both handler packages use it.

**Rationale**: The platform policy reads the list in 2 ways. The first method alone decides if a second factor is required. What the user must set up depends on if any method was `oidc`. A session that completed `password` and then `oidc` shows the difference. The table test pins it. The policy takes the first method from the list itself. So a sign-in cannot name a first method that its list does not start with.

`NewSignIn` reads the session through the nil-safe getters of the Kratos client. So a missing session is a sign-in with no methods. `handleConsent` used the same getters before. Without a session it finds no assurance level and answers 403.

### 4. The handlers ask once per request and keep their helpers

`handleCreateFlow` and `handleUpdateFlow` call `secondFactorPolicy.For(NewSignIn(session))` once. They call it where they read the first setting before. This is after the email verification check. In `handleCreateFlow` the call is made only when the tenant resolver does not defer the MFA checks.

`shouldEnforceMFAWithSession`, `shouldEnforceWebAuthnWithSession` and `shouldRegenerateBackupCodesWithSession` keep their names and their spans. They take the `Requirement` as an argument. `handleUpdateFlow` can reach its two helpers without a session, as before. They keep their `session == nil` checks. They look nothing up then, whatever the requirement says.

The control flow of `handleCreateFlow` for a session does not change, except for step 2:

1. Email verification check. The handler redirects a user who must verify.
2. `secondFactorPolicy.For`.
3. `SetUp` is `totp`: `HasTOTPAvailable`. The handler redirects a user without it to `/ui/setup_secure`.
4. `SetUp` is `webauthn`: `HasWebAuthnAvailable`. The handler redirects a user without it to `/ui/setup_passkey`.
5. The Hydra login continues (`MustReAuthenticate`, `AcceptLoginRequest`).

The control flow of `handleUpdateFlow` after `UpdateLoginFlow` and `CheckSession` does not change, except for step 2:

1. Email verification check.
2. `secondFactorPolicy.For`.
3. `SetUp` is `totp` and there is a session: `HasTOTPAvailable`.
4. `RegenerateBackupCodes`, a session and a `lookup_secret` second method: `HasNotEnoughLookupSecretsLeft`.
5. The first of these that applies: verification redirect, backup codes redirect, authenticator redirect, tenant selection or the Hydra login, Kratos's own redirect.

`handleConsent` does these steps: `CheckSession`, `secondFactorPolicy.For`, the AAL comparison, and then `GetConsent` and `AcceptConsent` as before.

**Alternative considered**: leave the handlers untouched and let each helper ask the policy itself. We rejected it. A helper that asks the policy itself makes the service ask the policy twice for each request.

### 5. The constructors keep their arguments

`kratos.NewAPI` and `extra.NewAPI` build `NewPlatformSecondFactorPolicy(mfaEnabled, oidcWebAuthnSequencingEnabled)` from the arguments that they already take. Both get a trailing `opts ...Option` with one option, `WithSecondFactorPolicy`. It follows the style of `pkg/web`'s options. The tests use it to give the handlers a mock policy and to check that the handlers follow it.

`pkg/web/router.go` gives no option. So each API builds its policy from the same two settings. Before, each API kept its own copy of them.

**Alternative considered**: build one policy in `pkg/web/router.go` and pass it to both constructors in place of the two booleans. We rejected it. It changes two constructor signatures and every call in the existing tests, for no change in behaviour.

### 6. The handlers depend on an interface; the types live in `pkg/kratos`

The handlers hold a `SecondFactorPolicyInterface` and not the platform policy itself. The repository asks this of every dependency. It lets the handler tests give a mock policy. The tests then check that the handlers act on its answer and do not work it out again from the session.

The interface is declared in `pkg/kratos/interfaces.go` and again in `pkg/extra/interfaces.go`. This follows the rule that a package declares the interfaces that it depends on. `SignIn`, `Requirement` and the platform policy live in `pkg/kratos/second_factor_policy.go`. `pkg/extra` already imports `pkg/kratos`. The existing `go:generate` lines produce the mocks.

### 7. The policy starts no span and logs nothing

The platform policy does no I/O, so there is nothing to trace. The spans of the 3 helpers and every log line stay as they were. This also lets the existing handler tests pass as they are.

### 8. `shouldEnforceMFA` is removed

`shouldEnforceMFA` checks the session from the request cookies. No handler called it. It was the last reader of `mfaEnabled` on `kratos.API`. It had to read the setting before it knew of any session. A policy that answers about a sign-in cannot replace this. We remove it together with its test, `TestShouldEnforceMFA`, and the `mfaEnabled` field. `kratos.NewAPI` still takes the argument and passes it to the platform policy.

`shouldEnforceVerification` is the counterpart for the email verification step. It also has no caller. It reads `verificationEnabled`, which this change does not touch. It stays.

**Alternative considered**: keep the function and the field so that no existing test changes. We rejected it. The function and the field leave a copy of the setting on the API for code that no request reaches.

### 9. What is left as it is

- Each handler still applies a part of the requirement. `handleCreateFlow` does not look at backup codes. `handleUpdateFlow` does not act on `webauthn`. The spec states both.
- The tenant resolver's `DeferMFAChecks` still decides if `handleCreateFlow` runs the MFA checks at all.

## Risks / Trade-offs

- [A difference in behaviour slips in] → The existing tests pass as they are, apart from the removed `TestShouldEnforceMFA`. `handlers_mfa_test.go` in both packages uses only the constructors' settings and strict mocks. For each decision of the rule and each failed lookup, it pins the lookups made and their order. It also pins the answers, the state cookie, the log lines and the spans. It passes unchanged on the code before the change.
- [One input gets a different answer] → A session can have an `authentication_methods` list that is present and empty. Before, `sessionRequiredAAL` read the first element of an empty list and `handleConsent` panicked. `NewSignIn` makes it a sign-in with no methods, which needs no second factor. So `handleConsent` now compares the AAL of the session with `aal1`. Kratos gives a session without methods the level `aal0`. The handler refuses it with 403.
- [The unit tests do not walk every combination] → They keep one row for each decision and each failure. This repository does not test the combinations of the two settings with the methods of the session. It does not test a returning session that has everything set up. This was true before the change and is true after it. The Playwright tests in `ui/tests` run with both settings at their defaults.
- [A handler ignores a part of a requirement] → A requirement of `webauthn` has no effect in `handleUpdateFlow`. The setting had no effect there either. The row `sequencing: oidc then a backup code follows kratos without a lookup` of `TestHandleUpdateFlowMFA` pins it.
- [The removal of `TestShouldEnforceMFA` loses coverage] → The rows of `TestHandleCreateFlowMFA`, `TestHandleUpdateFlowMFA` and `TestPlatformSecondFactorPolicyFor` cover its cases about the rule. These cases are: MFA off, an `oidc` session, a TOTP credential present, a TOTP credential missing, and a failed lookup. Its 2 cases about a failing session check tested only the removed function.

Rollout, observability and failure handling:

- The change has no configuration, schema or migration. It ships with any release. To roll it back, revert it.
- The spans and log lines on the login and consent paths are the same. The policy cannot fail, so the change adds no failure path.
- Kratos and Hydra receive the same calls in the same order. OpenFGA is not involved.

## Migration Plan

1. Merge. No operator action and no configuration change.
2. Developers run `make mocks` to regenerate `mock_interfaces.go` and `mock_extra.go`.

## Open Questions

None.

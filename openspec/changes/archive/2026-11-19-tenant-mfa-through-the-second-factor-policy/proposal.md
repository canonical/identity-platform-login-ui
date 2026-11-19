## Why

### Problem statement

The `pkg/kratos` and `pkg/extra` APIs now ask a second factor policy what a sign-in needs (`second-factor-policy`). Before, they read an MFA flag. With the extension enabled, `pkg/web` built the APIs with that flag off, and it gave `pkg/kratos` the option `WithBackupCodesRegeneration`. The flag and the option do not exist any more.

### Scope

- How `pkg/web` replaces the rule of `MFA_ENABLED` when the extension is enabled.

### Non-goals

- A change to what a sign-in needs, with the extension enabled or disabled.
- A change to the tenant checks. The tenant checks keep the MFA of each tenant. The policy sees only the methods of a sign-in, not the tenant.

### Success criteria

- No requirement names the MFA flag of the handlers or `WithBackupCodesRegeneration`.
- With the extension enabled, the handlers ask for no second factor and for no setup of one. They still ask a user with few backup codes left to make new ones.

## What Changes

- The extension has a second factor policy, `byosso.SecondFactorPolicy`. For every sign-in, it asks only for new backup codes when few are left.
- `pkg/web` gives this policy to the `pkg/kratos` and `pkg/extra` APIs with `WithSecondFactorPolicy`. It gives `MFA_ENABLED` to the APIs unchanged.
- `pkg/kratos` loses the option `WithBackupCodesRegeneration`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `tenant-mfa`: the policy of the extension replaces the policy of `MFA_ENABLED`.
- `byosso-disabled`: with the flag off, the handlers use the policy of `MFA_ENABLED`.

## Impact

- Backend: `pkg/byosso/mfa.go`, `pkg/web/byosso.go`, `pkg/web/router.go`, `pkg/kratos/extension.go`.
- Frontend: no change.
- Kratos, Hydra, cookies and OpenFGA: no change.

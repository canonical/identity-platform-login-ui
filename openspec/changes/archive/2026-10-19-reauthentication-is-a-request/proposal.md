## Why

### Problem statement

When an app asks for a fresh sign-in, Login UI starts the company sign-in with `reauthenticate`. The SSO service then sends `prompt=login` and `max_age=0` to the identity provider. Many identity providers ignore these parameters. The specification said that the identity provider asks the user to log in again. Login UI and the SSO service cannot make an external identity provider do that.

### Scope

- The text of the requirements that describe `reauthenticate`.

### Non-goals

- A change to the behaviour of Login UI.

### Success criteria

- No requirement says that the identity provider asks the user to log in again.

## What Changes

- `reauthenticate` is a request to the identity provider. The identity provider can ignore it, and no service checks what it did.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `re-authentication`: a company sign-in passes the request on, and nothing more.
- `connected-accounts`: the sign-in again before a settings change asks for a fresh login.

## Impact

No code changes. Kratos, Hydra and cookies do not change.

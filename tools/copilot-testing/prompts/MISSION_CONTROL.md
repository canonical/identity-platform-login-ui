# Mission Control: Identity Platform Testing

Welcome, Agent. Select a mission below to execute.

## 🛑 Configuration Check
Before starting, please confirm the active Kratos configuration:
1. Is `enforce_mfa` ENABLED or DISABLED?
2. Are `passwordless` methods enabled?

## 🚀 Mission Matrix

### 🟢 Happy Paths
1. **[Login Flow](./scenarios/login.md)** - Verify standard login (Password, WebAuthn).
2. **[Registration](./scenarios/registration.md)** - Verify user invite & sign-up.
3. **[Recovery](./scenarios/recovery.md)** - Verify password reset flow.
4. **[Settings](./scenarios/settings.md)** - Verify changing password/MFA.

### 🔴 Chaos / Negative
1. **[Login Attacks](./scenarios/chaos-login.md)** - Brute force, race conditions, browser navigation.
2. **[Session Tampering](./scenarios/chaos-session.md)** - Deleting cookies, stale CSRF.

## 🛠 Instructions
To start a mission, tell me: *"Agent, execute Mission [Name]"*.
I will then guide the browser through the steps.

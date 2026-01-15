# Identity Platform Chaos Agent XState

A deterministic, state-machine driven testing tool for the Identity Platform Login UI.

## Overview

This tool uses [XState v5](https://stately.ai/docs/xstate) to define strict authentication flows ("Happy Paths") and [Playwright](https://playwright.dev/) to execute them against the running UI. Unlike the original heuristic-based chaos agent, this tool ensures reproducible, step-by-step verification of complex scenarios like WebAuthn registration and login.

## Prerequisites

- **Node.js** v18+
- **Playwright** browsers (installed automatically via npm install or manually)
- **Identity Platform** running locally (Kratos, Hydra, UI)
- **Oathtool** (for TOTP generation in setup scripts)

## Installation

```bash
npm install
```

## Setup Identities

Before running tests, reset the Kratos identities to a clean state:

```bash
bash scripts/setup-identities.sh
```

This script creates standard test users like `webauthn-dynamic@example.com`.

## Usage

Run the default test scenario against localhost:

```bash
npm start
```

### CLI Options

The tool is fully configurable for use in different environments (local, staging, CI).

```bash
npx ts-node src/index.ts [options]
```

| Option | Description | Default |
|--------|-------------|---------|
| `--base-url <url>` | Base URL of the application (e.g., `https://login.staging.com`) | `http://localhost` |
| `--reset-user` | Reset WebAuthn credentials for the user before running (requires Kratos Admin) | `false` |
| `--username <email>` | User identifier for login | `webauthn-dynamic@example.com` |
| `--password <pwd>` | User password | `Password123!` |
| `--profile <name>` | Profile/Flow to run (`webauthn-flow`) | `webauthn-flow` |
| `--scenario <name>` | Scenario ID | `webauthn-login` |
| `--headed` | Run browser in headed mode (visible UI) | `false` |
| `--video` | Record video of the session | `false` |

### Example: Running against Staging

```bash
npm start -- \
  --base-url https://login.staging.canonical.com \
  --reset-user \
  --username qa-user@canonical.com \
  --password "SuperSecret123!"
```

## Scenarios

### `webauthn-login`
Verifies the complete lifecycle of a WebAuthn passkey:
1. **Login** with password (user has no 2FA initially).
2. **Skip TOTP** setup if prompted.
3. **Register WebAuthn** security key (using Chrome DevTools Protocol Virtual Authenticator).
4. **Logout**.
5. **Login** again.
6. **Verify** using the previously registered WebAuthn key.

### `totp-login`
Verifies Login with TOTP (Time-based One-Time Password):
1. **Prerequisite**: User must have TOTP configured (handled by `setup-identities.sh`).
2. **Login** with password.
3. **Analyze State** detects TOTP challenge.
4. **Verify** by generating a valid code from the saved secret.
5. **Success** if redirected to dashboard.

To run:
```bash
npm start -- --scenario=totp-login --profile=totp-flow --username=login-test@example.com --password="Test1234!"
```

### `recovery-login`
Verifies the Password Recovery (Forgot Password) flow:
1. **Start** on login page.
2. **Click** "Forgot your password?".
3. **Submit** email address.
4. **Fetch** recovery code from MailSlurper (localhost API).
5. **Enter** code in UI.
6. **Set** new password.
7. **Success** if validation passes and user is potentially logged in/redirected.

To run:
```bash
npm start -- --scenario=recovery-login --profile=recovery-flow
```

### `backup-code-login`
Verifies the Backup Code setup and usage flow:
1. **Login** with password and TOTP (Prerequisite: TOTP enabled).
2. **Navigate** (or check) Backup Codes setup.
3. **Deactivate** existing codes if present.
4. **Generate** new codes.
5. **Logout**.
6. **Login** with password.
7. **Use Backup Code** instead of TOTP.

To run (requires a user with TOTP, e.g. `login-test`):
```bash
npm start -- --scenario=backup-code-login --profile=backup-code-flow --username=login-test@example.com --password="Test1234!"
```

## Project Structure

- `src/machines/`: XState machine definitions.
  - `base.ts`: Shared setup, actors, and types.
  - `webauthn.machine.ts`: WebAuthn flow logic.
  - `totp.machine.ts`: TOTP flow logic.
  - `recovery.machine.ts`: Recovery flow logic.
  - `backup-code.machine.ts`: Backup code flow logic.
- `src/drivers/`: Domain-specific Playwright execution logic.
  - `root.ts`: Main entry point orchestrating sub-drivers.
  - `auth.ts`: Login, Logout, Session validation.
  - `mfa.ts`: WebAuthn and TOTP logic.
  - `recovery.ts`: Password recovery logic.
  - `settings.ts`: Account settings and backup codes.
- `src/index.ts`: The entry point. Selects the appropriate machine based on `--profile`, injects the driver actors, and runs the simulation.

## Key Implementation Details

- **Virtual Authenticators**: Uses CDP (`WebAuthn.addVirtualAuthenticator`) to simulate hardware keys.
- **Credential Persistence**: Since CDP sessions are ephemeral, the `Driver` exports the generated credential after registration and re-imports it into the browser session after logout to ensure the key is available for the second login.

## Troubleshooting

- **Timeout Errors**: If the standard timeout (30s) is exceeded, check if the UI is hanging or if the selector has changed.
- **"No credentials found"**: This means the export step failed during registration. Ensure the "Add security key" flow completed successfully before the driver attempts to export.

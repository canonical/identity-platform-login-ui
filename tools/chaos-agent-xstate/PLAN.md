# Plan: comprehensive "Happy Path" Testing

## Overview
We will extend the `chaos-agent-xstate` framework to cover all critical "Happy Path" scenarios. This involves porting logic from the heuristic-based `chaos-agent` and adapting it to the deterministic XState machine.

## Priorities

1.  **Recovery Flow** (Password Reset via Email)
2.  **Account Management** (Settings: Add/Remove Keys/TOTP)
3.  **Backup Codes** (Generation & Usage)
4.  **OIDC Provider Flow** (Login via 3rd party app)

## Detailed Implementation Plan

### Phase 1: Recovery Flow (`recovery-flow`)
**Goal:** Verify a user can reset their password using the email loop.

*   **Service Layer (`src/services/email.ts`)**:
    *   Port `EmailService` from `chaos-agent`.
    *   Ensure it can poll `http://localhost:4437/mail` for the specific user's email.
    *   Implement `getLatestCode(email)` to regex-match the 6-8 digit code.

*   **Driver (`src/driver.ts`)**:
    *   `clickForgotPassword()`: Finds link on login page.
    *   `enterRecoveryEmail(email)`: Fills form.
    *   `enterRecoveryCode(code)`: Fills code input.
    *   `enterNewPassword(password)`: Sets and confirms new password.

*   **Machine (`src/machine.ts`)**:
    *   Add states: `requestingRecovery`, `awaitingRecoveryCode`, `resettingPassword`.
    *   Logic:
        1.  Start -> `navigateToLogin`.
        2.  `clickForgotPassword`.
        3.  `enterRecoveryEmail`.
        4.  `invoke: fetchRecoveryCode` (calls EmailService).
        5.  `enterRecoveryCode`.
        6.  `enterNewPassword`.
        7.  Verify login with *new* password.

### Phase 2: Account Management (`settings-flow`)
**Goal:** Verify a logged-in user can add/remove MFA methods.

*   **Driver (`src/driver.ts`)**:
    *   `navigateToSettings()`: Go to `/ui/settings` (or click avatar -> settings).
    *   `removeWebAuthn(name?)`: Find trash icon, confirm modal.
    *   `removeTotp()`: Click remove button, confirm.
    *   `addTotpFromSettings()`: Reuse existing `setupTotp` logic but triggered from settings page.

*   **Machine**:
    *   Extend `dashboard` state to allow transitions to `settings`.
    *   New Scenario `manage-account`: Login -> Dashboard -> Settings -> (Add/Remove) -> Callback.

### Phase 3: Backup Codes
**Goal:** Verify generating and using backup codes.

*   **Driver**:
    *   `generateBackupCodes()`: Click "Generate", scrape codes from DOM, save to state/file.
    *   `loginWithBackupCode()`: During MFA step, click "Use backup code", enter one from saved list.

*   **Machine**:
    *   Add `generatingBackupCodes` state after `dashboard` in setup flow.
    *   Add `backupCodeVerifying` state parallel to `totpVerifying` / `webauthnVerifying`.

### Phase 4: OIDC Provider Login (`oidc-flow`)
**Goal:** Verify the platform works as an Identity Provider for the sample app.

*   **Driver**:
    *   `navigateToOidcApp(url)`: Go to `http://localhost:4446`.
    *   `clickOidcLogin()`: Click "Authorize" / "Login" on the OIDC app.
    *   `handleConsent()`: If Hydra shows consent screen, accept it.
    *   `verifyOidcSuccess()`: Check if redirected back to OIDC app callback URL with success state.

*   **Machine**:
    *   New entry point: `navigateToOidcApp` instead of `navigateToLogin`.
    *   The middle part (Login/MFA) remains the same (`performLogin`, `totpVerifying`, etc.).
    *   Final state checks for OIDC app success instead of Dashboard.

## Code Structure Refactoring (Priority #0)
**Goal:** Split monolithic files into a hierarchical structure to improve maintainability before adding new complexity.

### 1. Driver Refactoring
Split `src/driver.ts` into domain-specific modules in `src/drivers/`:
*   `drivers/base.ts`: Common utilities (logger, page wrappers).
*   `drivers/auth.ts`: Login, Logout, standard flow.
*   `drivers/mfa.ts`: TOTP, WebAuthn, and Backup Codes logic.
*   `drivers/recovery.ts`: Password reset logic.
*   `drivers/settings.ts`: Account management logic.
*   `drivers/root.ts`: Aggregator that initializes all sub-drivers.

### 2. Utilities Refactoring
*   Create `src/utils/` directory.
*   Move `src/kratos-admin.ts` to `src/utils/kratos-admin.ts`.
*   Move generic helpers (like OTP generation if extracted) there.

### 3. Machine Refactoring
Split `src/machine.ts` into composed actors/machines in `src/machines/`:
*   `machines/login-machine.ts`: The logic for standard authentication.
*   `machines/recovery-machine.ts`: Logic for recovery flow.
*   `machines/full-flow-machine.ts`: The top-level orchestrator that decides which sub-machine to run based on the `scenario` input.

## Detailed Implementation Plan

### Phase 1: Recovery Flow (`recovery-flow`)


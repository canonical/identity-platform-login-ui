# Mission: Account Recovery

**Objective:** Verify the "Forgot Password" flow works end-to-end.

## Target
- **URL:** `/ui/login`
- **User:** `recovery-agent@example.com`

## Execution Steps

### Phase 0: Setup (CRITICAL)
1. **Ensure User Exists:** Run the setup script to create a fresh user.
   ```bash
   ./tools/copilot-testing/scripts/setup-recovery-user.sh
   ```

### Phase 1: Request
1. Go to Login (`/ui/login`).
2. Click "Forgot password".
3. Enter email `recovery-agent@example.com`.
4. Submit.
5. Verify you see the "Enter code" screen.

### Phase 2: Retrieval
1. **Action:** Retrieve the recovery code from the email system using the helper script.
   ```bash
   ./tools/copilot-testing/scripts/fetch-recovery-code.sh "recovery-agent@example.com"
   ```
2. **Action:** Enter the returned 6-digit code into the UI.
3. Submit.

### Phase 3: Completion
1. Enter new password `NewPassword123!`.
2. Confirm new password.
3. Submit.
4. **Verification:** Ensure you are redirected to the "Secure your account" (MFA setup) page or the Login/Settings page.

# Mission: User Settings

**Objective:** Verify authenticated user can manage credentials.

## Execution Steps

### 1. Change Password
1. Navigate to `/ui/manage_password` (or find link in Dashboard).
2. Enter new password.
3. Verify success message.

### 2. Setup TOTP
1. Navigate to `/ui/manage_secure`.
2. Click "Add Authenticator".
3. **Action:** Ask User for a TOTP code corresponding to the secret shown on screen (or try to generate one if you can extract the secret).
   - *Tip:* `playwright_evaluate` can scrape the `innerText` of the secret code element.
4. Enter code.
5. Verify TOTP is active.

### 3. Backup Codes
1. Navigate to `/ui/manage_backup_codes`.
2. Click "Generate".
3. Verify codes are displayed.
4. **Chaos:** Reload the page. Do the codes persist or disappear (security check)?

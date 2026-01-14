# Mission: Session Tampering

**Objective:** Verify resilience against cookie theft and CSRF attacks.

## Execution Steps

### Attack 1: Stale CSRF
1. Open Login page.
2. Wait 10 minutes (or simulate time travel?). *Alternative:* Delete the `csrf_token_...` cookie using `playwright_evaluate`: `document.cookie = "csrf_token_...=; expires=Thu, 01 Jan 1970 00:00:00 UTC;"`.
3. Try to submit the form.
4. **Verify:** Should get a 401 or "Session expired", then redirect to a fresh flow. NOT a 500 error.

### Attack 2: Flow ID Tampering
1. Start Login.
2. Edit the URL: Change `?flow=123...` to `?flow=999...` (Invalid ID).
3. Hit Enter.
4. **Verify:** Should see "Flow expired" or 404/410 page.

### Attack 3: Unauthorized Access
1. Clear all cookies (`context.clearCookies()`).
2. Try to go directly to `/ui/manage_password`.
3. **Verify:** Immediate redirect to Login.

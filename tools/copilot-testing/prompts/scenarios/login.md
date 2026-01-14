# Mission: Login Verification

**Objective:** Verify that a user can log in under various configurations.

## Target
- **URL:** `http://127.0.0.1:4446/` (Hydra Test Client)
- **User:** `test@example.com` / `password123` (Create this user if missing!)

## Execution Steps

### Phase 1: Initiation
1. Navigate to the Hydra Test Client.
2. Start an "Authorization Code" flow.
3. Verify you land on the Login UI.

### Phase 2: Authentication
1. Enter credentials.
2. **Observation:** Check if you are redirected to MFA or straight to success.
   - If **MFA Enforced** -> Verify 2FA screen appears.
   - If **MFA Optional** -> Verify Success or MFA Setup prompt.

### Phase 3: Verification
1. Ensure you end up back at the Hydra Client.
2. Verify the "Authorization Code" is present in the URL or page.

## Agent Instructions
- Use `playwright_fill` for inputs.
- Use `playwright_click` for buttons.
- If the flow fails, take a `playwright_screenshot` and report the error text.

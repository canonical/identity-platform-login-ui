# Mission: Chaos Login

**Objective:** Attempt to break the login flow via browser manipulation.

## Execution Steps

### Attack 1: The "Back Button" Race
1. Start Login Flow.
2. Enter Email/Password.
3. Click "Sign In".
4. **IMMEDIATELY** click browser "Back".
5. Click browser "Forward".
6. **Verify:** Does the UI crash? Is the CSRF token still valid?

### Attack 2: Double Submit
1. Fill the form.
2. Try to click "Sign In" twice rapidly (use `playwright_evaluate` to run `document.querySelector('button').click(); document.querySelector('button').click();`).
3. **Verify:** Do we get a 500 error?

### Attack 3: Input Fuzzing
1. Enter a password with 5000 characters.
2. Enter an email with emojis: `test🤡@example.com`.
3. **Verify:** Graceful error message (400), NOT a server crash (500).

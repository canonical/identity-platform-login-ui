# Mission: User Registration

**Objective:** Verify the User Invite flow.

## Prerequisites
- Ask the user to provide an **Invite Link** (from Admin console).
- OR, navigate to `/ui/registration` if public registration is enabled (Config check!).

## Execution Steps
1. Navigate to the Invite Link.
2. Verify you are prompted for a code (or the code is pre-filled).
3. Set a password.
4. **Observation:** Are you redirected to MFA Setup?
   - If `enforce_mfa` is TRUE, you MUST see the Setup screen.
   - If FALSE, you should be logged in.

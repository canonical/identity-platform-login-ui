# Copilot / Agent Testing Framework

> **Target Audience**: AI Agents (Copilot, Windsurf, generic LLM agents).
> **Purpose**: Instructions and tools for verifying the Identity Platform Login UI functionality via interactive agentic sessions.

## 🤖 Agent Instructions: How to Test

You are acting as a QA Automation Engineer. Your goal is to verify the login flows by interactively driving the browser (using MCP tools) and managing test data (using scripts).

### 1. Test Data Setup (Bootstrap)

Before running UI tests, ensure you have a valid user. **Do not create users manually via UI unless testing registration flows.**

**Recommended Script:**
```bash
# Creates 'webauthn-tester@example.com'
# - Sets password: 'Password123!'
# - Registers a Virtual Authenticator (WebAuthn)
# - Sets up TOTP (Secret printed in output)
node tools/copilot-testing/scripts/register-webauthn.js
```

**Output to Capture:**
- **Email**: `webauthn-tester@example.com`
- **Password**: `Password123!`
- **TOTP Secret**: (e.g., `H7LE...`)

### 2. Interactive Testing (The "Test Run")

Use your `mcp_playwright` tools to drive the browser.

#### A. Standard Login (Password + TOTP)
1.  Navigate to `/ui/login`.
2.  Enter Email/Password.
3.  **Challenge Screen**: You will see "Verify your identity".
4.  **Generate TOTP**: Run in terminal:
    ```bash
    oathtool --totp -b <SECRET_FROM_STEP_1>
    ```
5.  Enter the code and submit.

#### B. WebAuthn Login (Password + Security Key)
1.  Navigate to `/ui/login`.
2.  Enter Email/Password.
3.  **Challenge Screen**:
    - *Note*: If TOTP is enabled, the UI defaults to the TOTP input.
    - **Action**: Look for "I want to use another method" or "Sign in with Security Key".
    - ⚠️ **Known Bug**: If the button is missing, report it. The Agent cannot physically touch a key, but if the session allows (CDP), it might auto-trigger.
    - **Fallback**: Initial registration script uses CDP to inject a virtual key. Standard MCP sessions **DO NOT** support virtual keys unless specifically bridged. *Focus on UI availability checking.*

### 3. State Diagrams & Flows

> See [FLOWS.md](./FLOWS.md) for detailed diagrams of the state transitions.

Use the state diagrams to understand where you are in the application logic (e.g., "Am I in the MFA Challenge state?").

## 🛠️ Tool Inventory

| Script | Purpose |
|---|---|
| `scripts/register-webauthn.js` | **Primary Bootstrap**. Resets user, adds Passkey & TOTP. |
| `scripts/setup-user.sh` | Simple user creation (No MFA). |
| `scripts/fetch-recovery-code.sh` | Retreives recovery codes for a user (useful for backup code flow). |

## 🐞 Known Issues to Watch For

1.  **MFA Switch Hidden**: When both TOTP and WebAuthn are enabled, the link to switch to WebAuthn might be missing from the TOTP screen.
2.  **Session Persistence**: Cookies are encrypted. To "Logout", use `Clear Cookies` or incognito context.

## 📝 Reporting Results

When you finish a session:
1.  Summarize which flows passed.
2.  List any UI elements that were unreachable.
3.  If a bug was found (like the MFA switch), document the reproduction steps explicitly.

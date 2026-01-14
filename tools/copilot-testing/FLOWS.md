# Identity Platform Login UI - State Diagrams

This document describes the state transitions of the Login UI to assist AI Agents in testing and navigation.

## 1. Standard Login Flow (MFA Enabled)

```mermaid
stateDiagram-v2
    [*] --> Login

    state Login {
        [*] --> EnterCredentials
        EnterCredentials --> MFA_Challenge : Valid Password
        EnterCredentials --> LoginError : Invalid Password
    }

    state MFA_Challenge {
        [*] --> CheckDefaultMethod
        CheckDefaultMethod --> TOTP_Screen : MFA Enabled (Default)
        CheckDefaultMethod --> WebAuthn_Screen : WebAuthn Auto-trigger

        TOTP_Screen --> WebAuthn_Screen : "Sign in with Security Key" (User Action)
        TOTP_Screen --> Success : Valid TOTP

        WebAuthn_Screen --> TOTP_Screen : "Use another method" (User Action)
        WebAuthn_Screen --> Success : Valid Assertion
    }

    Success --> [*]
```

## 2. 2FA Method Selection Logic

When a user lands on the MFA Challenge page, the UI decides which method to show:

1. **TOTP (Authenticator App)**:
   - Shown by default if `isAuthCode` is true (TOTP nodes exist).
   - *Known Issue*: If TOTP is shown, the "Sign in with Security Key" button might be hidden until manually triggered or fixed.

2. **WebAuthn (Security Key)**:
   - Shown if `isWebauthn` is true.
   - Triggers browser/hardware interaction immediately.

3. **Backup Codes**:
   - Accessed via "Use backup code" or "I cannot access my authenticator" links.

## 3. WebAuthn Registration Flow

```mermaid
stateDiagram-v2
    [*] --> Settings
    Settings --> AddKey : Click "Add Security Key"

    state AddKey {
        [*] --> NameKey : Enter Name
        NameKey --> PromptBrowser : Submit
        PromptBrowser --> Success : Touch Key
        PromptBrowser --> Error : Timeout/Cancel
    }

    Success --> Settings : Key Listed
```

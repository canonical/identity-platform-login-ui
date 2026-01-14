# AI Chaos Agent

A Heuristic State-Machine Agent for chaos testing the Identity Platform Login UI.

## Latest Updates ✨

**2026-01-13:** Phase 1.2 Complete - WebAuthn Flow  
- Full end-to-end WebAuthn (security key) testing
- Supports both 2FA and passwordless modes
- Virtual authenticator via Chrome DevTools Protocol
- Automatic user presence simulation
- See `PHASE_1.2_COMPLETE.md` for details

**2026-01-13:** Phase 1.1 Complete - Password Recovery Flow  
- Full end-to-end password reset testing
- Automatic email code extraction from Mailslurper
- Enhanced EmailService with robust polling
- See `PHASE_1.1_COMPLETE.md` for details

## Features

### Validated Flows ✅
1. **OIDC Flow** - OAuth2 authorization with Hydra
2. **Standalone Login** - Direct login with TOTP verification
3. **Password Recovery** - Full password reset with email verification
4. **WebAuthn** - Security key registration and login (2FA + passwordless) ✨ NEW

### Coming Soon 🚧
- Negative test scenarios (invalid inputs, expired codes)
- Configuration matrix testing (different feature flags)
- Playwright test generation from discovered flows

## Installation

```bash
cd tools/chaos-agent
npm install
```

## Quick Start

### Test WebAuthn Flows ✨ NEW
```bash
# 1. Registration + 2FA login (register keys, logout, re-login with WebAuthn)
npm run test:webauthn-register

# 2. WebAuthn-only login (user has WebAuthn, no TOTP)
npm run test:webauthn-login
```

**Note:** Run `npm run setup-identities` first to create test users!

### Test Password Recovery
```bash
# With visible browser
npm run test:recovery

# Headless (CI-friendly)
npm run test:recovery-headless
```

### Test Other Flows
```bash
# Standalone login
npm run test:login

# OIDC flow
npm run test:oidc
```

## Usage

### Validation Mode (Happy Paths)
Run specific happy path validations (Login, Recovery).

```bash
npm start -- --mode=validation --url=http://localhost/ui/login
```

### Exploration Mode (Monkey)
Run weighted random exploration.

```bash
npm start -- --mode=exploration --duration=300
```

### Custom Test
```bash
npm start -- --mode=validation --url=<URL> --duration=<seconds> [--headed]
```

## Debugging

### Using Playwright MCP Server
For complex debugging, use the Playwright MCP server tools to inspect the browser state while the agent is running:
- `playwright-browser_snapshot`: View the accessibility tree
- `playwright-browser_evaluate`: Check element properties
- `playwright-browser_take_screenshot`: See what the agent sees

### Timeouts
**Crucial:** Always run chaos agent tests with a system timeout to prevent hanging processes if the agent gets stuck:
```bash
timeout 60 npm start -- --mode=validation ...
```

## Documentation

- **Quick Start:** This file
- **Testing Guides:** 
  - `WEBAUTHN_TESTS.md` (4 comprehensive test scenarios) ✨ NEW
  - `WEBAUTHN_TEST.md` (security key authentication) 
  - `RECOVERY_TEST.md` (password recovery)
- **Implementation Details:** 
  - `WEBAUTHN_IMPLEMENTATION.md`
  - `RECOVERY_IMPLEMENTATION.md`
- **Complete Guide:** `IMPLEMENTATION_GUIDE.md`
- **State Flows:** `STATE_FLOWS.md`
- **All Fixes:** `FIXES_SUMMARY.md`

## Prerequisites

Ensure services are running:
```bash
cd ../..  # Go to project root
docker compose -f docker-compose.dev.yml up -d
```

Required services:
- **Kratos** (port 4433/4434) - Identity management
- **Hydra** (port 4444/4445) - OAuth2 server
- **Mailslurper** (port 4436/4437) - Email testing (required for recovery flow)

## QA Persona (Test Generation)
Generate a Playwright test scaffold using GitHub Copilot.

```bash
make generate-test
```

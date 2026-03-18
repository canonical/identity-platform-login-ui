# Skill: debug-browser-errors

Use this skill to diagnose a frontend bug where a server error is not being surfaced
correctly in the UI, or where unexpected behaviour is visible in the browser.

**Invoke with a known issue:**
> `Use the debug-browser-errors skill to investigate: <issue description>`

**Invoke in observe mode (watch me work):**
> `Use the debug-browser-errors skill — open http://localhost/ui/login in the browser,
> monitor the console and network while I walk through the steps, then help me understand
> what went wrong.`

---

## Feature Flags

These flags change which flow steps are active. Check them first when a redirect or
flow step is missing or unexpected.

| Env var                           | Default | Effect                                         |
|-----------------------------------|---------|------------------------------------------------|
| `MFA_ENABLED`                     | `true`  | Enables TOTP/WebAuthn/backup code steps        |
| `IDENTIFIER_FIRST_ENABLED`        | `true`  | Splits login into identifier then password     |
| `OIDC_WEBAUTHN_SEQUENCING_ENABLED`| `false` | Adds passkey sequencing step in OIDC login     |
| `FEATURE_FLAGS`                   | all     | Comma-separated list: password, webauthn, etc. |
| `AUTHORIZATION_ENABLED`           | `false` | Enables OpenFGA authz checks                   |

---

## Workflow

### Step 1 — Confirm the dev stack is running

```bash
docker compose -f docker-compose.dev.yml ps && curl -s http://localhost:4455/api/v0/status
```

If not running, use the `setup-dev-env` prompt.

### Step 2 — Reproduce the issue with Chrome DevTools MCP

Use the `chrome-devtools` MCP server to open the app and walk through the
reproduction steps from the issue. Capture:

1. **Console errors** — note the full message, stack trace, and source file
2. **Network failures** — for each failed request, record:
   - Method, URL, status code
   - Request body (form data or JSON)
   - Response body (the full JSON, especially any `error`, `id`, or `reason` fields)
3. **Screenshots** at the point of failure

Example prompt to the MCP:
```
Open http://localhost/ui/reset_password, fill in the current password field
with "test", fill the new password with "test", submit the form, and capture all
network requests and console errors.
```

In **observe mode**, open the URL and keep the DevTools session open. Ask the user to
perform the steps themselves, then call `console_messages` and `network_requests` after
they are done to capture what happened.

### Step 3 — Check the backend logs **before forming any hypothesis**

> **Critical**: the response body the Go backend returns on a 4xx/5xx is often a
> secondary error (e.g. a JSON unmarshal failure on an upstream Kratos error response).
> The true root cause — including the `reason`, `ory-error-id`, and the exact failing
> condition — is almost always in the Kratos or Hydra container log. **Do not attempt
> to explain the bug from the network response body alone.** Check logs first.

For every 4xx/5xx observed in step 2, immediately run:

```bash
docker logs $(docker ps -qf name=kratos) --since 5m 2>&1 | grep -v health | tail -20
docker logs $(docker ps -qf name=hydra) --since 5m 2>&1 | grep -v health | tail -20
```

Go backend logs go to the VS Code Debug Console (F5) or the `./app serve` terminal.
Set `LOG_LEVEL=debug` in `.vscode/launch.json` to log full upstream response bodies.

The Kratos entry contains `reason`, `error.id`, and `ory-error-id` — use those to
drive the hypothesis, not the browser response body.

**Caution — browser sessions**: if the DevTools MCP reuses an existing Chrome session,
you may land on a mid-flow page (e.g. TOTP setup). Open a fresh incognito/guest
window in Chrome and reconnect before repeating the reproduction steps if the
reproduced behaviour looks wrong.

### Step 4 — Map the error to the Go handler

From the failing network request URL and the backend logs, identify the Go handler:

| URL pattern                              | Handler package      | File                        |
|------------------------------------------|----------------------|-----------------------------|
| `/api/kratos/self-service/*`             | `pkg/kratos`         | `handlers.go`               |
| `/api/consent`                           | `pkg/extra`          | `handlers.go`               |
| `/api/device`                            | `pkg/device`         | `handlers.go`               |

Look at the handler's response when it receives an error from Kratos. Check whether
it returns the updated Kratos flow object (good — the frontend can display the error
nodes) or returns a plain HTTP error status (bad — the frontend has nothing to render).

### Step 5 — Map the error to the React component

If the server returned a flow object but the UI shows no error, check in order:
- `ui/components/Flow.tsx` — passes message nodes to `Node.tsx`?
- `ui/components/Node.tsx` — is the node `group` or `type` being filtered out?
- `NodeInput*.tsx` — renders `meta.messages` array?

If the server returned a plain HTTP error (no flow object), fix the Go handler to
forward the Kratos error flow instead of returning its own error response.

### Step 6 — Propose the fix

Provide:
- The exact file(s) and line(s) to change
- The minimal change needed (no refactors)
- A one-line test case to verify the fix: either a Go unit test asserting the handler
  returns the flow object on error, or a Playwright E2E step that checks the error
  message appears in the UI

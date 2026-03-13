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

## Recommended Entry Sequence

If you are starting from scratch — not from a known issue — use these steps in order:

1. **Start the dev stack** — use the `setup-dev-env` prompt if the stack is not running
2. **Open a monitored browser session** — use the `monitor-browser` prompt to open the
   app, capture traffic, and have Copilot watch while you work:
   > `Use the monitor-browser prompt — I want to walk through the registration flow`
3. **Diagnose a specific failure** — once you see something unexpected, invoke this skill:
   > `Use the debug-browser-errors skill to investigate what I just saw`

Without step 2, you need to know the reproduction steps in advance and write them
explicitly. Observe mode removes that requirement.

---

## When to Use This Skill

- A page shows no error when one is expected (e.g. form submission silently fails)
- A network request returns 4xx/5xx but the UI does not update
- A console error appears but the root cause is unclear
- A redirect happens unexpectedly
- You want to walk through a flow and have Copilot explain what is happening

---

## Workflow

### Step 1 — Confirm the dev stack is running

Before using the browser tools, verify the full stack is up:

```bash
docker compose -f docker-compose.dev.yml ps     # all services should be Up
curl -s http://localhost:4455/api/v0/status      # Go backend health check
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
# Kratos — identity flows (login, registration, recovery, settings, MFA)
docker logs $(docker ps -qf name=kratos) --tail 30 2>&1

# Hydra — OAuth2/OIDC consent and device flows
docker logs $(docker ps -qf name=hydra) --tail 30 2>&1
```

**Go backend** — logs go to the terminal where `./app serve` was started (or the
VS Code Debug Console if launched via F5). Look for lines at `error` or `warn` level.
Set `LOG_LEVEL=debug` in `.vscode/launch.json` to get the full upstream response body
logged on every Kratos/Hydra call.

Cross-reference timestamps between the browser network request and the container log
entries to confirm you are looking at the right event. The Kratos log entry will
contain fields like `reason`, `error.id`, and `ory-error-id` that identify the
specific failure precisely — use those to drive the hypothesis, not the browser response.

**Caution — browser sessions**: if the DevTools MCP reuses an existing Chrome session,
you may land on a mid-flow page (e.g. TOTP setup) instead of a fresh flow. If the
reproduced behaviour looks wrong, open an isolated context with no cookies before
repeating the reproduction steps.

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

If the server **did** return a flow object but the UI still shows no error:

1. Check `ui/components/Flow.tsx` — does it pass message nodes to `Node.tsx`?
2. Check `ui/components/Node.tsx` — is the node `group` or `type` being filtered out?
3. Check the relevant `NodeInput*.tsx` — does it render the `messages` array from the
   node's `meta.messages`?

If the server returned a **plain HTTP error** (no flow object), the fix is in the
Go handler: it should call Kratos, receive the error flow, and forward it to the
frontend rather than returning its own error response.

### Step 6 — Propose the fix

Provide:
- The exact file(s) and line(s) to change
- The minimal change needed (no refactors)
- A one-line test case to verify the fix: either a Go unit test asserting the handler
  returns the flow object on error, or a Playwright E2E step that checks the error
  message appears in the UI

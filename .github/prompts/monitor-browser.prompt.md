# Monitor Browser Session

Use this prompt to start a live monitored browser session before you know exactly
what is wrong. You walk through the app normally; Copilot watches the network and
console and summarises what happened.

---

## What Copilot will do

1. Use the `chrome-devtools` MCP to connect to the running Chrome instance
2. Open the requested URL in a **new tab** (use an incognito/guest window to avoid cached session cookies)
3. Keep the DevTools session open and monitor all console messages and network traffic
4. Wait while you navigate and interact with the app
5. **Proactively** capture and report if — without waiting for "analyse this":
   - The page URL changes to `/ui/error` (error page shown to the user)
   - Any 4xx/5xx network response is observed
6. When you say **"done"** or **"analyse this"**, capture the full console + network log and summarise: requests made, any 4xx/5xx responses, console errors, redirects.
7. Close the browser session early once the relevant requests are captured to save tokens.

## How to invoke

> `Use the monitor-browser prompt — I want to walk through the [registration / login /
> password reset / MFA setup] flow and see what happens`

Specify the starting URL if it is not the login page — e.g.:
> `Use the monitor-browser prompt, start at http://localhost/ui/recovery`

## After the session

Once Copilot summarises what it captured, if anything looks wrong:

> `Something went wrong when I submitted the form — help me understand why`

Use the `debug-browser-errors` skill from step 3 onwards — it has the log commands
and explains how to correlate Kratos/Hydra entries with the network response.

## Tips

- **Always use a clean session** — if you land on a mid-flow page (TOTP setup,
  settings) instead of the login screen, you are in an existing session. Open an
  incognito/guest window in Chrome first, then ask:
  > `Use the monitor-browser prompt, start at http://localhost/ui/login`
- **Enable debug logging** before starting: set `LOG_LEVEL=debug` in
  `.vscode/launch.json` so the Go backend logs full Kratos/Hydra request/response
  bodies. This makes log correlation much faster.
- **Combine with the setup prompt**: if the stack is not running, use the
  `setup-dev-env` prompt first, then come back here.

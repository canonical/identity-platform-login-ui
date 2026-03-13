# Monitor Browser Session

Use this prompt to start a live monitored browser session before you know exactly
what is wrong. You walk through the app normally; Copilot watches the network and
console and summarises what happened.

---

## What Copilot will do

1. Open the requested URL in an **isolated browser context** (no cached cookies or
   existing sessions) using the `chrome-devtools` MCP server
2. Keep the DevTools session open and monitor all console messages and network traffic
3. Wait while you navigate and interact with the app
4. **Proactively** capture and report if — without waiting for "analyse this":
   - The page URL changes to `/ui/error` (error page shown to the user)
   - Any 4xx/5xx network response is observed
5. When you say **"done"** or **"analyse this"**, capture the full console + network
   log and summarise: requests made, any 4xx/5xx responses, console errors, redirects

## How to invoke

> `Use the monitor-browser prompt — I want to walk through the [registration / login /
> password reset / MFA setup] flow and see what happens`

Specify the starting URL if it is not the login page — e.g.:
> `Use the monitor-browser prompt, start at http://localhost/ui/recovery`

## After the session

Once Copilot summarises what it captured, if anything looks wrong:

> `Something went wrong when I submitted the form — help me understand why`

For any 4xx/5xx or `/ui/error` redirect observed, **immediately** check container logs
before drawing conclusions from the network response body:

```bash
docker logs $(docker ps -qf name=kratos) --since 5m 2>&1 | grep -v health
docker logs $(docker ps -qf name=hydra) --since 5m 2>&1 | grep -v health
```

The Kratos log entry will contain `reason`, `ory-error-id`, `flow_method`, and
`identity_id` — these are the canonical source of truth for what went wrong, not
the Go backend's response body (which is often a secondary unmarshal error).

For the full diagnosis workflow see the `debug-browser-errors` skill.

## Tips

- **Always use an isolated context** — if you land on a mid-flow page (TOTP setup,
  settings) instead of the login screen, you are in an existing session. Ask:
  > `Open an isolated browser with no cookies and go to http://localhost/ui/login`
- **Enable debug logging** before starting: set `LOG_LEVEL=debug` in
  `.vscode/launch.json` so the Go backend logs full Kratos/Hydra request/response
  bodies. This makes log correlation much faster.
- **Combine with the setup prompt**: if the stack is not running, use the
  `setup-dev-env` prompt first, then come back here.

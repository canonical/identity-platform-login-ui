---
description: Quick-start reference for getting the full dev stack running locally.
---

# Setup Dev Environment

Use this prompt to get oriented quickly when setting up for the first time or after
a machine restart.

---

## Prerequisites

Create a `.env` file in the repo root with your GitHub OAuth app credentials (needed
for Kratos GitHub social login). Other flows (password, WebAuthn, TOTP) work without it.

```bash
echo "CLIENT_ID=<your_client_id>" >> .env
echo "CLIENT_SECRET=<your_client_secret>" >> .env
```

Register a new app at https://github.com/settings/applications/new with Authorization
callback URL: `http://localhost:4433/self-service/methods/oidc/callback/github`.

---

## Step 1 — Start the dependency stack

```bash
docker compose -f docker-compose.dev.yml up -d
```

Verify services are healthy:

```bash
curl -s http://localhost:4433/health/ready   # → {"status":"ok"}
curl -s http://localhost:4445/health/ready   # → {"status":"ok"}
```

Expected services:

| Service       | Port  | Ready indicator                          |
|---------------|-------|------------------------------------------|
| Traefik       | 80    | Reverse proxy — no explicit health check |
| Kratos        | 4433  | `GET http://localhost:4433/health/ready` |
| Kratos admin  | 4434  | Same health endpoint                     |
| Hydra admin   | 4445  | `GET http://localhost:4445/health/ready` |
| PostgreSQL    | 5432  | Kratos/Hydra migrations run on startup   |
| Mailslurper   | 4436  | Open `http://localhost:4436` to confirm  |
| OpenFGA       | 8080  | Internal — used if AUTHORIZATION_ENABLED |

## Step 2a — Frontend developer (Next.js dev server with HMR)

Install dependencies if `ui/node_modules` doesn't exist yet:

```bash
cd ui && npm install
```

Then start the dev server:

```bash
cd ui && DEV=true npm run dev   # Next.js on :3001, enables proxy rewrites to Go backend at :4455
```

`DEV=true` is **required** — without it the proxy rewrites are disabled and API calls fail.

Then also run the Go backend (Step 2b). Open the app at `http://localhost:3001` for
hot-module replacement during active frontend work.

## Step 2b — Build & run the Go backend

The easiest way is the VS Code **"Launch Package"** debug config (F5) — it has all env
vars pre-filled from `.vscode/launch.json` and supports breakpoints.

For a manual run, always use `make` (not `go build ./main.go` directly):

```bash
make npm-build build   # builds Next.js static export, then compiles Go binary
./app serve            # see env vars section below
```

> **Why not `go build ./main.go` directly?**
> `cmd/serve.go` embeds `ui/dist` via `//go:embed`. A plain `go build` fails unless
> `cmd/ui/dist/` contains the real Next.js output. `make build` copies it there
> automatically — `go build` skips this step.

If the binary was already running, **restart it** after rebuilding:

```bash
pkill -f "./app serve" && ./app serve   # (with env vars)
```

## Step 3 — Open the app

| Mode | URL | Notes |
|------|-----|-------|
| Frontend dev (HMR) | `http://localhost:3001` | Changes reflect instantly; use this during active frontend editing |
| Full stack via Traefik | `http://localhost` | Requires `make npm-build build`; needed for OAuth redirect flows |

Prefer `:3001` during active frontend editing. Use `http://localhost` when testing
OAuth/OIDC flows or anything that depends on Traefik routing.

## Required Environment Variables

The VS Code launch config pre-fills all of these. For manual runs, read them from
`.vscode/launch.json`. Key variables:

| Variable                   | Required | Default in `launch.json`        | Notes                              |
|----------------------------|----------|---------------------------------|------------------------------------|
| `COOKIES_ENCRYPTION_KEY` | Yes      | dev value in `launch.json`  | Exactly 32 bytes                   |
| `KRATOS_PUBLIC_URL`      | Yes      | `http://localhost:4433`     |                                    |
| `KRATOS_ADMIN_URL`       | Yes      | `http://localhost:4434`     |                                    |
| `HYDRA_ADMIN_URL`        | Yes      | `http://localhost:4445`     |                                    |
| `BASE_URL`               | Yes      | `http://localhost/`         |                                    |
| `PORT`                   | No       | `4455`                      | Binary default is `8080`           |
| `MFA_ENABLED`            | No       | `true`                      | Set to `false` to simplify login   |
| `IDENTIFIER_FIRST_ENABLED` | No     | `true`                      | Splits login into identifier + password steps |
| `AUTHORIZATION_ENABLED`  | No       | `false`                     | Set to `true` to enable OpenFGA    |
| `LOG_LEVEL`              | No       | `debug`                     | Already verbose in launch config   |

To change any of these for your session, edit `.vscode/launch.json` — the values persist across restarts.

## Quick Sanity Checks

```bash
curl -s http://localhost:4433/health/ready   # Kratos → {"status":"ok"}
curl -s http://localhost:4455/api/v0/status  # Go backend → {"status":"ok",...}
curl -s -o /dev/null -w "%{http_code}" http://localhost:3001/ui/login  # → 200
curl -s -o /dev/null -w "%{http_code}" http://localhost/ui/login       # → 200 (needs make npm-build build)
# Open http://localhost:4436 to check Mailslurper (view test emails)
```

## Useful Make Targets

```bash
make npm-build    # Build Next.js static export → ui/dist/ (run before make build)
make build        # Copy ui/dist → cmd/ui/dist/, compile Go binary → ./app
make test         # Go unit tests
make test-e2e     # Playwright E2E tests (full stack must be running)
make mocks        # Regenerate gomock mocks after changing an interface
```

---

## Next Steps

Once the stack is running, use these prompts and skills in order:

| What you want to do                              | Use                          |
|--------------------------------------------------|------------------------------|
| Open the app in a browser and watch what happens | `monitor-browser` prompt     |
| Debug a specific error you observed              | `debug-browser-errors` skill |
| Compare a page against its Figma design          | `compare-design` skill       |


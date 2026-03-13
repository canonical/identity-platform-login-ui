---
description: Quick-start reference for getting the full dev stack running locally.
---

# Setup Dev Environment

Use this prompt to get oriented quickly when setting up for the first time or after
a machine restart.

---

## Step 1 — Start the dependency stack

```bash
docker compose -f docker-compose.dev.yml up
```

Wait until all services are healthy. Expected services:

| Service       | Port  | Ready indicator                          |
|---------------|-------|------------------------------------------|
| Traefik       | 80    | Reverse proxy — no explicit health check |
| Kratos        | 4433  | `GET http://localhost:4433/health/ready` |
| Kratos admin  | 4434  | Same health endpoint                     |
| Hydra admin   | 4445  | `GET http://localhost:4445/health/ready` |
| PostgreSQL    | 5432  | Kratos/Hydra migrations run on startup   |
| Mailslurper   | 4436  | Open `<http://localhost:4436>` to confirm    |
| OpenFGA       | 8080  | Internal — used if AUTHORIZATION_ENABLED |

## Step 2a — Frontend developer (Next.js dev server)

```bash
cd ui && npm run dev     # Next.js on :3000, DEV=true enables proxy rewrites
```

Then also run the Go backend (Step 2b) — the frontend proxies API calls to it.

## Step 2b — Backend developer (Go)

Either use VS Code "Launch Package" debug config (F5), or:

```bash
make npm-build build && ./app serve
```

The launch config in `.vscode/launch.json` has all env vars pre-filled.

## Step 3 — Open the app

Always open `<http://localhost>` (Traefik on port 80), not `:3000` or `:4455` directly.
Kratos sets session cookies scoped to the Traefik domain.

## Required Environment Variables

The VS Code launch config pre-fills all of these. For manual runs:

| Variable                 | Required | Launch config default       | Notes                              |
|--------------------------|----------|-----------------------------|------------------------------------|
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
curl -s http://localhost:4433/health/ready   # Kratos
curl -s http://localhost:4455/api/v0/status  # Go backend
# Open <http://localhost:4436> to check Mailslurper (view test emails)
# Open <http://localhost/ui/login> to view the app
```

## Useful Make Targets

```bash
make test         # Go unit tests
make test-e2e     # Playwright E2E tests (full stack must be running)
make npm-build    # Build Next.js static export → ui/dist/
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


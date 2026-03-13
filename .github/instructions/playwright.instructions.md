---
applyTo: "ui/tests/**"
---

# Playwright E2E Tests — Scoped Instructions

These rules apply when Copilot is editing files under `ui/tests/`.

## Configuration Essentials

- **`baseURL`**: `http://localhost:2345/login/` as configured in `playwright.config.ts`.
  Port 2345 is an **OIDC client application** (e.g. Grafana) — tests exercise the full
  OAuth2/OIDC redirect flow: OIDC client → Hydra consent → Kratos login → redirect back.
  This service is **not** in `docker-compose.dev.yml`, so E2E tests currently require a
  Juju/Kubernetes environment (or a local port-forward). **TODO**: add a lightweight OIDC
  client to the dev compose stack.
  Either way, always target Traefik — never `:3000` directly.
- **`workers: 1`**: tests run sequentially on purpose. They share a live Kratos instance
  and manipulate real identities. Parallel execution causes state conflicts.
- **`retries: 2` in CI**, 0 locally. Allow for test flakiness in CI but fail fast locally.
- Screenshots saved to `tests/__screenshots__/{testFileName}/{arg}{ext}` — used for
  visual regression. Don't delete them manually; update with `--update-snapshots`.

## Available Helpers (`ui/tests/helpers/`)

Use these instead of re-implementing common operations:

| Helper file         | What it provides                                      |
|---------------------|-------------------------------------------------------|
| `login.ts`          | Log in a user through the full Kratos browser flow    |
| `kratosIdentities.ts` | Create / delete Kratos identities via the admin API |
| `totp.ts`           | Set up and supply TOTP codes                         |
| `backupCode.ts`     | Retrieve and use backup codes                        |
| `mail.ts`           | Read emails from Mailslurper (`:4436`)               |
| `password.ts`       | Password reset helpers                               |
| `oidc_client.ts`    | Start an OIDC client session                         |
| `name.ts`           | Generate random test identity names                  |

## Test Structure

- Each spec file covers exactly one user-facing flow (e.g. `login-first-time.spec.ts`).
- Create / clean up identities within the test using `kratosIdentities` helpers —
  never rely on pre-existing state in the database.
- Use `test.beforeEach` / `test.afterEach` for identity lifecycle, not `beforeAll`.
- `maxFailures: 3` — the suite aborts early to avoid cascading failures during CI.

## What Requires the Full Stack

E2E tests require all services running:
```bash
docker compose -f docker-compose.dev.yml up   # Kratos, Hydra, Traefik, Postgres, Mailslurper
./app serve                                   # Go backend on :4455
cd ui && npm run dev                          # Next.js on :3000 (if testing frontend dev build)
```

Or run against the production build:
```bash
make npm-build build && ./app serve
```

Both modes are covered by `make test-e2e`.

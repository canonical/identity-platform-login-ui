# Identity Platform Login UI - AI Coding Agent Instructions

> **Additional Documentation**: See `.github/instructions/` for detailed guides on architecture patterns, testing, and workflows.

## Project Overview

Full-stack authentication UI for Canonical Identity Platform. **Go backend (chi router)** + **React/Next.js frontend**. Integrates with Ory Kratos (identity), Ory Hydra (OAuth2), and OpenFGA (authorization).

**Key Technologies:**
- Backend: Go 1.24+, chi router, gomock, OpenTelemetry, Prometheus
- Frontend: Next.js 15.x, TypeScript 5.9+, Playwright, ESLint/Prettier
- Infrastructure: Kubernetes via Juju, Rockcraft for OCI images

## Architecture Quick Reference

**Directory Structure:**
- `cmd/`: Application entry (serve.go does dependency injection)
- `pkg/`: Public handlers (web, kratos, device, extra, metrics, status, ui)
- `internal/`: Private implementations (config, hydra, kratos, authorization, tracing, logging)
- `ui/`: React frontend (pages, components, api, tests)

**Key Patterns:**
- Service constructors: `NewService(deps..., tracer, monitor, logger)` - **tracer/monitor/logger always last**
- Interfaces: Every package has `interfaces.go` for dependencies
- Router: Option pattern in `pkg/web/router.go` with `RegisterEndpoints(mux)` methods
- Mocks: Generated via `//go:generate mockgen` directives, run `make mocks`

## Essential Commands

```bash
# Backend
make mocks          # Generate gomock mocks
make test           # Run Go tests
make build          # Build binary (requires UI built first)

# Frontend
make npm-build      # Build React app
make test-e2e       # Run Playwright E2E tests

# Combined
make npm-build build   # Full build (MUST be sequential, not parallel)

# Local dev
docker compose -f docker-compose.dev.yml up   # Start dependencies
./app serve                                    # Run application
```

## Code Conventions (Mandatory)

**Critical Rules - Non-Negotiable:**

1. **File Headers**: Every Go file starts with:
   ```go
   // Copyright 2026 Canonical Ltd.
   // SPDX-License-Identifier: AGPL-3.0
   ```

2. **Error Handling**:
   - Messages: lowercase verbs ("cannot fetch flow", not "Cannot Fetch Flow")
   - `%w` only if caller needs to inspect error; otherwise `%v`
   - Add context only at package boundaries
   - Log errors in handlers only, not in services
   - Return zero values with errors: `return nil, nil, err`

3. **Naming**:
   - Named receivers: `func (s *Service) Method()` not `func (*Service) Method()`
   - Concise variables: `flow` not `flowObject`
   - Doc comments on all functions: `"FunctionName does X."`

4. **Testing**:
   - Standard library `testing` only - NO testify/external assertions
   - Table-driven tests: `tests := []struct{...}{{...}}`
   - Mock all interfaces with gomock
   - Expect tracer.Start() with format: `"package.Type.Method"`

5. **Code Structure**:
   - Prefer `:=` over `var` unless zero value is intentional
   - Early returns, avoid nesting (no pyramids of doom)
   - Never use bare returns
   - Always pass/return struct pointers: `*Config` not `Config`

**Frontend (TypeScript/React)**:
- Functional components only: `FC<Props>` type
- No `any` types - use `unknown` or proper types
- Destructure props in signature
- No ESLint errors/warnings in production code
- 2-space indent, single quotes, trailing commas (Prettier enforced)

**See `.github/instructions/go.instructions.md` for complete details.**

### TypeScript/React Frontend Conventions

**Critical Rules:**
- Functional components only: `FC<Props>` type
- No `any` types - use `unknown` or proper types
- Destructure props in signature
- No ESLint errors/warnings in production code
- 2-space indent, single quotes, trailing commas (Prettier enforced)

**See `.github/instructions/react.instructions.md` for complete details.**

## Critical Domain Knowledge

### Kratos Integration

**Custom Extensions** (not in upstream SDK):
- `ExecuteIdentifierFirstUpdateLoginRequest()` - two-step login flow
- Cookie encryption via `pkg/kratos/cookie_manager.go` (AES-GCM)

**Flow Pattern** (all flows follow this):
```go
func (s *Service) CreateFlow(..., cookies []*http.Cookie) (*Flow, []*http.Cookie, error)
func (s *Service) UpdateFlow(..., cookies []*http.Cookie) (*Redirect, *Success, []*http.Cookie, error)
```
Input: `(context, params..., cookies)` → Output: `(data, cookies, error)`

### Hydra Integration

**Custom Device Flow** (`internal/hydra/device.go`):
- Manually implements OAuth device flow (not in upstream SDK)
- TODO: Remove when upstream supports it

### OpenFGA Authorization

- Enabled via `AUTHORIZATION_ENABLED` env var
- Model managed via `./app create-fga-model` command
- Schema in `internal/authorization/schema.openfga`

## Common Tasks

**Add New API Handler:**
1. Create `pkg/newfeature/interfaces.go`
2. Implement `pkg/newfeature/service.go` with `NewService(deps..., tracer, monitor, logger)`
3. Create `pkg/newfeature/handlers.go` with `RegisterEndpoints(mux)`
4. Wire into `pkg/web/router.go`
5. Add test files with `//go:generate mockgen` directives
6. Run `make mocks`

**Add Tracing:**
```go
ctx, span := s.tracer.Start(ctx, "package.Service.MethodName")
defer span.End()
```

**Add React Page:**
1. Create `ui/pages/newpage.tsx` (functional component)
2. Add `ui/api/newpage.ts` (API utilities)
3. Add `ui/tests/newpage.spec.ts` (Playwright E2E)

**See `.github/instructions/workflows.instructions.md` for detailed examples.**

## Environment Variables

**Required:**
- `KRATOS_PUBLIC_URL`, `KRATOS_ADMIN_URL`, `HYDRA_ADMIN_URL` - Service endpoints
- `BASE_URL` - Application base URL
- `COOKIES_ENCRYPTION_KEY` - 32-byte encryption key

**Key Settings:**
- `PORT` (default: 8080)
- `MFA_ENABLED` (default: true)
- `IDENTIFIER_FIRST_ENABLED` (default: true)
- `FEATURE_FLAGS` - Comma-separated: `password,webauthn,backup_codes,totp,account_linking`
- `LOG_LEVEL` - debug, info, error (default: error)
- `TRACING_ENABLED` (default: true)

**See `internal/config/specs.go` for complete list.**

## What NOT to Do

**Go Backend**:
- Don't create global mutable state - pass dependencies explicitly
- Don't use bare returns even with named return values
- Don't add context to every error - only at abstraction boundaries
- Don't mix `var` and `:=` declarations - use `:=` unless zero value is intentionally unread
- Don't use `testify` or other assertion libraries - use standard library `testing` only
- Don't modify working Kratos/Hydra integration code unless fixing a bug

**React Frontend**:
- Don't create class components - use functional components only
- Don't use `any` type - use `unknown` or proper types
- Don't commit with ESLint errors or warnings
- Don't use `console.log` in production code - remove debug statements
- Don't skip Playwright visual regression tests - they catch UI regressions

**General**:
- Don't make big refactors - make surgical, minimal changes
- Don't break existing API contracts between frontend and backend
- Don't change build scripts without testing both `make build` and `make npm-build build`
- Don't commit without running pre-commit hooks (`pre-commit install -t commit-msg`)

## Documentation Maintenance

**Self-Correction Directive**:
- If you establish a new pattern or convention during a conversation that isn't documented here, **you must update this file**.
- This ensures the instructions remain a living document and the single source of truth for project standards.
- Examples of updates: new error handling patterns, frontend component patterns, testing conventions, or architectural decisions.

---
applyTo: "ui/**"
---

# Frontend — Scoped Instructions

These rules apply whenever Copilot is editing files under `ui/`.

## How the Dev Server Proxies to the Go Backend

`next.config.js` configures rewrites **only when `DEV=true`** (i.e. `npm run dev`):

| Incoming request (Next.js :3000) | Forwarded to         |
|----------------------------------|----------------------|
| `/api/*`                         | Go backend `:4455`   |
| `/self-service/*`                | Go backend `:4455/api/kratos/self-service/*` |
| `/ui/*`                          | Go backend `:4455`   |
| `/.well-known/webauthn.js`       | Kratos `:4433`       |

**In production** there are no rewrites — Next.js emits a static export (`ui/dist/`) that the Go binary serves directly. Never use `next start`; this app uses `output: 'export'`.

The browser must always be opened at `http://localhost` (Traefik on port 80), not at `:3000` directly.

## Ory Flow/Node Component Architecture

Kratos returns a `UiFlow` with a `ui.nodes` array. Each node has a `type` and `group`
that determine which component renders it. **Form fields are never hand-coded.**

```
Flow.tsx        — receives the full flow object, iterates nodes, renders <Node> per node
Node.tsx        — dispatches to the correct NodeInput*, NodeAnchor, NodeImage, etc.
NodeInput*.tsx  — one file per HTML input type: Button, Checkbox, Email, Hidden,
                  Password, Submit, Text, Url
```

**Where to make changes:**
- Changing how a specific input *looks or behaves* → edit the relevant `NodeInput*.tsx`
- Adding a brand-new Kratos node type → create a new `NodeInput*.tsx` and wire it in `Node.tsx`
- Changing flow *lifecycle* (fetch, submit, redirect after submit) → edit the page in `ui/pages/*.tsx`

## Vanilla Framework & @canonical/react-components

- Use components from `@canonical/react-components` before writing custom HTML.
  Key components: `Button`, `Input`, `Notification`, `Spinner`, `Modal`, `Strip`.
- Styling uses Vanilla Framework utility classes and Sass. Do not add inline styles.
- Class names follow Vanilla Framework naming: `p-`, `u-`, `l-` prefixes.
- No CSS modules — styles live in `ui/static/css/` or component-level `.scss` files.

## TypeScript Rules

- No `any` — use `unknown` or the correct Ory SDK type (e.g. `UiNode`, `LoginFlow`).
- Ory SDK types live in `@ory/client`; import from there, don't redefine them.
- All page components are `NextPage` or `FC<Props>`; never use class components.
- Props interfaces use `PascalCase` with a `Props` suffix: `type LoginPageProps = { ... }`.

## Tracing a Kratos Error Back to the Codebase

When a Kratos API returns an error, it includes a numeric `id` field. The Go backend
maps these to named constants in `pkg/kratos/service.go`:

```go
IncorrectCredentials = 4000006
DuplicateIdentifier  = 4000007
InvalidBackupCode    = 4000016
// …
```

The frontend surfaces errors through the `UiNode` message array on the flow object —
the `Flow.tsx` / `Node.tsx` components render them automatically. If an error from the
server is **not** appearing in the UI, the issue is almost always that the Go handler
returned an HTTP error response instead of forwarding the updated Kratos flow object
back to the frontend.

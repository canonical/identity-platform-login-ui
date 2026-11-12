# Proposal

## Why

When `KRATOS_PUBLIC_URL` or `HYDRA_ADMIN_URL` is unset, the service starts health-check
goroutines for Kratos and Hydra with empty base URLs. Each goroutine emits
`Get "/health/ready": unsupported protocol scheme ""` every 10 seconds. This floods
logs in deployments that run without Kratos or Hydra (issue #908). No spec covers the
status subsystem.

## What Changes

- `status.Service` accepts the Kratos and Hydra base URLs as constructor parameters.
- When a dependency URL is empty, its readiness check returns `false, nil` immediately
  and does not call the Ory SDK. The status endpoint reports the dependency as unavailable.
- The service logs one warning at startup for each empty dependency URL.
- **BREAKING** (internal API): `status.NewService` gains two leading `string` parameters
  (`kratosURL`, `hydraURL`).
- The router passes `kratosPublicURL` through the existing `WithKratosPublicURL` option
  and the Hydra admin URL through a new `WithHydraAdminURL` option.
- The Ory Kratos and Hydra clients do not change.

## Capabilities

### New Capabilities

- `service-health`: dependency health checks for Kratos and Hydra, including behavior
  when their URLs are not configured.

### Modified Capabilities

- None.

## Impact

- **Backend (Go)**: `pkg/status/service.go`, `pkg/status/service_test.go`,
  `pkg/web/router.go`, `cmd/serve.go`. No frontend (Next.js) impact.
- **Kratos / Hydra**: no changes to how the service calls Kratos or Hydra. The service
  only stops calling their metadata endpoints when no URL is configured.
- **Cookies / OpenFGA**: no impact.
- **Non-goals**: no changes to readiness HTTP endpoint semantics, no changes to the
  Ory clients, no periodic re-evaluation of configuration.
- **Success criteria**: a deployment with only the metrics endpoint (no Kratos, no Hydra)
  emits no `unsupported protocol scheme` log entries, and each missing dependency logs
  exactly one warning at startup.

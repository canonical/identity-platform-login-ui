## Context

`pkg/tenants.Service` currently communicates with the tenant service via a hand-rolled HTTP client: it constructs `GET /api/v0/tenants/lookup` requests, handles HTTP status codes, and decodes JSON responses manually. The `github.com/canonical/identity-platform-api` repository (branch `IAM-1998`, to be merged to `main`) now ships generated gRPC client stubs for `TenantService`, including `LookupTenants(ctx, *LookupTenantsRequest) (*LookupTenantsResponse, error)`. The `LookupTenantsRequest` carries an optional `email` and an optional `identity_id`; the response contains `[]*tenant.Tenant` which has the same fields (`id`, `name`, `created_at`, `enabled`) as the local `Tenant` struct.

## Goals / Non-Goals

**Goals:**
- Replace the ad-hoc HTTP client with the generated `tenant.TenantServiceClient`.
- Remove the local `Tenant` struct and use `tenant.Tenant` from the SDK (or map at the boundary to keep the rest of the codebase decoupled).
- Eliminate custom JSON decoding and URL construction from `pkg/tenants`.
- Keep the `LookupTenantsByEmail`, `LookupTenantsByIdentityID`, and `LookupTenantsByFlow` behaviour identical from the caller's perspective.
- Wire up the gRPC connection in `cmd/serve.go`.

**Non-Goals:**
- Implementing any other `TenantService` RPC methods (e.g. `ListTenants`, `CreateTenant`).
- Frontend changes.
- Changing the Kratos, Hydra, or OpenFGA integration.

## Decisions

### 1. Inject `tenant.TenantServiceClient` interface into `Service`

**Decision**: Accept the generated `tenant.TenantServiceClient` interface directly in the `Service` constructor rather than a `grpc.ClientConn`.

**Rationale**: Injecting the client interface keeps the service testable with gomock without a real gRPC server. Injecting a raw connection would require a test gRPC server or embed the dial logic inside the service, violating the pattern used by `pkg/` packages (constructor receives ready-to-use dependencies).

**Alternative considered**: Accept `grpc.ClientConn` and call `tenant.NewTenantServiceClient` internally. Rejected: makes mocking harder and couples construction to the service.

### 2. Keep the local `Tenant` struct as a mapping boundary

**Decision**: Retain the `pkg/tenants.Tenant` struct and convert `*tenant.Tenant` responses from the SDK at the service boundary.

**Rationale**: Other packages in `pkg/` (handlers, resolver, cookie storage) already reference `*tenants.Tenant`. Changing all callers to the protobuf-generated type would spread the SDK dependency across the codebase and couple callers to protobuf types. A thin mapping function inside `service.go` keeps the blast radius minimal.

**Alternative considered**: Replace the local struct entirely with `*tenant.Tenant`. Rejected: protobuf-generated types carry `state`, `sizeCache`, `unknownFields` and are not idiomatic Go value types.

### 3. gRPC dial options and address config

**Decision**: Add a `TenantServiceGRPCAddress` config field (e.g. `tenant-svc:50051`). The existing `TenantsAPIURL` field is retired. In `cmd/serve.go` dial with `grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))` for development; TLS credentials can be added as a follow-up when the service is deployed with mTLS.

**Rationale**: Minimal change footprint; matches the pattern of other internal service addresses in the config. Using insecure credentials is acceptable for intra-cluster gRPC calls (same pattern used in admin-ui).

**Alternative considered**: Reuse `TenantsAPIURL` as the gRPC target. Rejected: HTTP URL and gRPC target have different schemes (`http://host:port` vs `host:port`); keeping a single field would require stripping the scheme, which is fragile.

### 4. Interface definition in `interfaces.go`

**Decision**: Define a minimal local `TenantServiceClientInterface` in `pkg/tenants/interfaces.go` with only the `LookupTenants` method needed by `Service`, rather than embedding the full generated `tenant.TenantServiceClient`.

**Rationale**: Follows the project convention: _"re-define a local interface with only the methods needed"_. Keeps coupling explicit and allows mocking with gomock via `//go:generate`.

## Risks / Trade-offs

- **gRPC dependency footprint**: adding `google.golang.org/grpc` and protobuf transitive deps increases the vendor directory. → Acceptable given the long-term removal of bespoke HTTP code.
- **Branch dependency**: the library is currently on `IAM-1998`. Go modules will pin the commit hash. After the branch is merged to `main` the `go.mod` replace/require must be updated. → Document in tasks; add a TODO comment in `go.mod` if needed.
- **Insecure gRPC for now**: intra-cluster traffic is insecure until mTLS is added. → Acceptable short-term; track as follow-up issue.

## Migration Plan

1. Add `github.com/canonical/identity-platform-api` to `go.mod` pointing at the `IAM-1998` commit (using `go get github.com/canonical/identity-platform-api@IAM-1998` or a replace directive).
2. Run `go mod vendor`.
3. Rewrite `pkg/tenants/service.go`.
4. Update `pkg/tenants/interfaces.go` and regenerate mocks (`make mocks`).
5. Update `internal/config/specs.go` and `cmd/serve.go`.
6. Ensure all existing unit tests pass; update test helpers/mocks.
7. After `IAM-1998` merges to `main`, update `go.mod` to reference `main` and re-vendor.

**Rollback**: The change is entirely in `pkg/tenants` and `cmd/serve.go`. Rolling back means reverting those files; no database or schema migrations are involved.

## Open Questions

_None — all questions resolved during review._

- **TLS/mTLS for gRPC**: not required for now; in-cluster plaintext is acceptable. TLS can be added as a follow-up when the service is deployed with mTLS.
- **gRPC timeout**: configurable via `TENANT_SERVICE_GRPC_TIMEOUT` (default **5 s**). 5 s matches the desired fast-path latency target and is consistent with the default enforced by `NewService`.

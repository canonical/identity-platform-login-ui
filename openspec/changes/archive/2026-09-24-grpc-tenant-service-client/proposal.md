## Why

The tenant service lookup in `pkg/tenants` is currently implemented with a hand-rolled HTTP client. Now that `github.com/canonical/identity-platform-api` provides generated gRPC client stubs for the `TenantService`, the ad-hoc HTTP client should be replaced with the official client, removing bespoke request construction and JSON decoding in favour of the generated SDK.

## What Changes

- Add `github.com/canonical/identity-platform-api` (branch `IAM-1998`, switching to `main` after merge) as a Go module dependency.
- Replace the `pkg/tenants.Service` HTTP client fields (`tenantsAPIURL`, `httpClient`) with a `tenant.TenantServiceClient` gRPC client from the SDK.
- Rewrite `LookupTenantsByEmail` and `LookupTenantsByIdentityID` to call `TenantService.LookupTenants` via gRPC instead of crafting HTTP requests.
- Keep the local `Tenant` struct and map `*tenant.Tenant` proto responses at the service boundary, so the rest of the codebase remains decoupled from protobuf-generated types.
- Update `NewService` constructor: accept a `tenant.TenantServiceClient` (or a `grpc.ClientConn` + address) instead of a raw URL and `*http.Client`.
- Update `cmd/serve.go` wiring to create the gRPC connection and inject it.
- Update `interfaces.go` local interface definition and mocks.
- Remove config field(s) no longer needed (e.g. `TenantsAPIURL` if replaced by a gRPC target address).

## Capabilities

### New Capabilities

- `tenant-grpc-client`: gRPC-backed tenant lookup — replaces the HTTP client in `pkg/tenants.Service` with a call to `TenantService.LookupTenants`, accepting either an `email` or `identity_id` filter.

### Modified Capabilities

<!-- No existing spec-level capability requirements change; the external behaviour (lookup by email / identity ID) is unchanged. Only the transport layer changes. -->

## Impact

- **Backend Go**: `pkg/tenants/service.go`, `pkg/tenants/interfaces.go`, `cmd/serve.go`, `internal/config/specs.go` (config field rename/add).
- **Dependencies**: new Go module `github.com/canonical/identity-platform-api` + transitive gRPC deps (`google.golang.org/grpc`, protobuf).
- **Tests**: `pkg/tenants` unit tests — mocks need to be regenerated; test setup changes from HTTP mock server to gomock gRPC client.
- **No Kratos, Hydra, or OpenFGA changes required.** Frontend changes (WebAuthn/TOTP flow selection on the login page) are also included in this PR, tracked separately under issue #839.

## ADDED Requirements

### Requirement: Tenant lookup via gRPC client
`pkg/tenants.Service` SHALL use the `TenantServiceClient` gRPC interface from `github.com/canonical/identity-platform-api` to resolve tenant lookups, replacing the hand-rolled HTTP client.

#### Scenario: Lookup by email succeeds
- **WHEN** `LookupTenantsByEmail` is called with a valid email address
- **THEN** the service SHALL call `TenantServiceClient.LookupTenants` with `email` set and return the mapped `[]*Tenant` slice

#### Scenario: Lookup by identity ID succeeds
- **WHEN** `LookupTenantsByIdentityID` is called with a valid Kratos identity ID
- **THEN** the service SHALL call `TenantServiceClient.LookupTenants` with `identity_id` set and return the mapped `[]*Tenant` slice

#### Scenario: Lookup by flow succeeds
- **WHEN** `LookupTenantsByFlow` is called with a valid Kratos login flow ID and cookies
- **THEN** the service SHALL extract the email from the flow and call `TenantServiceClient.LookupTenants` with `email` set

#### Scenario: gRPC error is propagated
- **WHEN** `TenantServiceClient.LookupTenants` returns a gRPC error
- **THEN** the service SHALL return a non-nil error to the caller and return `nil` for the tenant slice

### Requirement: Local TenantServiceClient interface
`pkg/tenants` SHALL define a local `TenantServiceClientInterface` in `interfaces.go` containing only the `LookupTenants` method, following the project's interface re-definition convention.

#### Scenario: Interface is narrow
- **WHEN** examining `pkg/tenants/interfaces.go`
- **THEN** the `TenantServiceClientInterface` SHALL declare exactly `LookupTenants(ctx context.Context, in *tenant.LookupTenantsRequest, opts ...grpc.CallOption) (*tenant.LookupTenantsResponse, error)` and no other methods

### Requirement: gRPC address configuration
The application SHALL accept a gRPC target address for the tenant service via configuration, replacing the previous HTTP URL field.

#### Scenario: Address is read from config
- **WHEN** the application starts with `TENANT_SERVICE_GRPC_ADDRESS` set (or equivalent config key)
- **THEN** the gRPC dial SHALL use that address to connect to the tenant service

#### Scenario: Default is empty / feature is disabled when address is unset
- **WHEN** no gRPC address is configured
- **THEN** the multi-tenancy feature behaves as if the tenant service is unavailable (no-op resolver), consistent with current behaviour when `TENANT_SERVICE_GRPC_ADDRESS` is unset

### Requirement: Mock regeneration
After changing `interfaces.go`, the generated mock for `TenantServiceClientInterface` SHALL be updated via `make mocks` so that unit tests continue to compile.

#### Scenario: Mocks compile after change
- **WHEN** `make mocks` is run after updating `interfaces.go`
- **THEN** `mock_interfaces.go` in `pkg/tenants` SHALL reflect the new interface and all existing tests SHALL compile and pass

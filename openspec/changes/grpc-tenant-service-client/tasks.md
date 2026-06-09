## 1. Dependency Setup

- [x] 1.1 Add `github.com/canonical/identity-platform-api` to `go.mod` pointing at the `IAM-1998` branch commit (`go get github.com/canonical/identity-platform-api@IAM-1998` or a `replace` directive)
- [x] 1.2 Run `go mod vendor` to pull all transitive gRPC / protobuf dependencies into `vendor/`
- [x] 1.3 Verify the build compiles after vendoring (`go build ./...`)

## 2. Config Changes

- [x] 2.1 Add `TenantServiceGRPCAddress` (string) to `internal/config/specs.go` (and any env-var binding in `cmd/serve.go`)
- [x] 2.2 Deprecate / remove `TenantsAPIURL` config field (or keep it as a no-op with a deprecation notice if a smooth rollout is needed)

## 3. Interface and Mock Updates

- [x] 3.1 Define `TenantServiceClientInterface` in `pkg/tenants/interfaces.go` with only `LookupTenants(ctx context.Context, in *tenant.LookupTenantsRequest, opts ...grpc.CallOption) (*tenant.LookupTenantsResponse, error)`
- [x] 3.2 Add `//go:generate mockgen` directive for the new interface in `pkg/tenants/interfaces.go`
- [x] 3.3 Run `make mocks` and verify `mock_interfaces.go` (or a new mock file) is regenerated correctly

## 4. Service Rewrite

- [x] 4.1 Replace `tenantsAPIURL string` and `httpClient *http.Client` fields in `pkg/tenants.Service` with `grpcClient TenantServiceClientInterface`
- [x] 4.2 Rewrite `lookupTenantsByEmail` to call `s.grpcClient.LookupTenants(ctx, &tenant.LookupTenantsRequest{Email: email})` and map the response to `[]*Tenant`
- [x] 4.3 Rewrite `LookupTenantsByIdentityID` to call `s.grpcClient.LookupTenants(ctx, &tenant.LookupTenantsRequest{IdentityId: identityID})` and map the response
- [x] 4.4 Add a `toLocalTenants` mapping helper that converts `[]*tenant.Tenant` → `[]*Tenant`
- [x] 4.5 Update `NewService` constructor signature to accept `grpcClient TenantServiceClientInterface` instead of `tenantsAPIURL` and remove `http.Client` initialization
- [x] 4.6 Remove unused imports (`encoding/json`, `net/http`, `net/url`, `time`) from `service.go`

## 5. Wiring in cmd/serve.go

- [x] 5.1 Add gRPC dial call in `cmd/serve.go`: `grpc.Dial(cfg.TenantServiceGRPCAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))`
- [x] 5.2 Create `tenant.NewTenantServiceClient(conn)` and pass it to `tenants.NewService`
- [x] 5.3 Handle the case where `TenantServiceGRPCAddress` is empty (skip dial, pass a no-op / nil client, or keep the `NoOpTenantResolver` path unchanged)
- [x] 5.4 Ensure the gRPC connection is closed on application shutdown

## 6. Unit Test Updates

- [x] 6.1 Update `pkg/tenants` unit tests to use the new gomock-generated `TenantServiceClientInterface` mock instead of an HTTP test server
- [x] 6.2 Verify `TestLookupTenantsByEmail` and `TestLookupTenantsByIdentityID` pass with the mock returning a `*tenant.LookupTenantsResponse`
- [x] 6.3 Verify `TestLookupTenantsByFlow` still passes (it calls `lookupTenantsByEmail` internally)
- [x] 6.4 Run `make test` and ensure all tests pass

## 7. Post-Merge Cleanup (deferred)

- [ ] 7.1 After `IAM-1998` is merged to `main` on `identity-platform-api`, update `go.mod` to point at the `main` branch / latest release tag
- [ ] 7.2 Re-run `go mod vendor` and open a follow-up PR to switch from the branch pin to `main`

## 1. Matching rule

- [x] 1.1 Add `audiencePermits(registered, resource string) bool` to `pkg/extra/resource.go`: parse both values; require equal `Scheme`, `Host` and `Opaque`; accept when the resource path equals the registered path, equals it with trailing `/` trimmed, or extends it across a `/` boundary (mirror `fosite.DefaultAudienceMatchingStrategy`, cite it in the doc comment). Return false when either value fails to parse.
- [x] 1.2 Add `permittedResources(registered, resources []string) (permitted, rejected []string)` to `pkg/extra/resource.go`: partition `resources` in input order by `audiencePermits` against any entry of `registered`; empty `registered` rejects everything. Allocate nothing when `resources` is empty.
- [x] 1.3 Add table-driven `TestAudiencePermits` in `pkg/extra/service_test.go` covering: exact match; registered `https://api.example/v1` vs resource `/v1/orders` (permit), `/v10` (reject), `/v1/` (permit), `/v1?x=1` (permit, query ignored); registered with trailing slash `https://api.example/v1/` vs `/v1` (permit); scheme mismatch `http` vs `https`; host mismatch; `urn:example:a` vs `urn:example:b` (reject) and vs itself (permit); unparseable registered entry (reject). Completion: `go test ./pkg/extra/ -run TestAudiencePermits` passes.
- [x] 1.4 Add table-driven `TestPermittedResources` covering empty registered audience, mixed permitted/rejected in order, and empty resources returning `nil, nil`. Completion: `go test ./pkg/extra/ -run TestPermittedResources` passes.

## 2. Wire the filter into consent acceptance

- [x] 2.1 In `pkg/extra/service.go` `grantAudience`: call `permittedResources(consent.GetClient().GetAudience(), resources)` after `resourceIndicators`; pass `permitted` to `mergeAudience`; add a second `s.logger.Debugf("dropping resource indicators not registered as client audience: %v", rejected)` guarded by `len(rejected) > 0`. Update the function's doc comment to state the whitelist.
- [x] 2.2 Update `TestAcceptConsentGrantsResourceIndicators` in `pkg/extra/service_test.go`: set `consent.Client` with `Audience: []string{"https://api.example.com", "https://mcp.example.com"}`, add a fourth `resource` that is valid but unregistered (`https://evil.example.com`), expect two `Debugf` calls, and assert the granted audience excludes the unregistered value.
- [x] 2.3 Add `TestAcceptConsentIgnoresResourcesWithoutClientAudience`: consent with a `resource` and a client whose `Audience` is nil; assert the granted audience equals Hydra's requested audience and one `Debugf` is logged. Completion: `go test ./pkg/extra/...` and `go vet ./pkg/extra/` pass.

## 3. Documentation

- [x] 3.1 In `README.md` section "Resource indicators (RFC 8707)", replace the sentence "Granted resource audiences are not checked against the client's registered `audience` ..." with the rule: an indicator is granted only when the client's registered `audience` permits it under the same matching Hydra applies to the `audience` parameter; point to `hydra create client --audience`.

## 4. Verification

- [x] 4.1 Live proof against an isolated Hydra 25.4.0 (same harness as the original branch proof: scratch database on the dev postgres, `exec hydra migrate sql` then `exec hydra serve all --dev`, throwaway Go program under `tmp_smoke/` deleted afterwards): register a client with `audience: ["https://mcp.example.com/sse"]`, request `resource=https://mcp.example.com/sse&resource=https://other.example.com`, complete login/consent through `extra.Service.AcceptConsent`, exchange the code, and confirm the JWT `aud` is exactly `["https://mcp.example.com/sse"]`. Repeat with a client that has no registered audience and confirm `aud` is absent.
- [x] 4.2 Run `go test ./...` and `go vet ./pkg/... ./internal/...`; remove `tmp_smoke/`, stop the scratch Hydra container and drop the scratch database.
- [x] 4.3 Commit on branch `IAM-2324` with a conventional message (`fix: restrict resource indicators to the client's registered audience`). The scope note goes in the PR description instead of an issue comment: `resource` values are granted only for audiences registered on the client; DCR clients without `audience` get none; device flow, JAR and PAR are not covered.

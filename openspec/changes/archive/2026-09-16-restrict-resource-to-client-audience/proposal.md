## Why

Branch `IAM-2324` (issue #966) grants every valid RFC 8707 `resource` indicator of the authorization request as an access token audience when the consent request is accepted. Hydra validates the equivalent non-standard `audience` request parameter against the client's registered `audience` list (fosite `validateAuthorizeAudience`) and rejects unlisted values, but the consent-time grant bypasses that check: any client can obtain an access token carrying an `aud` it was never registered for. The review of the branch rated this a high-severity broken-access-control finding (SEC-RES-001). The gap must be closed before the branch merges.

### Problem statement

- `pkg/extra` `Service.grantAudience` merges the parsed `resource` values into `grant_access_token_audience` without consulting `consent.client.audience`.
- Hydra applies consent-granted audience unconditionally, so the Login UI is the only place the check can happen.
- `/api/consent` auto-accepts consent for any user with a Kratos session, so no human gate mitigates the escalation.

### Scope

- Filter the parsed `resource` indicators through the client's registered `audience` before granting, using the same matching semantics fosite applies to the `audience` parameter.
- Report rejected indicators the same way syntactically invalid ones are reported today (dropped, debug-logged, consent still accepted).
- Document the rule.

### Non-goals

- Rejecting the consent request with `invalid_target` (RFC 8707 section 2.1). Failure semantics of `/api/consent` stay as they are.
- Bounding the number or length of `resource` values.
- Recovering `resource` values from JAR request objects, PAR, or the device flow POST body.
- Changing how Hydra's dynamic client registration handles `audience` (a dynamically registered client may still self-register any `audience`; this is Hydra behaviour and out of scope).
- Any frontend (`ui/`) change.

### Success criteria

- An authorization request with `resource=R` yields an access token with `R` in `aud` only if the client's registered `audience` permits `R` under fosite's matching rule.
- A client with an empty registered `audience` never receives a `resource`-derived audience.
- Existing behaviour for the `audience` request parameter and for clients that send no `resource` is byte-for-byte unchanged.
- Unit tests cover the matching rule boundaries; a live run against Hydra confirms the filtered `aud`.

## What Changes

- `pkg/extra`: the resource indicators recovered from the consent request's `request_url` are matched against `consent.GetClient().GetAudience()`; unmatched indicators are dropped and debug-logged together with the syntactically invalid ones.
- Matching rule mirrors fosite `DefaultAudienceMatchingStrategy`: an indicator is permitted when it equals a registered audience, or when scheme and host are equal and the registered audience's path is a segment-boundary prefix of the indicator's path. An empty registered audience permits nothing.
- **BREAKING** relative to the unmerged `IAM-2324` branch only: clients relying on unregistered `resource` values lose them. No released behaviour changes.
- `README.md` "Resource indicators (RFC 8707)": replace the "not checked against the client's registered `audience`" warning with the rule.

## Capabilities

### New Capabilities

- `resource-indicator-audience`: granting RFC 8707 `resource` indicators as access token audience at consent time, restricted to the client's registered `audience`.

### Modified Capabilities

<!-- No existing specs in openspec/specs/. -->

## Impact

- **Backend Go**: `pkg/extra/resource.go` (matching helper), `pkg/extra/service.go` (`grantAudience`), `pkg/extra/service_test.go`. No interface change, so no mock regeneration.
- **Hydra**: no configuration change. The Login UI reads `client.audience` from the consent request Hydra already returns; the accepted consent still carries `grant_access_token_audience`. Operators who want a client to receive `resource`-derived audiences must register those audiences on the client (`hydra create client --audience ...` or the admin API).
- **Kratos, cookies, OpenFGA**: unaffected.
- **Frontend**: unaffected.
- **Issue #966**: the fix serves operator-registered clients. MCP clients registering via DCR without `audience` receive no `resource`-derived audience unless the registration lists it; this is to be stated on the issue.

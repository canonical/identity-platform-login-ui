## Purpose

Ory Hydra binds access tokens to resource servers with the non-standard `audience` authorization request parameter, which fosite validates against the client's registered `audience` list, and ignores the RFC 8707 `resource` parameter. The Login UI recovers `resource` indicators from the authorization URL that Hydra keeps in the consent request and grants them as access token audience. Because Hydra applies consent-granted audience unconditionally, the Login UI is the only place that can hold `resource` to the same authorization rule Hydra applies to `audience`. This capability defines that rule so that a client cannot obtain an `aud` it was not registered for.

## Requirements

### Requirement: Resource indicators are recovered from the authorization request
When accepting a consent request, the Login UI SHALL read every `resource` query parameter from the consent request's `request_url` and SHALL treat each value as an RFC 8707 resource indicator candidate. A candidate SHALL be syntactically valid only if it is an absolute URI (RFC 3986 section 4.3) that contains no fragment component and no whitespace, control, or backslash characters.

#### Scenario: Multiple resource parameters
- **WHEN** the authorization URL contains `resource=https://a.example/api` and `resource=https://b.example/api`
- **THEN** both values are candidates for the granted audience

#### Scenario: Relative or fragment-bearing value
- **WHEN** the authorization URL contains `resource=/api` or `resource=https://a.example/api#frag`
- **THEN** the value is dropped and is never granted

#### Scenario: No request URL
- **WHEN** the consent request carries no `request_url`
- **THEN** no resource indicator is granted and the consent is accepted with Hydra's requested audience unchanged

### Requirement: Resource indicators are restricted to the client's registered audience
The Login UI SHALL grant a syntactically valid resource indicator as access token audience only if the OAuth 2.0 client of the consent request has a registered `audience` entry that permits it. An entry permits an indicator when both parse as URIs and either the two are byte-equal, or their scheme, host and opaque part are equal and the entry's path, with trailing slashes removed, equals the indicator's path or is a prefix of it ending on a path segment boundary. Query components SHALL NOT take part in the comparison. A client with no registered `audience` SHALL receive no resource-derived audience.

#### Scenario: Exact registered audience
- **WHEN** the client is registered with audience `https://mcp.example/sse` and the authorization URL contains `resource=https://mcp.example/sse`
- **THEN** the issued access token's `aud` contains `https://mcp.example/sse`

#### Scenario: Path below a registered audience
- **WHEN** the client is registered with audience `https://api.example/v1` and the authorization URL contains `resource=https://api.example/v1/orders`
- **THEN** `https://api.example/v1/orders` is granted

#### Scenario: Path that only shares a string prefix
- **WHEN** the client is registered with audience `https://api.example/v1` and the authorization URL contains `resource=https://api.example/v10`
- **THEN** `https://api.example/v10` is dropped

#### Scenario: Scheme or host mismatch
- **WHEN** the client is registered with audience `https://api.example` and the authorization URL contains `resource=http://api.example` or `resource=https://evil.example`
- **THEN** the value is dropped

#### Scenario: Opaque URI mismatch
- **WHEN** the client is registered with audience `urn:example:a` and the authorization URL contains `resource=urn:example:b`
- **THEN** `urn:example:b` is dropped

#### Scenario: Empty registered audience
- **WHEN** the client has no registered audience and the authorization URL contains any `resource` value
- **THEN** no resource-derived audience is granted

### Requirement: Granted audience composition
The granted access token audience SHALL be Hydra's `requested_access_token_audience` in its original order, followed by each permitted resource indicator in request order, with a permitted indicator omitted when the granted audience already contains an equal string. The Login UI SHALL NOT remove or reorder any audience Hydra requested.

#### Scenario: Resource duplicates the audience parameter
- **WHEN** Hydra requested audience `https://api.example` and the authorization URL also contains `resource=https://api.example` permitted by the client registration
- **THEN** the granted audience contains `https://api.example` exactly once

#### Scenario: No resource parameter
- **WHEN** the authorization URL contains no `resource` parameter
- **THEN** the granted audience is exactly Hydra's requested audience

### Requirement: Dropped indicators do not fail the consent
The Login UI SHALL accept the consent request with the remaining granted audience when one or more resource indicators are dropped, and SHALL record the dropped values in a debug-level log entry. The Login UI SHALL NOT return an error to the client because of a dropped resource indicator.

#### Scenario: Mixed permitted and unregistered indicators
- **WHEN** the client is registered with audience `https://a.example` and the authorization URL contains `resource=https://a.example` and `resource=https://b.example`
- **THEN** the consent is accepted, the granted audience contains `https://a.example` and not `https://b.example`, and `https://b.example` appears in a debug log entry

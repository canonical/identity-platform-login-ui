## Purpose

With multi-tenancy enabled, a login is accepted for a user and for one of that user's tenants, and the tenant ends up in the tokens the client receives. The Login UI learns the candidate tenant early, from the email the user types or from the tenant selection, and keeps it in an encrypted cookie bound to the login challenge until the login is accepted. That cookie belongs to a login request, not to a user: the email can be changed, the selection endpoint takes any id, and the account that signs in can differ from the email entered. This capability defines how the tenant of a login is decided so that it is always one of the tenants of the user who signs in, and what the recorded tenant is allowed to decide.

## ADDED Requirements

### Requirement: The tenants of the user decide the tenant of a login
With multi-tenancy enabled, every time the Login UI resolves the tenant of a client's login (when an email is submitted, after a credential is accepted, and when the login page is opened with a session) it SHALL look up the tenants of that email or of that session's identity, and SHALL decide from them: a user with no tenant gets no tenant, a user with exactly one tenant gets that tenant, and a user with several tenants selects one. It SHALL NOT skip the lookup because a tenant is already recorded for the login challenge. The tenant the Login UI passes to Hydra when it accepts the login SHALL be one of the tenants looked up for the session it accepts, or absent.

#### Scenario: A user with no tenant signs in after the email of a user with one
- **WHEN** the email of a user with one tenant is entered for a client's login, the login request is opened again, and a user with no tenant enters their email and signs in
- **THEN** the login is accepted with no tenant

#### Scenario: A user with one tenant signs in after the email of a user with none
- **WHEN** the email of a user with no tenant is entered for a client's login, the login request is opened again, and a user with one tenant enters their email and signs in
- **THEN** the login is accepted with that user's tenant

#### Scenario: Tenant id sent to the selection endpoint by a user with one tenant
- **WHEN** a tenant id that is not the user's is stored for the login challenge through the tenant selection endpoint, and a user with exactly one tenant signs in
- **THEN** the login is accepted with the user's own tenant

#### Scenario: Tenant Service cannot be reached
- **WHEN** the lookup fails at any of the three points
- **THEN** that step answers with a server error and the login is not accepted, also when a tenant is already recorded

### Requirement: A recorded tenant only chooses among the user's several tenants
The tenant recorded in the state cookie for a login challenge SHALL be used only when the user has several tenants and the recorded tenant is one of them. When the user has several tenants and the recorded tenant is not one of them, the Login UI SHALL drop the record and send the user to the tenant selection, and the cookie it writes at that point SHALL NOT hold the dropped tenant.

#### Scenario: Selection kept across the steps of a login
- **WHEN** a user with several tenants selects one and then enters their password and second factor
- **THEN** the user is not asked to select again and the login is accepted with the selected tenant

#### Scenario: Recorded tenant is not one of the user's several
- **WHEN** a tenant that is not the user's is recorded for the login challenge and a user with several tenants signs in
- **THEN** the user is sent to the tenant selection, and the login is accepted with the tenant they select there

#### Scenario: Selected tenant lost during the login
- **WHEN** a user selects one of several tenants, loses that membership before the login is accepted, and has exactly one tenant left
- **THEN** the login is accepted with the tenant that is left

### Requirement: An email submission starts from an empty record
When an email is submitted for a login challenge, the Login UI SHALL resolve the tenant from a new state for that challenge, holding no tenant recorded before.

#### Scenario: Second user shares one of several tenants with the first
- **WHEN** a user with several tenants selects one for a client's login, the login request is opened again, and a second user who has the same tenant among several enters their email
- **THEN** the second user is sent to the tenant selection and is not given the first user's choice

#### Scenario: Email submitted again after a selection
- **WHEN** a user with several tenants selects one and then submits their email again for the same login request
- **THEN** the user selects a tenant again

### Requirement: What the record still leaves open
The state cookie SHALL remain bound to a login challenge and not to a user. An account other than the email entered, such as one that signs in through an external provider, that has the recorded tenant among several SHALL keep it without being asked. The tenant id the Login UI sends to Kratos with the credentials, for the Tenant Service's login hook, SHALL remain the recorded one, unchecked by the Login UI.

#### Scenario: Another account that shares the recorded tenant
- **WHEN** the email of one user is entered and a tenant recorded for it, and a different account that has that tenant among several signs in through an external provider
- **THEN** the login is accepted with the recorded tenant, which is one of that account's own, without a selection

### Requirement: Logins without multi-tenancy are unchanged
With multi-tenancy disabled the Login UI SHALL NOT look up tenants and SHALL accept logins with no tenant, as before.

#### Scenario: Login without multi-tenancy
- **WHEN** a user signs in for a client's login with multi-tenancy disabled
- **THEN** no tenant is looked up and the login is accepted with no tenant

## Purpose

Kratos writes every link between an account and a company sign-in. For a subject it has no link for, whose address already belongs to an account, Kratos does not link on the word of the identity provider: the user first signs in to the account with a way it already has (Kratos's account-linking login flow). Login UI does not shorten that step. It adds what Kratos leaves off the page, and keeps track of which sign-in is being linked.

## ADDED Requirements

### Requirement: The account-linking page offers the account's own ways in
A login flow that carries the Kratos message `1010016` is the account-linking flow. For it the extension SHALL keep every sign-in Kratos lists, also when the tenant requires company sign-in (this sign-in only proves the account), and SHALL add the account's other company sign-ins, which Kratos hides because they share the provider used to get there:

- the account is the identity that holds the address of the flow (Kratos admin API); its company sign-ins are the links the SSO service lists for it (`ListLinks` with `identity_id`);
- each is a submit node of group `sso`, name `sso_account_link`, value the connection id, labelled "Continue with <label>";
- the connection being linked (the one in the sign-in cookie) MUST be left out: its identity provider answers with the subject that has no link, so it cannot prove the account;
- with no such company sign-in and no other first factor in the flow, the recovery prompt of `sign-in-screen` is shown.

#### Scenario: Account with a password, tenant that requires company sign-in
- **WHEN** a member with a portal password returns from their first company sign-in
- **THEN** the login page shows the message of Kratos and the password field, although the tenant offers no password otherwise

#### Scenario: Account whose only way in is another company sign-in
- **WHEN** the account has a link to connection A and returns from a first sign-in with connection B
- **THEN** the page offers "Continue with <label of A>" and does not offer B

### Requirement: Proving the account with another company sign-in
A submission of `sso_account_link` SHALL start that company sign-in on the account-linking flow itself, so that Kratos signs the user in through the existing link and then adds the pending one. The extension MUST first check that the flow is an account-linking flow (`400`), that the connection is one the account of the flow is linked to (`403` "Provider not allowed"), and that the sign-in cookie of this browser holds a pending attempt, with a ticket and a tenant, for another connection than the one picked (`400`). The new attempt SHALL be for the tenant of the pending attempt and the address of the flow, and the sign-in cookie SHALL keep the ticket being linked next to the new one.

#### Scenario: A connection the account is not linked to
- **WHEN** `sso_account_link` names a connection that is not among the account's links
- **THEN** the response is `403` and no ticket is asked for

### Requirement: The linked sign-in is the one the session counts for
When the session comes back and the sign-in cookie holds a ticket being linked, the extension SHALL ask the SSO service to confirm that ticket first, and the proving one only if the first is `NOT_APPLICABLE`. Each ticket SHALL be sent with its own receipt (`company-sign-in`): the browser holds one for the sign-in being linked, which the SSO service accepted before Kratos asked the user to prove the account, and one for the sign-in that proved it. The connection of the first ticket confirmed is the provenance of the session. A session to which Kratos added the company sign-in by account linking (`byo-sso` is not its first authentication method) SHALL be checked as a company-sign-in session.

#### Scenario: Linked after signing in with the password
- **WHEN** the session's methods are `password` then `oidc`/`byo-sso`, and the SSO service confirms the attempt
- **THEN** the session passes the checks of the tenant that requires company sign-in, with the linked connection as its provenance

#### Scenario: Linked through the account's other company sign-in
- **WHEN** the user proved the account with connection A while linking connection B
- **THEN** `CompleteAttempt` is called with the ticket of B and the receipt of B first, and B is the provenance

# account-linking Specification

## Purpose

Kratos writes every link between an account and a company sign-in. When a subject has no link but its address already belongs to an account, Kratos does not link on the word of the identity provider. The user first signs in to the account with a way that the account already has. This is the account-linking login flow of Kratos.

Login UI does not shorten that step. It adds what Kratos leaves off the page. It also keeps track of which sign-in the user links.

## Requirements
### Requirement: The account-linking page offers the account's own ways in
A login flow that carries the Kratos message `1010016` is the account-linking flow. For this flow the extension SHALL keep every sign-in that Kratos lists. This also applies when the tenant requires company sign-in, because this sign-in only proves the account. The extension SHALL also add the other company sign-ins of the account. Kratos hides them because they share the provider that brought the user there.

These rules apply to the added company sign-ins:

- The account is the identity that holds the address of the flow (Kratos admin API).
- The company sign-ins of the account are the links that the SSO service lists for it (`ListLinks` with `identity_id`).
- Each company sign-in is a submit node of group `sso`, with name `sso_account_link` and the connection id as value.
- The label of each node is "Continue with <label>".
- The extension MUST leave out the connection that the user links now (the connection in the sign-in cookie). Its identity provider answers with the subject that has no link, so it cannot prove the account.
- If the account has no such company sign-in and the flow has no other first factor, the page shows the recovery prompt of `sign-in-screen`.

#### Scenario: Account with a password, tenant that requires company sign-in
- **WHEN** a member with a portal password returns from their first company sign-in
- **THEN** the login page shows the message of Kratos and the password field
- **AND** this is true although the tenant offers no password otherwise

#### Scenario: Account whose only way in is another company sign-in
- **WHEN** the account has a link to connection A and returns from a first sign-in with connection B
- **THEN** the page offers "Continue with <label of A>"
- **AND** the page does not offer B

### Requirement: Proving the account with another company sign-in
A submission of `sso_account_link` SHALL start that company sign-in on the account-linking flow itself. This is so that Kratos signs the user in through the existing link and then adds the pending link. The extension MUST first check these points:

- The flow is an account-linking flow. If not, the answer is `400`.
- The connection is one that the account of the flow is linked to. If not, the answer is `403` "Provider not allowed".
- The sign-in cookie of this browser holds a pending attempt, with a ticket and a tenant, for another connection than the one picked. If not, the answer is `400`.

The new attempt SHALL be for the tenant of the pending attempt and the address of the flow. Next to the new ticket, the sign-in cookie SHALL keep the ticket that the user links.

#### Scenario: A connection the account is not linked to
- **WHEN** `sso_account_link` names a connection that is not among the links of the account
- **THEN** the response is `403`
- **AND** the extension asks for no ticket

### Requirement: The linked sign-in is the one the session counts for
When the session comes back and the sign-in cookie holds a ticket that the user links, the extension SHALL ask the SSO service to confirm that ticket first. The extension SHALL ask it to confirm the proving ticket only if the first answer is `NOT_APPLICABLE`.

The extension SHALL send each ticket with its own receipt (`company-sign-in`). The browser holds these receipts:

- one for the sign-in that the user links, which the SSO service accepted before Kratos asked the user to prove the account
- one for the sign-in that proved the account

The connection of the first ticket that the SSO service confirms is the provenance of the session.

When Kratos added the company sign-in to a session by account linking, the extension SHALL check that session as a company-sign-in session. In such a session `byo-sso` is not the first authentication method.

#### Scenario: Linked after signing in with the password
- **WHEN** the methods of the session are `password` then `oidc`/`byo-sso`
- **AND** the SSO service confirms the attempt
- **THEN** the session passes the checks of the tenant that requires company sign-in
- **AND** the linked connection is the provenance

#### Scenario: Linked through the account's other company sign-in
- **WHEN** the user proved the account with connection A while the user linked connection B
- **THEN** the extension calls `CompleteAttempt` first with the ticket of B and the receipt of B
- **AND** B is the provenance


## MODIFIED Requirements

### Requirement: Flow-level messages are shown
The login page SHALL render the messages of `flow.ui.messages` as notifications. It SHALL render an error as an alert. The Connected accounts page SHALL render the error messages among them. A page SHALL NOT render an error message as a flow-level message if the backend maps the id of the message to its own text. The keys of `uiErrorText` in `pkg/kratos/ui_errors.go` are these ids. The page shows the answer of the backend where the error happened.

#### Scenario: Second-factor prompt
- **WHEN** Kratos returns a login flow at `aal2` with the message "Please complete the second authentication challenge."
- **THEN** the login page shows that message

#### Scenario: A refusal on the way back from a provider
- **WHEN** the browser returns from an external provider to a login flow that carries an error message
- **THEN** the login page shows that error above the form

#### Scenario: An error the backend answers itself
- **WHEN** the frontend fetches a login flow again after a wrong password
- **AND** the flow still carries Kratos's message for the wrong password
- **THEN** the login page does not add that message to the one it showed for the refused submission

#### Scenario: Connected accounts shows errors only
- **WHEN** the settings flow of the Connected accounts page carries an error message and an informational message
- **THEN** the page shows the error and not the informational message

### Requirement: No automatic forward away from an error
If `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is on, the login page SHALL NOT forward to its single provider automatically while it shows a flow-level error. An error that the page does not show SHALL NOT stop the forward. While the page forwards, it SHALL NOT render flow-level messages. The page leaves before the user can read them.

#### Scenario: Single provider and an error
- **WHEN** sequencing is on
- **AND** the login flow has one provider and carries an error message that the page shows
- **THEN** the page shows the message and waits for the user

#### Scenario: Single provider and an error the backend answers itself
- **WHEN** sequencing is on
- **AND** the login flow has one provider and carries only an error that the page does not show
- **THEN** the page forwards to the provider

#### Scenario: Single provider and an informational message
- **WHEN** sequencing is on
- **AND** the login flow has one provider and carries only an informational message
- **THEN** the page forwards to the provider and does not show the message

### Requirement: A second-factor redirect with no way back gets one
When a flow request fails with `session_aal2_required` and the redirect names neither `return_to` nor `login_challenge`, the frontend SHALL add the current page as `return_to`. The frontend SHALL remove the `flow` parameter from that address, so that the page starts a new flow when the browser comes back. The frontend SHALL do this before it follows the redirect. When the redirect names either one, the frontend SHALL follow the redirect unchanged. When the redirect is empty or is not a URL, the frontend SHALL also follow it unchanged.

#### Scenario: Second factor asked from an account page
- **WHEN** the answer to a settings request is `session_aal2_required` with a redirect that has neither `return_to` nor `login_challenge`
- **THEN** the browser follows the redirect with `return_to` set to the page it was on, without its `flow` parameter

#### Scenario: A redirect that carries a login challenge
- **WHEN** the answer to a login request is `session_aal2_required` with a redirect that names a `login_challenge`
- **THEN** the browser follows the redirect as it is

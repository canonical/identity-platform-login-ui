## Purpose

Kratos tells the frontend about a flow in more than its form fields. It attaches messages to the flow as a whole. Two examples are the prompt for the second factor and the reason why a sign-in at an external provider was refused. Kratos also answers a flow request with an error that redirects the browser to the second factor.

This capability states what the frontend does with both. The frontend does this so that a user sees what Kratos said and is never left on a page that cannot continue.

Two boundaries are deliberate:

- When the backend already answers an error in its own words, the frontend shows the error once, where it happened. The frontend does not show it again as a flow-level message.
- Only the login page and the Connected accounts page show flow-level messages. The other pages are not part of this capability.

## ADDED Requirements

### Requirement: Flow-level messages are shown
The login page SHALL render the messages of `flow.ui.messages` as notifications. It SHALL render an error as an alert. The Connected accounts page SHALL render the error messages among them. A page SHALL NOT render a message as a flow-level message if the backend maps the id of the message to its own text. The keys of `uiErrorText` in `pkg/kratos/ui_errors.go` are these ids. The page shows the answer of the backend where the error happened.

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
If `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is on, the login page SHALL NOT forward to its single provider automatically while it shows a flow-level error. An error that the page does not show SHALL NOT stop the forward.

#### Scenario: Single provider and an error
- **WHEN** sequencing is on
- **AND** the login flow has one provider and carries an error message that the page shows
- **THEN** the page shows the message and waits for the user

#### Scenario: Single provider and an error the backend answers itself
- **WHEN** sequencing is on
- **AND** the login flow has one provider and carries only an error that the page does not show
- **THEN** the page forwards to the provider

### Requirement: A second-factor redirect with no way back gets one
When a flow request fails with `session_aal2_required` and the redirect names neither `return_to` nor `login_challenge`, the frontend SHALL add the current page as `return_to`. The frontend SHALL do this before it follows the redirect. When the redirect names either one, the frontend SHALL follow the redirect unchanged.

#### Scenario: Second factor asked from an account page
- **WHEN** the answer to a settings request is `session_aal2_required` with a redirect that has neither `return_to` nor `login_challenge`
- **THEN** the browser follows the redirect with `return_to` set to the page it was on

#### Scenario: A redirect that carries a login challenge
- **WHEN** the answer to a login request is `session_aal2_required` with a redirect that names a `login_challenge`
- **THEN** the browser follows the redirect as it is

### Requirement: The Connected accounts page follows a flow error
The Connected accounts page SHALL act on an error from the backend as on any flow error when both of these conditions are true:

- the settings flow of the page cannot be created
- the backend answers with the error as JSON

A response that is not a string SHALL NOT stop the page from handling it.

#### Scenario: Connected accounts with the second factor still owed
- **WHEN** the Connected accounts page asks for a settings flow
- **AND** the answer is `session_aal2_required` with a redirect
- **THEN** the browser follows the redirect to the second factor

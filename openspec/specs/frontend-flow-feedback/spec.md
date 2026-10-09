# frontend-flow-feedback Specification

## Purpose

Kratos tells the frontend about a flow in more than its form fields: it attaches messages to the flow as a whole (the prompt for the second factor, the reason a sign-in at an external provider was refused), and it answers a flow request with an error that redirects the browser to the second factor. This capability states what the frontend does with both, so that a user is told what Kratos said and is never left on a page that cannot continue.

Two boundaries are deliberate. An error the backend already answers in its own words is shown once, where it happened, and not again as a flow-level message. And only the login page and the Connected accounts page show flow-level messages: the other pages are not part of this capability.

## Requirements
### Requirement: Flow-level messages are shown
The login page SHALL render the messages of `flow.ui.messages` as notifications, an error as an alert, and the Connected accounts page SHALL render the error messages among them. A message whose id the backend maps to its own text (the keys of `uiErrorText` in `pkg/kratos/ui_errors.go`) SHALL NOT be rendered as a flow-level message: the page shows the backend's answer where the error happened.

#### Scenario: Second-factor prompt
- **WHEN** Kratos returns a login flow at `aal2` with the message "Please complete the second authentication challenge."
- **THEN** the login page shows that message

#### Scenario: A refusal on the way back from a provider
- **WHEN** the browser returns from an external provider to a login flow that carries an error message
- **THEN** the login page shows that error above the form

#### Scenario: An error the backend answers itself
- **WHEN** a login flow is fetched again after a wrong password, and still carries Kratos's message for it
- **THEN** the login page does not add that message to the one it showed for the refused submission

#### Scenario: Connected accounts shows errors only
- **WHEN** the settings flow of the Connected accounts page carries an error message and an informational one
- **THEN** the page shows the error and not the informational one

### Requirement: No automatic forward away from an error
With `OIDC_WEBAUTHN_SEQUENCING_ENABLED`, the login page SHALL NOT forward to its single provider automatically while it shows a flow-level error. An error the page leaves out SHALL NOT hold the forward back.

#### Scenario: Single provider and an error
- **WHEN** sequencing is on, and the login flow has one provider and carries an error message the page shows
- **THEN** the page shows the message and waits for the user

#### Scenario: Single provider and an error the backend answers itself
- **WHEN** sequencing is on, and the login flow has one provider and carries only an error the page leaves out
- **THEN** the page forwards to the provider

### Requirement: A second-factor redirect with no way back gets one
When a flow request fails with `session_aal2_required` and the redirect names neither `return_to` nor `login_challenge`, the frontend SHALL add the page it is on as `return_to` before it follows the redirect. A redirect that names either SHALL be followed unchanged.

#### Scenario: Second factor asked from an account page
- **WHEN** a settings request is answered `session_aal2_required` with a redirect that has neither `return_to` nor `login_challenge`
- **THEN** the browser follows the redirect with `return_to` set to the page it was on

#### Scenario: A redirect that carries a login challenge
- **WHEN** a login request is answered `session_aal2_required` with a redirect that names a `login_challenge`
- **THEN** the browser follows the redirect as it is

### Requirement: The Connected accounts page follows a flow error
When the settings flow of the Connected accounts page cannot be created and the backend answers with an error as JSON, the page SHALL act on it as on any flow error. A response that is not a string SHALL NOT stop the page from handling it.

#### Scenario: Connected accounts with the second factor still owed
- **WHEN** the Connected accounts page asks for a settings flow and the answer is `session_aal2_required` with a redirect
- **THEN** the browser follows the redirect to the second factor


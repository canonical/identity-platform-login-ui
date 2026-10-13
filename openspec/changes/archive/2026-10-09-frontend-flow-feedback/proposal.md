## Why

Kratos tells the frontend about a flow in two ways that the frontend does not act on:

- Messages that Kratos attaches to a flow as a whole.
- Errors that answer a flow request with a redirect that the frontend cannot follow.

### Problem statement

- `flow.ui.messages` holds the messages that Kratos attaches to a flow and not to one of its nodes. One example is the prompt for the second factor. Another example is the reason why an external provider refused a sign-in, when the browser comes back from it. No page renders these messages. A user who is sent back to the login page after a refusal sees the page again with no explanation.
- A flow request can be answered with `session_aal2_required`. The answer carries the URL of the second-factor login. A settings flow is created without a `return_to`. So that URL names neither `return_to` nor `login_challenge`. The frontend follows it as it is. The login endpoint answers 400 ("One of return_to or login_challenge must be provided").
- The Connected accounts page does not pass a failed creation of its settings flow to `handleFlowError`. Its own handler calls `trim()` on the response body. This throws when the body is an error as JSON. The page keeps loading.

The last two problems happen if Kratos lets a session at `aal1` reach the account pages (`session.whoami.required_aal: aal1`) for an account that has a second factor. With `highest_available`, Kratos refuses that session before anyone asks for a settings flow.

### Scope

- Render flow-level messages on the login page, and the errors among them on the Connected accounts page.
- Do not forward automatically to a single provider while the page shows such an error.
- Add a `return_to` to a second-factor redirect that has no way back.
- Let the Connected accounts page follow a flow error.

### Non-goals

- Flow-level messages on the other pages: registration, recovery, verification, and the settings pages other than Connected accounts.
- Any behaviour of the backend. The flows that the backend returns do not change.
- A change to the texts of Kratos's messages, or a translation of them.
- The errors that the backend already answers in its own words. They stay where they are shown today.

### Success criteria

- A user who must give the second factor reads Kratos's prompt for it on the page.
- No error is shown twice. The page does not render an error that the backend answers itself as a flow-level message.
- A session at `aal1` that opens an account page goes to the second factor and then back to that page.
- ESLint reports no error and the Playwright specs in `ui/tests` pass.

## What Changes

- New `ui/components/FlowMessages.tsx` renders `flow.ui.messages` as notifications. It renders an error as an alert. `ui/pages/login.tsx` renders every type. `ui/pages/manage_connected_accounts.tsx` renders the errors.
- `ui/util/constants.ts` lists the message ids that the backend maps to its own text (`uiErrorText` in `pkg/kratos/ui_errors.go`). `uiErrorText` gets a comment that points at the list. `FlowMessages` leaves those messages out.
- `ui/pages/login.tsx`: if `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is on, the automatic forward to a single provider waits while the page shows a flow-level error.
- `ui/util/handleFlowError.ts`: for `session_aal2_required`, the frontend adds the current page as `return_to` when the redirect names neither `return_to` nor `login_challenge`.
- `ui/pages/manage_connected_accounts.tsx`: a failed creation of the settings flow goes to `handleFlowError` first.
- Two more changes are visible to users. The line "Please complete the second authentication challenge." appears on the second-factor page. Kratos's other flow-level messages appear on the login page wherever Kratos sets them.

## Capabilities

### New Capabilities

- `frontend-flow-feedback`: what the frontend does with the messages that Kratos attaches to a flow, and with a flow error that redirects to the second factor.

### Modified Capabilities

<!-- None. -->

## Impact

- **Frontend**: `ui/components/FlowMessages.tsx` (new), `ui/pages/login.tsx`, `ui/pages/manage_connected_accounts.tsx`, `ui/util/constants.ts`, `ui/util/handleFlowError.ts`. One assertion is added to `ui/tests/use-backup-codes.spec.ts`.
- **Backend Go**: one comment in `pkg/kratos/ui_errors.go`. No behaviour changes.
- **Kratos**: no configuration change. Its flow-level messages become visible.
- **Hydra**, **cookies**, **OpenFGA**: not affected.

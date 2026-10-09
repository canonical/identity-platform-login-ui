## Why

Kratos tells the frontend about a flow in two ways that the frontend does not act on: messages attached to the flow as a whole, and errors that answer a flow request with a redirect it cannot follow.

### Problem statement

- `flow.ui.messages` holds the messages Kratos attaches to a flow and not to one of its nodes: the prompt for the second factor, or the reason an external provider's sign-in was refused when the browser comes back from it. No page renders them, so a user who is sent back to the login page after a refusal sees the page again with no explanation.
- A flow request answered with `session_aal2_required` carries the URL of the second-factor login. A settings flow is created without a `return_to`, and then that URL names neither `return_to` nor `login_challenge`. The frontend follows it as it is, and the login endpoint answers 400 ("One of return_to or login_challenge must be provided").
- The Connected accounts page does not pass a failed creation of its settings flow to `handleFlowError`. Its own handler calls `trim()` on the response body, which throws when the body is an error as JSON, and the page keeps loading.

The last two are reached where Kratos lets a session at `aal1` reach the account pages (`session.whoami.required_aal: aal1`) for an account that has a second factor. With `highest_available`, Kratos refuses that session before a settings flow is asked for.

### Scope

- Render flow-level messages on the login page, and the errors among them on the Connected accounts page.
- Do not forward automatically to a single provider while such an error is shown.
- Add a `return_to` to a second-factor redirect that has no way back.
- Let the Connected accounts page follow a flow error.

### Non-goals

- Flow-level messages on the other pages (registration, recovery, verification, the settings pages other than Connected accounts).
- Any backend behaviour: the flows the backend returns are unchanged.
- Changing the texts of Kratos's messages, or translating them.
- The errors the backend already answers in its own words: they stay where they are shown today.

### Success criteria

- A user who is asked for the second factor reads Kratos's prompt for it on the page.
- No error is shown twice: an error the backend answers itself is not also rendered as a flow-level message.
- A session at `aal1` that opens an account page is taken to the second factor and back to that page.
- ESLint reports no error and the Playwright specs in `ui/tests` pass.

## What Changes

- New `ui/components/FlowMessages.tsx`: renders `flow.ui.messages` as notifications, an error as an alert. `ui/pages/login.tsx` renders every type; `ui/pages/manage_connected_accounts.tsx` renders the errors.
- `ui/util/constants.ts` lists the message ids the backend maps to its own text (`uiErrorText` in `pkg/kratos/ui_errors.go`, which gains a comment pointing at the list). `FlowMessages` leaves those out.
- `ui/pages/login.tsx`: with `OIDC_WEBAUTHN_SEQUENCING_ENABLED`, the automatic forward to a single provider waits while a flow-level error is shown.
- `ui/util/handleFlowError.ts`: for `session_aal2_required`, the current page is added as `return_to` when the redirect names neither `return_to` nor `login_challenge`.
- `ui/pages/manage_connected_accounts.tsx`: a failed creation of the settings flow goes to `handleFlowError` first.
- User-visible besides the fixes: the line "Please complete the second authentication challenge." appears on the second-factor page, and Kratos's other flow-level messages on the login page wherever Kratos sets them.

## Capabilities

### New Capabilities

- `frontend-flow-feedback`: what the frontend does with the messages Kratos attaches to a flow and with a flow error that redirects to the second factor.

### Modified Capabilities

<!-- None. -->

## Impact

- **Frontend**: `ui/components/FlowMessages.tsx` (new), `ui/pages/login.tsx`, `ui/pages/manage_connected_accounts.tsx`, `ui/util/constants.ts`, `ui/util/handleFlowError.ts`; one assertion added to `ui/tests/use-backup-codes.spec.ts`.
- **Backend Go**: one comment in `pkg/kratos/ui_errors.go`. No behaviour changes.
- **Kratos**: no configuration change. Its flow-level messages become visible.
- **Hydra**, **cookies**, **OpenFGA**: unaffected.

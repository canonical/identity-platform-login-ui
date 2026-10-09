## Context

A Kratos flow carries messages in two places: on a node (`node.messages`), which the node components render next to the input, and on the flow (`flow.ui.messages`), which nothing rendered. Kratos uses the second for what concerns the whole step. In v25.4.0, `selfservice/flow/login/handler.go:262` adds "Please complete the second authentication challenge." to every login flow at `aal2`, and an error an external provider answers with comes back as a message on the login flow (`selfservice/strategy/oidc/strategy.go:396` makes it a 400 with the reason, which `ui/container/container.go:167-172` turns into a message of the flow).

The backend already turns some of Kratos's errors into its own answer: `uiErrorText` in `pkg/kratos/ui_errors.go` maps their ids to the text the pages show where the error happened (under the password field, on the code page). Those errors can also still sit in the flow the frontend fetches next.

## Decisions

### One component, and a list of what it leaves out

`ui/components/FlowMessages.tsx` renders the messages it is given as Vanilla notifications (`error` as an alert, `success` as positive, anything else as information). It leaves out a message whose id is in `ORY_ERR_ANSWERED_BY_BACKEND` (`ui/util/constants.ts`), the keys of `uiErrorText`: the page already shows the backend's text for it, and showing Kratos's wording as well would report one mistake twice.

The list is a copy of the backend's keys. A comment at `uiErrorText` names it, so that an id added there is added here. A copy that falls behind shows an error twice; it never hides one.

*Alternative, turned down:* removing those messages from the flow in the backend. It changes what the backend returns for every client of its API, for a concern that is the page's.

### Where they are shown

- `ui/pages/login.tsx`: every type, above the form. The login page is where Kratos sends the browser back with a refusal or a prompt.
- `ui/pages/manage_connected_accounts.tsx`: errors only. The page reports its successes with toasts already, and an informational line of Kratos's next to them would say the same thing twice.

### No automatic forward while an error is shown

With `OIDC_WEBAUTHN_SEQUENCING_ENABLED`, a login page whose only option is one provider forwards to it at once. A user who has just come back from that provider with a refusal would be sent there again before reading why. The forward waits while a flow-level error is shown; an error the component leaves out does not hold it back, so the pages that forwarded before still do.

### The second-factor redirect

`handleFlowError` follows the redirect of `session_aal2_required`. For that case only, it first adds the current page as `return_to` when the URL has neither `return_to` nor `login_challenge`. Kratos names a way back only when the flow was created with one (v25.4.0 `selfservice/flow/settings/handler.go:313-319`), and the login endpoint needs one (`handleCreateFlow` in `pkg/kratos/handlers.go` answers 400 without). A redirect that names either is followed unchanged.

### Connected accounts

The page's request for a settings flow gets `handleFlowError("settings", setFlow)` in front of its own handler, as `reset_password.tsx` and `setup_backup_codes.tsx` have. Its own handler now checks that the body is a string before it calls `trim()`.

## Risks / Trade-offs

- **A visible change for every deployment.** Kratos's flow-level messages appear where they did not. On the paths the Playwright specs walk, that is the second-factor prompt. Other messages Kratos sets on a login flow become visible too, in Kratos's English wording.
- **Two lists to keep in step**, the backend's map and the frontend's ids. Accepted for now; the comment is the guard.
- **Not covered by a Playwright spec:** the second-factor redirect and the Connected accounts fix need Kratos at `aal1`, which the dev stack does not run.

## Rollout

No setting and no migration. Nothing to observe in logs or metrics: the backend is unchanged.

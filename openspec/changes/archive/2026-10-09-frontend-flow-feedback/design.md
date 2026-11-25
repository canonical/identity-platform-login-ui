## Context

A Kratos flow carries messages in two places:

- On a node (`node.messages`). The node components render these next to the input.
- On the flow (`flow.ui.messages`). Nothing rendered these.

Kratos uses the second place for what concerns the whole step. In v25.4.0, `selfservice/flow/login/handler.go:262` adds "Please complete the second authentication challenge." to every login flow at `aal2`. An external provider can answer with an error. That error comes back as a message on the login flow. `selfservice/strategy/oidc/strategy.go:396` makes it a 400 with the reason. `ui/container/container.go:167-172` turns it into a message of the flow.

The backend already turns some of Kratos's errors into its own answer. `uiErrorText` in `pkg/kratos/ui_errors.go` maps the ids of these errors to the text that the pages show where the error happened. Two examples of such places are under the password field and on the code page. These errors can also still be in the flow that the frontend fetches next.

## Decisions

### One component, and a list of what it leaves out

`ui/components/FlowMessages.tsx` renders the messages that it gets as Vanilla notifications. An `error` is an alert, a `success` is positive, and any other type is information.

The component leaves out a message if its id is in `ORY_ERR_ANSWERED_BY_BACKEND` (`ui/util/constants.ts`). This list holds the keys of `uiErrorText`. The page already shows the backend's text for such a message. If the page also shows Kratos's wording, it reports one mistake twice.

The list is a copy of the backend's keys. A comment at `uiErrorText` names the list. The comment is there so that whoever adds an id there adds it here. A copy that falls behind shows an error twice. It never hides an error.

*Alternative, turned down:* the backend removes those messages from the flow. We turned this down because it changes what the backend returns for every client of its API. The concern belongs to the page.

### Where they are shown

- `ui/pages/login.tsx` shows every type, above the form. Kratos sends the browser back to the login page with a refusal or a prompt.
- `ui/pages/manage_connected_accounts.tsx` shows errors only. The page already reports its successes with toasts. Kratos's informational line next to them says the same thing twice.

### No automatic forward while an error is shown

If `OIDC_WEBAUTHN_SEQUENCING_ENABLED` is on and the only option of a login page is one provider, the page forwards to it at once. Without the wait, a user can come back from that provider with a refusal. The page then sends this user there again before the user reads the reason.

So the forward waits while the page shows a flow-level error. An error that the component leaves out does not stop the forward. The pages that forwarded before still do.

### The second-factor redirect

`handleFlowError` follows the redirect of `session_aal2_required`. For that case only, it first adds the current page as `return_to`. It does this when the URL has neither `return_to` nor `login_challenge`.

Kratos names a way back only when the flow was created with one (v25.4.0 `selfservice/flow/settings/handler.go:313-319`). The login endpoint needs a way back. Without one, `handleCreateFlow` in `pkg/kratos/handlers.go` answers 400. The frontend follows a redirect that names either one unchanged.

### Connected accounts

The page's request for a settings flow gets `handleFlowError("settings", setFlow)` in front of its own handler. `reset_password.tsx` and `setup_backup_codes.tsx` have the same call. The page's own handler now checks that the body is a string before it calls `trim()`.

## Risks / Trade-offs

- **A visible change for every deployment.** Kratos's flow-level messages appear where they did not appear before. On the paths that the Playwright specs walk, this is the second-factor prompt. Other messages that Kratos sets on a login flow also become visible. They are in Kratos's English wording.
- **Two lists to keep in step.** These are the backend's map and the frontend's ids. We accept this for now. The comment is the guard.
- **No Playwright spec covers two changes.** These are the second-factor redirect and the Connected accounts fix. Both need Kratos at `aal1`. The dev stack does not run Kratos at `aal1`.

## Rollout

There is no setting and no migration. There is nothing to observe in logs or metrics, because the backend is unchanged.

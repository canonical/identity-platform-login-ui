## 1. Flow errors

- [x] 1.1 `ui/util/handleFlowError.ts`: `getSecondFactorRedirect` adds the current page as `return_to` to a `session_aal2_required` redirect that names neither `return_to` nor `login_challenge`; the other cases keep `getRedirectToFromError`.
- [x] 1.2 `ui/pages/manage_connected_accounts.tsx`: `handleFlowError("settings", setFlow)` in front of the page's own handler of a failed settings flow, which checks that the body is a string before `trim()`.

## 2. Flow-level messages

- [x] 2.1 `ui/components/FlowMessages.tsx`: notifications for the messages it is given, optionally of some types only, without those `isErrorAnsweredByBackend` names.
- [x] 2.2 `ui/util/constants.ts`: `ORY_ERR_ANSWERED_BY_BACKEND`, the keys of `uiErrorText`, and `isErrorAnsweredByBackend`; a comment at `uiErrorText` in `pkg/kratos/ui_errors.go` that names the list.
- [x] 2.3 `ui/pages/login.tsx`: `FlowMessages` above the form; `isSingleOidcOption` is false while a flow-level error is shown.
- [x] 2.4 `ui/pages/manage_connected_accounts.tsx`: `FlowMessages` with the errors of the settings flow.

## 3. Tests and validation

- [x] 3.1 `ui/tests/use-backup-codes.spec.ts`: the second sign-in expects "Please complete the second authentication challenge." on the second-factor page.
- [x] 3.2 `npm run lint` in `ui/` reports no error and `npx tsc --noEmit` passes.
- [x] 3.3 `npx @fission-ai/openspec validate frontend-flow-feedback --strict` passes, and the change is archived so that `openspec/specs/frontend-flow-feedback/spec.md` holds the requirements.

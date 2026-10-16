## 1. Flow errors

- [x] 1.1 `ui/util/handleFlowError.ts`: `getSecondFactorRedirect` adds the current page as `return_to` to a `session_aal2_required` redirect that has neither `return_to` nor `login_challenge`. The other cases keep `getRedirectToFromError`.
- [x] 1.2 `ui/pages/manage_connected_accounts.tsx`: put `handleFlowError("settings", setFlow)` in front of the page's own handler of a failed settings flow. The handler checks that the body is a string before `trim()`.

## 2. Flow-level messages

- [x] 2.1 `ui/components/FlowMessages.tsx`: show notifications for the messages it gets, for some types only if asked. Leave out the messages that `isErrorAnsweredByBackend` names.
- [x] 2.2 `ui/util/constants.ts`: add `ORY_ERR_ANSWERED_BY_BACKEND` (the keys of `uiErrorText`) and `isErrorAnsweredByBackend`. Add a comment at `uiErrorText` in `pkg/kratos/ui_errors.go` that names the list.
- [x] 2.3 `ui/pages/login.tsx`: put `FlowMessages` above the form. Make `isSingleOidcOption` false while the page shows a flow-level error.
- [x] 2.4 `ui/pages/manage_connected_accounts.tsx`: put `FlowMessages` with the settings flow's errors on the page.

## 3. Tests and validation

- [x] 3.1 `ui/tests/use-backup-codes.spec.ts`: the second sign-in expects "Please complete the second authentication challenge." on the second-factor page.
- [x] 3.2 `npm run lint` in `ui/` reports no error and `npx tsc --noEmit` passes.
- [x] 3.3 `npx @fission-ai/openspec validate frontend-flow-feedback --strict` passes. The change is archived, so `openspec/specs/frontend-flow-feedback/spec.md` holds the requirements.

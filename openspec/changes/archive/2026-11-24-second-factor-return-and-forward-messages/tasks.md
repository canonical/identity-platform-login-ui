## 1. Specification

- [x] 1.1 In `frontend-flow-feedback`, change the three requirements. Check with `openspec validate --all --strict`.

## 2. Code

- [x] 2.1 In `ui/util/handleFlowError.ts`, remove `flow` from the added `return_to`, and follow an empty or unreadable redirect unchanged.
- [x] 2.2 In `ui/components/FlowMessages.tsx`, export the function that finds the shown messages. Use it in `ui/pages/login.tsx`.
- [x] 2.3 In `ui/pages/login.tsx`, show no flow-level message while the page forwards.
- [x] 2.4 In `ui/util/constants.ts`, make `isErrorAnsweredByBackend` true only for error messages.
- [x] 2.5 Check with `npm run lint`, `npx tsc --noEmit` and `npm run build`. The Playwright test in `ui/tests/use-backup-codes.spec.ts` matches the second-factor prompt exactly.

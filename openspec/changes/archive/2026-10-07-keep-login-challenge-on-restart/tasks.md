## 1. Login steps carry the challenge

- [x] 1.1 In `ui/pages/login.tsx`, in the branch that fetches a flow by id (`kratos.getLoginFlow`): after `setFlow`, when the flow has `oauth2_login_challenge` and the URL has no `login_challenge`, add it with `router.replace({ query: { ...router.query, login_challenge } }, undefined, { shallow: true })`. Completion: on the dev stack the password step of a client's login shows `login_challenge` in its URL.
- [x] 1.2 Leave the redirect after the email (`loginIdentifierFirst`) as it is on `main`: the step it leads to gets the challenge from 1.1.

## 2. A restart keeps the challenge

- [x] 2.1 In `ui/util/handleFlowError.ts` `newFlowUrl`, add a `login` case: return `./login?login_challenge=<encoded>` when `window.location.search` has both `login_challenge` and `flow`, and `./login` otherwise. Update the comment above the function. Other flow types keep their branches.
- [x] 2.2 `npx eslint pages/login.tsx util/handleFlowError.ts`, `npx prettier --check` on both and `npx tsc --noEmit -p .` in `ui/`. Completion: all clean.

## 3. Tests

- [x] 3.1 Add `ui/tests/back-on-second-factor.spec.ts` with a helper that presses Back, expects the email step and expects `login_challenge` in its URL, and two tests: Back on the authenticator setup of a first sign-in, and Back on the second-factor page of a later sign-in; both sign in again and finish at the OIDC client with tokens.
- [x] 3.2 On the `docker-compose.dev.yml` stack (`ui/tests/scripts/01-start-cluster.sh`, `02-start-ui.sh`, `03-start-oidc-app.sh`): `npx playwright test tests/back-on-second-factor.spec.ts` passes with the change, and both tests fail without it at the URL assertion (the restarted login is at `/ui/login?flow=…` with no challenge).

## 4. Verification beyond the dev stack

- [x] 4.1 Run the walks where the backend does not pass the challenge to Kratos, with an image built from the branch on the identity platform's test plane (`canonical/canonical-identity-platform-testing`, `tests/browser/scenarios/resilience-scenarios.ts`): Back on the second factor reaches the client with `OIDC_WEBAUTHN_SEQUENCING_ENABLED=true` and with `MULTI_TENANCY_ENABLED=true`, and so does Back after a tenant selection for a user with several tenants, whose token carries the tenant picked. The rest of that suite gives the same results as on `main`.
- [x] 4.2 Read, not run: a restart with an expired challenge, and a flow creation refused with `self_service_flow_return_to_forbidden`. The outcomes are in `design.md` (D2, D3) and in the spec.

## 5. Specification

- [x] 5.1 `npx @fission-ai/openspec validate keep-login-challenge-on-restart --strict` passes.
- [x] 5.2 Archive the change so that `openspec/specs/login-restart/spec.md` holds the requirements and no active change is left (`npx @fission-ai/openspec list --json` returns no change).

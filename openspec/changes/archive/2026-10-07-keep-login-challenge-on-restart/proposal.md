## Why

A login started by an OAuth 2.0 client is identified by Hydra's login challenge. Issue #984: when the user presses the browser's Back button on the second-factor page of such a login, the Login UI shows the email step of a new login that has no challenge. Signing in there ends on the account page, and the client never receives its authorization code. The same happens with Back on the authenticator setup of a first sign-in.

### Problem statement

- Kratos regenerates its CSRF token when the first factor creates the session, so the flow of the step that Back returns to can no longer be fetched (`security_csrf_violation`).
- `ui/util/handleFlowError.ts` answers that by starting a new flow at `./login`, with no query parameters, so the new flow is not tied to the client's request.
- The URL of the step that Back returns to does not carry the challenge in the first place. Kratos redirects to the login UI with only the flow id, the tenant selection sends the browser to `/ui/login?flow=<id>`, and so does the redirect to the second factor.

### Scope

- Every login step that the browser can return to carries the client's login challenge in its URL.
- A login that is restarted because its flow can no longer be used is started for the same login challenge.
- State what happens when the client's request can no longer be completed by the time of the restart.

### Non-goals

- Answering "already signed in" when the login for the challenge has already completed. Hydra answers the same for a completed and a pending challenge (the challenge is the encoded flow), so this needs the Login UI to remember the challenges it accepted. The outcome of that case is stated in the spec and accepted as it is.
- A dedicated page for a login request that expired.
- Keeping the destination (`return_to`) of a restarted login that has no client.
- Any change to the restart of registration, recovery, settings or verification flows.
- Any backend (`pkg/`) change.

### Success criteria

- Back on the second-factor page, and Back on the authenticator setup of a first sign-in, followed by signing in again, return the user to the client with an authorization code.
- The same holds for a user with several tenants, whose password step is reached from the tenant selection.
- It holds with `OIDC_WEBAUTHN_SEQUENCING_ENABLED` and `MULTI_TENANCY_ENABLED` off, and with either on.
- A login that cannot be created for its challenge does not make the page reload itself forever.
- A login with no client restarts as it did before.

## What Changes

- `ui/pages/login.tsx`: when the page loads a flow by its id and the URL has no `login_challenge`, it adds the flow's `oauth2_login_challenge` to the URL by replacing the current history entry.
- `ui/util/handleFlowError.ts`: a restarted login keeps the `login_challenge` of the current URL when that URL also names a flow. With no flow in the URL the restart is for a flow that could not be created, and it goes to `./login` without the challenge, as before.
- User-visible besides the fix, and accepted: a restart is now a login for the client's request whatever the state of that request.
  - After the login has completed, going back to a login step and signing in again ends at the client with `access_denied` ("The consent verifier has already been used"). Before, that sign-in had no client and ended on the account page.
  - After the challenge has expired, the restarted login ends with an error instead of on the account page.

## Capabilities

### New Capabilities

- `login-restart`: how a login started by an OAuth 2.0 client keeps its login challenge across its steps and across a restart, and how a restart ends when the client's request can no longer be completed.

### Modified Capabilities

<!-- None: no existing spec covers the login flow. -->

## Impact

- **Frontend**: `ui/pages/login.tsx`, `ui/util/handleFlowError.ts`, new Playwright spec `ui/tests/back-on-second-factor.spec.ts`.
- **Backend Go**: unaffected. The frontend reads `oauth2_login_challenge` from the flow the backend already returns: Kratos sets it when it is given the challenge, and `hydrateKratosLoginFlow` in `pkg/kratos/service.go` derives it from the flow's `return_to` otherwise.
- **Kratos**: no configuration change. A restart creates a new login flow for the same challenge, as opening `/ui/login?login_challenge=<challenge>` does.
- **Hydra**: no configuration change. A restart can accept the same login challenge a second time; Hydra refuses the replay of a completed login when the consent verifier is redeemed.
- **Cookies**: unaffected. The backend already deletes the Kratos session cookie when a flow fetch fails with a CSRF violation, which is why a restarted login asks for the credentials again.
- **OpenFGA**: unaffected.
- **URLs**: the login challenge now appears in the URL of every login step, not only of the first. It is added on the Login UI's own page, never to another origin.

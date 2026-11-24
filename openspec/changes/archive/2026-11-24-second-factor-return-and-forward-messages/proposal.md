## Why

### Problem statement

A review of `frontend-flow-feedback` found three gaps:

- The `return_to` that the frontend adds for the second factor keeps the `flow` parameter of the page. When the browser comes back, the page loads that old flow again, which can be expired.
- An empty redirect, or a redirect that is not a URL, became a `return_to` to the same page, or an exception.
- While the login page forwards to its single provider, it shows informational messages for a moment before it leaves.

Also, the frontend hides a message by its id alone. It must hide only error messages.

### Scope

- The `return_to` that `ui/util/handleFlowError.ts` adds for `session_aal2_required`.
- The flow-level messages of the login page while it forwards to its single provider.

### Non-goals

- A change to the messages of the Connected accounts page. That page always creates a new settings flow. A new flow carries no error that the backend answers itself.
- A change to the redirects that Kratos sends.

### Success criteria

- After the second factor, the browser comes back to the page with no `flow` parameter, and the page starts a new flow.
- An empty redirect or a redirect that is not a URL is followed as before this capability.
- The login page shows no flow-level message while it forwards.

## What Changes

- `getSecondFactorRedirect` removes `flow` from the `return_to` that it adds. It follows an empty or unreadable redirect unchanged.
- The login page and `FlowMessages` use one function to find the messages that a page shows.
- The login page shows no flow-level message while it forwards to its single provider.
- `isErrorAnsweredByBackend` is true only for error messages.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `frontend-flow-feedback`: the three requirements above.

## Impact

- Frontend: `ui/util/handleFlowError.ts`, `ui/components/FlowMessages.tsx`, `ui/pages/login.tsx`, `ui/util/constants.ts`.
- Backend, Kratos and Hydra: no change.

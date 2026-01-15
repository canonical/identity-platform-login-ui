# AI Agent Instructions for Chaos Agent XState

This document is intended for AI coding assistants working on this tool.

## Philosophy
This tool replaces the unpredictable "monkey-testing" approach with **Deterministic State Machines**.
- **Goal**: Verify specific "Happy Paths" rigorously.
- **Method**: XState controls the logic flow; Playwright acts as the side-effect handler.

## Architecture

### 1. The Machines (`src/machines/*.machine.ts`)
- **Base Setup**: `src/machines/base.ts` contains the shared `setup()` definition with all actors and their input types.
- **Scenarios**: Each scenario has its own file (e.g., `recovery.machine.ts`) exporting a specific state machine.
- **Logic Only**: The machine should NOT contain Puppeteer/Playwright code. It only manages state transitions and "invokes" promises.

### 2. The Drivers (`src/drivers/*.ts`)
- **Execution Layer**: Split into domain-specific files (`auth.ts`, `mfa.ts`, etc.).
- **CDP Integration**: Heavy use of Chrome DevTools Protocol for WebAuthn in `mfa.ts`.
  - *Pattern*: `enableWebAuthn()` -> `addVirtualAuthenticator` -> `addCredential` (if saving/restoring).
- **Persistence**: Virtual Authenticators die with the session/navigation. You MUST export credentials after creation and re-import them when needed later in the flow.
- **Shared State**: All drivers share a `DriverState` object managed by `root.ts`.

### 3. The Runner (`src/index.ts`)
- Bridges the two using XState v5 Actor model.
- Uses `authMachine.provide({ actors: { ... } })` to inject the driver implementations into the machine's strict logic.
- Defines commandline options to switch profiles/scenarios.

## How to Add a New Scenario

1.  **Update `base.ts`**: Add any new actors (implementation stubs) or types to the `setup()` definition.
2.  **Create Machine**: Create `src/machines/your-scenario.machine.ts`. Import `authSetup` from `./base` and define the linear flow using `.createMachine(...)`.
3.  **Update Drivers**:
    - Implement the actual UI interaction methods in the appropriate driver file.
    - Expose it via `src/drivers/root.ts`.
4.  **Register in `index.ts`**:
    - Import your new machine.
    - Add it to the `machines` dictionary.
    - Add the actor implementation to the `actors` object passed to `provide()`.

## Common Pitfalls

- **WebAuthn Timing**: The UI often transitions *after* the browser handles the WebAuthn signal. Use `Promise.race` or careful `waitFor` logic.
- **Selectors**: The UI uses specific text labels. If tests fail, check if copy changed (e.g., "Sign in" vs "Log in").
- **CDP Session Loss**: If `driver.page.reload()` or navigation happens, the CDP session might need re-attaching or at least the authenticator needs re-adding.

## Debugging
- Use `console.log(chalk.color(...))` for visibility.
- The `index.ts` logs every state transition.

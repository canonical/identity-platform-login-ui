# Future Work

## Completed
- [x] **Parameterization**: Added CLI flags for Base URL, Credentials, and optional Admin API usage.
- [x] **State Management**: Implemented `KratosAdmin` helper to reset user credentials between runs.
- [x] **TOTP Flow**: Implemented `setupTotp` and `verifyTotp` in Driver and Machine.
    - Extract secret from QR code/text.
    - Generate token using `otpauth`.

## Short Term
- [ ] **Password Recovery**: Implement "Forgot Password" flow validation.
- [ ] **Dockerization**: Create `Dockerfile` for portable execution (Deferred).
- [ ] **Parallel Execution**: Allow running multiple scenarios in parallel (Deferred).
- [ ] **CI Integration**: Add a GitHub Action workflow to run these tests on PRs (Deferred).

## Medium Term
- [ ] **State State Explosion Management**: As scenarios grow, the single `machine.ts` file will become unmanageable. Refactor into hierarchical state nodes or separate machine files.
- [ ] **Visual Regression**: Integrate visual snapshots into the Driver state checks.

## Long Term
- [ ] **Model-Based Testing (MBT)**: Use XState's `@xstate/test` to generate paths automatically instead of hardcoding the inputs. This would restore the "Chaos" aspect but in a controlled manner.
- [ ] **Fuzzing Inputs**: Inject unusual strings into email/password fields during the `loggingIn` states.

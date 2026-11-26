## 1. Specification

- [x] 1.1 In `tenant-mfa`, say that `pkg/web` gives the APIs the policy of the extension. Check with `openspec validate --all --strict`.
- [x] 1.2 In `byosso-disabled`, say that the handlers use the policy of `MFA_ENABLED` with the flag off.

## 2. Code

- [x] 2.1 Add `SecondFactorPolicy` to `pkg/byosso/mfa.go`, with one test in `pkg/byosso/mfa_test.go`.
- [x] 2.2 In `pkg/web/byosso.go`, give the policy to both APIs with `WithSecondFactorPolicy`. In `pkg/web/router.go`, give `MFA_ENABLED` to the APIs unchanged.
- [x] 2.3 Remove `WithBackupCodesRegeneration` and its test from `pkg/kratos`. Check with `go vet ./...` and `go test ./... -short`.

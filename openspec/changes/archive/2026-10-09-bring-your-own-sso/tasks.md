## 1. Settings and dependencies

- [x] 1.1 Use the `v0/sso` package and the `v0/tenant` additions of `github.com/canonical/identity-platform-api`. Until a released version has them, `go.mod` requires the commit of the pull request that adds them. `google.golang.org/genproto/googleapis/rpc` becomes a direct requirement.
- [x] 1.2 Add `BYOSSOEnabled`, `SSOServiceGRPCAddress`, `SSOServiceGRPCTimeout` (3s), `SSOServiceTLSEnabled` and `KratosPrivilegedSessionMaxAge` (1h) to `internal/config/specs.go`.
- [x] 1.3 In `serve` (`cmd/serve.go`, `validateTenantSettings`, tested by `TestValidateTenantSettings`) refuse incomplete settings. It refuses BYO-SSO without multi-tenancy. It refuses BYO-SSO without the SSO service address. It refuses BYO-SSO with OIDC-WebAuthn sequencing.
- [x] 1.4 Add `TenantChoice` (`tc`) to `FlowStateCookie` in `internal/cookies/cookies.go`.
- [x] 1.5 Check that `go build ./...` compiles.

## 2. gRPC dial helper

- [x] 2.1 Move `NewGRPCConn` from `pkg/tenants/grpc.go` to `internal/grpc/client.go` as `NewConn(service, address, tlsEnabled, opts...)`. Keep the same keepalive and transport settings. Append the extra dial options. Delete `pkg/tenants/grpc.go`.
- [x] 2.2 Test it in `internal/grpc/client_test.go` (`TestNewConn`, `TestNewConnWithTLS`, `TestNewConnFailOnDialOption`).
- [x] 2.3 Check with `go test ./internal/grpc/...`.

## 3. Extension hooks in `pkg/kratos`

- [x] 3.1 Define `ExtensionInterface` (nine hooks) in `pkg/kratos/interfaces.go`.
- [x] 3.2 Add `pkg/kratos/extension.go` with `Option`, `WithExtension`, `NoOpExtension`, `errResponseWritten` and `UnsetSessionCookie`. Add `WithBackupCodesRegeneration` (the prompt to regenerate backup codes without `mfaEnabled`, tested by `TestWithBackupCodesRegeneration`). Add `WriteGetFlowError` and `WriteUpdateFlowError` (the handlers' answers to a Kratos error, for the extension's own calls). Make `NewAPI` take options and default to the `NoOpExtension`.
- [x] 3.3 In `handleCreateFlow` (`pkg/kratos/handlers.go`) move the verification, MFA and WebAuthn session checks into `enforceSessionChecks` unchanged. With a session and a login challenge, call `HandleSessionLogin` after them if the extension handles session logins. Pass a created flow with no login challenge through `HydrateLoginFlow`.
- [x] 3.4 In `handleCreateFlowWithSession` call `BeforeAcceptLogin` before `AcceptLoginRequest`. Return `errResponseWritten` when the hook answered. Make `handleCreateFlow` and `handleUpdateFlow` return on that error.
- [x] 3.5 In `handleGetLoginFlow` pass the flow through `HydrateLoginFlow`.
- [x] 3.6 In `handleUpdateIdentifierFirstFlow` call `BeforeTenantSelection` before the tenant selection by email.
- [x] 3.7 In `handleUpdateFlow` call `InterceptLoginSubmission` after the flow is fetched. Go on with the request that it returns. If the extension handles session logins, do not ask the tenant resolver's `NeedsTenantSelection`.
- [x] 3.8 In `handleUpdateRegistrationFlow` call `InterceptRegistrationSubmission` before the parse.
- [x] 3.9 In the settings handlers call `InterceptSettingsSubmission` before the parse. Call `HydrateSettingsFlow` on the flows of `handleCreateSettingsFlow`, `handleGetSettingsFlow` and `handleUpdateSettingsFlow`.
- [x] 3.10 Export `RedirectResponse` for the extension's redirects.
- [x] 3.11 Add tests in `pkg/kratos/extension_test.go` (`TestWithExtension`, `TestWithBackupCodesRegeneration`, `TestUnsetSessionCookie`). Every handler test that existed runs with the no-op. Add a test for each call site in `pkg/kratos/handlers_test.go` (`TestHandleGetLoginFlowWithExtension`, `…WhenExtensionAnswers`, `TestHandleCreateFlowWithSessionAndExtension`, `…RedirectToVerification`, `TestHandleCreateFlowWithoutSessionAndExtension`, `TestHandleUpdateIdentifierFirstFlowWhenExtensionAnswers`, `TestHandleUpdateFlowWhenExtensionAnswers`, `TestHandleUpdateFlowWithRequestOfExtension`, `TestHandleUpdateFlowWithSessionAndExtension`, `TestHandleUpdateRegistrationFlowWhenExtensionAnswers`, `TestHandleGetSettingsFlowWithExtension`, `TestHandleUpdateSettingsFlowWhenExtensionAnswers`).
- [x] 3.12 Check that `go test ./pkg/kratos/...` passes. The tests that existed are unchanged.

## 4. Consent hook in `pkg/extra`

- [x] 4.1 Define `ExtensionInterface` (`GateConsent`) in `pkg/extra/interfaces.go`. Add `pkg/extra/extension.go` (`Option`, `WithExtension`, `NoOpExtension`). Make `NewAPI` take options.
- [x] 4.2 In `handleConsent` (`pkg/extra/handlers.go`) call `GateConsent` after the handler reads the consent request. Answer with its redirect, or with `403` on its error. Do not accept the consent.
- [x] 4.3 Add tests in `pkg/extra/extension_test.go`. Add `TestHandleConsentWhenExtensionRejects`, `…Approves`, `TestHandleConsentFailOnGateConsent` and `TestHandleConsentInvalidPasswordAALWithExtension` in `pkg/extra/handlers_test.go`.
- [x] 4.4 Check with `go test ./pkg/extra/...`.

## 5. Options in `pkg/tenants`

- [x] 5.1 In `pkg/tenants/service.go` and `interfaces.go` add `Tenant.Invited` (`invited`). `WithSignInTenants` takes a client of the Tenant Service's `TenantSignInService` (`TenantSignInServiceClientInterface`, only `ListSignInTenants`). A lookup by email then lists the sign-in tenants of the address, with `invited`, in place of `LookupTenants`.
- [x] 5.2 In `pkg/tenants/handlers.go` add `WithSessionLookupByEmail` and `lookupSessionTenants`. `GET /api/v0/tenants` and the empty-selection check use them.
- [x] 5.3 Add tests: `TestLookupTenantsByEmailWithSignInTenants` and `…Error` (`service_test.go`), `TestHandleLookupTenantsBySessionEmail` and `…WithoutEmail` (`handlers_test.go`), and `TestCookieTenantResolverStoreTenantKeepsTenantChoice` (`resolver_test.go`).
- [x] 5.4 Check with `go test ./pkg/tenants/...`.

## 6. The extension: clients and building blocks (`pkg/byosso`)

- [x] 6.1 Add `constants.go` (provider id, node groups and fields, message ids, error ids, cookie names, deadlines). Add `interfaces.go` (the interfaces of the package, mocked with `go:generate` directives in its test files). Mocks are not committed.
- [x] 6.2 In `grpc.go` add the reconnect parameters and the retry service config. Test them in `grpc_test.go` (`TestDialOptionsRetry`).
- [x] 6.3 In `directory.go` add `Directory` over the `TenantSignInService` of the Tenant Service (`SignInTenants`, `SignInContext`, `JoinTenant`, `CreatePersonalTenant`) and the retried methods. `toSignInContext` fails on unspecified and unknown enum values. Test it in `directory_test.go`.
- [x] 6.4 In `sso.go` add `SSO` over the SSO Service (`Options`, `StartAttempt`, `CompleteAttempt`, `Links`, `DeleteLink`) and the retried methods. Turn error reasons into sentinel errors. Test it in `sso_test.go`.
- [x] 6.5 In `hydra.go` add `LoginRequest` (`prompt`, `max_age`, remembered subject), `Reauthenticate`, `NeedsFreshFirstFactor` and `RemembersAnother`. Add `Hydra` (`LoginRequest`, `RevokeLoginSession`, `RejectConsent`). Test it in `hydra_test.go`.
- [x] 6.6 In `kratos.go` add `Verification.Start` and `IdentityFinder` (`IdentityID`, `IdentityExists`, `HasRecoveryCodes`). Test it in `kratos_test.go`.
- [x] 6.7 In `kratos_decorator.go` add `KratosServiceDecorator`, which removes the `byo-sso` nodes from login, registration and settings flows. Test it in `kratos_decorator_test.go`.
- [x] 6.8 In `cookies.go` add `CookieStore` and the four cookies, each sealed together with its name. Add the name of the SSO Service's receipt cookie, and the code that reads it and clears it. Test it in `cookies_test.go` (`TestSetSignIn`, `TestSetProvenance`, `TestGetFreshWithValueOfAnotherCookie`, `TestGetProvenanceFailOnDecrypt`, `TestClearReceipt`).
- [x] 6.9 In `flow.go` and `helpers.go` add the node builders and the flow predicates. Add the redirect and error answers, `isUnavailable` and `budget`. Add `requestFields`. It matches fields in any case, as the handler's decoder does. A field named twice is an error. Add `traitsEmail` (a nested and a flat address that differ are an error), `sameOrigin` and the URL builders. Test them in `flow_test.go` and `helpers_test.go`.

## 7. Sign-in screen

- [x] 7.1 In `handlers.go` add the `API` type, `NewAPI`, `HydrateLoginFlow`, `BeforeTenantSelection`, `InterceptLoginSubmission` and `answerPick`.
- [x] 7.2 In `signin.go` add `decideSignIn`, `routeToCompanySignIn`, `hydrateSignInScreen`, `hydrateNoTenant`, `hydrateChosenTenant`, `chooseTenant`, `resetTenant`, `bindTenant`, `clearTenant` and `withRecoverPrompt`. Add `pick` and `pickFor`. A pick with no tenant bound any more goes back to the page of the flow.
- [x] 7.3 Add tests in `signin_test.go` (`TestHydrateChosenTenant…`, `TestHydrateSignInScreen…`, `TestHydrateNoTenant`, `TestPick…`, `TestChooseTenant…`, `TestResetTenant`, `TestBindTenant`). Add tests in `handlers_test.go` (`TestHydrateLoginFlowUnchanged`, `TestBeforeTenantSelection…`, `TestInterceptLoginSubmission…`).

## 8. Company sign-in

- [x] 8.1 In `attempt.go` add `startAttempt` (ticket, `byo-sso` submission with `login_hint`, cookies, expired Kratos session cookie). Add `completePending` (confirmation, each ticket with its receipt, provenance, and the receipt cookies cleared with the sign-in cookie). Add `markFresh` and `isFresh`. The mark carries the time of its making.
- [x] 8.2 In `handlers.go` add `handleComplete` and `RegisterEndpoints` (`GET /api/v0/sso/complete`).
- [x] 8.3 Add tests in `attempt_test.go` (`TestStartAttempt…`, `TestCompletePending…`, `TestIsFresh`). Add tests in `handlers_test.go` (`TestHandleComplete…`, with another `return_to` and with the join).

## 9. Checks before the accept, and tenant MFA

- [x] 9.1 In `decide.go` add `decide` and its helpers (`includesCompanySignIn`, `mfaNeeded`, `mfaDone`, `provenanceFor`, `fresh`, `privileged`).
- [x] 9.2 In `accept.go` add `checkAndAccept`, the tenant resolution, the personal tenant, `JoinTenant` before `accept`, `askFor`, `askCompanySignIn`, `freshFlow`, `selectTenant` and `verifyAddress`. Add the revocation of another subject's Hydra login session. Add `HandleSessionLogin` and `BeforeAcceptLogin` in `handlers.go`.
- [x] 9.3 In `mfa.go` add `askMFA` and `hasSecondFactor`.
- [x] 9.4 Add tests in `decide_test.go` (`TestDecide`, one case for each decision, and the helpers). Add tests in `accept_test.go` (`TestCheckAndAccept…`). They cover a completed sign-in, a linked attempt, refusals, joins and no tenant. They also cover pending invitations, another subject, a fresh first factor, address verification and every failing backend. Add tests in `mfa_test.go`.

## 10. Account linking

- [x] 10.1 In `linking.go` add `hydrateAccountLinking`, `accountLinks` and `linkingPick`.
- [x] 10.2 Add tests in `linking_test.go` (`TestHydrateAccountLinking`, `…FailOnLinks`, `TestLinkingPick`, `…Refused`, `…FailOnLinks`).

## 11. Registration

- [x] 11.1 In `registration.go` add `admittingTenants`, `registrationSignIn`, `registrationChoice`, `registrationLoginFlow` and `chooseRegistrationTenant`. Add `hydrateRegistrationChoice`, which gives an error message when no tenant admits the address any more. Add `InterceptRegistrationSubmission` in `handlers.go`.
- [x] 11.2 Add tests in `registration_test.go` (`TestRegistrationSignIn…`, `TestAdmittingTenants…`, `TestHydrateRegistrationChoiceWhenNoTenantAdmits`, `TestChooseRegistrationTenant…`, `TestRegistrationFor`). Add tests in `handlers_test.go` (`TestInterceptRegistrationSubmission…`).

## 12. Connected accounts

- [x] 12.1 In `settings.go` add `withLinks`, `unlink`, `hydrateRefresh` and `refreshPick`. In `handlers.go` add `HydrateSettingsFlow` and `InterceptSettingsSubmission` (with the same-origin check).
- [x] 12.2 Add tests in `settings_test.go` (`TestUnlinkRefused`, `TestUnlinkWithoutPrivilegedSession`, `TestUnlinkWhenSettingsFlowRedirects`, `TestUnlinkFails`, `TestHydrateRefresh…`, `TestRefreshPick`, `…Refused`). Add tests in `handlers_test.go` (`TestHydrateSettingsFlow…`, `TestInterceptSettingsSubmission…`).

## 13. Consent check

- [x] 13.1 In `consent.go` add `GateConsent` and `consentRefusal`.
- [x] 13.2 Add tests in `consent_test.go` (`TestConsentRefusal`, `TestGateConsent`, `TestGateConsentFailOnRejectConsent`).
- [x] 13.3 Check groups 6 to 13 with `go test ./pkg/byosso/...`.

## 14. Wiring

- [x] 14.1 In `pkg/web/byosso.go` add the router options (`WithBYOSSOEnabled`, `WithBYOSSOClients`, `WithSSOGRPCTimeout`, `WithCookieEncryption`, `WithKratosPrivilegedSessionMaxAge`). Add `registerBYOSSO`. It returns the Kratos service, the MFA flag and the options for the `pkg/kratos` and `pkg/extra` APIs. When enabled, the MFA flag is off, and the options are `WithExtension` and `WithBackupCodesRegeneration`. It returns an error when the extension is enabled and cannot be built.
- [x] 14.2 In `pkg/web/router.go` make `NewRouter` and `registerAPIs` return an error. Give the extension options to the tenants service and API.
- [x] 14.3 In `cmd/serve.go` add the dial options of both channels when enabled. Dial and close the channel to the SSO Service. Add a `TenantSignInService` client on the channel to the Tenant Service. Pass the clients, the flag, the timeout, the cookie cipher and the privileged session age to the router.
- [x] 14.4 Add tests in `pkg/web/byosso_test.go` (`TestNewRouterWithBYOSSO`, `TestNewRouterFailOnBYOSSO`, `TestNewRouterFailOnBaseURL`, `TestRegisterBYOSSO`, `TestRegisterBYOSSODisabled`).
- [x] 14.5 Check with `go test ./pkg/web/...`.

## 15. Frontend

- [x] 15.1 Add `ui/components/RedirectingNotice.tsx` ("Redirecting to <label>…").
- [x] 15.2 In `ui/util/constants.ts` add the node groups and fields that the backend adds (`isTenantNode`, `isSsoNode`, `isTenantChoice`, `isSsoUnlinkBtn`, `getSsoLinkNodeId`, the invitation label).
- [x] 15.3 In `ui/util/redirectTo.ts` add `getRedirectLabel` and `useLabelledRedirect`. In `ui/api/kratos.ts` add `IdentifierFirstError`, which keeps the status and body of a failed identifier step. In `ui/api/tenants.ts` add `invited`.
- [x] 15.4 In `ui/util/handleFlowError.ts` add `getInPlaceErrorMessage` for `sso_unavailable`, `sso_not_applicable` and `tenant_not_a_member`. Add `isKratosError` for a Kratos error that the backend passes on as JSON.
- [x] 15.5 Change `ui/pages/login.tsx` as follows. Show in-place errors and a failed pick with the flow messages. Post the tenant picks and company sign-in picks on their own, and disable the buttons while one is in flight. Sort the tenant list last under its heading. Use labelled redirects. Show an in-place error where the auto-submit of the security key rendered nothing. Show the error page for a Kratos error that the backend passes on with its own status.
- [x] 15.6 In `ui/pages/register.tsx` add the labelled redirect to a company sign-in. A Kratos error that the backend passes on as JSON is not a flow. If the page does not know its id, it goes to the error page.
- [x] 15.7 In `ui/pages/manage_connected_accounts.tsx` add the "Company sign-ins" section and its Disconnect, which acts on a Kratos error that it knows. In `ui/pages/select_tenant.tsx` add the invitation mark.
- [x] 15.8 Check that `npx eslint` on the changed files in `ui/` reports no error.

## 16. Validation

- [x] 16.1 Make sure that `make mocks`, then `go vet ./...` and `go test ./...` pass.
- [x] 16.2 With `BYOSSO_ENABLED=false`, compare the backend with upstream `main` in a browser. It is the same at every step.
- [x] 16.3 Make sure that `openspec validate bring-your-own-sso --strict` passes.

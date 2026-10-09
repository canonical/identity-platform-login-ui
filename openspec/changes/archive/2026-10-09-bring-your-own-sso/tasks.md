## 1. Settings and dependencies

- [x] 1.1 Use the `v0/sso` package and the `v0/tenant` additions of `github.com/canonical/identity-platform-api` (`go.mod`: a `replace` directive to a local checkout until a version with them is published); `golang.org/x/oauth2` and `google.golang.org/genproto/googleapis/rpc` become direct requirements
- [x] 1.2 Add `BYOSSOEnabled`, `SSOServiceGRPCAddress`, `SSOServiceGRPCTimeout` (3s), `SSOServiceTLSEnabled`, `ServiceTokenURL`, `ServiceClientID`, `ServiceClientSecret`, `ServiceTokenScopes` and `KratosPrivilegedSessionMaxAge` (1h) to `internal/config/specs.go`
- [x] 1.3 Refuse incomplete settings in `serve` (`cmd/serve.go`, `validateTenantSettings`, tested by `TestValidateTenantSettings`): BYO-SSO without multi-tenancy; BYO-SSO without the SSO service address or the service client; BYO-SSO with OIDC-WebAuthn sequencing
- [x] 1.4 Add `TenantChoice` (`tc`) to `FlowStateCookie` in `internal/cookies/cookies.go`
- [x] 1.5 Check: `go build ./...` compiles

## 2. gRPC dial helper

- [x] 2.1 Move `NewGRPCConn` from `pkg/tenants/grpc.go` to `internal/grpc/client.go` as `NewConn(service, address, tlsEnabled, opts...)`, with the same keepalive and transport settings and extra dial options appended; delete `pkg/tenants/grpc.go`
- [x] 2.2 Test it in `internal/grpc/client_test.go` (`TestNewConn`, `TestNewConnWithTLS`, `TestNewConnFailOnDialOption`)
- [x] 2.3 Check: `go test ./internal/grpc/...`

## 3. Extension hooks in `pkg/kratos`

- [x] 3.1 Define `ExtensionInterface` (nine hooks) in `pkg/kratos/interfaces.go`
- [x] 3.2 Add `pkg/kratos/extension.go`: `Option`, `WithExtension`, `WithBackupCodesRegeneration` (the prompt to regenerate backup codes without `mfaEnabled`; tested by `TestWithBackupCodesRegeneration`), `NoOpExtension`, `errResponseWritten`, `UnsetSessionCookie`, and `WriteGetFlowError` and `WriteUpdateFlowError` (the handlers' answers to an error of Kratos, for the extension's own calls); make `NewAPI` take options and default to the `NoOpExtension`
- [x] 3.3 `handleCreateFlow` (`pkg/kratos/handlers.go`): move the verification, MFA and WebAuthn session checks into `enforceSessionChecks` unchanged; with a session and a login challenge call `HandleSessionLogin` after them when the extension handles session logins; pass a created flow with no login challenge through `HydrateLoginFlow`
- [x] 3.4 `handleCreateFlowWithSession`: call `BeforeAcceptLogin` before `AcceptLoginRequest`; return `errResponseWritten` when it answered, and have `handleCreateFlow` and `handleUpdateFlow` return on that error
- [x] 3.5 `handleGetLoginFlow`: pass the flow through `HydrateLoginFlow`
- [x] 3.6 `handleUpdateIdentifierFirstFlow`: call `BeforeTenantSelection` before the tenant selection by email
- [x] 3.7 `handleUpdateFlow`: call `InterceptLoginSubmission` after the flow is fetched and go on with the request it returns; do not ask the tenant resolver's `NeedsTenantSelection` when the extension handles session logins
- [x] 3.8 `handleUpdateRegistrationFlow`: call `InterceptRegistrationSubmission` before parsing
- [x] 3.9 Settings handlers: `InterceptSettingsSubmission` before parsing; `HydrateSettingsFlow` on the flows of `handleCreateSettingsFlow`, `handleGetSettingsFlow` and `handleUpdateSettingsFlow`
- [x] 3.10 Export `RedirectResponse` for the extension's redirects
- [x] 3.11 Tests: `pkg/kratos/extension_test.go` (`TestWithExtension`, `TestWithBackupCodesRegeneration`, `TestUnsetSessionCookie`); the no-op is what every handler test that existed runs with; each call site in `pkg/kratos/handlers_test.go` (`TestHandleGetLoginFlowWithExtension`, `…WhenExtensionAnswers`, `TestHandleCreateFlowWithSessionAndExtension`, `…RedirectToVerification`, `TestHandleCreateFlowWithoutSessionAndExtension`, `TestHandleUpdateIdentifierFirstFlowWhenExtensionAnswers`, `TestHandleUpdateFlowWhenExtensionAnswers`, `TestHandleUpdateFlowWithRequestOfExtension`, `TestHandleUpdateFlowWithSessionAndExtension`, `TestHandleUpdateRegistrationFlowWhenExtensionAnswers`, `TestHandleGetSettingsFlowWithExtension`, `TestHandleUpdateSettingsFlowWhenExtensionAnswers`)
- [x] 3.12 Check: `go test ./pkg/kratos/...` passes, the tests that existed unmodified

## 4. Consent hook in `pkg/extra`

- [x] 4.1 Define `ExtensionInterface` (`GateConsent`) in `pkg/extra/interfaces.go`; add `pkg/extra/extension.go` (`Option`, `WithExtension`, `NoOpExtension`); make `NewAPI` take options
- [x] 4.2 `handleConsent` (`pkg/extra/handlers.go`): call `GateConsent` after the consent request is read; answer its redirect, or `403` on its error, without accepting
- [x] 4.3 Tests: `pkg/extra/extension_test.go`; `TestHandleConsentWhenExtensionRejects`, `…Approves`, `TestHandleConsentFailOnGateConsent`, `TestHandleConsentInvalidPasswordAALWithExtension` in `pkg/extra/handlers_test.go`
- [x] 4.4 Check: `go test ./pkg/extra/...`

## 5. Options in `pkg/tenants`

- [x] 5.1 `pkg/tenants/service.go` and `interfaces.go`: `Tenant.Invited` (`invited`); `WithSignInTenants` takes a client of the tenant service's `TenantSignInService` (`TenantSignInServiceClientInterface`: `ListSignInTenants` only) and makes lookups by email list the sign-in tenants of the address, `invited` included, in place of `LookupTenants`
- [x] 5.2 `pkg/tenants/handlers.go`: `WithSessionLookupByEmail` and `lookupSessionTenants`, used by `GET /api/v0/tenants` and the empty-selection check
- [x] 5.3 Tests: `TestLookupTenantsByEmailWithSignInTenants`, `…Error` (`service_test.go`); `TestHandleLookupTenantsBySessionEmail`, `…WithoutEmail` (`handlers_test.go`); `TestCookieTenantResolverStoreTenantKeepsTenantChoice` (`resolver_test.go`)
- [x] 5.4 Check: `go test ./pkg/tenants/...`

## 6. The extension: clients and building blocks (`pkg/byosso`)

- [x] 6.1 `constants.go` (provider id, node groups and fields, message ids, error ids, cookie names, deadlines) and `interfaces.go` (the interfaces the package uses, mocked with `go:generate` directives in its test files; mocks are not committed)
- [x] 6.2 `grpc.go`: reconnect parameters, the retry service config, `NewServiceTokenSource` with its own HTTP timeout, bearer credentials that map token failures to `UNAVAILABLE` or `UNAUTHENTICATED`; `grpc_test.go` (`TestDialOptionsRetry`, `TestDialOptionsFailOnToken`, `TestNewServiceTokenSource`, `TestNewTokenSourceTimeout`)
- [x] 6.3 `directory.go`: `Directory` over the tenant service's `TenantSignInService` (`SignInTenants`, `SignInContext`, `JoinTenant`, `CreatePersonalTenant`), the retried methods, `toSignInContext` failing on unspecified and unknown enum values; `directory_test.go`
- [x] 6.4 `sso.go`: `SSO` over the SSO service (`Options`, `StartAttempt`, `CompleteAttempt`, `Links`, `DeleteLink`), the retried methods, error reasons turned into sentinel errors; `sso_test.go`
- [x] 6.5 `hydra.go`: `LoginRequest` (`prompt`, `max_age`, remembered subject), `Reauthenticate`, `NeedsFreshFirstFactor`, `RemembersAnother`, and `Hydra` (`LoginRequest`, `RevokeLoginSession`, `RejectConsent`); `hydra_test.go`
- [x] 6.6 `kratos.go`: `Verification.Start` and `IdentityFinder` (`IdentityID`, `IdentityExists`, `HasRecoveryCodes`); `kratos_test.go`
- [x] 6.7 `kratos_decorator.go`: `KratosServiceDecorator` removing the `byo-sso` nodes from login, registration and settings flows; `kratos_decorator_test.go`
- [x] 6.8 `cookies.go`: `CookieStore` and the four cookies, each sealed together with its name; the name of the SSO service's receipt cookie, reading it and clearing it; `cookies_test.go` (`TestSetSignIn`, `TestSetProvenance`, `TestGetFreshWithValueOfAnotherCookie`, `TestGetProvenanceFailOnDecrypt`, `TestClearReceipt`)
- [x] 6.9 `flow.go` and `helpers.go`: node builders and flow predicates; redirect and error answers, `isUnavailable`, `budget`, `requestFields` (fields matched whatever their case, as the handler's decoder matches them; a field named twice is an error), `traitsEmail` (a nested and a flat address that differ are an error), `sameOrigin`, URL builders; `flow_test.go`, `helpers_test.go`

## 7. Sign-in screen

- [x] 7.1 `handlers.go`: the `API` type and `NewAPI`; `HydrateLoginFlow`, `BeforeTenantSelection`, `InterceptLoginSubmission` and `answerPick`
- [x] 7.2 `signin.go`: `decideSignIn`, `routeToCompanySignIn`, `hydrateSignInScreen`, `hydrateNoTenant`, `hydrateChosenTenant`, `pick`/`pickFor` (a pick with no tenant bound any more goes back to the page of the flow), `chooseTenant`, `resetTenant`, `bindTenant`/`clearTenant`, `withRecoverPrompt`
- [x] 7.3 Tests: `signin_test.go` (`TestHydrateChosenTenant…`, `TestHydrateSignInScreen…`, `TestHydrateNoTenant`, `TestPick…`, `TestChooseTenant…`, `TestResetTenant`, `TestBindTenant`); `handlers_test.go` (`TestHydrateLoginFlowUnchanged`, `TestBeforeTenantSelection…`, `TestInterceptLoginSubmission…`)

## 8. Company sign-in

- [x] 8.1 `attempt.go`: `startAttempt` (ticket, `byo-sso` submission with `login_hint`, cookies, Kratos session cookie expired), `completePending` (confirmation, each ticket with its receipt; provenance; the receipt cookies cleared with the sign-in cookie), `markFresh`/`isFresh` (the mark carries the time it was made)
- [x] 8.2 `handlers.go`: `handleComplete` and `RegisterEndpoints` (`GET /api/v0/sso/complete`)
- [x] 8.3 Tests: `attempt_test.go` (`TestStartAttempt…`, `TestCompletePending…`, `TestIsFresh`); `handlers_test.go` (`TestHandleComplete…`, including another `return_to` and the join)

## 9. Checks before the accept, and tenant MFA

- [x] 9.1 `decide.go`: `decide` and its helpers (`includesCompanySignIn`, `mfaNeeded`, `mfaDone`, `provenanceFor`, `fresh`, `privileged`)
- [x] 9.2 `accept.go`: `checkAndAccept`, tenant resolution and the personal tenant, `JoinTenant` before `accept`, `askFor`, `askCompanySignIn`, `freshFlow`, `selectTenant`, `verifyAddress`, the revocation of another subject's Hydra login session; `HandleSessionLogin` and `BeforeAcceptLogin` in `handlers.go`
- [x] 9.3 `mfa.go`: `askMFA` and `hasSecondFactor`
- [x] 9.4 Tests: `decide_test.go` (`TestDecide`, one case per decision, and the helpers); `accept_test.go` (`TestCheckAndAccept…`: completing a sign-in, the attempt being linked, refusals, joining, no tenant, pending invitations, another subject, fresh first factor, address verification, every failing backend); `mfa_test.go`

## 10. Account linking

- [x] 10.1 `linking.go`: `hydrateAccountLinking`, `accountLinks`, `linkingPick`
- [x] 10.2 Tests: `linking_test.go` (`TestHydrateAccountLinking`, `…FailOnLinks`, `TestLinkingPick`, `…Refused`, `…FailOnLinks`)

## 11. Registration

- [x] 11.1 `registration.go`: `admittingTenants`, `registrationSignIn`, `registrationChoice`, `registrationLoginFlow`, `hydrateRegistrationChoice` (an error message when no tenant admits the address any more), `chooseRegistrationTenant`; `InterceptRegistrationSubmission` in `handlers.go`
- [x] 11.2 Tests: `registration_test.go` (`TestRegistrationSignIn…`, `TestAdmittingTenants…`, `TestHydrateRegistrationChoiceWhenNoTenantAdmits`, `TestChooseRegistrationTenant…`, `TestRegistrationFor`); `handlers_test.go` (`TestInterceptRegistrationSubmission…`)

## 12. Connected accounts

- [x] 12.1 `settings.go`: `withLinks`, `unlink`, `hydrateRefresh`, `refreshPick`; `HydrateSettingsFlow` and `InterceptSettingsSubmission` (same-origin check) in `handlers.go`
- [x] 12.2 Tests: `settings_test.go` (`TestUnlinkRefused`, `TestUnlinkWithoutPrivilegedSession`, `TestUnlinkWhenSettingsFlowRedirects`, `TestUnlinkFails`, `TestHydrateRefresh…`, `TestRefreshPick`, `…Refused`); `handlers_test.go` (`TestHydrateSettingsFlow…`, `TestInterceptSettingsSubmission…`)

## 13. Consent check

- [x] 13.1 `consent.go`: `GateConsent` and `consentRefusal`
- [x] 13.2 Tests: `consent_test.go` (`TestConsentRefusal`, `TestGateConsent`, `TestGateConsentFailOnRejectConsent`)
- [x] 13.3 Check for groups 6 to 13: `go test ./pkg/byosso/...`

## 14. Wiring

- [x] 14.1 `pkg/web/byosso.go`: the router options (`WithBYOSSOEnabled`, `WithBYOSSOClients`, `WithSSOGRPCTimeout`, `WithCookieEncryption`, `WithKratosPrivilegedSessionMaxAge`) and `registerBYOSSO`, which returns the Kratos service, MFA flag and options the `pkg/kratos` and `pkg/extra` APIs are built with (enabled: the MFA flag off, `WithExtension` and `WithBackupCodesRegeneration`), or an error when the extension is enabled and cannot be built
- [x] 14.2 `pkg/web/router.go`: `NewRouter` and `registerAPIs` return an error; the tenants service and API get the extension's options
- [x] 14.3 `cmd/serve.go`: the service token source and the dial options of both channels when enabled; dial and close the SSO service channel; a `TenantSignInService` client on the channel to the tenant service; pass the clients, the flag, timeout, cookie cipher and privileged session age to the router
- [x] 14.4 Tests: `pkg/web/byosso_test.go` (`TestNewRouterWithBYOSSO`, `TestNewRouterFailOnBYOSSO`, `TestNewRouterFailOnBaseURL`, `TestRegisterBYOSSO`, `TestRegisterBYOSSODisabled`)
- [x] 14.5 Check: `go test ./pkg/web/...`

## 15. Frontend

- [x] 15.1 `ui/components/RedirectingNotice.tsx` ("Redirecting to <label>…")
- [x] 15.2 `ui/util/constants.ts`: the node groups and fields the backend adds (`isTenantNode`, `isSsoNode`, `isTenantChoice`, `isSsoUnlinkBtn`, `getSsoLinkNodeId`, the invitation label)
- [x] 15.3 `ui/util/redirectTo.ts`: `getRedirectLabel` and `useLabelledRedirect`; `ui/api/kratos.ts`: `IdentifierFirstError` keeping the status and body of a failed identifier step; `ui/api/tenants.ts`: `invited`
- [x] 15.4 `ui/util/handleFlowError.ts`: `getInPlaceErrorMessage` for `sso_unavailable`, `sso_not_applicable` and `tenant_not_a_member`; `isKratosError` for an error of Kratos passed on as JSON
- [x] 15.5 `ui/pages/login.tsx`: in-place errors and a failed pick, shown with the flow messages; tenant and company sign-in picks posted on their own, with the buttons disabled while one is in flight; the tenant list sorted last under its heading; labelled redirects; an in-place error shown where the security-key auto-submit rendered nothing; the error page for an error of Kratos passed on with its own status
- [x] 15.6 `ui/pages/register.tsx`: labelled redirect to a company sign-in; an error of Kratos passed on as JSON is not taken for a flow, and goes to the error page when its id is not known
- [x] 15.7 `ui/pages/manage_connected_accounts.tsx`: the "Company sign-ins" section and its Disconnect, which acts on an error of Kratos it knows; `ui/pages/select_tenant.tsx`: the invitation mark
- [x] 15.8 Check: `npx eslint` on the changed files in `ui/` reports no error

## 16. Validation

- [x] 16.1 `make mocks`, then `go vet ./...` and `go test ./...` pass
- [x] 16.2 With `BYOSSO_ENABLED=false`, the backend compared with upstream `main` in a browser: the same at every step
- [x] 16.3 `openspec validate bring-your-own-sso --strict` passes

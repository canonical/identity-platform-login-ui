## 1. Shift-Left Go Vulnerability Scanning

- [x] 1.1-govulncheck-target Add `govulncheck` target to `Makefile` touching [Makefile](Makefile) (installs and runs `govulncheck ./...` directly)
- [x] 1.2-unittest-step Add vulnerability check step to [.github/workflows/unittest.yaml](.github/workflows/unittest.yaml) invoking `make govulncheck`
- [x] 1.3-ci-sequence Sequence pull request validation in [.github/workflows/ci.yaml](.github/workflows/ci.yaml) so that `build` declares `needs: [ unit-test ]` to fail fast before rock construction

## 2. Release Quarantine Configuration

- [x] 2.1-prerelease-config Update [release-please-config.json](release-please-config.json) to set `"prerelease": true` for the root package

## 3. Candidate Staging, Gated Scanning & Automated Promotion

- [x] 3.1-candidate-tags Update publish step in [.github/workflows/ci.yaml](.github/workflows/ci.yaml) to publish candidate image tags (e.g. `${{ github.ref_name }}-candidate`) instead of `stable` on release tags
- [x] 3.2-scan-gate Update scan step in [.github/workflows/ci.yaml](.github/workflows/ci.yaml) to target the candidate image and fail the job on detected HIGH or CRITICAL CVEs
- [x] 3.3-promote-job Add promotion job to [.github/workflows/ci.yaml](.github/workflows/ci.yaml) that runs on scan success, promotes candidate tags to `${{ github.ref_name }}` and `stable` via Skopeo, and executes `gh release edit ${{ github.ref_name }} --prerelease=false --latest`

## 4. Scheduled Continuous Monitoring

- [x] 4.1-cves-monitoring Add scheduled workflow or job in [.github/workflows/cves.yaml](.github/workflows/cves.yaml) to run periodic Trivy scans against `ghcr.io/canonical/identity-platform-login-ui:stable`

## 5. Upstream Alignment (`canonical/identity-team`)

- [x] 5.1-upstream-alignment Document and prepare patch/issue for `canonical/identity-team` reusable workflows (`_rock-gh-publish.yaml` candidate mode, `_rock-scan.yaml` fail-on-CVE support, and `_rock-promote.yaml`)

## 6. Verification Suite

- [x] 6.1-verify-vulncheck Run `make govulncheck` locally and ensure it exits cleanly
- [x] 6.2-verify-test-unit Run `make test-unit` locally and ensure it exits with zero diagnostic failures
- [x] 6.3-verify-workflow-syntax Verify GitHub Actions workflow syntax for [.github/workflows/ci.yaml](.github/workflows/ci.yaml) and [.github/workflows/unittest.yaml](.github/workflows/unittest.yaml)

## 7. Documentation & Rollout

- [x] 7.1-document-release-flow Document the pre-release, scan, and promotion release flow in repository docs or developer guide

## Implementation Notes

- Shift-left vulnerability check target `make govulncheck` added directly to `Makefile` and wired into `unittest.yaml` before unit test execution.
- PR validation in `ci.yaml` sequenced so that `build` rock creation declares `needs: [ unit-test ]`.
- Release quarantine configured in `release-please-config.json` via `"prerelease": true`.
- Candidate image publishing (`<tag>-candidate`), Trivy security gating (`exit-code: 1`), and automated Skopeo promotion + GitHub release graduation implemented in `ci.yaml`.
- Scheduled weekly scan of `stable` added to `cves.yaml`.
- Upstream reusable workflow specification documented in `docs/upstream-identity-team-workflows.md` for `canonical/identity-team`.

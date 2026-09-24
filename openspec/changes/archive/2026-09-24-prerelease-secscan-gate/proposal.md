## Why

Currently, release tags trigger immediate publication of container images to the `stable` registry tag before vulnerability scanning completes. If a security vulnerability is detected in the rock image, the artifact has already been made publicly available as stable. Furthermore, pull request workflows run full container builds before testing for vulnerable Go dependencies, leading to slow CI feedback loops and unnecessary resource consumption.

## What Changes

- **Release Quarantine & Gated Promotion**: Releases cut by release automation are initially marked as pre-releases. Publication to the public `stable` container tag and graduation of the GitHub Release to stable are strictly gated behind passing vulnerability scans.
- **Shift-Left Vulnerability Checking**: Introduce Go dependency and reachable symbol vulnerability scanning (`govulncheck`) into the test lifecycle.
- **Fail-Fast PR Pipeline**: Sequence pull request validation so heavy container builds execute only after unit tests and vulnerability checks pass.
- **Continuous Security Monitoring**: Establish scheduled vulnerability scans against the active stable release artifact to catch newly published CVEs.

## Capabilities

### New Capabilities
- `artifact-security-gate`: Defines requirements for vulnerability gating, pre-release quarantine, automated promotion to stable, and continuous artifact scanning.

### Modified Capabilities
None.

## Non-goals

- Modifying the underlying login UI or authentication runtime logic.
- Replacing Trivy or govulncheck with proprietary scanning engines.
- Automating upstream dependency version bumps (managed separately via dependabot/manual updates).

## Success Metrics

- 100% of container images tagged as `stable` have zero un-triaged high/critical CVEs at publication time.
- 0 runner minutes spent building container images on PRs that contain known vulnerable Go dependencies.
- Detection of newly disclosed CVEs in the stable artifact within 24 hours via scheduled scans.

## Impact

- **Affected components**: [Makefile](Makefile), [.github/workflows/unittest.yaml](.github/workflows/unittest.yaml), [.github/workflows/ci.yaml](.github/workflows/ci.yaml), [release-please-config.json](release-please-config.json).
- **External dependencies**: Upstream reusable workflows in `canonical/identity-team` (`_rock-gh-publish.yaml`, `_rock-scan.yaml`).

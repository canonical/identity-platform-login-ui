# artifact-security-gate Specification

## Purpose
TBD - created by archiving change prerelease-secscan-gate. Update Purpose after archive.
## Requirements
### Requirement: Pull request vulnerability pre-check
The test verification workflow SHALL execute static vulnerability analysis on Go dependencies prior to initiating container image builds. If any high or critical vulnerability with a reachable execution path is detected, the workflow SHALL terminate with an error and skip container image construction.

#### Scenario: Clean dependencies in pull request
- **WHEN** a pull request is submitted with no known vulnerable Go dependencies
- **THEN** vulnerability checks exit with code 0
- **THEN** the workflow proceeds to build the container rock artifact

#### Scenario: Vulnerability detected in pull request
- **WHEN** a pull request introduces a dependency containing a known reachable vulnerability
- **THEN** vulnerability checks exit with a non-zero exit code
- **THEN** the container rock build job is skipped and does not consume runner time

### Requirement: Release quarantine and pre-release tagging
When a release PR is merged, the release automation SHALL create the GitHub Release marked as a pre-release (`prerelease: true`) and SHALL NOT mark it as latest or stable.

#### Scenario: Release cut marks pre-release
- **WHEN** an automated release PR is merged into main
- **THEN** release automation cuts a release with `prerelease: true`
- **THEN** the release is not marked as the latest stable release in GitHub

### Requirement: Candidate artifact scanning and promotion gate
The release pipeline SHALL publish container artifacts under temporary candidate tags and perform security scanning against the candidate image before publishing to `stable`. Promotion to `stable` and graduation of the GitHub Release to a non-prerelease state SHALL occur ONLY when the security scan completes with zero blocking vulnerabilities.

#### Scenario: Clean candidate promoted to stable
- **WHEN** the candidate container image scan completes with zero blocking vulnerabilities
- **THEN** the image is copied and tagged as `stable` and `<version>` in the container registry
- **THEN** the GitHub Release is updated to `prerelease: false` and marked as latest

#### Scenario: Vulnerable candidate blocked from promotion
- **WHEN** the candidate container image scan detects an un-triaged blocking vulnerability
- **THEN** the security scan job fails with a non-zero exit code
- **THEN** the image is NOT tagged as `stable` in the container registry
- **THEN** the GitHub Release remains quarantined as a pre-release

### Requirement: Continuous security scanning of published artifacts
The CI system SHALL execute scheduled vulnerability scans against the active `stable` container image and the default branch to detect newly disclosed CVEs.

#### Scenario: Scheduled scan on stable image
- **WHEN** the scheduled security workflow triggers
- **THEN** the scanner evaluates `ghcr.io/canonical/identity-platform-login-ui:stable` for newly disclosed vulnerabilities
- **THEN** any findings are published to the repository security advisory system


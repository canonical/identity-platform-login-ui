---
name: cve-resolver
description: Resolve CVEs in rocks by updating dependencies in rockcraft.yaml, verifying with govulncheck and rockcraft, and creating specific conventional commits.
---

# CVE Resolver Skill

This skill guides the process of addressing Common Vulnerabilities and Exposures (CVEs) in rockcraft projects.

## Workflow

### 1. Identify CVEs
CVEs are typically reported via GitHub issues (e.g., https://github.com/canonical/openfga-rock/issues/59). Each issue will list the affected packages and the corresponding CVE IDs.

### 2. Update Dependencies
Address the CVEs by modifying `rockcraft.yaml`. There are two main approaches:
- **Update Source Version**: If a part's `source-tag` or `source-branch` can be moved to a patched version, update it there.
- **Go Mod Replace**: For Go-based parts, use `go mod edit -replace` within the `override-build` section to force a specific patched version of a sub-dependency.

Example `override-build` snippet:
```yaml
    override-build: |
      # Addressing CVE
      go mod edit -replace go.opentelemetry.io/otel/sdk=go.opentelemetry.io/otel/sdk@v1.43.0
      go mod edit -replace google.golang.org/grpc=google.golang.org/grpc@v1.79.3
      go mod tidy
      # ... rest of the build commands
```

### 3. Verification
Run the following checks to ensure the quality and effectiveness of the changes:

- **Vulnerability Check**: If the part is Go-based, run `govulncheck ./...` within the part's context (if possible) or simulate the build to see if vulnerabilities persist.
- **Clean Build**: Run `rockcraft clean` to ensure a fresh environment.
- **Pack Rock**: Run `rockcraft pack` to verify that the rock still builds correctly with the new dependency versions.

### 4. Commit Changes
Create a conventional commit following this exact format:

**Title**: `fix: address CVEs`

**Extended Message**: A bulleted list of each CVE ID and the package being patched, including a brief description if applicable. **The message must also include a reference to the related GitHub issue using the hash format (e.g., `#123`).**

Example:
```text
fix: address CVEs

- CVE-2024-45334: google.golang.org/grpc patched to v1.79.3
- CVE-2024-45337: go.opentelemetry.io/otel/sdk patched to v1.43.0

Fixes #59
```

## Tips
- Always check the latest stable versions of the packages mentioned in the CVE report.
- Ensure `go mod tidy` is called after `go mod edit -replace`.
- If a `git patch` is preferred over `go mod edit`, apply it in the `override-build` section.
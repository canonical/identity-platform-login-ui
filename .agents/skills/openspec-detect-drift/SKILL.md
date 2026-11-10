---
name: openspec-detect-drift
description: >
  Detect drift between code changes and OpenSpec specifications. Use when
  verifying whether recent commits, pull requests, or working tree diffs
  diverge from active OpenSpec specifications in openspec/specs/, introduce
  undocumented behaviors, violate existing requirements, or leave specs
  outdated.
license: AGPL-3.0-only
metadata:
  author: Canonical
  version: "1.0.0"
  summary: >
    Detects drift between differential code changes and active OpenSpec
    specifications.
  tags:
    - openspec
    - drift
    - specification
    - verification
---

# Detect OpenSpec Drift

Audit differential code changes against the active specification baseline in
`openspec/specs/` to identify unmanaged drift, contradictions, undocumented
behaviors, or outdated scenarios.

## Application and Scope

This skill is invoked during code reviews, pre-commit checks, or CI workflows
to verify that code changes remain aligned with the project's OpenSpec
specifications located in `openspec/specs/`.

## Workflow

Follow these steps in sequence:

1. **Extract Differential Code Diff**
   - Run `git diff origin/main...HEAD` (or against a specified base ref).
   - If in an uncommitted working tree or detached state, run `git diff HEAD`.
   - Exclude vendored directories, third-party libraries, virtual environments,
     and auto-generated lock/cache files.

2. **Discover Active Specifications**
   - Locate all specification files matching `openspec/specs/**/spec.md`.
   - Extract the capability name, `Purpose`, `### Requirement:` sections,
     and `#### Scenario:` blocks for each capability.

3. **Perform Semantic Drift Evaluation**
   - Compare changed files, symbols, endpoints, and behaviors against the
     specification requirements:
     - **Contradictions (CRITICAL)**: Code changes violate an explicit `SHALL`
       requirement or scenario expectation in an existing spec.
     - **Undocumented Behaviors (WARNING)**: Code introduces new public APIs,
       configuration options, routes, or behaviors not defined in any spec.
     - **Outdated Specifications (WARNING)**: Code removes, renames, or alters
       behavior that is still described as active in `openspec/specs/`.
     - **Conforming Refactoring (OK)**: Code changes are internal details,
       performance improvements, or bugfixes conforming to existing specs.

4. **Generate Drift Scorecard**
   - Produce a structured markdown report table summarizing the alignment
     status of each touched capability or area.
   - For every finding, provide exact `file:line` references and the affected
     requirement titles.

5. **Generate Remediation Delta Spec**
   - For any detected drift, draft a copy-pasteable OpenSpec delta spec snippet
     using standard `## ADDED Requirements`, `## MODIFIED Requirements`, or
     `## REMOVED Requirements` sections.

## Gotchas

- **CRITICAL**: The generated report MUST always start with the exact anchor tag
  `<!-- OPENSPEC_DRIFT_REPORT -->` on the very first line before any markdown
  headings.
- Ignore changes in test files, documentation, and dependencies when checking
  for contract changes, unless tests reveal a specification divergence.
- When evaluating deletions or renames, confirm whether the old functionality
  is deprecated or completely removed, and flag outdated spec scenarios.
- Do not assume missing specs are intentional; always report undocumented
  public interface additions as drift requiring a spec update.

## Output Format Template

Use the following format for the evaluation report:

````markdown
<!-- OPENSPEC_DRIFT_REPORT -->
## 🔍 OpenSpec Drift Report

### Summary Scorecard

| Capability / Area | Status | Drift Type | Summary |
|---|---|---|---|
| `<capability>` | `🚨 Drift Detected` / `✅ Aligned` | `Contradiction` / `Missing Spec` / `Outdated Spec` / `None` | Brief summary |

### Detailed Findings

#### [CRITICAL/WARNING/OK] Finding Title

- **Spec / Requirement**: `<capability>` -> `### Requirement: ...`
- **Changed Code**: `<file>:<line>`
- **Description**: Detailed description of what drifted and why it matters.

### Proposed OpenSpec Delta Spec / Remediation

```markdown
# Delta Spec for <capability>

## MODIFIED Requirements

### Requirement: Requirement title
#### Scenario: Scenario title
- **WHEN** condition occurs
- **THEN** expected outcome
```
````

## Definition of Done

The drift evaluation is complete when:

1. The report begins with the exact anchor tag `<!-- OPENSPEC_DRIFT_REPORT -->`
   on line 1.
2. All modified source files have been cross-checked against `openspec/specs/`.
3. Every identified drift item includes a classification, `file:line`
   reference, and explanation.
4. A copy-pasteable OpenSpec remediation delta snippet is provided for any
   unaligned changes.

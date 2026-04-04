# Implementation Plan: Agentic Release Process

**Branch**: `002-agentic-release` | **Date**: 2026-04-04 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-agentic-release/spec.md`

## Summary

Automate the nfs-provisioner-operator release pipeline by creating a `make bump-version` script, enhancing the existing `/operator-release` skill with dry-run support and human approval gates, and orchestrating the full flow: verify → bump → build → push → commit → community-operators PR. Modeled after kserve's release process v3 with adaptations for OLM operator workflows.

## Technical Context

**Language/Version**: Go 1.24 (operator), Bash (release scripts), YAML (GH Actions workflows)  
**Primary Dependencies**: make, podman/docker, opm, gh CLI, kustomize  
**Storage**: N/A (file-based version references only)  
**Testing**: Bash script tests (manual verification), existing CI pipeline (unit/integration/e2e)  
**Target Platform**: Linux (release manager workstation), GitHub Actions (CI)  
**Project Type**: Single project — release tooling added to existing operator repo  
**Performance Goals**: bump-version completes in <10 seconds, full release in <15 minutes  
**Constraints**: Must work with existing Makefile structure, existing Claude Code skills, existing GH Actions  
**Scale/Scope**: Single operator, single channel (alpha), 2 community-operators repos

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Code Quality & Go Standards | PASS | Release scripts are Bash, not Go. Go standards apply to operator code which is unchanged. |
| II. Testing Standards & Ginkgo | PASS | No test code changes. Existing tests are prerequisites for release. |
| III. User Experience Consistency | PASS | Release tooling improves operator experience for release managers. |
| IV. Performance & Resource Efficiency | PASS | No runtime operator changes. Script performance target: <10s for bump. |
| Security & RBAC | PASS | Image signing addressed in existing CI. No new RBAC changes. |
| Development Workflow | PASS | Enhances existing workflow with automation. Signed commits preserved (FR-008 approval gates). |

**Gate Result**: PASS — No violations. Proceed to Phase 0.

## Project Structure

### Documentation (this feature)

```text
specs/002-agentic-release/
├── plan.md              # This file
├── research.md          # Phase 0: research findings
├── data-model.md        # Phase 1: release entity model
├── quickstart.md        # Phase 1: how to use the release process
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
hack/
└── release/
    ├── bump-version.sh          # Version bump script (FR-001, FR-002, FR-003)
    └── validate-release.sh      # Pre-release validation (FR-009)

Makefile                          # bump-version target added

.claude/skills/
├── operator-release.md           # Enhanced with dry-run, approval gates, orchestration
└── operator-verify.md            # Existing (used as prerequisite)
```

**Structure Decision**: Release scripts go in `hack/release/` following kserve convention. The existing skill files are enhanced in place. No new source directories needed — this is tooling, not application code.

## Complexity Tracking

> No constitution violations. Table intentionally empty.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| (none)    |            |                                     |

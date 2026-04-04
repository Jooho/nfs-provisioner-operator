# Tasks: Agentic Release Process

**Input**: Design documents from `/specs/002-agentic-release/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: Not explicitly requested. Test tasks omitted. Validation is done via manual dry-run and existing CI.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: Create release tooling directory structure

- [x] T001 Create release scripts directory at hack/release/
- [x] T002 [P] Read and document current version references in env.sh, config/manifests/bases/nfs-provisioner-operator.clusterserviceversion.yaml, catalog/nfs-provisioner-operator/channel.yaml

**Checkpoint**: Directory structure ready, current state documented

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Utility functions and validation logic shared by all user stories

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T003 Create version validation functions (semver regex, version ordering via sort -V, PRIOR matches current env.sh) in hack/release/bump-version.sh
- [x] T004 [P] Create prerequisite checker script (registry login, gh auth, opm/kustomize availability, fork repo paths) in hack/release/validate-release.sh
- [x] T005 Add bump-version and validate-release targets to Makefile

**Checkpoint**: Foundation ready — `make validate-release` runs and reports prerequisites status

---

## Phase 3: User Story 1 — One-Command Version Bump (Priority: P1) MVP

**Goal**: `make bump-version NEW_VERSION=0.0.10 PRIOR_VERSION=0.0.9` updates all version references in one command

**Independent Test**: Run bump-version with a test version, verify all files are updated via `git diff`, then revert with `git checkout -- .`

### Implementation for User Story 1

- [x] T006 [US1] Implement pre-build version bump logic in hack/release/bump-version.sh: update VERSION in env.sh
- [x] T007 [US1] Implement replaces field update in config/manifests/bases/nfs-provisioner-operator.clusterserviceversion.yaml via sed in hack/release/bump-version.sh
- [x] T008 [US1] Add `make bundle` execution and CRD sync (cp config/crd/bases/*.yaml bundle/manifests/) to hack/release/bump-version.sh
- [x] T009 [US1] Add diff display at end of bump-version.sh (show all changed files for review)
- [x] T010 [US1] Handle macOS vs Linux sed -i compatibility in hack/release/bump-version.sh
- [x] T011 [US1] Add error handling: invalid version format, same version, version not greater, PRIOR mismatch with current env.sh

**Checkpoint**: `make bump-version NEW_VERSION=0.0.10 PRIOR_VERSION=0.0.9` updates env.sh, CSV replaces, regenerates bundle, syncs CRD, shows diff. Revertable via git.

---

## Phase 4: User Story 2 — Dry-Run Release Preview (Priority: P2)

**Goal**: `--dry-run` flag shows all planned actions without executing anything

**Independent Test**: Run `/operator-release 0.0.10 --dry-run`, verify no files changed (git status clean), verify summary output lists all planned actions

### Implementation for User Story 2

- [x] T012 [US2] Add --dry-run flag parsing to hack/release/bump-version.sh (show what would change without modifying files)
- [x] T013 [US2] Create dry-run summary output format in hack/release/bump-version.sh: list files to modify, images to build, PRs to create
- [x] T014 [US2] Update .claude/skills/operator-release.md Phase 1 to call validate-release.sh and support --dry-run flow
- [x] T015 [US2] Update .claude/skills/operator-release.md to add dry-run path through all phases (show plan, skip execution)

**Checkpoint**: `/operator-release 0.0.10 --dry-run` shows complete release plan with zero side effects

---

## Phase 5: User Story 3 — Automated Image Build and Push (Priority: P3)

**Goal**: Image build/push integrated into release flow with post-build digest updates

**Independent Test**: Run image build phase with a test version, verify 3 images appear in registry, verify digest is captured

### Implementation for User Story 3

- [x] T016 [US3] Add image build commands (operator, bundle, catalog) to .claude/skills/operator-release.md Phase 3
- [x] T017 [US3] Add post-push digest capture logic: get digest via `podman inspect` or `skopeo inspect`, update env.sh NFS_OPERATOR_PINNED_DIGESTS
- [x] T018 [US3] Add post-push digest update for config/manifests/bases/nfs-provisioner-operator.clusterserviceversion.yaml containerImage annotation
- [x] T019 [US3] Add post-push digest update for config/manager/kustomization.yaml
- [x] T020 [US3] Add re-run `make bundle` after digest updates to propagate to bundle CSV
- [x] T021 [US3] Add FBC generation: `opm render` new version yaml, update catalog/nfs-provisioner-operator/channel.yaml, run `opm validate catalog/`

**Checkpoint**: After image push, all digest references updated, FBC catalog valid, `opm validate` passes

---

## Phase 6: User Story 4 — Community Operators PR Submission (Priority: P4)

**Goal**: Automated PR creation to both community-operators repos with correct content

**Independent Test**: Run PR submission phase against fork repos, verify directory structure and PR content via `gh pr view`

### Implementation for User Story 4

- [x] T022 [US4] Update .claude/skills/operator-release.md Phase 6a: automate k8s-operatorhub/community-operators PR (branch, copy bundle, commit, push, gh pr create)
- [x] T023 [US4] Update .claude/skills/operator-release.md Phase 6b: automate community-operators-prod PR with FBC (bundle copy + catalog-templates update + catalog render across OCP versions + ci.yaml update)
- [x] T024 [US4] Add fork repo freshness check (git fetch upstream, warn if behind) to validate-release.sh
- [x] T025 [US4] Add error handling for existing branches (from previous failed attempts) in operator-release.md Phase 6

**Checkpoint**: Two PRs created with correct structure. `gh pr view` shows expected content for both repos.

---

## Phase 7: User Story 5 — End-to-End Orchestrated Release (Priority: P5)

**Goal**: Single `/operator-release 0.0.10` command runs full pipeline with 3 approval gates

**Independent Test**: Run full release end-to-end with a new version, verify all artifacts created and all gates respected

### Implementation for User Story 5

- [x] T026 [US5] Restructure .claude/skills/operator-release.md to follow 7-phase pipeline: prerequisites → bump → G1 → build → G2 → push/digest/FBC → commit → G3 → PRs → report
- [x] T027 [US5] Add G1 (Version Review) gate: show diff, ask for approval, revert on reject
- [x] T028 [US5] Add G2 (Push Approval) gate: show image list, ask for approval, skip push on reject
- [x] T029 [US5] Add G3 (PR Approval) gate: show PR plan, ask for approval, skip PRs on reject
- [x] T030 [US5] Add abort handling: clean up partial state at any gate rejection (revert uncommitted changes, delete local branches)
- [x] T031 [US5] Add summary report generation (Phase 7): list all images with URLs, PR links, completed actions
- [x] T032 [US5] Integrate /operator-verify as prerequisite call at start of pipeline (run before bump-version)

**Checkpoint**: Full release from `/operator-release 0.0.10` — verify → bump → build → push → commit → PR with 3 human gates

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, cleanup, and validation

- [x] T033 [P] Update specs/002-agentic-release/quickstart.md with final usage examples reflecting implemented flow
- [ ] T034 [P] Add release process documentation to docs/release.md
- [x] T035 Run full dry-run validation: `/operator-release 0.0.10 --dry-run` end-to-end
- [ ] T036 Run full release validation with a real version bump (actual release test)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Phase 2 — MVP, must complete first
- **US2 (Phase 4)**: Depends on US1 (extends bump-version with dry-run)
- **US3 (Phase 5)**: Depends on US1 (needs bump-version to set version before build)
- **US4 (Phase 6)**: Depends on US3 (needs pushed images for PR content)
- **US5 (Phase 7)**: Depends on US1-US4 (orchestrates all phases)
- **Polish (Phase 8)**: Depends on US5

### User Story Dependencies

- **US1 (P1)**: Independent after Foundational — MVP
- **US2 (P2)**: Extends US1 (adds dry-run to existing script)
- **US3 (P3)**: Extends US1 (adds build/push after bump)
- **US4 (P4)**: Requires US3 (needs images pushed to create PRs)
- **US5 (P5)**: Requires US1-US4 (orchestrates everything)

### Within Each User Story

- Foundational tasks before story-specific tasks
- Script implementation before skill updates
- Core logic before error handling
- Story complete before moving to next priority

### Parallel Opportunities

- T001 and T002 can run in parallel (Phase 1)
- T003 and T004 can run in parallel (Phase 2)
- T033 and T034 can run in parallel (Phase 8)
- Within US3: T017, T018, T019 can run in parallel (different files)

---

## Parallel Example: User Story 3

```bash
# After T017 captures digest, these can run in parallel:
Task: "T018 [US3] Update CSV containerImage digest"
Task: "T019 [US3] Update kustomization.yaml digest"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1 (bump-version)
4. **STOP and VALIDATE**: Run `make bump-version NEW_VERSION=test PRIOR_VERSION=0.0.9`, verify diff, revert
5. This alone eliminates the most error-prone manual step

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. US1 (bump-version) → MVP! Most manual work eliminated
3. US2 (dry-run) → Safety net for reviewing changes
4. US3 (image build/push) → Full build automation
5. US4 (community PRs) → Full PR automation
6. US5 (orchestration) → Full agentic release
7. Each story adds value without breaking previous stories

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story
- This is release tooling (Bash scripts + Claude skill updates), not Go application code
- Existing CI (unit/integration/e2e) is a prerequisite, not modified by this feature
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently

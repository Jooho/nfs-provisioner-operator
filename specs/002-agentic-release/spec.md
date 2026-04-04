# Feature Specification: Agentic Release Process

**Feature Branch**: `002-agentic-release`  
**Created**: 2026-04-04  
**Status**: Draft  
**Input**: User description: "Agentic release process for nfs-provisioner-operator, modeled after kserve release process v3. This repo serves as a testbed before applying to kserve."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - One-Command Version Bump (Priority: P1)

Release manager wants to update all version references across the project by running a single command, instead of manually editing multiple files (env, CSV, bundle metadata).

**Why this priority**: Version bumping is the foundation of every release. Without this, every subsequent step is manual and error-prone. This is the most frequent pain point in the current process.

**Independent Test**: Can be fully tested by running the bump-version command with a test version and verifying all files are correctly updated. Delivers immediate value by eliminating manual edits.

**Acceptance Scenarios**:

1. **Given** current version is 0.0.9, **When** operator runs `make bump-version NEW_VERSION=0.0.10`, **Then** the `env` file, CSV, and bundle metadata all reflect 0.0.10 with correct `replaces` pointing to 0.0.9
2. **Given** version bump has been run, **When** operator inspects the changed files, **Then** only version-related fields are modified; no unintended changes exist
3. **Given** an invalid version format (e.g., "abc"), **When** operator runs bump-version, **Then** the command fails with a clear error message and makes no changes

---

### User Story 2 - Dry-Run Release Preview (Priority: P2)

Release manager wants to preview everything that a release would do — version changes, images to build, PRs to create — without actually executing anything, so they can review before committing.

**Why this priority**: Dry-run builds trust in the automation. It is the human approval gate that makes fully agentic execution safe. Without it, operators won't trust the process enough to let an agent run it.

**Independent Test**: Can be fully tested by running the release process with a dry-run flag and verifying that no files are modified, no images are pushed, and no PRs are created, while a complete summary of intended actions is displayed.

**Acceptance Scenarios**:

1. **Given** operator invokes `/operator-release 0.0.10 --dry-run`, **When** the process completes, **Then** a summary report shows all planned actions (version changes, image builds, PR targets) without executing any of them
2. **Given** dry-run shows an issue (e.g., missing prerequisite), **When** operator reviews the output, **Then** the specific blocker is clearly identified with a remediation suggestion
3. **Given** dry-run succeeds, **When** operator then runs without `--dry-run`, **Then** the actual execution matches exactly what was previewed

---

### User Story 3 - Automated Image Build and Push (Priority: P3)

Release manager wants operator, bundle, and FBC catalog images to be built and pushed to the registry as part of the release flow, without running individual podman commands manually.

**Why this priority**: Image build/push is the most time-consuming manual step. Automating it saves significant effort per release and eliminates human errors in image tagging.

**Independent Test**: Can be fully tested by triggering the image build phase with a test version and verifying all three images (operator, bundle, catalog) appear in the target registry with correct tags.

**Acceptance Scenarios**:

1. **Given** version has been bumped to 0.0.10, **When** the image build phase executes, **Then** three images are built and pushed: `operator:0.0.10`, `operator-bundle:0.0.10`, `operator-catalog:0.0.10`
2. **Given** the container registry is unreachable, **When** push fails, **Then** the process stops with a clear error and does not proceed to subsequent phases
3. **Given** images already exist for the target version, **When** build phase runs, **Then** the operator is warned about overwriting existing images and asked to confirm

---

### User Story 4 - Community Operators PR Submission (Priority: P4)

Release manager wants PRs to both community-operators repos (k8s-operatorhub and community-operators-prod) to be created automatically with correct bundle content and FBC catalog data.

**Why this priority**: This is the most complex and error-prone manual step. The FBC format for community-operators-prod requires precise directory structure and catalog rendering. Automating this reduces the risk of PR rejection.

**Independent Test**: Can be fully tested by running the PR submission phase against forked repos and verifying the correct directory structure, file content, and PR metadata.

**Acceptance Scenarios**:

1. **Given** images have been pushed for version 0.0.10, **When** PR submission phase runs, **Then** two PRs are created: one to k8s-operatorhub/community-operators and one to redhat-openshift-ecosystem/community-operators-prod
2. **Given** the community-operators-prod repo requires FBC format, **When** the PR is created, **Then** it includes correct FBC catalog data rendered across all supported OCP versions
3. **Given** a PR submission fails (e.g., branch already exists), **When** the error occurs, **Then** the process reports the failure clearly and does not affect the other PR

---

### User Story 5 - End-to-End Orchestrated Release (Priority: P5)

Release manager wants to invoke a single command that orchestrates the entire release pipeline — verify, bump, build, push, commit, PR — with human approval gates at critical decision points.

**Why this priority**: This is the ultimate goal — the fully agentic release. However, it depends on all previous stories being solid. The value is in the orchestration and the trust model (agent proposes, human approves).

**Independent Test**: Can be fully tested by running the full release flow end-to-end with a new version and verifying that all artifacts are created, all gates are respected, and all PRs are submitted.

**Acceptance Scenarios**:

1. **Given** operator invokes `/operator-release 0.0.10`, **When** the process runs, **Then** it executes: verify → bump-version → build/push images → commit → create community-operators PRs, pausing for human approval before irreversible actions (push, PR creation)
2. **Given** verification fails (e.g., unit tests fail), **When** the process encounters the failure, **Then** it stops immediately, reports the failure, and does not proceed to bump-version or subsequent phases
3. **Given** the operator approves all gates, **When** the release completes, **Then** a summary report shows all completed actions with links to images and PRs

---

### Edge Cases

- What happens when the target version already has images in the registry?
- What happens when a community-operators PR branch already exists from a previous failed attempt?
- What happens when the `replaces` version in the CSV doesn't exist in the channel?
- What happens when `opm validate` fails after rendering the FBC catalog?
- What happens when the operator is not logged into the container registry?
- What happens when the community-operators fork is not up-to-date with upstream?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a `make bump-version` target that updates all version references (env file, CSV metadata, CSV deployment image, bundle annotations) given NEW_VERSION and PRIOR_VERSION parameters
- **FR-002**: System MUST validate version format (semver) before making any changes
- **FR-003**: System MUST sync CRD files from `config/crd/bases/` to `bundle/manifests/` during version bump
- **FR-004**: System MUST support a `--dry-run` mode that shows all planned actions without executing them
- **FR-005**: System MUST build three container images per release: operator, bundle, and FBC catalog
- **FR-006**: System MUST render FBC catalog data using `opm render` and validate using `opm validate`
- **FR-007**: System MUST create PRs to both k8s-operatorhub/community-operators and redhat-openshift-ecosystem/community-operators-prod with correct directory structure and content
- **FR-008**: System MUST pause for human approval before irreversible actions: image push, git push, and PR creation
- **FR-009**: System MUST verify prerequisites before starting: registry login, gh auth, fork repos existence, all tests passing
- **FR-010**: System MUST produce a summary report at the end of each release showing all completed actions, image URLs, and PR links
- **FR-011**: System MUST support aborting the release at any approval gate without leaving partial state
- **FR-012**: System MUST update the FBC channel entries with correct `replaces` chain when bumping version

### Key Entities

- **Release Version**: A semver string (e.g., 0.0.10) that drives all version references across the project
- **Release Phase**: A discrete step in the release pipeline (verify, bump, build, push, commit, PR) that can succeed or fail independently
- **Approval Gate**: A human decision point where the process pauses and presents a summary for approval before proceeding
- **Release Report**: A final summary of all actions taken, artifacts produced, and links generated during a release

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Release manager can bump all version references by running a single command, completing in under 10 seconds
- **SC-002**: Dry-run mode accurately previews all release actions with zero side effects
- **SC-003**: Full release (bump → build → push → PR) completes in under 15 minutes with human gates, compared to current 30+ minutes of manual work
- **SC-004**: Zero manual file edits required during a standard release (all changes produced by automation)
- **SC-005**: Community-operators PRs pass upstream CI on first submission in 90%+ of releases
- **SC-006**: Release process can be aborted at any gate without leaving orphaned branches, partial pushes, or inconsistent state

## Assumptions

- Release manager has `podman` (or `docker`) logged into `quay.io`
- Release manager has `gh` CLI authenticated with push access
- Community operator fork repos are cloned at known paths (`~/temp/20260213_SPECKIT/k8s-community-operators` and `~/temp/20260213_SPECKIT/community-operators-prod`)
- `opm` CLI is available for FBC catalog operations
- All CI checks (unit, integration, e2e) pass before release is initiated
- The operator follows a single-channel (alpha) upgrade path
- This process targets the nfs-provisioner-operator first, then will be adapted for kserve

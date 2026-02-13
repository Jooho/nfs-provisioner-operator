---
description: "Task list for Production Quality Codebase Refactoring"
---

# Tasks: Production Quality Codebase Refactoring

**Input**: Design documents from `/specs/001-production-refactor/`
**Prerequisites**: plan.md (required), spec.md (required), data-model.md, contracts/module-interfaces.md, research.md, quickstart.md

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story. Each story represents a complete, independently deliverable increment of value.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

- **Go packages**: `pkg/` for internal packages, `controllers/` for controller wrappers
- **Tests**: `pkg/<module>/<module>_test.go` for unit tests, `test/integration/` and `test/e2e/` for integration/E2E tests
- **Config**: `config/` for Kubernetes manifests, `.golangci.yml` for linting config

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [ ] T001 Verify Go 1.24 installation and set PATH=~/dev/lang/go/bin:$PATH
- [ ] T002 Run go mod download and go mod tidy to ensure dependencies are current
- [ ] T003 Run go mod vendor to vendor dependencies
- [ ] T004 [P] Create pkg/ directory structure for new modules
- [ ] T005 [P] Create test/integration/ directory for integration tests
- [ ] T006 [P] Create test/e2e/ directory for E2E tests

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T007 Configure golangci-lint by creating .golangci.yml in repository root
- [x] T008 Add lint target to Makefile (make lint: golangci-lint run --fix)
- [x] T009 [P] Update Makefile test target to enforce 80% coverage threshold
- [x] T010 [P] Add coverage-report target to Makefile (generates HTML coverage report)
- [x] T011 Run make lint to establish baseline and fix existing linting issues in controllers/
- [x] T012 Commit baseline: "chore: add golangci-lint configuration and fix baseline issues"

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Modular Code Architecture (Priority: P1) 🎯 MVP

**Goal**: Refactor codebase from monolithic controller to modular pkg/-based architecture with clear separation of concerns

**Independent Test**: Code review confirms each module has single responsibility, interfaces are well-defined, and new contributors can navigate the structure in under 30 minutes

### Implementation for User Story 1

#### Step 1: Create pkg/validation Module

- [x] T013 [P] [US1] Create pkg/validation/validator.go with Validator interface
- [x] T014 [US1] Implement validation logic for storage option mutual exclusivity in pkg/validation/validator.go
- [x] T015 [US1] Improve error messages to be actionable (e.g., "exactly one of spec.hostPathDir, spec.pvc, or spec.scForNFSPvc must be set")
- [ ] T016 [US1] Add field format validation (storageSize, scForNFS, image) in pkg/validation/validator.go

#### Step 2: Create pkg/defaults Module

- [x] T017 [P] [US1] Create pkg/defaults/defaults.go with ApplyDefaults function
- [x] T018 [US1] Implement default value application logic (storageSize=10Gi, scForNFS=nfs, image, imagePullPolicy)
- [x] T019 [US1] Ensure ApplyDefaults is idempotent and doesn't override user-provided values

#### Step 3: Create pkg/builder Module

- [x] T020 [P] [US1] Move builder/ to pkg/builder/ directory
- [x] T021 [US1] Refactor pkg/builder/builder.go to provide resource-specific builder functions
- [x] T022 [US1] Add BuildDeployment function in pkg/builder/builder.go
- [x] T023 [P] [US1] Add BuildService function in pkg/builder/builder.go
- [x] T024 [P] [US1] Add BuildServiceAccount function in pkg/builder/builder.go
- [ ] T025 [P] [US1] Add BuildPVC function in pkg/builder/builder.go
- [x] T026 [P] [US1] Add BuildStorageClass function in pkg/builder/builder.go
- [x] T027 [P] [US1] Add BuildRBAC functions (ClusterRole, ClusterRoleBinding, Role, RoleBinding) in pkg/builder/builder.go
- [x] T028 [P] [US1] Add BuildSCC function (OpenShift only) in pkg/builder/builder.go

#### Step 4: Refactor pkg/resources Module

- [ ] T029 [US1] Move controllers/resources/ to pkg/resources/ directory
- [ ] T030 [US1] Update pkg/resources/manager.go to implement ResourceManager interface from contracts/
- [ ] T031 [US1] Refactor pkg/resources/deployment.go to use pkg/builder functions
- [ ] T032 [P] [US1] Refactor pkg/resources/service.go to use pkg/builder functions
- [ ] T033 [P] [US1] Refactor pkg/resources/pvc.go to use pkg/builder functions
- [ ] T034 [P] [US1] Refactor pkg/resources/rbac.go to use pkg/builder functions
- [ ] T035 [P] [US1] Refactor pkg/resources/scc.go to use pkg/builder functions
- [ ] T036 [P] [US1] Refactor pkg/resources/storageclass.go to use pkg/builder functions
- [ ] T037 [P] [US1] Refactor pkg/resources/serviceaccount.go to use pkg/builder functions

#### Step 5: Create pkg/reconciler Module

- [ ] T038 [US1] Create pkg/reconciler/reconciler.go with Reconciler interface
- [ ] T039 [US1] Implement NewReconciler constructor with dependency injection (Validator, ResourceManager, K8s client, logger)
- [ ] T040 [US1] Implement Reconcile method orchestrating: validation → defaults → resource creation → status update
- [ ] T041 [US1] Add proper error classification (validation vs. transient vs. permanent errors) in pkg/reconciler/reconciler.go
- [ ] T042 [US1] Implement requeue logic (no requeue for validation errors, exponential backoff for transient errors)

#### Step 6: Update Controller to Delegate

- [ ] T043 [US1] Simplify controllers/nfsprovisioner_controller.go to thin wrapper
- [ ] T044 [US1] Update controller Reconcile method to fetch CR and delegate to pkg/reconciler
- [ ] T045 [US1] Remove inline validation logic from controller (now in pkg/validation)
- [ ] T046 [US1] Update controller initialization to construct pkg/reconciler.Reconciler

#### Step 7: Move Defaults Module

- [ ] T047 [US1] Move controllers/defaults/ to pkg/defaults/ directory
- [ ] T048 [US1] Update all import paths in pkg/reconciler and pkg/resources to use pkg/defaults

#### Step 8: Update Documentation

- [ ] T049 [P] [US1] Update README.md with new pkg/ structure and module descriptions
- [ ] T050 [P] [US1] Update docs/ to reflect refactored architecture
- [ ] T051 [US1] Add godoc comments to all exported functions and types in pkg/ modules

**Checkpoint**: Modular architecture complete - codebase is navigable and modules have clear responsibilities

---

## Phase 4: User Story 2 - Robust Error Handling and Validation (Priority: P2)

**Goal**: Implement comprehensive error handling with Kubernetes Conditions, structured logging, and actionable error messages

**Independent Test**: Trigger error scenarios (invalid CR, API unavailable, resource conflict) and verify system responds with clear status conditions and graceful degradation

### Implementation for User Story 2

#### Step 1: Enhance Status Structure (CRD Update)

- [ ] T052 [US2] Add Conditions field to NFSProvisionerStatus in api/v1alpha1/nfsprovisioner_types.go
- [ ] T053 [P] [US2] Add ObservedGeneration field to NFSProvisionerStatus in api/v1alpha1/nfsprovisioner_types.go
- [ ] T054 [P] [US2] Add Phase field to NFSProvisionerStatus in api/v1alpha1/nfsprovisioner_types.go
- [ ] T055 [US2] Run make manifests generate to update CRD with new status fields

#### Step 2: Implement Status Condition Management

- [ ] T056 [US2] Create pkg/reconciler/conditions.go with condition helper functions
- [ ] T057 [US2] Add SetReadyCondition function (sets Ready=True when reconciliation succeeds)
- [ ] T058 [P] [US2] Add SetProgressingCondition function (sets Progressing=True during reconciliation)
- [ ] T059 [P] [US2] Add SetDegradedCondition function (sets Degraded=True on transient errors)
- [ ] T060 [P] [US2] Add SetAvailableCondition function (sets Available=True when Deployment has replicas)

#### Step 3: Enhance Error Handling in Reconciler

- [ ] T061 [US2] Update pkg/reconciler/reconciler.go to set Progressing=True at start of reconciliation
- [ ] T062 [US2] Add status condition updates for validation errors in pkg/reconciler/reconciler.go
- [ ] T063 [US2] Add status condition updates for resource creation failures in pkg/reconciler/reconciler.go
- [ ] T064 [US2] Update status with ObservedGeneration and Phase on each reconciliation in pkg/reconciler/reconciler.go
- [ ] T065 [US2] Implement graceful error recovery (set Degraded=True, log error, requeue with delay)

#### Step 4: Improve Validation Error Messages

- [ ] T066 [US2] Update pkg/validation/validator.go to return structured errors with field paths
- [ ] T067 [US2] Add examples of valid values in error messages (e.g., "must be a valid Kubernetes quantity like '10Gi' or '1Ti'")
- [ ] T068 [US2] Add validation for nodeSelector labels existence (warning if labels don't match cluster nodes)

#### Step 5: Add Structured Logging

- [ ] T069 [US2] Update pkg/reconciler/reconciler.go to use structured logging with consistent key names
- [ ] T070 [US2] Add log statements for key reconciliation events (validation success/failure, resource creation, status update)
- [ ] T071 [P] [US2] Update pkg/resources/manager.go to add structured logging for resource operations
- [ ] T072 [US2] Ensure all log statements include CR name and namespace as keys

#### Step 6: Implement Exponential Backoff

- [ ] T073 [US2] Verify controller-runtime's default exponential backoff is enabled in controllers/nfsprovisioner_controller.go
- [ ] T074 [US2] Add custom requeue logic for permanent errors (RequeueAfter: 5 minutes) in pkg/reconciler/reconciler.go

**Checkpoint**: Error handling complete - system fails gracefully with actionable messages and proper status conditions

---

## Phase 5: User Story 3 - Comprehensive Automated Testing (Priority: P3)

**Goal**: Achieve 80%+ code coverage with comprehensive unit and integration tests using Ginkgo/Gomega

**Independent Test**: Run `make test` and verify 80%+ coverage, all tests pass in under 10 minutes, CI blocks PRs below coverage threshold

### Unit Tests for User Story 3

> **NOTE: Write these tests using Ginkgo v2 with SpecContext pattern**

#### Step 1: Unit Tests for pkg/validation

- [ ] T075 [P] [US3] Create pkg/validation/validator_test.go with Ginkgo test suite
- [ ] T076 [P] [US3] Add unit test for valid hostPathDir configuration in pkg/validation/validator_test.go
- [ ] T077 [P] [US3] Add unit test for valid pvc configuration in pkg/validation/validator_test.go
- [ ] T078 [P] [US3] Add unit test for valid scForNFSPvc configuration in pkg/validation/validator_test.go
- [ ] T079 [P] [US3] Add unit test rejecting multiple storage options in pkg/validation/validator_test.go
- [ ] T080 [P] [US3] Add unit test rejecting no storage options in pkg/validation/validator_test.go
- [ ] T081 [P] [US3] Add unit test validating storageSize format in pkg/validation/validator_test.go
- [ ] T082 [P] [US3] Add unit test validating scForNFS name format in pkg/validation/validator_test.go
- [ ] T083 [US3] Verify pkg/validation coverage ≥90% using go test -coverprofile

#### Step 2: Unit Tests for pkg/defaults

- [ ] T084 [P] [US3] Create pkg/defaults/defaults_test.go with Ginkgo test suite
- [ ] T085 [P] [US3] Add unit test applying default storageSize in pkg/defaults/defaults_test.go
- [ ] T086 [P] [US3] Add unit test applying default scForNFS in pkg/defaults/defaults_test.go
- [ ] T087 [P] [US3] Add unit test applying default image configuration in pkg/defaults/defaults_test.go
- [ ] T088 [P] [US3] Add unit test not overriding user-provided values in pkg/defaults/defaults_test.go
- [ ] T089 [US3] Verify pkg/defaults coverage = 100% using go test -coverprofile

#### Step 3: Unit Tests for pkg/builder

- [ ] T090 [P] [US3] Create pkg/builder/builder_test.go with Ginkgo test suite
- [ ] T091 [P] [US3] Add unit test for BuildDeployment with hostPathDir volume in pkg/builder/builder_test.go
- [ ] T092 [P] [US3] Add unit test for BuildDeployment with PVC volume in pkg/builder/builder_test.go
- [ ] T093 [P] [US3] Add unit test for BuildService generating correct ports in pkg/builder/builder_test.go
- [ ] T094 [P] [US3] Add unit test for BuildStorageClass with correct provisioner in pkg/builder/builder_test.go
- [ ] T095 [P] [US3] Add unit test for BuildServiceAccount with correct metadata in pkg/builder/builder_test.go
- [ ] T096 [P] [US3] Add unit test for BuildPVC (returns nil when not needed) in pkg/builder/builder_test.go
- [ ] T097 [P] [US3] Add unit test for BuildSCC (OpenShift only) in pkg/builder/builder_test.go
- [ ] T098 [US3] Verify pkg/builder coverage ≥90% using go test -coverprofile

#### Step 4: Unit Tests for pkg/resources

- [ ] T099 [P] [US3] Update pkg/resources/manager_test.go to use SpecContext pattern
- [ ] T100 [P] [US3] Create pkg/resources/deployment_test.go with unit tests for deployment creation
- [ ] T101 [P] [US3] Create pkg/resources/service_test.go with unit tests for service creation
- [ ] T102 [P] [US3] Create pkg/resources/pvc_test.go with unit tests for PVC creation
- [ ] T103 [P] [US3] Create pkg/resources/rbac_test.go with unit tests for RBAC creation
- [ ] T104 [P] [US3] Create pkg/resources/scc_test.go with unit tests for SCC creation
- [ ] T105 [P] [US3] Create pkg/resources/storageclass_test.go with unit tests for StorageClass creation
- [ ] T106 [US3] Add unit test for idempotency (calling EnsureResources twice) in pkg/resources/manager_test.go
- [ ] T107 [US3] Verify pkg/resources coverage ≥80% using go test -coverprofile

#### Step 5: Unit Tests for pkg/reconciler

- [ ] T108 [P] [US3] Create pkg/reconciler/reconciler_test.go with Ginkgo test suite and mocks
- [ ] T109 [P] [US3] Add unit test for successful reconciliation (valid CR → resources created) in pkg/reconciler/reconciler_test.go
- [ ] T110 [P] [US3] Add unit test for validation failure (invalid CR → status error, no requeue) in pkg/reconciler/reconciler_test.go
- [ ] T111 [P] [US3] Add unit test for resource creation failure (API error → requeue with backoff) in pkg/reconciler/reconciler_test.go
- [ ] T112 [P] [US3] Add unit test for CR update scenario in pkg/reconciler/reconciler_test.go
- [ ] T113 [P] [US3] Add unit test for status condition updates in pkg/reconciler/reconciler_test.go
- [ ] T114 [US3] Verify pkg/reconciler coverage ≥85% using go test -coverprofile

#### Step 6: Integration Tests

- [ ] T115 [US3] Create test/integration/integration_suite_test.go with Ginkgo suite and envtest setup
- [ ] T116 [US3] Configure envtest to start Kubernetes API server in test/integration/integration_suite_test.go
- [ ] T117 [US3] Create test/integration/reconcile_integration_test.go for end-to-end reconciliation tests
- [ ] T118 [P] [US3] Add integration test: Create NFSProvisioner CR → verify all resources created
- [ ] T119 [P] [US3] Add integration test: Update NFSProvisioner CR → verify resources updated
- [ ] T120 [P] [US3] Add integration test: Delete NFSProvisioner CR → verify resources garbage collected
- [ ] T121 [P] [US3] Add integration test: Invalid CR → verify status error condition set
- [ ] T122 [US3] Verify integration tests complete in under 10 minutes

#### Step 7: E2E Tests (Optional but Recommended)

- [ ] T123 [P] [US3] Create test/e2e/e2e_suite_test.go with Ginkgo suite
- [ ] T124 [US3] Add E2E test helper to provision Kind cluster with OLM in test/e2e/cluster_setup.go
- [ ] T125 [US3] Create test/e2e/nfsprovisioner_e2e_test.go for full workflow tests
- [ ] T126 [P] [US3] Add E2E test: Deploy operator via OLM → create CR → verify NFS provisioner works
- [ ] T127 [P] [US3] Add E2E test: Create PVC using NFS StorageClass → verify PV created
- [ ] T128 [US3] Add E2E test: Mount NFS PVC in pod → write data → verify persistence

#### Step 8: Coverage Enforcement

- [ ] T129 [US3] Update Makefile test target to fail if coverage < 80%
- [ ] T130 [US3] Add make coverage-report target generating HTML coverage report
- [ ] T131 [US3] Run make test and verify overall coverage ≥80%
- [ ] T132 [US3] Run make coverage-report and review uncovered lines, add tests as needed

**Checkpoint**: Testing complete - 80%+ coverage, tests pass consistently, CI enforces coverage

---

## Phase 6: User Story 4 - OpenShift 4.19+ Platform Support (Priority: P4)

**Goal**: Update OLM bundle metadata to declare OpenShift 4.19+ compatibility and ensure platform-specific features work

**Independent Test**: Deploy operator on OpenShift 4.19+ cluster via OperatorHub and verify installation succeeds, SCC is applied, and operator functions correctly

### Implementation for User Story 4

#### Step 1: Update Bundle Metadata

- [ ] T133 [US4] Update bundle/manifests/nfs-provisioner-operator.clusterserviceversion.yaml to declare support for OpenShift 4.19+
- [ ] T134 [P] [US4] Add supported versions annotation (e.g., "olm.properties": supports 4.16-4.20)
- [ ] T135 [P] [US4] Update bundle/metadata/annotations.yaml with OpenShift 4.19+ compatibility metadata
- [ ] T136 [US4] Run operator-sdk bundle validate to verify bundle correctness

#### Step 2: Test SCC Detection on OpenShift 4.19+

- [ ] T137 [US4] Add integration test for SCC CRD detection logic in test/integration/scc_detection_test.go
- [ ] T138 [US4] Verify pkg/resources/scc.go detects SCC CRD presence correctly on OpenShift clusters
- [ ] T139 [US4] Verify pkg/resources/scc.go gracefully skips SCC creation on vanilla Kubernetes

#### Step 3: OpenShift Certification Preparation

- [ ] T140 [P] [US4] Review Red Hat OpenShift certification requirements for operators
- [ ] T141 [P] [US4] Ensure all required CSV fields are populated (description, keywords, maintainers, links)
- [ ] T142 [US4] Add OWNERS file if required for certification
- [ ] T143 [US4] Document certification process in docs/certification.md

#### Step 4: Deploy and Validate on OpenShift 4.19+

- [ ] T144 [US4] Provision OpenShift 4.19 test cluster (or use existing cluster)
- [ ] T145 [US4] Build and push operator bundle image to quay.io or internal registry
- [ ] T146 [US4] Create CatalogSource pointing to bundle image in test cluster
- [ ] T147 [US4] Install operator via OperatorHub UI or kubectl apply -f subscription.yaml
- [ ] T148 [US4] Verify operator pod starts successfully and is Running
- [ ] T149 [US4] Create NFSProvisioner CR and verify reconciliation succeeds
- [ ] T150 [US4] Verify SCC is created and applied to NFS server pod on OpenShift
- [ ] T151 [US4] Verify NFS provisioner creates PVs correctly using NFS StorageClass

#### Step 5: Test Upgrade from 4.18 to 4.19

- [ ] T152 [US4] Deploy operator on OpenShift 4.18 cluster
- [ ] T153 [US4] Create NFSProvisioner CR and verify functionality
- [ ] T154 [US4] Upgrade cluster from 4.18 to 4.19
- [ ] T155 [US4] Verify operator continues running without issues post-upgrade
- [ ] T156 [US4] Verify existing NFSProvisioner CR continues to function

**Checkpoint**: OpenShift 4.19+ support complete - operator installs via OLM and functions correctly on OpenShift 4.19+

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final improvements and documentation updates affecting multiple user stories

- [ ] T157 [P] Update README.md with refactored architecture overview, quick start, and testing instructions
- [ ] T158 [P] Update docs/makefile_playbook.md with new Makefile targets (lint, coverage-report)
- [ ] T159 [P] Create docs/architecture.md documenting pkg/ module structure and interfaces
- [ ] T160 [P] Update docs/test_script.md with Ginkgo/Gomega testing guidance and coverage requirements
- [ ] T161 Verify all godoc comments are present on exported functions and types in pkg/
- [ ] T162 Run golangci-lint and fix any remaining linting issues
- [ ] T163 Run make test and ensure all tests pass with ≥80% coverage
- [ ] T164 Run make build and verify binary builds successfully
- [ ] T165 Review quickstart.md in specs/001-production-refactor/ for accuracy after refactoring
- [ ] T166 Update CHANGELOG.md or RELEASE_NOTES.md with refactoring changes

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational - Modular architecture (FOUNDATIONAL for US2, US3, US4)
- **User Story 2 (Phase 4)**: Depends on US1 completion - Error handling builds on modular architecture
- **User Story 3 (Phase 5)**: Depends on US1 and US2 - Tests validate modular code and error handling
- **User Story 4 (Phase 6)**: Depends on US1, US2, US3 - OpenShift support needs stable, tested codebase
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: MUST complete first - foundational for all other stories
- **User Story 2 (P2)**: Depends on US1 (needs modular architecture)
- **User Story 3 (P3)**: Depends on US1 and US2 (tests validate modules and error handling)
- **User Story 4 (P4)**: Depends on US1, US2, US3 (needs stable, tested codebase)

**CRITICAL PATH**: Setup → Foundational → US1 → US2 → US3 → US4 → Polish

### Within Each User Story

**User Story 1** (Modular Architecture):
- Create pkg/validation and pkg/defaults in parallel ([P] tasks T013, T017)
- Create pkg/builder after validation/defaults
- Refactor pkg/resources after builder exists
- Create pkg/reconciler after all modules exist
- Update controller last (depends on reconciler)

**User Story 2** (Error Handling):
- Update CRD status fields first (T052-T055)
- Implement condition management functions in parallel ([P] tasks T057-T060)
- Update reconciler to use conditions
- Improve validation error messages in parallel with structured logging

**User Story 3** (Testing):
- Write unit tests for each module in parallel ([P] tasks within each step)
- Integration tests depend on all unit tests passing
- E2E tests depend on integration tests passing

**User Story 4** (OpenShift 4.19+):
- Update bundle metadata in parallel ([P] tasks T134, T135)
- Test SCC detection in parallel with bundle updates
- Deploy and validate sequentially (T144-T151)

### Parallel Opportunities

- **Setup Phase**: Tasks T004, T005, T006 can run in parallel
- **Foundational Phase**: Tasks T009, T010 can run in parallel
- **US1 - Validation & Defaults**: Tasks T013 and T017 can run in parallel
- **US1 - Builder Functions**: Tasks T023-T028 can run in parallel (different files)
- **US1 - Resources Refactor**: Tasks T032-T037 can run in parallel (different files)
- **US1 - Documentation**: Tasks T049, T050 can run in parallel
- **US2 - Status Fields**: Tasks T053, T054 can run in parallel
- **US2 - Condition Functions**: Tasks T058-T060 can run in parallel
- **US2 - Logging**: Tasks T071 can run in parallel with T069-T070
- **US3 - All Unit Tests**: Most unit test tasks within each module can run in parallel ([P] marker)
- **US3 - E2E Tests**: Tasks T126, T127 can run in parallel
- **US4 - Bundle Metadata**: Tasks T134, T135 can run in parallel
- **US4 - Certification Prep**: Tasks T140, T141 can run in parallel
- **Polish Phase**: Tasks T157-T161 can run in parallel

---

## Parallel Example: User Story 1

```bash
# Launch validation and defaults modules in parallel:
# Terminal 1:
cd pkg/validation && code validator.go validator_test.go

# Terminal 2:
cd pkg/defaults && code defaults.go defaults_test.go

# After both complete, proceed to builder module:
cd pkg/builder && code builder.go builder_test.go

# Launch resource refactoring in parallel (different files):
# Terminal 1: deployment.go
# Terminal 2: service.go
# Terminal 3: rbac.go
# Terminal 4: scc.go
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T006)
2. Complete Phase 2: Foundational (T007-T012) - CRITICAL GATE
3. Complete Phase 3: User Story 1 (T013-T051) - Modular architecture
4. **STOP and VALIDATE**: Code review confirms modular structure
5. Commit MVP: "feat: refactor to modular pkg/-based architecture"

**MVP Deliverable**: Codebase with clear module separation, improved navigability, ready for error handling and testing

### Incremental Delivery

1. **Foundation** (Setup + Foundational) → Linting and coverage enforcement in place
2. **Add US1** (Modular Architecture) → Commit and deploy → Validate structure
3. **Add US2** (Error Handling) → Commit and deploy → Test error scenarios
4. **Add US3** (Testing) → Commit and deploy → Verify 80%+ coverage
5. **Add US4** (OpenShift 4.19+) → Commit and deploy → Test on OpenShift cluster
6. **Polish** → Final documentation and cleanup

Each story adds value without breaking previous stories.

### Sequential Team Strategy

**Recommended for single developer**:

1. Complete Setup + Foundational together (1 day)
2. Focus on User Story 1 completely (1-2 weeks)
3. Move to User Story 2 (1 week)
4. Implement User Story 3 (1-2 weeks for comprehensive tests)
5. Finalize User Story 4 (2-3 days)
6. Polish and documentation (2-3 days)

**Total Estimate**: 4-6 weeks for complete refactoring

### Parallel Team Strategy (if multiple developers available)

Not recommended for this feature. Due to the nature of refactoring, User Story 1 (modular architecture) MUST be complete before other stories can begin effectively. However, within US1, certain tasks can be parallelized:

- **Developer A**: pkg/validation and pkg/defaults modules
- **Developer B**: pkg/builder module
- **Developer C**: pkg/resources refactoring

After US1 completion:
- **Developer A**: User Story 2 (Error Handling)
- **Developer B**: User Story 3 (Testing)
- **Developer C**: User Story 4 (OpenShift 4.19+)

---

## Notes

- **[P] tasks**: Different files, no dependencies - can run in parallel
- **[Story] label**: Maps task to specific user story for traceability
- **SpecContext**: All Ginkgo tests MUST use `func(ctx SpecContext)` parameter per constitution
- **SpecTimeout**: Set timeout for long-running tests: `SpecTimeout(30*time.Second)`
- **Coverage**: Target 80% minimum (exceeds constitution's 70%)
- **Go Path**: Always set `PATH=~/dev/lang/go/bin:$PATH` before build/test
- **Commit frequency**: Commit after completing each module or logical group of tasks
- **Testing workflow**: Tests written → Tests fail (Red) → Implement → Tests pass (Green) → Refactor
- **Constitution compliance**: All code must pass `make lint` with zero warnings

## Success Metrics

- ✅ 80%+ code coverage across all pkg/ modules
- ✅ All tests pass in under 2 minutes (unit) and 10 minutes (integration)
- ✅ golangci-lint runs with zero warnings
- ✅ New contributors navigate codebase in under 30 minutes
- ✅ Code review cycle time decreases by 40%
- ✅ Operator deploys successfully on OpenShift 4.19+ via OLM

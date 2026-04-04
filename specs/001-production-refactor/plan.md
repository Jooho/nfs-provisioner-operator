# Implementation Plan: Production Quality Codebase Refactoring

**Branch**: `001-production-refactor` | **Date**: 2026-02-13 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-production-refactor/spec.md`

**Note**: This plan addresses the transition of the NFS Provisioner Operator from proof-of-concept quality to production-ready, maintainable, and well-tested codebase.

## Summary

**Primary Requirement**: Refactor the NFS Provisioner Operator codebase to production quality by implementing modular architecture, comprehensive error handling, automated testing (80%+ coverage), and OpenShift 4.19+ support.

**Technical Approach** (from research):
- Refactor existing Go codebase into clearly separated modules following controller-runtime best practices
- Implement comprehensive error handling with structured logging and status condition updates
- Establish Ginkgo/Gomega test suite with unit tests (business logic) and integration tests (envtest-based)
- Integrate golangci-lint for code quality enforcement
- Automate test execution via Makefile targets for consistent developer workflows and CI integration
- Update OLM bundle metadata to declare OpenShift 4.19+ compatibility

## Technical Context

**Language/Version**: Go 1.24 (as specified in go.mod)

**Primary Dependencies**:
- controller-runtime v0.18.4 (Kubernetes operator framework)
- Ginkgo v2.19.0 + Gomega v1.33.1 (testing framework)
- kubebuilder (code generation and scaffolding)
- operator-sdk (OLM bundle management)

**Storage**: N/A (this feature refactors code, does not introduce new data storage)

**Testing**:
- Ginkgo v2 + Gomega for unit and integration tests
- envtest v1.30.0 for integration testing against simulated Kubernetes API
- Go built-in coverage tooling (`go test -coverprofile`)
- Target: 80% code coverage minimum

**Target Platform**:
- Linux (operator controller container)
- Kubernetes 1.30+ and OpenShift 4.16-4.19+ (target compatibility range)

**Project Type**: Single Go project (Kubernetes operator)

**Performance Goals**:
- Reconciliation loops complete within 30 seconds for simple operations (per constitution)
- Unit test suite executes in under 2 minutes
- Integration test suite completes in under 10 minutes

**Constraints**:
- Memory limit ≤256Mi for operator controller (per constitution)
- CPU requests ≤100m (per constitution)
- Maintain backward compatibility with existing CRD schema
- Zero downtime during operator upgrade
- No breaking changes to existing NFSProvisioner CR API

**Scale/Scope**:
- Support 100 NFSProvisioner CR instances per cluster
- Handle 50 concurrent PVC creation requests
- Codebase size: ~2000-3000 lines of production code (estimated post-refactor)

**Note**: The user mentioned Python in the planning input, but code inspection confirms this is a **Go-only project**. No Python code exists in the repository. All testing and linting tooling will be Go-based (golangci-lint, go test, Ginkgo/Gomega).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Code Quality & Go Standards (Principle I)

- ✅ **PASS**: Refactoring will organize code into logical modules with single responsibilities
- ✅ **PASS**: golangci-lint will be integrated to enforce Go standards
- ✅ **PASS**: Go 1.24 version consistency maintained (already in go.mod)
- ✅ **PASS**: All exported functions will receive godoc comments as part of refactoring
- ✅ **PASS**: Error handling will be explicit throughout refactored code

**Compliance**: This feature directly addresses Code Quality principle by transforming PoC-level code into production-grade, modular architecture.

### Testing Standards & Ginkgo Practices (Principle II - NON-NEGOTIABLE)

- ✅ **PASS**: Ginkgo v2.19.0 already in use, will be expanded for comprehensive coverage
- ✅ **PASS**: All new tests MUST use `SpecContext` parameter (per constitution)
- ✅ **PASS**: SpecTimeout will be set appropriately (30s default, adjusted per test)
- ✅ **PASS**: Unit tests will be added to `pkg/` modules and existing `controllers/resources/`
- ✅ **PASS**: Integration tests will use envtest (already configured with ENVTEST_K8S_VERSION=1.30.0)
- ⚠️ **REQUIRES ATTENTION**: Target 80% coverage (spec requirement) vs. constitution's 70% minimum
  - **Resolution**: Adopt the higher standard (80%) from spec as it exceeds constitutional minimum
- ✅ **PASS**: Test failure scenarios will be explicitly covered (API unavailability, invalid CRs, conflicts)

**Compliance**: Feature requirements exceed constitutional minimums. 80% coverage target is more stringent than 70% constitutional requirement.

### User Experience Consistency (Principle III)

- ✅ **PASS**: CR API schema remains unchanged (backward compatibility maintained)
- ✅ **PASS**: Error handling improvements will provide actionable status condition messages
- ✅ **PASS**: Structured logging with consistent key names will be implemented
- ✅ **PASS**: Documentation (README.md, docs/) will be updated to reflect refactored structure

**Compliance**: Refactoring enhances UX through better error messages and documentation without breaking existing APIs.

### Performance & Resource Efficiency (Principle IV)

- ✅ **PASS**: Reconciliation performance goals align with constitution (30s for simple operations)
- ✅ **PASS**: Resource constraints met (≤256Mi memory, ≤100m CPU)
- ✅ **PASS**: Scalability targets maintained (100 CRs, 50 concurrent PVCs)

**Compliance**: Feature preserves existing performance characteristics while improving code quality.

### Security & RBAC Requirements

- ✅ **PASS**: Existing RBAC permissions preserved (no changes to security model)
- ✅ **PASS**: SCC detection logic for OpenShift vs Kubernetes remains intact
- ✅ **PASS**: No changes to container security posture

**Compliance**: Security model unchanged; refactoring is internal code quality improvement.

### Development Workflow

- ✅ **PASS**: Feature branch naming follows `###-feature-name` format (001-production-refactor)
- ✅ **PASS**: CI integration will be enhanced with test coverage checks
- ✅ **PASS**: Makefile targets will standardize test execution

**Compliance**: Feature enhances development workflow through better testing infrastructure.

**GATE STATUS**: ✅ **PASSED** - All constitutional requirements met or exceeded. Proceed to Phase 0 research.

## Project Structure

### Documentation (this feature)

```text
specs/001-production-refactor/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

**Current Structure** (as-is):
```text
.
├── api/v1alpha1/                # CRD type definitions
│   ├── nfsprovisioner_types.go
│   └── groupversion_info.go
├── cmd/
│   └── main.go                  # Operator entrypoint
├── controllers/
│   ├── nfsprovisioner_controller.go  # Main reconciler (244 lines)
│   ├── defaults/
│   │   └── defaults.go          # Default values
│   └── resources/               # Resource management module
│       ├── common.go
│       ├── deployment.go
│       ├── manager.go
│       ├── manager_test.go      # Existing Ginkgo test
│       ├── pvc.go
│       ├── rbac.go
│       ├── scc.go
│       ├── service.go
│       ├── serviceaccount.go
│       └── storageclass.go
├── builder/                     # Object builder utilities
│   ├── namespace_builder.go
│   └── object_builder.go
├── config/                      # Kubernetes manifests and kustomize
│   ├── crd/
│   ├── manager/
│   ├── rbac/
│   ├── samples/
│   └── ...
└── bundle/                      # OLM bundle metadata
```

**Target Structure** (to-be, post-refactor):
```text
.
├── api/v1alpha1/                # CRD type definitions (unchanged)
├── cmd/
│   └── main.go                  # Entrypoint (minimal, delegates to pkg/)
├── pkg/                         # NEW: Internal packages for better organization
│   ├── reconciler/              # NEW: Reconciliation logic
│   │   ├── reconciler.go        # Core reconciliation loop
│   │   └── reconciler_test.go   # Ginkgo unit tests
│   ├── validation/              # NEW: Input validation module
│   │   ├── validator.go         # CR validation logic
│   │   └── validator_test.go
│   ├── resources/               # REFACTORED: Resource management (from controllers/resources)
│   │   ├── manager.go           # Resource manager orchestration
│   │   ├── manager_test.go
│   │   ├── deployment.go
│   │   ├── deployment_test.go   # NEW: Per-resource tests
│   │   ├── service.go
│   │   ├── service_test.go      # NEW
│   │   ├── rbac.go
│   │   ├── rbac_test.go         # NEW
│   │   ├── scc.go
│   │   ├── scc_test.go          # NEW
│   │   ├── pvc.go
│   │   ├── pvc_test.go          # NEW
│   │   └── storageclass.go
│   │       └── storageclass_test.go  # NEW
│   ├── defaults/                # MOVED: Default values (from controllers/defaults)
│   │   ├── defaults.go
│   │   └── defaults_test.go     # NEW
│   └── builder/                 # MOVED: Object builders (from builder/)
│       ├── builder.go
│       └── builder_test.go      # NEW
├── controllers/                 # SIMPLIFIED: Thin controller wrapper
│   └── nfsprovisioner_controller.go  # Delegates to pkg/reconciler
├── test/                        # NEW: Integration and E2E tests
│   ├── e2e/
│   │   ├── e2e_suite_test.go    # Ginkgo E2E suite
│   │   └── nfsprovisioner_e2e_test.go
│   └── integration/
│       ├── integration_suite_test.go  # envtest-based integration suite
│       └── reconcile_integration_test.go
├── config/                      # Kubernetes manifests (unchanged structure)
└── bundle/                      # OLM bundle (metadata updated for 4.19+)
```

**Structure Decision**:
The refactoring moves from a **controllers-centric layout** to a **pkg-centric layout** that better separates concerns:

1. **pkg/reconciler**: Isolates reconciliation logic from controller-runtime plumbing
2. **pkg/validation**: Centralizes CR validation (currently inline in controller)
3. **pkg/resources**: Keeps resource management but adds per-resource unit tests
4. **test/**: Dedicated directory for integration and E2E tests (separate from unit tests)

This structure follows Kubernetes operator best practices (e.g., operator-sdk project layout recommendations) and enables independent testing of each module.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

**No violations detected**. All constitutional requirements are met or exceeded by this refactoring feature.

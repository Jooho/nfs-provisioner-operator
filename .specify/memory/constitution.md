<!--
Sync Impact Report:
Version: INITIAL → 1.0.0
Principles Established:
  - I. Code Quality & Go Standards (NEW)
  - II. Testing Standards & Ginkgo Practices (NEW)
  - III. User Experience Consistency (NEW)
  - IV. Performance & Resource Efficiency (NEW)
Added Sections:
  - Security & RBAC Requirements (NEW)
  - Development Workflow (NEW)
Removed Sections: None
Templates Requiring Updates:
  ✅ plan-template.md - Constitution Check section aligns with new principles
  ✅ spec-template.md - User scenarios and requirements align with UX principles
  ✅ tasks-template.md - Task categorization supports testing and quality gates
Follow-up TODOs: None
-->

# NFS Provisioner Operator Constitution

## Core Principles

### I. Code Quality & Go Standards

**MUST adhere to the following Go development practices:**

- Follow official Go code review standards and effective Go guidelines
- Use `gofmt` and `golangci-lint` for consistent formatting and linting
- Maintain Go version consistency: use Go 1.24 as specified in go.mod
- Set `PATH=~/dev/lang/go/bin:$PATH` when custom Go installation required
- All exported functions, types, and packages MUST have godoc comments
- Avoid premature optimization; clarity over cleverness
- Use meaningful variable and function names that convey intent
- Handle errors explicitly; never ignore error return values
- Prefer composition over inheritance; use interfaces judiciously

**Rationale**: Kubernetes operators are complex distributed systems. Consistent code quality reduces cognitive load, improves maintainability, and prevents subtle bugs in reconciliation logic.

### II. Testing Standards & Ginkgo Practices (NON-NEGOTIABLE)

**MUST follow these testing requirements:**

- Use Ginkgo v2 test framework with Gomega matchers for all controller tests
- **CRITICAL**: All Ginkgo tests MUST use `SpecContext` parameter (not plain `Context`)
- Example: `It("should reconcile", func(ctx SpecContext) { ... }, SpecTimeout(30*time.Second))`
- Set appropriate `SpecTimeout` for long-running operations (default 30 seconds)
- Write unit tests for business logic in `internal/controller/resources/`
- Write integration tests using `envtest` for controller reconciliation loops
- Achieve minimum 70% code coverage for controller logic
- Test failure scenarios: resource conflicts, API server unavailability, invalid CRs
- Mock external dependencies; use real Kubernetes API via envtest for integration tests

**Rationale**: SpecContext enables proper cancellation handling and prevents test timeouts. Kubernetes operators require rigorous testing due to asynchronous reconciliation and complex state management.

### III. User Experience Consistency

**MUST ensure consistent operator experience:**

- Custom Resource (CR) API design:
  - Use clear, self-documenting field names in CRD spec
  - Provide comprehensive validation via OpenAPI schema (kubebuilder markers)
  - Document all CR fields with `+kubebuilder:validation` and description comments
  - Use status conditions following Kubernetes conventions (Ready, Progressing, Degraded)
- Error handling and messaging:
  - Return actionable error messages to CR status conditions
  - Use structured logging (controller-runtime's `logr`) with consistent key names
  - Emit Kubernetes events for important state changes (creation, errors, warnings)
- Documentation:
  - Maintain up-to-date CR examples in `config/samples/`
  - Update README.md with feature changes and usage instructions
  - Keep `docs/` directory synchronized with implementation

**Rationale**: Operators are infrastructure components consumed by platform engineers. Clear APIs, helpful errors, and thorough documentation reduce operational burden and improve adoption.

### IV. Performance & Resource Efficiency

**MUST meet these performance standards:**

- Reconciliation performance:
  - Reconcile loops MUST complete within 30 seconds for simple operations
  - Use requeue strategies appropriately; avoid tight reconciliation loops (<5 seconds)
  - Implement exponential backoff for error retry (controller-runtime default)
- Resource constraints:
  - Operator controller MUST run with memory limit ≤256Mi in production
  - CPU requests SHOULD be ≤100m; burst to 500m acceptable
  - NFS Provisioner pods MUST specify resource requests/limits in CR
- Scalability targets:
  - Support up to 100 NFSProvisioner CR instances per cluster
  - Handle 50 concurrent PVC creation requests without degradation
  - Minimize watch cache memory footprint using field selectors where applicable

**Rationale**: Operators run continuously in cluster control planes. Inefficient reconciliation or excessive resource usage impacts cluster stability and operational costs.

## Security & RBAC Requirements

**MUST implement security best practices:**

- Follow principle of least privilege for ServiceAccount RBAC:
  - Grant only required permissions (specified in `config/rbac/`)
  - Use namespace-scoped roles where possible; avoid ClusterRole unless necessary
  - Document RBAC requirements in role manifests with comments
- Security Context Constraints (SCC) for OpenShift:
  - Detect SCC CRD presence for OpenShift vs Kubernetes environments
  - Apply appropriate SCC for NFS server pods (may require privileged for NFS kernel modules)
  - Check SCC availability before creating pods; fail gracefully with clear error
- Container security:
  - Run operator controller as non-root user (UID 65532)
  - Use distroless or minimal base images (gcr.io/distroless/static recommended)
  - Avoid privileged containers in operator; isolate privilege to workload pods only
- Supply chain security:
  - Pin dependency versions in go.mod; use `go mod tidy && go mod vendor`
  - Scan container images for CVEs using approved tooling
  - Sign and verify container images in CI/CD pipeline

**Rationale**: Operators have elevated Kubernetes API privileges. Security vulnerabilities can compromise entire clusters. Defense in depth is mandatory.

## Development Workflow

**MUST follow these development practices:**

- Version control and branching:
  - Use feature branches: `###-feature-name` format
  - Main branch (`main`) MUST always be deployable
  - Squash commits for feature branches; preserve semantic commit history on main
- Code review requirements:
  - All changes require PR review and approval
  - PR MUST pass CI checks: lint, unit tests, integration tests
  - Include test evidence (coverage report, envtest output) in PR description
- Testing workflow:
  - Run tests with Go version matching go.mod: `export PATH=~/dev/lang/go/bin:$PATH && go test ./...`
  - Execute `make lint` before pushing changes
  - Run `make test` for unit tests; `make test-e2e` for integration tests
- Build and release:
  - Use `make docker-build` with semantic version tags (vMAJOR.MINOR.PATCH)
  - Update `config/manager/kustomization.yaml` with new image tags
  - Follow documentation in `docs/new_image.md` for image publication

**Rationale**: Systematic workflow prevents integration issues and ensures reproducible builds. Operator releases impact production clusters; discipline is non-negotiable.

## Governance

**Constitution Authority:**
This constitution supersedes all other development practices and guidelines. When conflicts arise between this document and team conventions, this constitution takes precedence.

**Amendment Procedure:**

1. Propose amendment with rationale and impact analysis
2. Update version number following semantic versioning:
   - **MAJOR**: Backward-incompatible principle changes or removals
   - **MINOR**: New principles or sections added
   - **PATCH**: Clarifications, wording fixes, non-semantic updates
3. Update `LAST_AMENDED_DATE` to amendment date
4. Propagate changes to dependent templates (plan, spec, tasks)
5. Document changes in Sync Impact Report (HTML comment at top of file)

**Compliance & Review:**

- All PRs MUST reference constitution compliance in review checklist
- Quarterly constitution review to assess relevance and effectiveness
- Complexity or deviations MUST be justified in `plan.md` Complexity Tracking table
- Use `.specify/templates/plan-template.md` Constitution Check section to validate features

**Versioning Policy:**

- Version format: MAJOR.MINOR.PATCH
- Track changes in Sync Impact Report
- Maintain backward compatibility for MINOR and PATCH changes

**Version**: 1.0.0 | **Ratified**: 2026-02-13 | **Last Amended**: 2026-02-13

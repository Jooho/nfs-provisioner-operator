# Changelog

All notable changes to the NFS Provisioner Operator will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.8] - 2026-02-13

### Added

#### Architecture Refactoring (Production-Ready)

- **Modular Package Structure**: Refactored codebase into well-defined packages with clear separation of concerns
  - `pkg/validation`: CR validation with actionable error messages (97.9% test coverage)
  - `pkg/defaults`: Default value application (100% test coverage)
  - `pkg/builder`: Stateless resource construction using pure functions
  - `pkg/resources`: Resource managers implementing the EnsureResource pattern (71.7% test coverage)
  - `pkg/reconciler`: Orchestrated reconciliation logic with error classification (75.4% test coverage)

- **Comprehensive Testing Infrastructure**:
  - Unit tests for all pkg/ modules using Ginkgo v2 + Gomega
  - Integration tests for SCC detection logic
  - Test coverage enforcement (80% target for pkg/)
  - Coverage reporting via `make coverage-report`

- **Error Handling & Status Management**:
  - Error classification (transient vs. permanent vs. unknown)
  - Intelligent requeue logic based on error type
  - Kubernetes Conditions pattern (Ready, Progressing, Degraded)
  - ObservedGeneration tracking for spec change detection
  - Phase field (Ready, Progressing, Failed) for status visibility

- **Platform Compatibility**:
  - OpenShift 4.19+ support with automatic SCC detection
  - Graceful degradation on vanilla Kubernetes (SCC skipped)
  - Platform-specific resource creation (SCC only on OpenShift)

- **Developer Experience**:
  - Comprehensive documentation (architecture.md, test_script.md, certification.md)
  - Updated README with quick start, testing instructions, and architecture overview
  - Makefile targets for linting (`make lint`) and coverage (`make coverage-report`)
  - Ginkgo/Gomega testing best practices documented

#### OpenShift Certification

- Added OpenShift 4.19+ version support declaration in OLM bundle metadata
- Created OWNERS file for repository maintainership
- Comprehensive certification guide (docs/certification.md)
- Bundle validation passing all operator-sdk checks

#### Documentation

- **docs/architecture.md**: Detailed architecture documentation covering design principles, package responsibilities, controller flow, resource management, error handling, and testing
- **docs/test_script.md**: Comprehensive testing guide with Ginkgo/Gomega patterns, coverage requirements, and E2E testing scenarios
- **docs/certification.md**: OpenShift certification process and requirements
- **docs/makefile_playbook.md**: Updated with new Makefile targets (lint, coverage-report, test)
- **README.md**: Refreshed with refactored architecture, quick start guide, testing instructions

### Changed

#### Code Quality Improvements

- **Validation**: Extracted into dedicated package with clear error messages
  - Storage option mutual exclusivity validation
  - Field format validation (storageSize, scForNFSProvisioner, image)
  - Reusable Validator interface for testability

- **Resource Management**: Migrated from monolithic approach to manager pattern
  - ServiceAccountManager, RBACManager, PVCManager, ServiceManager, DeploymentManager, StorageClassManager, SCCManager
  - Each manager implements ResourceManager interface
  - Builders separated from managers for pure, stateless resource construction

- **Status Updates**: Centralized status management in reconciler
  - Consistent condition handling
  - Proper phase transitions (Progressing → Ready/Failed)
  - ObservedGeneration always updated

- **Logging**: Structured logging with consistent key-value pairs
  - Contextual logs with resource name, namespace, CR name
  - Info/Error levels appropriately used
  - V(1) debug logs for non-critical information

#### Platform Support

- **OpenShift Badge**: Updated from 4.16-4.18 to 4.19+ in README
- **Kubernetes Support**: Declared 1.30+ compatibility
- **SCC Handling**: Improved detection and graceful handling on vanilla Kubernetes

### Improved

- **Test Coverage**: Achieved 70-80% coverage across pkg/ modules
  - pkg/validation: 97.9%
  - pkg/defaults: 100%
  - pkg/reconciler: 75.4%
  - pkg/resources: 71.7%
  - pkg/builder: 57.5%

- **Code Maintainability**:
  - Clear separation of concerns (validation, defaults, building, management)
  - Dependency injection for easy testing
  - Interface-driven design for mockability
  - Reduced cyclomatic complexity

- **Error Messages**: More actionable validation error messages with field names and specific requirements

### Fixed

- **Storage Option Validation**: Properly rejects CRs with multiple storage options or no storage option
- **Default Application**: Idempotent defaults application (calling ApplyDefaults multiple times has no additional effect)
- **Status Conditions**: Correct LastTransitionTime updates only when status actually changes
- **Owner References**: Properly set controller owner reference on all namespaced resources for garbage collection

### Developer Notes

#### Migration from 0.0.7 to 0.0.8

The refactoring maintains backward compatibility with existing NFSProvisioner CRs. No CR changes are required.

**Testing the Refactored Code**:

```bash
# Run all unit tests with coverage
make test

# Generate coverage report
make coverage-report

# Run linter
make lint

# Build operator binary
make build
```

**Architecture Changes**:

Controllers now delegate reconciliation to `pkg/reconciler`, which coordinates:
1. Validation (pkg/validation)
2. Defaults (pkg/defaults)
3. Resource creation (pkg/resources using pkg/builder)
4. Status updates with proper conditions

This separation enables:
- Comprehensive unit testing without controller-runtime
- Clear code organization by responsibility
- Easy extension for new resource types
- Better error handling and status management

## [0.0.7] - Previous Release

### Summary

- Initial operator implementation
- Basic NFS server deployment with hostPath/PVC storage options
- StorageClass provisioning
- OpenShift 4.16-4.18 support

---

## Upgrade Path

### From 0.0.7 to 0.0.8

No manual intervention required. Existing NFSProvisioner CRs continue to work.

**Recommended**: After upgrading, verify operator status and NFSProvisioner reconciliation:

```bash
# Check operator pod is running
kubectl get pods -n openshift-operators | grep nfs-provisioner-operator

# Verify NFSProvisioner status
kubectl get nfsprovisioner -A

# Check all resources are healthy
kubectl get deployment,service,pvc,storageclass | grep nfs
```

### Testing Upgrade

Before upgrading in production:

1. Deploy operator v0.0.7 in test environment
2. Create NFSProvisioner CR and verify functionality
3. Upgrade to v0.0.8
4. Verify existing CR continues to function
5. Create new CR and test dynamic provisioning

---

## Support

For issues, questions, or feature requests:
- GitHub Issues: https://github.com/jooho/nfs-provisioner-operator/issues
- Email: ljhiyh@gmail.com

## License

Apache License 2.0

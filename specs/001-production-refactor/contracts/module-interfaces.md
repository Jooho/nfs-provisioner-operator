# Module Interface Contracts

**Feature**: Production Quality Codebase Refactoring
**Date**: 2026-02-13

## Overview

This document defines the interface contracts between internal Go modules in the refactored NFS Provisioner Operator codebase. These contracts ensure loose coupling, testability, and clear separation of concerns.

**Note**: These are **internal** Go package interfaces, not external APIs. The external API (NFSProvisioner CRD) remains unchanged and is documented in [data-model.md](../data-model.md).

## Module Architecture

```text
┌──────────────────────────────────────────────────────────┐
│           controllers/nfsprovisioner_controller.go       │
│                    (Thin controller layer)               │
└────────────────────────┬─────────────────────────────────┘
                         │
                         │ delegates to
                         ▼
           ┌─────────────────────────────┐
           │    pkg/reconciler           │
           │  (Orchestrates reconciliation)│
           └──┬────────────┬─────────┬───┘
              │            │         │
        calls │      calls │   calls │
              ▼            ▼         ▼
      ┌─────────────┐ ┌────────────────┐ ┌──────────────┐
      │pkg/validation│ │pkg/resources   │ │ pkg/defaults │
      │             │ │  (ResourceManager) │ │              │
      └─────────────┘ └────────────────┘ └──────────────┘
                              │
                        uses  │
                              ▼
                      ┌──────────────┐
                      │ pkg/builder  │
                      └──────────────┘
```

## Contract 1: Reconciler Interface

**Package**: `pkg/reconciler`
**Purpose**: Orchestrate the reconciliation logic independently of controller-runtime plumbing

### Interface Definition

```go
package reconciler

import (
    "context"
    cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
    ctrl "sigs.k8s.io/controller-runtime"
)

// Reconciler encapsulates the business logic for reconciling NFSProvisioner resources.
// This interface allows the controller-runtime Controller to delegate reconciliation
// to a testable, framework-independent component.
type Reconciler interface {
    // Reconcile implements the reconciliation logic for a single NFSProvisioner resource.
    // It returns a Result indicating requeue behavior and an error if reconciliation fails.
    //
    // Parameters:
    //   - ctx: Context for cancellation and timeout
    //   - nfsProvisioner: The NFSProvisioner resource to reconcile
    //
    // Returns:
    //   - ctrl.Result: Requeue behavior (immediate, after delay, or no requeue)
    //   - error: Non-nil if reconciliation failed (triggers exponential backoff retry)
    Reconcile(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner) (ctrl.Result, error)
}
```

### Implementation Contract

**File**: `pkg/reconciler/reconciler.go`

**Responsibilities**:
1. Validate the NFSProvisioner resource using `pkg/validation.Validator`
2. Apply default values using `pkg/defaults.ApplyDefaults()`
3. Create/update Kubernetes resources using `pkg/resources.ResourceManager`
4. Update NFSProvisioner status with conditions and phase
5. Return appropriate requeue behavior based on outcome

**Error Handling**:
- Validation errors: Update status with error message, return `ctrl.Result{Requeue: false}, nil` (don't retry invalid CR)
- Transient errors (e.g., API unavailable): Return `ctrl.Result{}, err` (controller-runtime retries with exponential backoff)
- Permanent errors (e.g., missing required resource): Update status, return `ctrl.Result{RequeueAfter: 5 * time.Minute}, nil` (retry with delay)

**Dependencies**:
- `pkg/validation.Validator` (injected via constructor)
- `pkg/resources.ResourceManager` (injected via constructor)
- `pkg/defaults` (static functions, no injection)
- `sigs.k8s.io/controller-runtime/pkg/client` (Kubernetes API client, injected)

### Testing Strategy

**Unit Tests** (`pkg/reconciler/reconciler_test.go`):
- Mock `Validator` and `ResourceManager` interfaces
- Test reconciliation logic for all scenarios:
  - Happy path: Valid CR → resources created → status updated
  - Validation failure: Invalid CR → status error → no requeue
  - Resource creation failure: API error → error returned → retry
  - Update scenario: CR changed → resources updated → status updated

**Test Example**:
```go
var _ = Describe("Reconciler", func() {
    var (
        reconciler Reconciler
        mockValidator *MockValidator
        mockResourceManager *MockResourceManager
        ctx context.Context
    )

    BeforeEach(func(ctx SpecContext) {
        mockValidator = &MockValidator{}
        mockResourceManager = &MockResourceManager{}
        reconciler = NewReconciler(mockValidator, mockResourceManager, k8sClient, logger)
    }, SpecTimeout(30*time.Second))

    It("should successfully reconcile a valid NFSProvisioner", func(ctx SpecContext) {
        nfs := &cachev1alpha1.NFSProvisioner{...}
        mockValidator.EXPECT().Validate(nfs).Return(nil)
        mockResourceManager.EXPECT().EnsureResources(ctx, nfs).Return(nil)

        result, err := reconciler.Reconcile(ctx, nfs)

        Expect(err).ToNot(HaveOccurred())
        Expect(result.Requeue).To(BeFalse())
    }, SpecTimeout(10*time.Second))
})
```

## Contract 2: Validator Interface

**Package**: `pkg/validation`
**Purpose**: Validate NFSProvisioner custom resources

### Interface Definition

```go
package validation

import (
    cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
)

// Validator validates NFSProvisioner resources according to business rules.
// Validation failures should result in user-friendly error messages that guide
// users to fix the issue.
type Validator interface {
    // Validate checks if the NFSProvisioner resource is valid.
    //
    // Parameters:
    //   - nfs: The NFSProvisioner resource to validate
    //
    // Returns:
    //   - error: Non-nil if validation fails, with a descriptive message
    Validate(nfs *cachev1alpha1.NFSProvisioner) error
}
```

### Implementation Contract

**File**: `pkg/validation/validator.go`

**Validation Rules** (see [data-model.md](../data-model.md#data-validation-rules)):
1. **Storage Mutual Exclusivity**: Exactly one of `hostPathDir`, `pvc`, or `scForNFSPvc` must be set
2. **Field Formats**:
   - `storageSize`: Must be a valid Kubernetes quantity (e.g., "10Gi")
   - `scForNFS`, `scForNFSPvc`: Must be valid DNS subdomain names (RFC 1123)
   - `image`: Must be a valid container image reference
3. **Reference Existence** (optional, warnings only):
   - If `pvc` is set, the PVC should exist in the same namespace
   - If `scForNFSPvc` is set, the StorageClass should exist

**Error Messages**:
- Must be **actionable**: Tell the user what's wrong and how to fix it
- Must be **specific**: Include field names and invalid values

**Example Error Messages**:
- ❌ Bad: `"invalid configuration"`
- ✅ Good: `"exactly one of spec.hostPathDir, spec.pvc, or spec.scForNFSPvc must be set; currently none are set"`
- ✅ Good: `"spec.storageSize '10G' is invalid; must be a valid Kubernetes quantity like '10Gi' or '1Ti'"`

**Dependencies**: None (pure validation logic)

### Testing Strategy

**Unit Tests** (`pkg/validation/validator_test.go`):
- Test valid configurations (all three storage options)
- Test invalid configurations (none set, multiple set, invalid formats)
- Verify error messages are descriptive

**Test Example**:
```go
var _ = Describe("Validator", func() {
    var validator Validator

    BeforeEach(func() {
        validator = NewValidator()
    })

    It("should accept hostPathDir storage option", func() {
        nfs := &cachev1alpha1.NFSProvisioner{
            Spec: cachev1alpha1.NFSProvisionerSpec{
                HostPathDir: "/mnt/nfs",
            },
        }
        Expect(validator.Validate(nfs)).To(Succeed())
    })

    It("should reject multiple storage options", func() {
        nfs := &cachev1alpha1.NFSProvisioner{
            Spec: cachev1alpha1.NFSProvisionerSpec{
                HostPathDir: "/mnt/nfs",
                Pvc: "my-pvc",
            },
        }
        err := validator.Validate(nfs)
        Expect(err).To(HaveOccurred())
        Expect(err.Error()).To(ContainSubstring("mutually exclusive"))
    })
})
```

## Contract 3: ResourceManager Interface

**Package**: `pkg/resources`
**Purpose**: Manage Kubernetes resources owned by NFSProvisioner CRs

### Interface Definition

```go
package resources

import (
    "context"
    cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
)

// ResourceManager creates and updates Kubernetes resources for an NFSProvisioner.
// It ensures that all required resources (Deployment, Service, RBAC, etc.) exist
// and match the desired state defined in the NFSProvisioner spec.
type ResourceManager interface {
    // EnsureResources creates or updates all Kubernetes resources for the NFSProvisioner.
    // This method is idempotent and can be called repeatedly.
    //
    // Parameters:
    //   - ctx: Context for cancellation and timeout
    //   - nfs: The NFSProvisioner resource defining the desired state
    //
    // Returns:
    //   - error: Non-nil if resource creation/update fails
    EnsureResources(ctx context.Context, nfs *cachev1alpha1.NFSProvisioner) error

    // GetManagedPods returns the names of pods managed by this NFSProvisioner.
    // This is used to populate the status.nodes field.
    //
    // Parameters:
    //   - ctx: Context for cancellation and timeout
    //   - nfs: The NFSProvisioner resource
    //
    // Returns:
    //   - []string: Pod names
    //   - error: Non-nil if pod listing fails
    GetManagedPods(ctx context.Context, nfs *cachev1alpha1.NFSProvisioner) ([]string, error)
}
```

### Implementation Contract

**File**: `pkg/resources/manager.go`

**Current Implementation**: The existing `controllers/resources/manager.go` already implements a similar pattern via `ResourceManagerSet`. The refactored version will:
1. Move to `pkg/resources/manager.go`
2. Implement the above interface explicitly
3. Continue using the modular per-resource approach (deployment.go, service.go, etc.)

**Resource Creation Order**:
1. ServiceAccount (required for pod identity)
2. RBAC (ClusterRole, ClusterRoleBinding, Role, RoleBinding)
3. SCC (OpenShift only, conditional on SCC CRD detection)
4. PVC (if `scForNFSPvc` is set)
5. Deployment (depends on ServiceAccount and PVC)
6. Service (exposes Deployment)
7. StorageClass (for end users to request NFS volumes)

**Idempotency**:
- Use `controllerutil.CreateOrUpdate()` for all resources
- Compare desired state vs. actual state
- Only update if differences detected

**Error Handling**:
- API errors: Return error (caller retries)
- Resource conflicts: Attempt to reconcile ownership, else return error
- Missing dependencies (e.g., PVC in spec.pvc doesn't exist): Return descriptive error

**Dependencies**:
- `pkg/builder` (constructs resource objects)
- `sigs.k8s.io/controller-runtime/pkg/client` (Kubernetes API client)
- `sigs.k8s.io/controller-runtime/pkg/controller/controllerutil` (CreateOrUpdate, SetControllerReference)

### Testing Strategy

**Unit Tests** (`pkg/resources/manager_test.go`):
- Mock Kubernetes client
- Test resource creation for each resource type
- Test idempotency (calling EnsureResources twice produces same result)
- Test error scenarios (API failures, missing dependencies)

**Integration Tests** (`test/integration/reconcile_integration_test.go`):
- Use envtest with real Kubernetes API server
- Create NFSProvisioner CR
- Verify all resources are created with correct ownership
- Update CR and verify resources are updated
- Delete CR and verify resources are garbage collected

## Contract 4: Builder Interface

**Package**: `pkg/builder`
**Purpose**: Construct Kubernetes resource objects from NFSProvisioner spec

### Interface Definition

```go
package builder

import (
    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    rbacv1 "k8s.io/api/rbac/v1"
    storagev1 "k8s.io/api/storage/v1"
    cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
)

// Builder constructs Kubernetes resource objects for NFSProvisioner resources.
// Builders are stateless and pure functions that transform NFSProvisioner spec
// into Kubernetes resource specs.
type Builder interface {
    // BuildDeployment constructs a Deployment resource for the NFS server.
    BuildDeployment(nfs *cachev1alpha1.NFSProvisioner) *appsv1.Deployment

    // BuildService constructs a Service resource exposing the NFS server.
    BuildService(nfs *cachev1alpha1.NFSProvisioner) *corev1.Service

    // BuildServiceAccount constructs a ServiceAccount for the NFS provisioner pod.
    BuildServiceAccount(nfs *cachev1alpha1.NFSProvisioner) *corev1.ServiceAccount

    // BuildPVC constructs a PVC for NFS server storage (if spec.scForNFSPvc is set).
    // Returns nil if PVC should not be created.
    BuildPVC(nfs *cachev1alpha1.NFSProvisioner) *corev1.PersistentVolumeClaim

    // BuildStorageClass constructs a StorageClass for end users.
    BuildStorageClass(nfs *cachev1alpha1.NFSProvisioner) *storagev1.StorageClass

    // BuildClusterRole, BuildClusterRoleBinding, BuildRole, BuildRoleBinding
    // (Additional methods for RBAC resources - omitted for brevity)
}
```

**Alternative Design**: Instead of an interface, these could be **stateless functions** in the `pkg/builder` package (e.g., `builder.Deployment(nfs)`, `builder.Service(nfs)`). This may be simpler and more Go-idiomatic.

### Implementation Contract

**Files**: `pkg/builder/builder.go` (or separate files per resource type)

**Current Implementation**: The existing `builder/` package already provides `NewNamespaceBuilder()` and `NewObjectBuilder()`. The refactored version will:
1. Move to `pkg/builder/`
2. Add resource-specific builder functions
3. Centralize naming conventions (e.g., resource name = "nfs-provisioner-" + CR name)

**Responsibilities**:
- Construct resource objects with correct metadata (name, namespace, labels, annotations)
- Set owner references (for garbage collection)
- Apply default values from `pkg/defaults`
- Handle conditional logic (e.g., PVC volume mount vs. hostPath)

**Dependencies**:
- `pkg/defaults` (for default values)
- Kubernetes API types (appsv1, corev1, etc.)

### Testing Strategy

**Unit Tests** (`pkg/builder/builder_test.go`):
- Test each builder function with various NFSProvisioner configurations
- Verify generated resources have correct fields
- Test edge cases (nil values, empty strings, etc.)

**Test Example**:
```go
var _ = Describe("Builder", func() {
    It("should build Deployment with correct image", func() {
        nfs := &cachev1alpha1.NFSProvisioner{
            ObjectMeta: metav1.ObjectMeta{Name: "test-nfs", Namespace: "default"},
            Spec: cachev1alpha1.NFSProvisionerSpec{
                NFSImageConfiguration: &cachev1alpha1.ImageConfiguration{
                    Image: pointer.String("custom-image:v1"),
                },
            },
        }

        deployment := BuildDeployment(nfs)

        Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("custom-image:v1"))
    })
})
```

## Contract 5: Defaults Interface (Optional)

**Package**: `pkg/defaults`
**Purpose**: Apply default values to NFSProvisioner spec

### Function Signatures

```go
package defaults

import cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"

// ApplyDefaults mutates the NFSProvisioner resource to apply default values
// for any unset optional fields.
//
// Defaults:
//   - spec.storageSize: "10Gi"
//   - spec.scForNFS: "nfs"
//   - spec.nfsImageConfiguration.image: (default NFS provisioner image)
//   - spec.nfsImageConfiguration.imagePullPolicy: "IfNotPresent"
func ApplyDefaults(nfs *cachev1alpha1.NFSProvisioner)
```

**Note**: This is a **stateless function**, not an interface. Defaults are simple enough that dependency injection is unnecessary.

### Testing Strategy

```go
var _ = Describe("Defaults", func() {
    It("should apply default storageSize", func() {
        nfs := &cachev1alpha1.NFSProvisioner{Spec: cachev1alpha1.NFSProvisionerSpec{}}
        ApplyDefaults(nfs)
        Expect(nfs.Spec.StorageSize).To(Equal("10Gi"))
    })

    It("should not override existing storageSize", func() {
        nfs := &cachev1alpha1.NFSProvisioner{
            Spec: cachev1alpha1.NFSProvisionerSpec{StorageSize: "20Gi"},
        }
        ApplyDefaults(nfs)
        Expect(nfs.Spec.StorageSize).To(Equal("20Gi"))
    })
})
```

## Integration Contract: Controller → Reconciler

**File**: `controllers/nfsprovisioner_controller.go`

The controller-runtime `Controller` delegates to `pkg/reconciler.Reconciler`:

```go
func (r *NFSProvisionerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := r.Log.WithValues("nfsprovisioner", req.NamespacedName)

    // Fetch the NFSProvisioner instance
    nfsProvisioner := &cachev1alpha1.NFSProvisioner{}
    if err := r.Get(ctx, req.NamespacedName, nfsProvisioner); err != nil {
        if errors.IsNotFound(err) {
            log.Info("NFSProvisioner resource not found, ignoring")
            return ctrl.Result{}, nil
        }
        log.Error(err, "Failed to get NFSProvisioner")
        return ctrl.Result{}, err
    }

    // Delegate to business logic reconciler
    return r.Reconciler.Reconcile(ctx, nfsProvisioner)
}
```

**Benefits**:
- Controller is thin wrapper around framework
- Business logic is testable without framework mocks
- Clear separation of concerns

## Testing Contract Summary

| Module | Test Type | Tool | Coverage Target | Key Scenarios |
|--------|-----------|------|-----------------|---------------|
| pkg/reconciler | Unit | Ginkgo + Gomega + Mocks | 80%+ | Valid CR, invalid CR, API errors, updates |
| pkg/validation | Unit | Ginkgo + Gomega | 90%+ | All validation rules, error messages |
| pkg/resources | Unit | Ginkgo + Gomega + Mocks | 80%+ | Resource creation, idempotency, errors |
| pkg/builder | Unit | Ginkgo + Gomega | 90%+ | Resource construction, defaults, conditionals |
| pkg/defaults | Unit | Ginkgo + Gomega | 100% | Default application, no overrides |
| controllers | Integration | Ginkgo + envtest | 70%+ | Full reconciliation loop, CRD validation |
| - | E2E | Ginkgo + Kind | 60%+ | Operator deployment, CR lifecycle, NFS provisioning |

## References

- **Current Code**:
  - `controllers/nfsprovisioner_controller.go` (reconciliation logic)
  - `controllers/resources/manager.go` (resource management)
  - `builder/` (object construction)
- **Kubernetes Patterns**:
  - [controller-runtime patterns](https://github.com/kubernetes-sigs/controller-runtime/blob/main/designs/use-cases.md)
  - [operator best practices](https://sdk.operatorframework.io/docs/best-practices/)
- **Refactoring Research**: `../research.md`

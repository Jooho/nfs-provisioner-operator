# NFS Provisioner Operator Architecture

This document provides a detailed overview of the NFS Provisioner Operator's architecture, focusing on the production-ready refactored codebase.

## Table of Contents

- [Design Principles](#design-principles)
- [Package Structure](#package-structure)
- [Module Responsibilities](#module-responsibilities)
- [Controller Flow](#controller-flow)
- [Resource Management](#resource-management)
- [Error Handling Strategy](#error-handling-strategy)
- [Testing Architecture](#testing-architecture)
- [Platform Compatibility](#platform-compatibility)

## Design Principles

The operator follows these core principles:

1. **Separation of Concerns**: Each package has a single, well-defined responsibility
2. **Dependency Injection**: Components receive dependencies explicitly for easy testing
3. **Stateless Builders**: Resource construction is pure, side-effect-free
4. **Interface-Driven**: Managers implement interfaces for mockability
5. **Fail-Fast Validation**: Invalid CRs are rejected early with clear error messages
6. **Structured Logging**: Consistent logging with contextual key-value pairs
7. **Test-Driven**: 80%+ code coverage with unit and integration tests

## Package Structure

```
pkg/
├── validation/       # CR validation with field format checking
│   ├── validator.go       # Validator interface and implementation
│   └── validator_test.go  # Comprehensive validation tests (97.9% coverage)
├── defaults/         # Default value application
│   ├── defaults.go        # ApplyDefaults function and constants
│   └── defaults_test.go   # Default application tests (100% coverage)
├── builder/          # Stateless resource builders
│   ├── builder.go         # BuildDeployment, BuildService, BuildPVC, etc.
│   ├── namespace_builder.go   # Namespace builder with functional options
│   ├── object_builder.go      # ObjectMeta builder with functional options
│   └── builder_test.go        # Builder tests (57.5% coverage)
├── resources/        # Resource managers using builder pattern
│   ├── manager.go         # ResourceManager interface + base implementation
│   ├── deployment.go      # DeploymentManager
│   ├── service.go         # ServiceManager
│   ├── pvc.go             # PVCManager
│   ├── rbac.go            # RBACManager (ClusterRole, ClusterRoleBinding, Role, RoleBinding)
│   ├── scc.go             # SCCManager (OpenShift SecurityContextConstraints)
│   ├── storageclass.go    # StorageClassManager
│   ├── serviceaccount.go  # ServiceAccountManager
│   └── *_test.go          # Manager tests (71.7% coverage)
└── reconciler/       # Reconciliation orchestration
    ├── reconciler.go      # Reconciler interface with dependency injection
    ├── status.go          # Status and condition management
    ├── errors.go          # Error classification logic
    └── reconciler_test.go # Reconciler tests (75.4% coverage)
```

### Non-pkg Directories

```
controllers/
└── nfsprovisioner_controller.go  # Controller entrypoint (delegates to pkg/reconciler)

api/v1alpha1/
├── nfsprovisioner_types.go       # NFSProvisioner CRD definition
├── nfsprovisioner_webhook.go     # Webhook validation (optional)
└── zz_generated.deepcopy.go      # Auto-generated DeepCopy methods

test/
├── integration/                   # Integration tests with envtest
│   └── scc_detection_test.go
└── e2e/                           # End-to-end tests (future)

config/
├── crd/                           # CRD manifests
├── rbac/                          # RBAC manifests for operator
├── manager/                       # Operator deployment manifests
└── samples/                       # Sample NFSProvisioner CRs

bundle/                            # OLM bundle for certification
├── manifests/
│   └── nfs-provisioner-operator.clusterserviceversion.yaml
└── metadata/
    └── annotations.yaml
```

## Module Responsibilities

### pkg/validation

**Purpose**: Validate NFSProvisioner custom resources before processing

**Key Features**:
- Storage option mutual exclusivity validation
  - Only one of: `hostPathDir`, `pvc`, `scForNFSPvc` can be set
  - At least one storage option must be configured
- Field format validation:
  - `storageSize`: Must be valid quantity (e.g., "10Gi", "500Mi")
  - `scForNFSProvisioner`: Non-empty string
  - `image`: Valid container image reference
- Actionable error messages with field names

**Interface**:
```go
type Validator interface {
    Validate(nfs *cachev1alpha1.NFSProvisioner) error
}
```

**Usage**:
```go
validator := validation.NewValidator()
if err := validator.Validate(nfsProvisioner); err != nil {
    // Handle validation error
}
```

**Test Coverage**: 97.9%

### pkg/defaults

**Purpose**: Apply default values to optional NFSProvisioner fields

**Default Values**:
- `storageSize`: "10Gi"
- `scForNFSProvisioner`: "nfs"
- `nfsImageConfiguration.image`: k8s.gcr.io/sig-storage/nfs-provisioner:...
- `nfsImageConfiguration.imagePullPolicy`: "IfNotPresent"

**Key Feature**: Idempotent - calling ApplyDefaults multiple times has no additional effect

**Function Signature**:
```go
func ApplyDefaults(nfs *cachev1alpha1.NFSProvisioner)
```

**Test Coverage**: 100%

### pkg/builder

**Purpose**: Construct Kubernetes resources as stateless, pure functions

**Key Features**:
- **Stateless**: No side effects, no API calls
- **Pure Functions**: Same input → same output
- **Testable**: Easy to unit test without mocks
- **Reusable**: Shared across managers

**Resource Builders**:
- `BuildDeployment(nfs)` → Deployment for NFS server
- `BuildService(nfs)` → Service exposing NFS ports
- `BuildPVC(nfs)` → PersistentVolumeClaim for NFS storage
- `BuildServiceAccount(nfs)` → ServiceAccount for NFS server
- `BuildClusterRole()` → ClusterRole for provisioner permissions
- `BuildClusterRoleBinding(namespace)` → ClusterRoleBinding
- `BuildRole()` → Role for leader election
- `BuildRoleBinding(namespace)` → RoleBinding
- `BuildStorageClass(nfs)` → StorageClass for dynamic provisioning
- `BuildSCC(nfs)` → SecurityContextConstraints (OpenShift only)

**Functional Options Pattern**:
```go
// ObjectMeta builder
obj := BuildObjectMeta(name,
    WithNamespace(namespace),
    WithLabels(labels),
    WithOwnerReference(owner),
)

// Namespace builder
ns := BuildNamespace(name,
    WithLabels(labels),
)
```

**Test Coverage**: 57.5%

### pkg/resources

**Purpose**: Manage Kubernetes resource lifecycle

**ResourceManager Interface**:
```go
type ResourceManager interface {
    GetResourceName() string
    EnsureResource(ctx context.Context, nfs *cachev1alpha1.NFSProvisioner) error
}
```

**BaseResourceManager**:
Provides common functionality to all managers:
- Kubernetes client
- Scheme for owner references
- Structured logger
- EnsureNamespacedResource helper
- EnsureClusterResource helper

**Resource Managers**:

1. **ServiceAccountManager**: Manages ServiceAccount for NFS server pod
2. **RBACManager**: Manages ClusterRole, ClusterRoleBinding, Role, RoleBinding
3. **PVCManager**: Manages PersistentVolumeClaim (when using PVC storage option)
4. **ServiceManager**: Manages Service exposing NFS ports
5. **DeploymentManager**: Manages Deployment for NFS server pod
6. **StorageClassManager**: Manages StorageClass for dynamic provisioning
7. **SCCManager**: Manages SecurityContextConstraints (OpenShift only)

**Manager Pattern**:
```go
type DeploymentManager struct {
    BaseResourceManager
}

func (m *DeploymentManager) EnsureResource(ctx context.Context, nfs *cachev1alpha1.NFSProvisioner) error {
    deployment := builder.BuildDeployment(nfs)
    return m.EnsureNamespacedResource(ctx, nfs, deployment)
}
```

**Test Coverage**: 71.7%

### pkg/reconciler

**Purpose**: Orchestrate reconciliation logic and manage CR status

**Reconciler Interface**:
```go
type Reconciler interface {
    Reconcile(ctx context.Context, nfs *cachev1alpha1.NFSProvisioner) (ctrl.Result, error)
}
```

**Reconciliation Flow**:

1. **Validation**: Validate CR using `pkg/validation`
   - On error: Set `Degraded=True`, `Ready=False`, `Phase=Failed`
   - Return: No requeue (user must fix CR)

2. **Defaults**: Apply defaults using `pkg/defaults`

3. **Resource Creation**: Call EnsureResource for each manager
   - On transient error: Set `Degraded=True`, `Progressing=True`, return error (controller-runtime requeues)
   - On permanent error: Set `Degraded=True`, `Ready=False`, `Phase=Failed`, requeue after 5min

4. **Success**: Set `Ready=True`, `Degraded=False`, `Progressing=False`, `Phase=Ready`

**Error Classification**:

```go
type errorType int

const (
    errorTypeTransient  errorType = iota  // Network errors, conflicts, timeouts
    errorTypePermanent                     // Forbidden, invalid, not found
    errorTypeUnknown                       // Default to transient for safety
)

func classifyError(err error) errorType {
    if apierrors.IsConflict(err) || apierrors.IsTimeout(err) ||
       apierrors.IsServerTimeout(err) || apierrors.IsServiceUnavailable(err) {
        return errorTypeTransient
    }
    if apierrors.IsForbidden(err) || apierrors.IsInvalid(err) ||
       apierrors.IsMethodNotSupported(err) {
        return errorTypePermanent
    }
    return errorTypeUnknown  // Treat unknown as transient for safety
}
```

**Status Management**:

Uses Kubernetes Conditions pattern:

- **Ready**: Overall readiness of NFSProvisioner
  - `True`: All resources created, reconciliation succeeded
  - `False`: Validation failed, permanent error, or not yet reconciled

- **Progressing**: Whether reconciliation is in progress
  - `True`: Resources being created, transient error occurred
  - `False`: Reconciliation succeeded or permanently failed

- **Degraded**: Whether NFSProvisioner is degraded
  - `True`: Validation failed, any error occurred
  - `False`: Reconciliation succeeded

- **Available**: Whether NFS server is running (future enhancement)

**Phase Field**:
- `Ready`: All resources created and healthy
- `Progressing`: Reconciliation in progress
- `Failed`: Validation or permanent error

**ObservedGeneration**:
Tracks which CR generation was last reconciled (for detecting spec changes)

**Test Coverage**: 75.4%

## Controller Flow

The controller ([controllers/nfsprovisioner_controller.go](../controllers/nfsprovisioner_controller.go)) is a thin wrapper:

```go
func (r *NFSProvisionerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Fetch NFSProvisioner CR
    nfs := &cachev1alpha1.NFSProvisioner{}
    if err := r.Get(ctx, req.NamespacedName, nfs); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // 2. Handle finalizer logic for deletion
    if nfs.GetDeletionTimestamp() != nil {
        return r.handleFinalization(ctx, nfs)
    }

    // 3. Delegate to pkg/reconciler
    return r.reconciler.Reconcile(ctx, nfs)
}
```

**Separation Benefits**:
- Controller focuses on: fetching CR, finalizers, controller-runtime integration
- Reconciler focuses on: business logic, resource management, status updates
- Easy to unit test reconciler without controller-runtime

## Resource Management

### Resource Creation Order

Resources are created in this order for dependency resolution:

1. **ServiceAccount**: Required by Deployment
2. **RBAC**: ClusterRole, ClusterRoleBinding, Role, RoleBinding
3. **PVC**: Required by Deployment (if using PVC storage)
4. **Service**: Exposes NFS server
5. **Deployment**: NFS server pod (depends on SA, PVC, Service)
6. **StorageClass**: For dynamic PV provisioning
7. **SCC**: OpenShift SecurityContextConstraints (if on OpenShift)

### Idempotency

All resource managers are idempotent:
- Creating existing resource: No-op
- Updating existing resource: Only if needed (future enhancement)
- Deleting non-existent resource: No-op

Achieved via `client.IgnoreAlreadyExists(err)`

### Owner References

All namespaced resources set controller owner reference to NFSProvisioner CR:
```go
ownerRef := metav1.OwnerReference{
    APIVersion: nfs.APIVersion,
    Kind:       nfs.Kind,
    Name:       nfs.Name,
    UID:        nfs.UID,
    Controller: ptr.To(true),
}
```

**Benefits**:
- Garbage collection: Deleting NFSProvisioner deletes all owned resources
- RBAC: Operator only needs delete permission on NFSProvisioner, not each resource
- Clear ownership in `kubectl get ... -o yaml`

**Cluster-scoped resources** (ClusterRole, ClusterRoleBinding, StorageClass, SCC):
- Cannot set owner reference (different namespace scope)
- Must be manually cleaned up or managed separately
- Reused across multiple NFSProvisioner instances

## Error Handling Strategy

### Validation Errors

**Classification**: Permanent
**Handling**: No requeue, user must fix CR
**Status**: `Degraded=True`, `Ready=False`, `Phase=Failed`

**Example**:
```
Error: validation failed: exactly one storage option must be set (hostPathDir, pvc, or scForNFSPvc)
```

### Resource Creation Errors

**Transient Errors** (network, conflict, timeout):
- **Handling**: Return error, controller-runtime requeues with exponential backoff
- **Status**: `Progressing=True`, `Degraded=True`, `Phase=Progressing`
- **Examples**: `IsConflict`, `IsTimeout`, `IsServiceUnavailable`

**Permanent Errors** (forbidden, invalid):
- **Handling**: Requeue after 5 minutes (maybe permissions will be fixed)
- **Status**: `Ready=False`, `Degraded=True`, `Phase=Failed`
- **Examples**: `IsForbidden`, `IsInvalid`, `IsMethodNotSupported`

**Unknown Errors**:
- **Handling**: Treat as transient (fail-safe)
- **Status**: Same as transient

### Reconciliation Success

**Status**: `Ready=True`, `Progressing=False`, `Degraded=False`, `Phase=Ready`

## Testing Architecture

### Unit Tests

**Location**: `*_test.go` files alongside implementation

**Framework**: Ginkgo v2 + Gomega

**Pattern**: SpecContext for interruptible tests

```go
var _ = Describe("Validator", func() {
    It("should reject multiple storage options", func(ctx SpecContext) {
        nfs := &cachev1alpha1.NFSProvisioner{
            Spec: cachev1alpha1.NFSProvisionerSpec{
                HostPathDir: "/mnt/nfs",
                SCForNFSPvc: "local-storage",  // Both set - invalid
            },
        }
        err := validator.Validate(nfs)
        Expect(err).To(HaveOccurred())
        Expect(err.Error()).To(ContainSubstring("exactly one storage option"))
    })
})
```

**Fake Client**: controller-runtime's `fake.NewClientBuilder()`
- In-memory Kubernetes API server
- Supports status subresource
- Fast, no external dependencies

**Coverage Target**: 80%

**Current Coverage**:
- pkg/validation: 97.9%
- pkg/defaults: 100%
- pkg/reconciler: 75.4%
- pkg/resources: 71.7%
- pkg/builder: 57.5%

### Integration Tests

**Location**: `test/integration/`

**Framework**: Ginkgo v2 + controller-runtime envtest

**Purpose**: Test SCC detection logic with actual CRD presence/absence

**Example**:
```go
Context("on vanilla Kubernetes (no SCC CRD)", func() {
    It("should gracefully skip SCC creation", func(ctx SpecContext) {
        // envtest does not have SCC CRD by default
        err := sccManager.EnsureResource(ctx, nfsProvisioner)
        Expect(err).NotTo(HaveOccurred())

        // Verify SCC was NOT created
        scc := &securityv1.SecurityContextConstraints{}
        err = k8sClient.Get(ctx, types.NamespacedName{Name: "nfs-provisioner"}, scc)
        Expect(err).To(HaveOccurred())  // Should not exist
    })
})
```

**Envtest**: Lightweight Kubernetes API server for integration testing
- Real etcd + kube-apiserver
- No kubelet, scheduler, or controller-manager
- Fast startup (~1s)

### End-to-End Tests

**Location**: `test/e2e/` (future)

**Purpose**: Test operator on real cluster
- Deploy operator via OLM
- Create NFSProvisioner CR
- Verify PVC dynamic provisioning works
- Test upgrade scenarios

## Platform Compatibility

### Kubernetes (Vanilla)

**Supported Versions**: 1.30+

**Features**:
- Deployment, Service, ServiceAccount, PVC, StorageClass
- RBAC (ClusterRole, ClusterRoleBinding, Role, RoleBinding)

**Not Used**:
- SecurityContextConstraints (OpenShift-only)

### OpenShift

**Supported Versions**: 4.19+

**Additional Features**:
- SecurityContextConstraints for NFS server pod permissions

**SCC Detection**:
```go
func (m *SCCManager) isSCCCRDAvailable(ctx context.Context) bool {
    crd := &apiextensionsv1.CustomResourceDefinition{}
    err := m.Client.Get(ctx, types.NamespacedName{
        Name: "securitycontextconstraints.security.openshift.io",
    }, crd)
    return err == nil
}
```

**Graceful Degradation**:
- On Kubernetes: SCC creation skipped (logged at V(1) level)
- On OpenShift: SCC created and service account added to SCC users

**OLM Bundle**:
- Declares `com.redhat.openshift.versions: v4.19+`
- OperatorHub shows operator only on OpenShift 4.19+
- Certification-ready metadata in CSV

## Future Enhancements

### Operator Level

- [ ] Webhooks for defaulting and validation (replace pkg/validation)
- [ ] Metrics endpoint for Prometheus
- [ ] Health/readiness probes for operator pod
- [ ] Leader election for HA deployment

### Reconciler Level

- [ ] Status condition: `Available` (check NFS server pod health)
- [ ] Reconcile existing resources (detect drift, update if needed)
- [ ] Resource deletion on CR deletion (finalizers for cluster resources)
- [ ] Handle PVC resize (if storageSize changes)

### Testing

- [ ] E2E tests on real clusters (Kubernetes + OpenShift)
- [ ] Performance tests (reconciliation latency)
- [ ] Chaos testing (pod deletion, network partition)

### Platform Support

- [ ] Helm chart for non-OLM deployment
- [ ] Kustomize overlays for different environments
- [ ] Multi-arch images (amd64, arm64)

## References

- [Kubernetes Operator Pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Operator SDK](https://sdk.operatorframework.io/)
- [OLM Bundle Format](https://olm.operatorframework.io/docs/tasks/creating-a-bundle/)
- [Ginkgo Testing Framework](https://onsi.github.io/ginkgo/)
- [OpenShift Operator Best Practices](https://docs.openshift.com/container-platform/latest/operators/operator_sdk/osdk-about.html)

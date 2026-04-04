# Kubernetes Operator Refactoring Best Practices in Go

This document provides comprehensive best practices for refactoring Kubernetes operators written in Go, based on 2026 industry standards, controller-runtime patterns, and analysis of the nfs-provisioner-operator codebase.

---

## 1. Module Organization

### Recommended Approach

**Standard Project Structure:**
```
operator-project/
├── api/
│   └── v1alpha1/           # API types and CRD definitions
│       ├── groupversion_info.go
│       └── *_types.go
├── controllers/            # Controller implementations
│   ├── defaults/          # Default values and constants
│   └── resources/         # Resource-specific managers
├── internal/              # Private application code
│   ├── reconciler/       # Business logic (sans-IO)
│   ├── builder/          # Resource builders
│   └── validation/       # Validation logic
├── pkg/                   # Public libraries
├── cmd/                   # Main applications
│   └── main.go
├── config/                # Kustomize manifests
└── tests/
    ├── e2e/
    └── integration/
```

**Key Principles:**

1. **Separation of Concerns**: Organize packages by functional responsibility rather than technical layers
   - `api/` contains only type definitions and generated code
   - `controllers/` contains controller-runtime integration code
   - `internal/reconciler/` contains pure business logic (sans-IO)
   - `internal/builder/` contains resource construction logic
   - `internal/validation/` contains validation logic

2. **Resource Manager Pattern**:
   - Current implementation in `controllers/resources/` is well-structured
   - Each resource type has its own manager (DeploymentManager, PVCManager, etc.)
   - Base interface provides common functionality
   - Enables independent testing of each resource type

3. **Modular Design**:
   - Single responsibility per package
   - Clear package boundaries
   - Minimal dependencies between packages

### Rationale

- **Testability**: Separating business logic from controller-runtime allows for pure unit tests without Kubernetes API server
- **Maintainability**: Clear package structure makes it easier to locate and modify code
- **Reusability**: Well-defined interfaces enable code reuse across different controllers
- **Scalability**: Modular structure makes it easier to add new resource types or controllers

### Examples from Kubernetes Ecosystem

- **controller-runtime**: Uses manager/reconciler separation
- **Tekton Operator**: Implements resource manager pattern similar to this codebase
- **Crossplane**: Separates reconciliation logic from API interactions

### Common Pitfalls to Avoid

1. **Monolithic controller files**: Don't put all reconciliation logic in a single file (current `nfsprovisioner_controller.go` could be refactored)
2. **Mixing concerns**: Don't mix business logic with controller-runtime framework code
3. **Tight coupling**: Avoid direct dependencies between resource managers
4. **God packages**: Don't create packages that do everything

### Recommendations for Current Codebase

**Current State Analysis:**
- ✅ Good: Resource manager pattern in `controllers/resources/`
- ✅ Good: Separate `defaults/` package for constants
- ⚠️  Improve: Move validation logic out of controller
- ⚠️  Improve: Extract business logic from `Reconcile()` method

**Suggested Improvements:**
```go
// Move to internal/reconciler/nfs_provisioner.go
type NFSProvisionerReconciler struct {
    ResourceManagers *resources.ResourceManagerSet
    Validator        *validation.NFSProvisionerValidator
}

func (r *NFSProvisionerReconciler) ReconcileNFSProvisioner(
    ctx context.Context,
    nfsProvisioner *cachev1alpha1.NFSProvisioner,
) error {
    // Pure business logic without controller-runtime dependencies
}
```

---

## 2. Controller-Runtime Patterns

### Recommended Approach

**Separation of Reconciliation Logic:**

1. **Thin Controller Layer**: Keep controller minimal, delegating to business logic
```go
// controllers/nfsprovisioner_controller.go
func (r *NFSProvisionerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := r.Log.WithValues("nfsprovisioner", req.NamespacedName)

    // 1. Fetch resource
    nfsProvisioner := &cachev1alpha1.NFSProvisioner{}
    if err := r.Get(ctx, req.NamespacedName, nfsProvisioner); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // 2. Delegate to business logic
    result, err := r.reconciler.Reconcile(ctx, nfsProvisioner)

    // 3. Handle finalizers (framework concern)
    if err := r.handleFinalizers(ctx, nfsProvisioner); err != nil {
        return ctrl.Result{}, err
    }

    return result, err
}
```

2. **Resource Manager Pattern** (already implemented well):
```go
// controllers/resources/common.go
type ResourceManager interface {
    EnsureResource(ctx context.Context, nfsProvisioner *cachev1alpha1.NFSProvisioner) error
    GetResourceName() string
}

type ResourceManagerSet struct {
    SCC            ResourceManager
    PVC            ResourceManager
    ServiceAccount ResourceManager
    RBAC           ResourceManager
    Deployment     ResourceManager
    Service        ResourceManager
    StorageClass   ResourceManager
}
```

3. **Idempotent Reconciliation**:
```go
// Each manager should be idempotent
func (m *DeploymentManager) EnsureResource(ctx context.Context, cr *cachev1alpha1.NFSProvisioner) error {
    desired := m.buildDeployment(cr)

    existing := &appsv1.Deployment{}
    err := m.Client.Get(ctx, client.ObjectKeyFromObject(desired), existing)

    if errors.IsNotFound(err) {
        return m.Client.Create(ctx, desired)
    }
    if err != nil {
        return err
    }

    // Update if needed (check for drift)
    if !equality.Semantic.DeepEqual(existing.Spec, desired.Spec) {
        existing.Spec = desired.Spec
        return m.Client.Update(ctx, existing)
    }

    return nil
}
```

### Rationale

- **Level-based reconciliation**: Action driven by actual cluster state, not individual events
- **Idempotency**: Multiple reconciliations produce same result
- **Testability**: Business logic can be tested without Kubernetes API server
- **Composability**: Resource managers can be composed and reused

### Examples from Kubernetes Ecosystem

- **Cluster API**: Uses comprehensive resource manager pattern
- **OpenShift Operators**: Separate reconciliation phases into distinct managers
- **Operator SDK samples**: Demonstrate thin controller layer with delegated business logic

### Common Pitfalls to Avoid

1. **Event-driven logic**: Don't write logic based on specific events (create/update/delete)
2. **Non-idempotent operations**: Ensure reconciliation can be safely retried
3. **State leakage**: Don't store state in the controller struct (use status subresource)
4. **Missing error handling**: Always handle and classify errors appropriately
5. **Ignoring context cancellation**: Always respect context cancellation

### Recommendations for Current Codebase

**Current Issues:**
- ✅ Good: Resource manager pattern implemented
- ✅ Good: Idempotent resource creation
- ⚠️  Missing: Update detection and drift correction
- ⚠️  Missing: Status condition updates
- ⚠️  Issue: Finalizer logic mixed with reconciliation
- ⚠️  Issue: Manual deletion of cluster-scoped resources in finalizer

**Improvements:**
```go
// Add drift detection to managers
type ResourceManager interface {
    EnsureResource(ctx context.Context, cr *cachev1alpha1.NFSProvisioner) error
    GetResourceName() string
    NeedsUpdate(existing, desired client.Object) bool  // NEW
}

// Separate finalizer logic
type FinalizerHandler struct {
    Client client.Client
    Log    logr.Logger
}

func (h *FinalizerHandler) Handle(ctx context.Context, obj client.Object, finalizerName string) error {
    // Dedicated finalizer handling logic
}
```

---

## 3. Error Handling

### Recommended Approach

**1. Comprehensive Error Classification:**
```go
package errors

import (
    "fmt"
    apierrors "k8s.io/apimachinery/pkg/api/errors"
    "k8s.io/apimachinery/pkg/runtime/schema"
)

// ErrorType classifies errors for different handling strategies
type ErrorType string

const (
    ErrorTypeTransient  ErrorType = "Transient"   // Retry with backoff
    ErrorTypePermanent  ErrorType = "Permanent"   // Don't retry
    ErrorTypeValidation ErrorType = "Validation"  // User error
    ErrorTypeConflict   ErrorType = "Conflict"    // Optimistic concurrency
)

type OperatorError struct {
    Type    ErrorType
    Message string
    Cause   error
}

func (e *OperatorError) Error() string {
    return fmt.Sprintf("%s error: %s: %v", e.Type, e.Message, e.Cause)
}

// Classification helpers
func IsTransient(err error) bool {
    if apierrors.IsTimeout(err) || apierrors.IsServerTimeout(err) {
        return true
    }
    if apierrors.IsServiceUnavailable(err) || apierrors.IsTooManyRequests(err) {
        return true
    }
    return false
}
```

**2. Status Conditions (Kubernetes Standard):**
```go
// api/v1alpha1/nfsprovisioner_types.go
type NFSProvisionerStatus struct {
    // Conditions represent the latest available observations of an object's state
    Conditions []metav1.Condition `json:"conditions,omitempty"`

    // Nodes are the names of the NFS provisioner pods
    Nodes []string `json:"nodes,omitempty"`

    // ObservedGeneration reflects the generation of the most recently observed NFSProvisioner
    ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// Condition types
const (
    ConditionTypeReady       = "Ready"
    ConditionTypeProgressing = "Progressing"
    ConditionTypeDegraded    = "Degraded"
    ConditionTypeAvailable   = "Available"
)

// Condition reasons
const (
    ReasonReconciling      = "Reconciling"
    ReasonReconcileSuccess = "ReconcileSuccess"
    ReasonReconcileError   = "ReconcileError"
    ReasonValidationFailed = "ValidationFailed"
)
```

**3. Condition Management Helper:**
```go
package conditions

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/api/meta"
)

// SetCondition updates the condition status
func SetCondition(conditions *[]metav1.Condition, conditionType string, status metav1.ConditionStatus, reason, message string) {
    condition := metav1.Condition{
        Type:               conditionType,
        Status:             status,
        ObservedGeneration: 0, // Set from CR
        LastTransitionTime: metav1.Now(),
        Reason:             reason,
        Message:            message,
    }
    meta.SetStatusCondition(conditions, condition)
}

// IsConditionTrue checks if condition is true
func IsConditionTrue(conditions []metav1.Condition, conditionType string) bool {
    condition := meta.FindStatusCondition(conditions, conditionType)
    return condition != nil && condition.Status == metav1.ConditionTrue
}
```

**4. Structured Logging with logr:**
```go
package reconciler

import (
    "context"
    "github.com/go-logr/logr"
    ctrl "sigs.k8s.io/controller-runtime"
)

func (r *NFSProvisionerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := ctrl.LoggerFrom(ctx).WithValues(
        "nfsprovisioner", req.NamespacedName,
        "reconcileID", uuid.New().String(),
    )

    // Use structured logging
    log.Info("Starting reconciliation",
        "generation", nfsProvisioner.Generation,
        "resourceVersion", nfsProvisioner.ResourceVersion,
    )

    if err := r.validate(nfsProvisioner); err != nil {
        log.Error(err, "Validation failed",
            "pvc", nfsProvisioner.Spec.Pvc,
            "storageClass", nfsProvisioner.Spec.SCForNFSPvc,
            "hostPath", nfsProvisioner.Spec.HostPathDir,
        )
        return ctrl.Result{}, err
    }

    // Log with different levels
    log.V(1).Info("Debug information", "details", debugData)

    return ctrl.Result{}, nil
}
```

**5. Error Handling in Reconcile Loop:**
```go
func (r *NFSProvisionerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := ctrl.LoggerFrom(ctx)

    nfsProvisioner := &cachev1alpha1.NFSProvisioner{}
    if err := r.Get(ctx, req.NamespacedName, nfsProvisioner); err != nil {
        if apierrors.IsNotFound(err) {
            log.Info("Resource not found, ignoring since object must be deleted")
            return ctrl.Result{}, nil
        }
        log.Error(err, "Failed to get NFSProvisioner")
        return ctrl.Result{}, err
    }

    // Set progressing condition
    conditions.SetCondition(&nfsProvisioner.Status.Conditions,
        ConditionTypeProgressing, metav1.ConditionTrue,
        ReasonReconciling, "Reconciliation in progress")

    // Validate
    if err := r.validator.Validate(nfsProvisioner); err != nil {
        conditions.SetCondition(&nfsProvisioner.Status.Conditions,
            ConditionTypeReady, metav1.ConditionFalse,
            ReasonValidationFailed, err.Error())
        if updateErr := r.Status().Update(ctx, nfsProvisioner); updateErr != nil {
            log.Error(updateErr, "Failed to update status")
            return ctrl.Result{}, updateErr
        }
        return ctrl.Result{}, err
    }

    // Ensure resources
    if err := r.ResourceManager.EnsureAllResources(ctx, nfsProvisioner); err != nil {
        log.Error(err, "Failed to ensure resources")
        conditions.SetCondition(&nfsProvisioner.Status.Conditions,
            ConditionTypeReady, metav1.ConditionFalse,
            ReasonReconcileError, fmt.Sprintf("Failed to ensure resources: %v", err))
        if updateErr := r.Status().Update(ctx, nfsProvisioner); updateErr != nil {
            log.Error(updateErr, "Failed to update status")
        }

        // Determine if we should retry
        if IsTransient(err) {
            return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
        }
        return ctrl.Result{}, err
    }

    // Success
    conditions.SetCondition(&nfsProvisioner.Status.Conditions,
        ConditionTypeReady, metav1.ConditionTrue,
        ReasonReconcileSuccess, "All resources created successfully")
    conditions.SetCondition(&nfsProvisioner.Status.Conditions,
        ConditionTypeProgressing, metav1.ConditionFalse,
        ReasonReconcileSuccess, "Reconciliation complete")

    nfsProvisioner.Status.ObservedGeneration = nfsProvisioner.Generation

    if err := r.Status().Update(ctx, nfsProvisioner); err != nil {
        log.Error(err, "Failed to update status")
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}
```

### Rationale

- **Observability**: Structured logging and conditions provide clear system state
- **Debugging**: Rich error context helps troubleshoot issues in production
- **User Experience**: Status conditions follow Kubernetes conventions
- **Reliability**: Proper error classification enables appropriate retry strategies

### Examples from Kubernetes Ecosystem

- **Cluster API**: Comprehensive condition management
- **Cert-Manager**: Excellent error handling and status reporting
- **Knative**: Structured logging with detailed conditions

### Common Pitfalls to Avoid

1. **Plain string errors**: Don't use simple error messages without context
2. **Missing status updates**: Always update status on errors
3. **Unstructured logging**: Don't use `fmt.Printf` or plain string logs
4. **Ignoring error types**: Don't treat all errors the same
5. **Logging sensitive data**: Never log secrets, tokens, or credentials

### Recommendations for Current Codebase

**Current Issues:**
- ⚠️  Missing: Status conditions (only has `Error` string and `Nodes` slice)
- ⚠️  Missing: Proper error classification
- ⚠️  Missing: ObservedGeneration tracking
- ⚠️  Issue: Simple error strings instead of structured errors
- ✅ Good: Uses logr for logging
- ✅ Good: Logs errors with context

**Immediate Improvements:**
1. Add `Conditions []metav1.Condition` to NFSProvisionerStatus
2. Create condition helper package
3. Update reconciliation loop to set conditions
4. Add error classification
5. Track ObservedGeneration

---

## 4. Testing Strategy

### 4.1 Unit Testing Approaches

**Recommended Approach:**

**1. Sans-IO Business Logic Testing:**
```go
// internal/reconciler/nfs_provisioner_test.go
func TestReconcileNFSProvisioner_Validation(t *testing.T) {
    tests := []struct {
        name    string
        spec    cachev1alpha1.NFSProvisionerSpec
        wantErr bool
    }{
        {
            name: "PVC and StorageClass both set",
            spec: cachev1alpha1.NFSProvisionerSpec{
                Pvc:         "my-pvc",
                SCForNFSPvc: "my-sc",
            },
            wantErr: true,
        },
        {
            name: "Valid PVC only",
            spec: cachev1alpha1.NFSProvisionerSpec{
                Pvc: "my-pvc",
            },
            wantErr: false,
        },
    }

    validator := validation.NewNFSProvisionerValidator()

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            nfs := &cachev1alpha1.NFSProvisioner{Spec: tt.spec}
            err := validator.Validate(nfs)
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

**2. Resource Builder Testing:**
```go
// internal/builder/deployment_test.go
func TestBuildDeployment(t *testing.T) {
    tests := []struct {
        name        string
        nfs         *cachev1alpha1.NFSProvisioner
        storageType string
        validate    func(*testing.T, *appsv1.Deployment)
    }{
        {
            name: "PVC storage",
            nfs: &cachev1alpha1.NFSProvisioner{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test",
                    Namespace: "default",
                },
            },
            storageType: "PVC",
            validate: func(t *testing.T, dep *appsv1.Deployment) {
                vol := dep.Spec.Template.Spec.Volumes[0]
                if vol.PersistentVolumeClaim == nil {
                    t.Error("Expected PVC volume source")
                }
            },
        },
        {
            name: "HostPath storage",
            nfs: &cachev1alpha1.NFSProvisioner{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test",
                    Namespace: "default",
                },
                Spec: cachev1alpha1.NFSProvisionerSpec{
                    HostPathDir: "/data/nfs",
                },
            },
            storageType: "HOSTPATH",
            validate: func(t *testing.T, dep *appsv1.Deployment) {
                vol := dep.Spec.Template.Spec.Volumes[0]
                if vol.HostPath == nil {
                    t.Error("Expected HostPath volume source")
                }
                if vol.HostPath.Path != "/data/nfs" {
                    t.Errorf("Expected path /data/nfs, got %s", vol.HostPath.Path)
                }
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            builder := NewDeploymentBuilder()
            dep := builder.Build(tt.nfs, tt.storageType)
            tt.validate(t, dep)
        })
    }
}
```

**3. Mock-Based Unit Testing:**
```go
// controllers/resources/deployment_test.go (with mocks)
func TestDeploymentManager_EnsureResource_WithMocks(t *testing.T) {
    // Use testify/mock or gomock
    mockClient := new(MockClient)
    manager := &DeploymentManager{
        BaseResourceManager: BaseResourceManager{
            Client: mockClient,
            Log:    logr.Discard(),
            Scheme: runtime.NewScheme(),
        },
    }

    nfs := &cachev1alpha1.NFSProvisioner{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "test",
            Namespace: "default",
        },
    }

    // Mock: Deployment doesn't exist
    mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything).
        Return(apierrors.NewNotFound(schema.GroupResource{}, "test"))

    // Mock: Create succeeds
    mockClient.On("Create", mock.Anything, mock.Anything).
        Return(nil)

    err := manager.EnsureResource(context.TODO(), nfs)
    if err != nil {
        t.Errorf("Expected no error, got %v", err)
    }

    mockClient.AssertExpectations(t)
}
```

### 4.2 Integration Testing with EnvTest

**Recommended Approach:**

**1. Ginkgo/Gomega Setup with SpecContext:**
```go
// controllers/suite_test.go
package controllers

import (
    "context"
    "path/filepath"
    "testing"
    "time"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"

    "k8s.io/client-go/kubernetes/scheme"
    "k8s.io/client-go/rest"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/envtest"
    logf "sigs.k8s.io/controller-runtime/pkg/log"
    "sigs.k8s.io/controller-runtime/pkg/log/zap"

    cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
)

var (
    cfg       *rest.Config
    k8sClient client.Client
    testEnv   *envtest.Environment
    ctx       context.Context
    cancel    context.CancelFunc
)

func TestControllers(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func(ctx SpecContext) {
    logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

    By("bootstrapping test environment")
    testEnv = &envtest.Environment{
        CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
        ErrorIfCRDPathMissing: true,
    }

    var err error
    cfg, err = testEnv.Start()
    Expect(err).NotTo(HaveOccurred())
    Expect(cfg).NotTo(BeNil())

    err = cachev1alpha1.AddToScheme(scheme.Scheme)
    Expect(err).NotTo(HaveOccurred())

    k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
    Expect(err).NotTo(HaveOccurred())
    Expect(k8sClient).NotTo(BeNil())

    // Start controller manager
    k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
        Scheme: scheme.Scheme,
    })
    Expect(err).ToNot(HaveOccurred())

    err = (&NFSProvisionerReconciler{
        Client: k8sManager.GetClient(),
        Log:    ctrl.Log.WithName("controllers").WithName("NFSProvisioner"),
        Scheme: k8sManager.GetScheme(),
    }).SetupWithManager(k8sManager)
    Expect(err).ToNot(HaveOccurred())

    go func() {
        defer GinkgoRecover()
        err = k8sManager.Start(ctx)
        Expect(err).ToNot(HaveOccurred(), "failed to run manager")
    }()

}, SpecTimeout(60*time.Second))

var _ = AfterSuite(func() {
    cancel()
    By("tearing down the test environment")
    err := testEnv.Stop()
    Expect(err).NotTo(HaveOccurred())
})
```

**2. Integration Tests with SpecContext (Ginkgo v2 Pattern):**
```go
// controllers/nfsprovisioner_controller_test.go
var _ = Describe("NFSProvisioner Controller", func() {
    const (
        timeout  = time.Second * 10
        interval = time.Millisecond * 250
    )

    Context("When creating NFSProvisioner", func() {
        It("Should create all required resources", func(ctx SpecContext) {
            nfs := &cachev1alpha1.NFSProvisioner{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-nfs",
                    Namespace: "default",
                },
                Spec: cachev1alpha1.NFSProvisionerSpec{
                    StorageSize: "10Gi",
                },
            }

            Expect(k8sClient.Create(ctx, nfs)).Should(Succeed())

            nfsLookupKey := types.NamespacedName{
                Name:      nfs.Name,
                Namespace: nfs.Namespace,
            }

            createdNFS := &cachev1alpha1.NFSProvisioner{}

            // Use Eventually for async assertions
            Eventually(func(g Gomega) {
                g.Expect(k8sClient.Get(ctx, nfsLookupKey, createdNFS)).Should(Succeed())
                g.Expect(createdNFS.Status.Conditions).ShouldNot(BeEmpty())
            }).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

            // Verify Deployment created
            deployment := &appsv1.Deployment{}
            Eventually(func(g Gomega) {
                err := k8sClient.Get(ctx, types.NamespacedName{
                    Name:      "nfs-provisioner",
                    Namespace: nfs.Namespace,
                }, deployment)
                g.Expect(err).NotTo(HaveOccurred())
            }).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

        }, SpecTimeout(30*time.Second))

        It("Should fail validation when both PVC and HostPath are set", func(ctx SpecContext) {
            nfs := &cachev1alpha1.NFSProvisioner{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "invalid-nfs",
                    Namespace: "default",
                },
                Spec: cachev1alpha1.NFSProvisionerSpec{
                    Pvc:         "my-pvc",
                    HostPathDir: "/data",
                },
            }

            Expect(k8sClient.Create(ctx, nfs)).Should(Succeed())

            // Verify error condition is set
            Eventually(func(g Gomega) {
                createdNFS := &cachev1alpha1.NFSProvisioner{}
                g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(nfs), createdNFS)).Should(Succeed())

                condition := meta.FindStatusCondition(createdNFS.Status.Conditions, ConditionTypeReady)
                g.Expect(condition).NotTo(BeNil())
                g.Expect(condition.Status).To(Equal(metav1.ConditionFalse))
                g.Expect(condition.Reason).To(Equal(ReasonValidationFailed))
            }).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

        }, SpecTimeout(30*time.Second))
    })
})
```

**3. Best Practices with Eventually:**
```go
// Always use Eventually for assertions against API server
Eventually(func(g Gomega) {
    obj := &corev1.ConfigMap{}
    g.Expect(k8sClient.Get(ctx, key, obj)).Should(Succeed())
    g.Expect(obj.Data["key"]).To(Equal("value"))
}).WithContext(ctx).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

// DON'T: Assert against cache (can be stale)
// obj := manager.GetClient().Get(...) // Uses cache
// Expect(obj.Status).To(Equal(...))   // Flaky!
```

### 4.3 E2E Testing Patterns

**Recommended Approach:**

**1. E2E Framework with Kind:**
```go
// tests/e2e/e2e_test.go
package e2e

import (
    "context"
    "testing"
    "time"

    "sigs.k8s.io/e2e-framework/pkg/env"
    "sigs.k8s.io/e2e-framework/pkg/envconf"
    "sigs.k8s.io/e2e-framework/pkg/envfuncs"
    "sigs.k8s.io/e2e-framework/pkg/features"

    cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
)

var testenv env.Environment

func TestMain(m *testing.M) {
    testenv = env.New()
    kindClusterName := "nfs-operator-e2e"
    namespace := "nfs-system"

    // Setup: Create kind cluster
    testenv.Setup(
        envfuncs.CreateKindCluster(kindClusterName),
        envfuncs.CreateNamespace(namespace),
        envfuncs.LoadDockerImageToCluster(kindClusterName, "nfs-provisioner-operator:test"),
        deployOperator(namespace),
    )

    // Teardown: Destroy kind cluster
    testenv.Finish(
        envfuncs.DeleteNamespace(namespace),
        envfuncs.DestroyKindCluster(kindClusterName),
    )

    testenv.Run(m)
}

func TestNFSProvisionerE2E(t *testing.T) {
    feature := features.New("NFS Provisioner").
        Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
            // Deploy NFSProvisioner CR
            nfs := &cachev1alpha1.NFSProvisioner{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "e2e-test",
                    Namespace: "nfs-system",
                },
                Spec: cachev1alpha1.NFSProvisionerSpec{
                    StorageSize: "10Gi",
                },
            }

            client, err := cfg.NewClient()
            if err != nil {
                t.Fatal(err)
            }

            if err := client.Resources().Create(ctx, nfs); err != nil {
                t.Fatal(err)
            }

            return ctx
        }).
        Assess("NFS server deployment ready", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
            client, err := cfg.NewClient()
            if err != nil {
                t.Fatal(err)
            }

            // Wait for deployment to be ready
            deployment := &appsv1.Deployment{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "nfs-provisioner",
                    Namespace: "nfs-system",
                },
            }

            err = wait.For(
                conditions.New(client.Resources()).DeploymentConditionMatch(
                    deployment,
                    appsv1.DeploymentAvailable,
                    corev1.ConditionTrue,
                ),
                wait.WithTimeout(time.Minute*3),
            )

            if err != nil {
                t.Fatal(err)
            }

            return ctx
        }).
        Assess("StorageClass created", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
            client, err := cfg.NewClient()
            if err != nil {
                t.Fatal(err)
            }

            sc := &storagev1.StorageClass{}
            err = client.Resources().Get(ctx, "example-nfs", "", sc)
            if err != nil {
                t.Fatalf("StorageClass not found: %v", err)
            }

            return ctx
        }).
        Assess("Can provision volume", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
            client, err := cfg.NewClient()
            if err != nil {
                t.Fatal(err)
            }

            // Create test PVC
            pvc := &corev1.PersistentVolumeClaim{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-pvc",
                    Namespace: "nfs-system",
                },
                Spec: corev1.PersistentVolumeClaimSpec{
                    StorageClassName: pointer.String("example-nfs"),
                    AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
                    Resources: corev1.ResourceRequirements{
                        Requests: corev1.ResourceList{
                            corev1.ResourceStorage: resource.MustParse("1Gi"),
                        },
                    },
                },
            }

            if err := client.Resources().Create(ctx, pvc); err != nil {
                t.Fatal(err)
            }

            // Wait for PVC to be bound
            err = wait.For(
                conditions.New(client.Resources()).ResourceMatch(pvc, func(object k8s.Object) bool {
                    p := object.(*corev1.PersistentVolumeClaim)
                    return p.Status.Phase == corev1.ClaimBound
                }),
                wait.WithTimeout(time.Minute*2),
            )

            if err != nil {
                t.Fatalf("PVC not bound: %v", err)
            }

            return ctx
        }).
        Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
            // Cleanup
            client, err := cfg.NewClient()
            if err != nil {
                t.Fatal(err)
            }

            nfs := &cachev1alpha1.NFSProvisioner{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "e2e-test",
                    Namespace: "nfs-system",
                },
            }

            if err := client.Resources().Delete(ctx, nfs); err != nil {
                t.Logf("Failed to delete NFSProvisioner: %v", err)
            }

            return ctx
        }).Feature()

    testenv.Test(t, feature)
}

func deployOperator(namespace string) env.Func {
    return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
        // Deploy operator using kustomize or kubectl
        // This would typically run: kubectl apply -k config/default
        return ctx, nil
    }
}
```

**2. KUTTL-based E2E Testing:**
```yaml
# tests/e2e/kuttl/nfs-provisioner/00-install.yaml
apiVersion: kuttl.dev/v1beta1
kind: TestStep
commands:
  - command: kubectl apply -f ../../../config/crd/bases
  - command: kubectl apply -f ../../../config/samples/cache_v1alpha1_nfsprovisioner.yaml
```

```yaml
# tests/e2e/kuttl/nfs-provisioner/00-assert.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nfs-provisioner
status:
  availableReplicas: 1
  readyReplicas: 1
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: example-nfs
```

```yaml
# tests/e2e/kuttl/kuttl-test.yaml
apiVersion: kuttl.dev/v1beta1
kind: TestSuite
kindNodeCache: true
startKIND: true
kindContext: nfs-operator-e2e
timeout: 120
parallel: 1
```

### 4.4 Testing Best Practices Summary

**Test Pyramid:**
```
        /\
       /E2\     E2E: Kind cluster, full deployment (slow, few tests)
      /____\
     /      \
    /  Integ \  Integration: EnvTest, controller+API (medium, moderate tests)
   /__________\
  /            \
 /     Unit     \ Unit: Pure logic, mocks (fast, many tests)
/________________\
```

**Coverage Guidelines:**
- Unit tests: 70-80% coverage of business logic
- Integration tests: Critical reconciliation paths
- E2E tests: Happy path + critical failure scenarios

### Rationale

- **Fast feedback**: Unit tests run in milliseconds
- **Confidence**: Integration tests verify controller behavior
- **Reality check**: E2E tests validate full system
- **Maintainability**: Layered testing catches issues early

### Examples from Kubernetes Ecosystem

- **controller-runtime**: Extensive envtest examples
- **Cluster API**: Comprehensive E2E framework usage
- **Tekton**: KUTTL-based E2E tests
- **Crossplane**: Multi-layered testing strategy

### Common Pitfalls to Avoid

1. **No unit tests**: Don't skip unit tests in favor of only integration tests
2. **Flaky tests**: Don't assert against cache, use Eventually with API server
3. **Missing timeouts**: Always use SpecTimeout and Eventually timeouts
4. **Slow tests**: Don't use E2E tests for every scenario
5. **No cleanup**: Always cleanup resources in tests
6. **Ignoring context**: Don't ignore SpecContext in Ginkgo v2

### Recommendations for Current Codebase

**Current State:**
- ✅ Good: Has integration tests with Ginkgo/Gomega
- ✅ Good: Uses envtest
- ✅ Good: Tests resource managers
- ⚠️  Missing: Unit tests for business logic
- ⚠️  Missing: E2E tests with Kind
- ⚠️  Issue: Not using SpecContext consistently (should migrate to Ginkgo v2 pattern)

**Migration to SpecContext:**
```go
// OLD (Ginkgo v1 style)
It("should create deployment", func() {
    ctx := context.Background()
    // ...
})

// NEW (Ginkgo v2 style)
It("should create deployment", func(ctx SpecContext) {
    // ctx is provided by Ginkgo
    // ...
}, SpecTimeout(30*time.Second))
```

---

## 5. Code Quality Tools: golangci-lint Configuration

### Recommended Approach

**1. Modern Configuration (Version 2 Schema):**
```yaml
# .golangci.yml
version: "2"

run:
  timeout: 10m
  skip-files:
    - "zz_generated.*\\.go$"
    - ".*conversion.*\\.go$"
  allow-parallel-runners: true
  go: "1.24"

output:
  formats:
    - format: colored-line-number
  print-issued-lines: true
  print-linter-name: true
  uniq-by-line: true

linters:
  disable-all: true
  enable:
    # Essential linters
    - errcheck          # Check for unchecked errors
    - gosimple          # Simplify code
    - govet             # Go vet
    - ineffassign       # Detect ineffective assignments
    - staticcheck       # Go static analysis
    - unused            # Check for unused code
    - typecheck         # Type check

    # Style and quality
    - gofmt             # Check formatting
    - goimports         # Check imports
    - misspell          # Fix spelling errors
    - revive            # golint replacement
    - stylecheck        # Style checker
    - unconvert         # Remove unnecessary conversions
    - unparam           # Unused function parameters

    # Bug detection
    - bodyclose         # Check HTTP body closure
    - contextcheck      # Check context usage (important for SpecContext!)
    - errorlint         # Error wrapping issues
    - nilerr            # Returns nil even if err != nil
    - rowserrcheck      # Check sql rows.Err

    # Security
    - gosec             # Security checker

    # Complexity
    - gocyclo           # Cyclomatic complexity
    - gocognit          # Cognitive complexity

    # Kubernetes-specific
    - importas          # Enforce import aliases
    - godot             # Check comment formatting

    # Code quality
    - goconst           # Repeated strings that could be constants
    - gocritic          # Comprehensive Go linter
    - goprintffuncname  # Check printf function names
    - nakedret          # Naked returns in long functions
    - prealloc          # Preallocate slices
    - whitespace        # Whitespace issues
    - nolintlint        # Ill-formed nolint directives

linters-settings:
  errcheck:
    check-type-assertions: true
    check-blank: true
    exclude-functions:
      - (io.Closer).Close
      - (*database/sql.Rows).Close

  govet:
    enable-all: true
    disable:
      - fieldalignment  # Too strict for operators

  gocyclo:
    min-complexity: 15

  gocognit:
    min-complexity: 20

  importas:
    no-unaliased: true
    alias:
      # Kubernetes
      - pkg: k8s.io/api/core/v1
        alias: corev1
      - pkg: k8s.io/api/apps/v1
        alias: appsv1
      - pkg: k8s.io/api/rbac/v1
        alias: rbacv1
      - pkg: k8s.io/api/storage/v1
        alias: storagev1
      - pkg: k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1
        alias: apiextensionsv1
      - pkg: k8s.io/apimachinery/pkg/apis/meta/v1
        alias: metav1
      - pkg: k8s.io/apimachinery/pkg/api/errors
        alias: apierrors
      - pkg: k8s.io/apimachinery/pkg/util/errors
        alias: kerrors
      # Controller Runtime
      - pkg: sigs.k8s.io/controller-runtime
        alias: ctrl
      # OpenShift
      - pkg: github.com/openshift/api/security/v1
        alias: securityv1

  gocritic:
    enabled-tags:
      - diagnostic
      - style
      - performance
      - experimental
      - opinionated
    disabled-checks:
      - whyNoLint  # We use nolintlint instead
      - unnamedResult  # Allow unnamed returns

  staticcheck:
    checks: ["all"]

  stylecheck:
    checks: ["all", "-ST1000", "-ST1003"]
    # ST1000: Package comments - not required for all packages
    # ST1003: ALL_CAPS naming - allow for constants

  gosec:
    excludes:
      - G104  # Errors unhandled (covered by errcheck)
      - G304  # File path from variable (too strict)
    config:
      global:
        audit: true

  revive:
    rules:
      - name: exported
        arguments:
          - disableStutteringCheck
      - name: package-comments
        disabled: true

  nakedret:
    max-func-lines: 30

  goconst:
    min-len: 3
    min-occurrences: 3
    ignore-tests: true

issues:
  max-issues-per-linter: 0
  max-same-issues: 0
  exclude-use-default: false

  exclude-rules:
    # Exclude some linters from test files
    - path: _test\.go
      linters:
        - gocyclo
        - gocognit
        - errcheck
        - goconst
        - funlen

    # Allow dot imports in tests (for Ginkgo/Gomega)
    - path: _test\.go
      text: "should not use dot imports"

    # Ignore HTTP request made with variable url in tests
    - path: _test\.go
      linters:
        - gosec
      text: "G107: Potential HTTP request made with variable url"

    # Exclude generated files
    - path: zz_generated.*\.go
      linters:
        - all

    # Exclude conversion files
    - path: .*conversion.*\.go
      linters:
        - all

    # Allow embed package with underscore import
    - linters:
        - revive
      source: "_ \"embed\""

    # Allow Reconcile and SetupWithManager without comments
    - linters:
        - revive
      text: "exported: exported method .*\\.(Reconcile|SetupWithManager|SetupWebhookWithManager) should have comment or be unexported"

    # Append can assign to different variable
    - linters:
        - gocritic
      text: "appendAssign: append result not assigned to the same slice"

    # Allow single case switch
    - linters:
        - gocritic
      text: "singleCaseSwitch: should rewrite switch statement to if statement"
```

**2. Makefile Integration:**
```makefile
# Makefile additions
.PHONY: lint
lint: golangci-lint ## Run golangci-lint
	$(GOLANGCI_LINT) run --config .golangci.yml

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint and fix issues
	$(GOLANGCI_LINT) run --config .golangci.yml --fix

## Tool Binaries
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

## Tool Versions
GOLANGCI_LINT_VERSION ?= v1.61.0

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
```

**3. CI/CD Integration:**
```yaml
# .github/workflows/lint.yml
name: Lint

on:
  pull_request:
  push:
    branches:
      - main

jobs:
  golangci:
    name: golangci-lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: v1.61
          args: --timeout=10m --config=.golangci.yml
```

**4. Pre-commit Hook:**
```bash
# .git/hooks/pre-commit
#!/bin/bash

# Run golangci-lint before commit
if ! make lint; then
    echo "❌ Linting failed. Please fix errors before committing."
    exit 1
fi

echo "✅ Linting passed"
```

### Rationale

- **Consistency**: Enforce coding standards across team
- **Bug Prevention**: Catch common mistakes early
- **Code Quality**: Maintain high quality codebase
- **Performance**: Identify performance issues
- **Security**: Detect security vulnerabilities

### Examples from Kubernetes Ecosystem

- **kubernetes/kubernetes**: Uses comprehensive golangci-lint config
- **controller-runtime**: Strict linting with many enabled linters
- **Cluster API**: Similar configuration to recommended above

### Common Pitfalls to Avoid

1. **Too strict**: Don't enable every linter, balance with productivity
2. **Ignoring generated code**: Always exclude zz_generated files
3. **Test file restrictions**: Allow more flexibility in test files
4. **No CI integration**: Always run linting in CI
5. **Outdated version**: Keep golangci-lint updated

### Recommendations for Current Codebase

**Current State:**
- ⚠️  Missing: .golangci.yml configuration file
- ⚠️  Missing: Lint target in Makefile
- ⚠️  Missing: CI integration for linting

**Immediate Actions:**
1. Create `.golangci.yml` with recommended configuration
2. Add `lint` and `lint-fix` targets to Makefile
3. Install golangci-lint in CI
4. Fix existing linting issues
5. Add pre-commit hook (optional)

---

## Summary and Implementation Roadmap

### Quick Reference Matrix

| Area | Current State | Priority | Effort |
|------|---------------|----------|--------|
| Module Organization | Good resource manager pattern | Medium | Medium |
| Controller-Runtime Patterns | Basic implementation | High | Medium |
| Error Handling | Basic logging | High | High |
| Unit Testing | Missing | High | High |
| Integration Testing | Basic envtest | Medium | Low |
| E2E Testing | Missing | Medium | High |
| golangci-lint | Not configured | High | Low |

### Phased Implementation Approach

**Phase 1: Quick Wins (1-2 days)**
1. Add .golangci.yml configuration
2. Fix linting issues
3. Add lint to Makefile and CI
4. Update integration tests to use SpecContext

**Phase 2: Error Handling & Status (3-5 days)**
1. Add Status Conditions to NFSProvisionerStatus
2. Create condition helper package
3. Implement structured error handling
4. Update reconciliation loop with proper error handling

**Phase 3: Testing Infrastructure (1 week)**
1. Extract business logic to testable functions
2. Add unit tests for validation and builders
3. Improve integration test coverage
4. Set up E2E test framework

**Phase 4: Refactoring (1-2 weeks)**
1. Separate reconciliation logic from controller
2. Implement drift detection in resource managers
3. Add proper finalizer handling
4. Refactor controller to thin layer

### Key Takeaways

1. **Module Organization**: Follow standard Kubernetes operator structure with clear separation of concerns
2. **Controller-Runtime**: Keep controller thin, delegate to business logic, use resource manager pattern
3. **Error Handling**: Use status conditions, structured logging, and proper error classification
4. **Testing**: Layer tests (unit → integration → e2e), use SpecContext, leverage Eventually
5. **Code Quality**: Configure golangci-lint with Kubernetes-specific rules

### Additional Resources

**Official Documentation:**
- [The Kubebuilder Book](https://book.kubebuilder.io/reference/good-practices)
- [controller-runtime GoDoc](https://pkg.go.dev/sigs.k8s.io/controller-runtime)
- [Operator SDK Documentation](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/)

**Community Resources:**
- [IBM Operator Sample Go Documentation](https://ibm.github.io/operator-sample-go-documentation/)
- [Kubernetes Operator Patterns](https://www.chainguard.dev/unchained/the-principle-of-reconciliation)
- [E2E Framework](https://github.com/kubernetes-sigs/e2e-framework)

**Testing Guides:**
- [Ginkgo Documentation](https://onsi.github.io/ginkgo/#interruptible-nodes-and-speccontext)
- [Testing Kubernetes Operators with EnvTest](https://www.infracloud.io/blogs/testing-kubernetes-operator-envtest/)
- [E2E Testing Best Practices](https://www.kubernetes.dev/blog/2023/04/12/e2e-testing-best-practices-reloaded/)

---

## Sources

This research document is based on the following sources:

**Module Organization:**
- [Go Operator Tutorial | Operator SDK](https://sdk.operatorframework.io/docs/building-operators/golang/tutorial/)
- [Kubernetes Operator Patterns and Best Practises Documentation](https://ibm.github.io/operator-sample-go-documentation/)
- [GitHub - IBM/operator-sample-go](https://github.com/IBM/operator-sample-go)

**Controller-Runtime Patterns:**
- [reconcile package - sigs.k8s.io/controller-runtime](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/reconcile)
- [The Principle of Reconciliation](https://www.chainguard.dev/unchained/the-principle-of-reconciliation)
- [Good Practices - The Kubebuilder Book](https://book.kubebuilder.io/reference/good-practices)
- [Understanding and Implementing the Reconciliation Loop Pattern](https://oneuptime.com/blog/post/2026-02-09-operator-reconciliation-loop/view)

**Error Handling:**
- [Logging | Operator SDK](https://sdk.operatorframework.io/docs/building-operators/golang/references/logging/)
- [Status conditions :: WebLogic Kubernetes Operator](https://oracle.github.io/weblogic-kubernetes-operator/managing-domains/accessing-the-domain/status-conditions/)
- [Logging Architecture | Kubernetes](https://kubernetes.io/docs/concepts/cluster-administration/logging/)

**Testing Strategy:**
- [Testing Kubernetes Operators using EnvTest](https://www.infracloud.io/blogs/testing-kubernetes-operator-envtest/)
- [Writing tests - The Kubebuilder Book](https://book.kubebuilder.io/cronjob-tutorial/writing-tests.html?highlight=ginkgo)
- [Testing Kubernetes Operators with Ginkgo, Gomega and the Operator Runtime](https://itnext.io/testing-kubernetes-operators-with-ginkgo-gomega-and-the-operator-runtime-6ad4c2492379)
- [Ginkgo Testing for Operator SDK](https://liederbach.dev/blog/ginkgo-testing-for-operator-sdk/)
- [Testing - kube](https://kube.rs/controllers/testing/)
- [Unit testing Kubernetes operators using mocks](https://itnext.io/unit-testing-kubernetes-operators-using-mocks-ba3ba2483ba3)
- [E2E Testing Best Practices, Reloaded](https://www.kubernetes.dev/blog/2023/04/12/e2e-testing-best-practices-reloaded/)
- [GitHub - kubernetes-sigs/e2e-framework](https://github.com/kubernetes-sigs/e2e-framework)
- [Testing Kubernetes Controllers with the E2E-Framework](https://medium.com/programming-kubernetes/testing-kubernetes-controllers-with-the-e2e-framework-fac232843dc6)

**Code Quality Tools:**
- [Configuration – Golangci-lint](https://golangci-lint.run/docs/configuration/)
- [Settings – Golangci-lint](https://golangci-lint.run/docs/linters/configuration/)
- [kubernetes/hack/golangci.yaml](https://github.com/kubernetes/kubernetes/blob/master/hack/golangci.yaml)
- [Go Linters: Essential Tools for Code Quality](https://www.glukhov.org/post/2025/11/linters-for-go/)
- [Migrating to GolangCI-Lint v2 Configuration](https://www.khajaomer.com/blog/level-up-your-go-linting)

---

*Document created: 2026-02-13*
*Based on: nfs-provisioner-operator codebase analysis and 2026 industry best practices*

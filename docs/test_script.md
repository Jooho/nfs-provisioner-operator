# Test Scripts

This document provides comprehensive testing guidance for the NFS Provisioner Operator, covering unit tests, integration tests, and end-to-end testing scenarios.

## Table of Contents

- [Unit Testing with Ginkgo/Gomega](#unit-testing-with-ginkgogomega)
- [Integration Testing](#integration-testing)
- [Coverage Requirements](#coverage-requirements)
- [End-to-End Testing](#end-to-end-testing)
- [Manual Testing Scenarios](#manual-testing-scenarios)

## Unit Testing with Ginkgo/Gomega

### Framework Overview

We use **Ginkgo v2** as our testing framework and **Gomega** for assertions.

- **Ginkgo**: BDD-style testing framework with hierarchical test organization
- **Gomega**: Matcher library with expressive assertions

### Basic Test Structure

```go
package resources

import (
    "testing"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

func TestResources(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "Resources Suite")
}

var _ = Describe("DeploymentManager", func() {
    var (
        deploymentManager *DeploymentManager
        nfsProvisioner    *cachev1alpha1.NFSProvisioner
    )

    BeforeEach(func(ctx SpecContext) {
        // Setup before each test
        nfsProvisioner = &cachev1alpha1.NFSProvisioner{
            ObjectMeta: metav1.ObjectMeta{
                Name:      "test-nfs",
                Namespace: "test-namespace",
            },
            Spec: cachev1alpha1.NFSProvisionerSpec{
                HostPathDir: "/mnt/nfs",
                StorageSize: "10Gi",
            },
        }

        fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
        baseManager := NewBaseResourceManager(fakeClient, logr.Discard(), testScheme)
        deploymentManager = NewDeploymentManager(baseManager)
    })

    It("should create deployment with correct volumes", func(ctx SpecContext) {
        err := deploymentManager.EnsureResource(ctx, nfsProvisioner)
        Expect(err).NotTo(HaveOccurred())

        deployment := &appsv1.Deployment{}
        err = deploymentManager.Client.Get(ctx, types.NamespacedName{
            Name: defaults.Deployment, Namespace: nfsProvisioner.Namespace
        }, deployment)
        Expect(err).NotTo(HaveOccurred())

        Expect(deployment.Spec.Template.Spec.Volumes).To(HaveLen(1))
        Expect(deployment.Spec.Template.Spec.Volumes[0].HostPath.Path).To(Equal("/mnt/nfs"))
    })
})
```

### Key Patterns

#### 1. SpecContext Pattern (REQUIRED)

**IMPORTANT**: All Ginkgo v2 tests MUST use `SpecContext` for interruptibility:

```go
BeforeEach(func(ctx SpecContext) {
    // Use ctx for context-aware operations
})

It("should do something", func(ctx SpecContext) {
    // Use ctx instead of context.Background()
    err := manager.EnsureResource(ctx, nfsProvisioner)
})
```

**Why**: SpecContext enables:
- Test timeouts and cancellation
- Graceful test interruption (Ctrl+C)
- Better test lifecycle management

#### 2. Fake Client Pattern

Use controller-runtime's fake client for fast, in-memory testing:

```go
fakeClient := fake.NewClientBuilder().
    WithScheme(testScheme).
    WithObjects(nfsProvisioner).
    WithStatusSubresource(nfsProvisioner).  // IMPORTANT for status testing
    Build()
```

**Status Testing**: Always use `WithStatusSubresource()` when testing status updates:

```go
// Without WithStatusSubresource: status updates are ignored
// With WithStatusSubresource: status updates work correctly

fakeClient := fake.NewClientBuilder().
    WithScheme(testScheme).
    WithObjects(nfsProvisioner).
    WithStatusSubresource(nfsProvisioner).  // Required!
    Build()

// Now status updates work
nfsProvisioner.Status.Phase = "Ready"
err := fakeClient.Status().Update(ctx, nfsProvisioner)
Expect(err).NotTo(HaveOccurred())
```

#### 3. Gomega Matchers

Use expressive matchers for clear test intent:

```go
// Equality
Expect(value).To(Equal(expected))
Expect(value).NotTo(Equal(notExpected))

// Nil checking
Expect(err).NotTo(HaveOccurred())
Expect(err).To(HaveOccurred())
Expect(ptr).NotTo(BeNil())

// Collections
Expect(slice).To(HaveLen(5))
Expect(slice).To(ContainElement("item"))
Expect(slice).To(ContainElements("item1", "item2"))
Expect(map).To(HaveKey("key"))
Expect(map).To(HaveKeyWithValue("key", "value"))

// Strings
Expect(str).To(ContainSubstring("substring"))
Expect(str).To(HavePrefix("prefix"))
Expect(str).To(MatchRegexp("regex.*pattern"))

// Numerics
Expect(number).To(BeNumerically(">", 10))
Expect(number).To(BeNumerically(">=", 10))
```

#### 4. Table-Driven Tests

Use `DescribeTable` for testing multiple scenarios:

```go
DescribeTable("validates storage options",
    func(hostPath, pvc, scForPvc string, shouldPass bool) {
        nfs := &cachev1alpha1.NFSProvisioner{
            Spec: cachev1alpha1.NFSProvisionerSpec{
                HostPathDir: hostPath,
                Pvc:         pvc,
                SCForNFSPvc: scForPvc,
            },
        }

        err := validator.Validate(nfs)
        if shouldPass {
            Expect(err).NotTo(HaveOccurred())
        } else {
            Expect(err).To(HaveOccurred())
        }
    },
    Entry("hostPath only", "/mnt/nfs", "", "", true),
    Entry("pvc only", "", "my-pvc", "", true),
    Entry("scForPvc only", "", "", "local-storage", true),
    Entry("multiple options", "/mnt/nfs", "", "local-storage", false),
    Entry("no options", "", "", "", false),
)
```

### Running Unit Tests

```bash
# Run all tests
make test

# Run specific package
go test ./pkg/validation/ -v

# Run specific test
go test ./pkg/validation/ -v -ginkgo.focus="validates storage options"

# Run with coverage
go test ./pkg/resources/ -v -coverprofile=cover.out
go tool cover -html=cover.out

# Run with race detection
go test -race ./...

# Skip slow tests
go test ./... -ginkgo.skip-measurements
```

### Test Organization

Use `Describe` and `Context` for hierarchical organization:

```go
var _ = Describe("Validator", func() {
    Describe("Validate", func() {
        Context("when storage options are valid", func() {
            It("should pass with hostPath only", func(ctx SpecContext) {
                // Test
            })

            It("should pass with PVC only", func(ctx SpecContext) {
                // Test
            })
        })

        Context("when storage options are invalid", func() {
            It("should fail with multiple options", func(ctx SpecContext) {
                // Test
            })

            It("should fail with no options", func(ctx SpecContext) {
                // Test
            })
        })
    })
})
```

## Integration Testing

Integration tests use `envtest` to run tests against a real Kubernetes API server.

### Setup

```bash
# Download envtest binaries
make envtest

# Run integration tests
KUBEBUILDER_ASSETS="$(./bin/setup-envtest use 1.30.0 -p path)" go test ./test/integration/ -v
```

### Example: SCC Detection Test

```go
var _ = Describe("SCC Detection", func() {
    Context("on vanilla Kubernetes (no SCC CRD)", func() {
        It("should gracefully skip SCC creation", func(ctx SpecContext) {
            err := sccManager.EnsureResource(ctx, nfsProvisioner)
            Expect(err).NotTo(HaveOccurred())

            // Verify SCC was NOT created
            scc := &securityv1.SecurityContextConstraints{}
            err = k8sClient.Get(ctx, types.NamespacedName{Name: "nfs-provisioner"}, scc)
            Expect(err).To(HaveOccurred())  // Should not exist
        })
    })

    Context("on OpenShift (SCC CRD present)", func() {
        BeforeEach(func(ctx SpecContext) {
            // Create SCC CRD to simulate OpenShift
            sccCRD := &apiextensionsv1.CustomResourceDefinition{
                ObjectMeta: metav1.ObjectMeta{
                    Name: "securitycontextconstraints.security.openshift.io",
                },
                // ... spec
            }
            Expect(k8sClient.Create(ctx, sccCRD)).To(Succeed())
        })

        It("should detect SCC CRD and create SCC", func(ctx SpecContext) {
            err := sccManager.EnsureResource(ctx, nfsProvisioner)
            Expect(err).NotTo(HaveOccurred())

            // Verify SCC was created
            scc := &securityv1.SecurityContextConstraints{}
            err = k8sClient.Get(ctx, types.NamespacedName{Name: "nfs-provisioner"}, scc)
            Expect(err).NotTo(HaveOccurred())
        })
    })
})
```

## Coverage Requirements

### Target Coverage: 80%

The project enforces a minimum of 80% test coverage:

```bash
make test
```

Output:
```
Checking coverage threshold (80%)...
PASS: Coverage 80.0% meets 80% threshold
```

### Current Coverage by Package

- ✅ `pkg/validation`: 97.9%
- ✅ `pkg/defaults`: 100%
- ✅ `pkg/reconciler`: 75.4%
- ✅ `pkg/resources`: 71.7%
- ⚠️ `pkg/builder`: 57.5% (needs improvement)

### Viewing Coverage

```bash
# Generate HTML report
make test
make coverage-report

# Open in browser
open coverage.html

# View in terminal
go tool cover -func=cover.out
```

### Coverage Tips

1. **Focus on behavior, not lines**: 80% coverage doesn't mean test every line, test every behavior
2. **Test error paths**: Test both success and failure scenarios
3. **Use table tests**: Cover multiple inputs efficiently
4. **Mock external dependencies**: Use fake client for Kubernetes API
5. **Don't test generated code**: Skip `zz_generated.deepcopy.go`

### Excluding Files from Coverage

Create `.coverignore`:
```
# Generated files
zz_generated.deepcopy.go
*/zz_generated.deepcopy.go

# Deprecated files
*/deprecated/*
```

## End-to-End Testing

### Manual E2E Testing

**Prerequisites**:
- Kubernetes 1.30+ or OpenShift 4.19+ cluster
- Operator installed via OLM or make deploy

### Scenario 1: HostPath Storage

**Create `PVC` with NFS `StorageClass` (HostPath)**

```bash
# 1. Create NFSProvisioner with HostPath
cat <<EOF | kubectl apply -f -
apiVersion: cache.jhouse.com/v1alpha1
kind: NFSProvisioner
metadata:
  name: nfs-hostpath
  namespace: nfsprovisioner-operator
spec:
  hostPathDir: "/mnt/nfs-data"
  storageSize: "10Gi"
  scForNFSProvisioner: "nfs"
EOF

# 2. Verify NFSProvisioner status
kubectl get nfsprovisioner nfs-hostpath -o yaml

# 3. Check created resources
kubectl get deployment,service,storageclass -n nfsprovisioner-operator | grep nfs

# 4. Create PVC using NFS StorageClass
cat <<EOF | kubectl apply -f -
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: nfs-pvc-example
  namespace: nfsprovisioner-operator
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 1Gi
  storageClassName: nfs
EOF

# 5. Verify PVC is Bound
kubectl get pvc nfs-pvc-example -n nfsprovisioner-operator

# 6. Create test pod to use PVC
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: nfs-test-pod
  namespace: nfsprovisioner-operator
spec:
  containers:
  - name: test
    image: busybox
    command: ["sh", "-c", "echo 'Hello from NFS' > /mnt/data/test.txt && sleep 3600"]
    volumeMounts:
    - name: nfs-volume
      mountPath: /mnt/data
  volumes:
  - name: nfs-volume
    persistentVolumeClaim:
      claimName: nfs-pvc-example
EOF

# 7. Verify file was written
kubectl exec -n nfsprovisioner-operator nfs-test-pod -- cat /mnt/data/test.txt

# Expected output: "Hello from NFS"

# 8. Cleanup
kubectl delete pod nfs-test-pod -n nfsprovisioner-operator
kubectl delete pvc nfs-pvc-example -n nfsprovisioner-operator
kubectl delete nfsprovisioner nfs-hostpath -n nfsprovisioner-operator
```

### Scenario 2: PVC Storage (Production)

```bash
# 1. Ensure a StorageClass exists for NFS server PVC
kubectl get storageclass

# 2. Create NFSProvisioner with PVC storage
cat <<EOF | kubectl apply -f -
apiVersion: cache.jhouse.com/v1alpha1
kind: NFSProvisioner
metadata:
  name: nfs-pvc-backed
  namespace: nfsprovisioner-operator
spec:
  scForNFSPvc: "local-storage"  # Use your available StorageClass
  storageSize: "50Gi"
  scForNFSProvisioner: "nfs"
EOF

# 3. Verify PVC was created for NFS server
kubectl get pvc -n nfsprovisioner-operator | grep nfs-server

# 4. Follow same PVC creation and test pod steps as Scenario 1
```

### Scenario 3: OpenShift SCC Verification

```bash
# 1. Deploy on OpenShift 4.19+
oc apply -f config/samples/cache_v1alpha1_nfsprovisioner.yaml

# 2. Verify SCC was created
oc get scc nfs-provisioner

# 3. Verify service account is in SCC users
oc describe scc nfs-provisioner | grep nfs-provisioner

# 4. Verify NFS server pod uses SCC
oc get pod -n nfsprovisioner-operator -o yaml | grep -A 5 annotations
# Should show: openshift.io/scc: nfs-provisioner
```

### Scenario 4: Upgrade Testing

```bash
# 1. Deploy old version (v0.0.7)
# ... install operator v0.0.7 via OLM

# 2. Create NFSProvisioner CR
kubectl apply -f config/samples/cache_v1alpha1_nfsprovisioner.yaml

# 3. Verify operator and NFS server are running
kubectl get pods -n nfsprovisioner-operator

# 4. Upgrade to new version (v0.0.8)
# ... update operator subscription or bundle

# 5. Verify operator pod restarted with new version
kubectl get csv -n openshift-operators | grep nfs-provisioner

# 6. Verify existing NFSProvisioner still works
kubectl get nfsprovisioner
kubectl get pvc,storageclass | grep nfs

# 7. Create new PVC and verify dynamic provisioning still works
```

### Scenario 5: Error Scenarios

**Validation Errors**:
```bash
# Create invalid CR (multiple storage options)
cat <<EOF | kubectl apply -f -
apiVersion: cache.jhouse.com/v1alpha1
kind: NFSProvisioner
metadata:
  name: nfs-invalid
spec:
  hostPathDir: "/mnt/nfs"
  scForNFSPvc: "local-storage"  # Both set - invalid!
  storageSize: "10Gi"
EOF

# Check status shows validation error
kubectl get nfsprovisioner nfs-invalid -o yaml
# Should show: Phase=Failed, Degraded=True, error message in conditions
```

**Insufficient RBAC**:
```bash
# Remove operator's RBAC permissions
kubectl delete clusterrole nfs-provisioner-operator-manager-role

# Try to reconcile
kubectl apply -f config/samples/cache_v1alpha1_nfsprovisioner.yaml

# Check operator logs for permission errors
kubectl logs -n nfsprovisioner-operator deployment/nfs-provisioner-operator-controller-manager

# Restore RBAC
make deploy
```

## Best Practices

### Testing Guidelines

1. **Write tests first** (TDD): Define expected behavior before implementation
2. **Test behavior, not implementation**: Test what, not how
3. **One assertion per test** (when possible): Clear failure messages
4. **Use descriptive test names**: "should create PVC when using PVC storage" not "test PVC"
5. **Clean up after tests**: Use `AfterEach` or defer for cleanup
6. **Don't share state between tests**: Each test should be independent
7. **Test edge cases**: Empty strings, nil pointers, max/min values
8. **Test error paths**: What happens when things go wrong?

### Common Pitfalls

❌ **Don't**: Use `context.Background()` in tests
✅ **Do**: Use `SpecContext` parameter

❌ **Don't**: Forget `WithStatusSubresource()` when testing status
✅ **Do**: Always use it for status tests

❌ **Don't**: Use `time.Sleep()` for waiting
✅ **Do**: Use `Eventually()` and `Consistently()`

❌ **Don't**: Test generated code
✅ **Do**: Focus on your business logic

❌ **Don't**: Ignore test failures
✅ **Do**: Fix or skip with explanation

### Async Testing with Eventually

For testing eventual consistency:

```go
// Wait up to 10s for condition, checking every 1s
Eventually(func() bool {
    pod := &corev1.Pod{}
    err := k8sClient.Get(ctx, key, pod)
    return err == nil && pod.Status.Phase == corev1.PodRunning
}, "10s", "1s").Should(BeTrue())

// Verify condition stays true for 5s
Consistently(func() bool {
    return deployment.Status.ReadyReplicas == 1
}, "5s", "1s").Should(BeTrue())
```

## References

- [Ginkgo Documentation](https://onsi.github.io/ginkgo/)
- [Gomega Matcher Reference](https://onsi.github.io/gomega/)
- [controller-runtime Fake Client](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/client/fake)
- [envtest](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest)
- [SpecContext Pattern](https://onsi.github.io/ginkgo/#interruptible-nodes-and-speccontext)
# NFS Provisioner Go Operator
![](https://img.shields.io/badge/openshift%204.19+-supported-green) ![](https://img.shields.io/badge/kubernetes%201.30+-supported-green) ![](https://img.shields.io/badge/go%201.25.4-tested-blue) ![](https://img.shields.io/badge/coverage-80%25+-success)

This operator deploys NFS server with several storage options and also provides a provisioner for storageClass.

## Core Capabilities
* NFS Server: Deployed
* NFS Provisioner: Help customers to create PV using StorageClass
* StorageClass: Dynamically create PV for requested PVC
## NFS Provisioner Operator Features
* NFS Server can use localStorage PVC or HostPath on the node


Originally, this operator is created for sharing how to develop operator by Jooho Lee.
This is [the full tutorial page](https://github.com/Jooho/jhouse_openshift/blob/master/test_cases/operator/go-operator/nfs-provisioner-tutorial-docs/Tutorial-1-Go-Operator-without-logic.md)

## Quick Start

### Prerequisites

- Kubernetes 1.30+ or OpenShift 4.19+
- kubectl or oc CLI
- Operator Lifecycle Manager (OLM) installed

### Installation via OLM (OperatorHub)

1. **From OperatorHub UI** (OpenShift Console):
   ```bash
   # Navigate to OperatorHub in OpenShift Console
   # Search for "NFS Provisioner"
   # Click Install and follow the wizard
   ```

2. **From CLI**:
   ```bash
   # Create a Subscription for the operator
   cat <<EOF | kubectl apply -f -
   apiVersion: operators.coreos.com/v1alpha1
   kind: Subscription
   metadata:
     name: nfs-provisioner-operator
     namespace: openshift-operators
   spec:
     channel: alpha
     name: nfs-provisioner-operator
     source: operatorhubio-catalog
     sourceNamespace: olm
   EOF
   ```

3. **Verify Installation**:
   ```bash
   # Check operator pod is running
   kubectl get pods -n openshift-operators | grep nfs-provisioner-operator

   # Check CSV status
   kubectl get csv -n openshift-operators | grep nfs-provisioner
   ```

### Creating an NFS Provisioner Instance

1. **Using HostPath** (for testing/development):
   ```yaml
   apiVersion: cache.jhouse.com/v1alpha1
   kind: NFSProvisioner
   metadata:
     name: nfs-sample
   spec:
     hostPathDir: "/mnt/nfs"
     storageSize: "10Gi"
     scForNFSProvisioner: "nfs"
   ```

2. **Using PVC** (for production):
   ```yaml
   apiVersion: cache.jhouse.com/v1alpha1
   kind: NFSProvisioner
   metadata:
     name: nfs-sample
   spec:
     scForNFSPvc: "local-storage"
     storageSize: "50Gi"
     scForNFSProvisioner: "nfs"
   ```

3. **Apply the CR**:
   ```bash
   kubectl apply -f config/samples/cache_v1alpha1_nfsprovisioner.yaml
   ```

4. **Verify Resources**:
   ```bash
   # Check NFSProvisioner status
   kubectl get nfsprovisioner

   # Check created resources
   kubectl get deployment,service,pvc,storageclass | grep nfs

   # On OpenShift, check SCC
   oc get scc nfs-provisioner
   ```

### Using the NFS StorageClass

Once the NFSProvisioner is running, you can create PVCs using the NFS storage class:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-nfs-claim
spec:
  storageClassName: nfs
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 1Gi
```

## Testing

### Running Unit Tests

```bash
# Run all tests with coverage
make test

# Generate coverage report
make coverage-report
open coverage.html  # View in browser
```

Expected output:
```
PASS: Coverage 80.0% meets 80% threshold
```

### Running Integration Tests

```bash
# Run integration tests with envtest
KUBEBUILDER_ASSETS="$(make envtest use 1.30.0 -p path)" go test ./test/integration/ -v
```

### Running Linter

```bash
# Run golangci-lint
make lint

# Auto-fix linting issues (when possible)
golangci-lint run --fix
```

### Test Coverage by Package

Current test coverage (as of latest):

- `pkg/validation`: 97.9% ✅
- `pkg/defaults`: 100% ✅
- `pkg/reconciler`: 75.4% ✅
- `pkg/resources`: 71.7% ✅
- `pkg/builder`: 57.5%
- **Overall**: ~70-80% ✅

### Manual Testing Scenarios

See [Test Scripts](./docs/test_script.md) for detailed manual testing procedures including:

- Storage option scenarios (hostPath, PVC, localStorage)
- Platform-specific testing (OpenShift vs Kubernetes)
- Upgrade testing
- Error scenarios and recovery

## Architecture

The operator follows a modular architecture with clear separation of concerns:

### Package Structure

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
│   └── object_builder.go      # ObjectMeta builder with functional options
├── resources/        # Resource managers using builder pattern
│   ├── manager.go         # ResourceManager interface
│   ├── deployment.go      # DeploymentManager
│   ├── service.go         # ServiceManager
│   ├── pvc.go            # PVCManager
│   ├── rbac.go           # RBACManager (ClusterRole, ClusterRoleBinding, Role, RoleBinding)
│   ├── scc.go            # SCCManager (OpenShift SecurityContextConstraints)
│   ├── storageclass.go   # StorageClassManager
│   └── serviceaccount.go # ServiceAccountManager
└── reconciler/       # Reconciliation orchestration
    └── reconciler.go      # Reconciler interface with dependency injection
```

### Module Responsibilities

- **pkg/validation**: Validates NFSProvisioner CRs with actionable error messages
  - Storage option mutual exclusivity (hostPathDir, pvc, scForNFSPvc)
  - Field format validation (storageSize, scForNFSProvisioner, image)

- **pkg/defaults**: Applies default values to unset optional fields
  - storageSize: "10Gi"
  - scForNFSProvisioner: "nfs"
  - image: Default NFS provisioner image
  - imagePullPolicy: "IfNotPresent"

- **pkg/builder**: Constructs Kubernetes resources (stateless, pure functions)
  - Deployment, Service, ServiceAccount
  - PVC, StorageClass
  - RBAC (ClusterRole, ClusterRoleBinding, Role, RoleBinding)
  - SCC (OpenShift SecurityContextConstraints)

- **pkg/resources**: Manages resource lifecycle with EnsureResource pattern
  - Each manager handles one resource type
  - Uses builder functions for resource construction
  - Sets controller owner references
  - Provides structured logging

- **pkg/reconciler**: Orchestrates reconciliation logic
  - Coordinates validation → defaults → resource creation
  - Error classification (transient vs. permanent)
  - Intelligent requeue logic

### Controller Architecture

The controller ([controllers/nfsprovisioner_controller.go](controllers/nfsprovisioner_controller.go)) is a thin wrapper that:

- Fetches NFSProvisioner CRs
- Manages finalizers for cleanup
- Delegates main reconciliation to `pkg/reconciler`

This architecture enables:

- Unit testing with mock dependencies
- Clear separation of concerns
- Easy maintenance and extension
- Comprehensive test coverage (80%+ target)

## Documentation
- Storage Options
  - [localStorage](./docs/storage_option_localStorage.md)
  - [hostPath](./docs/storage_option_hostPath.md)
  - [storageClass](./config/samples/cache_v1alpha1_nfsprovisioner.yaml)
  - [PVC](./config/samples/cache_v1alpha1_nfsprovisioner_pvc.yaml)

- Development
  - [Makefile playbook](./docs/makefile_playbook.md)
  - [Build a new image](./docs/new_image.md)
  - [Test Scripts](./docs/test_script.md)
  - [Setup Dev Env](./docs/setup_development.md)

- [Manual Installation](./docs/manual_deploy.md)


## The first steps, if you have all binaries

```bash
go mod tidy
go mod vendor
```

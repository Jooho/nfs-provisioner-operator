# NFS Provisioner Go Operator 
![](https://img.shields.io/badge/openshift%204.16-tested-green)  ![](https://img.shields.io/badge/openshift%204.17-tested-green) ![](https://img.shields.io/badge/openshift%204.18-tested-green)

This operator deploy NFS server with serveral storage options and also provide provisioner for storageClass.

## Core Capabilities
* NFS Server: Deployed
* NFS Provisioner: Help customers to create PV using StorageClass
* StorageClass: Dynamically create PV for requested PVC
## NFS Provisioner Operator Features
* NFS Server can use localStorage PVC or HostPath on the node


Originally, this operator is created for sharing how to develop operator by Jooho Lee.
This is [the full tutorial page](https://github.com/Jooho/jhouse_openshift/blob/master/test_cases/operator/go-operator/nfs-provisioner-tutorial-docs/Tutorial-1-Go-Operator-without-logic.md)

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

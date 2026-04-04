# Data Model: Production Quality Codebase Refactoring

**Feature**: Production Quality Codebase Refactoring
**Date**: 2026-02-13

## Overview

This document describes the data model for the NFS Provisioner Operator. While this refactoring feature does not introduce new data entities, it documents the existing data structures to ensure the refactored code maintains backward compatibility and improves validation/handling of these entities.

## Custom Resource Definitions (CRDs)

### NFSProvisioner (v1alpha1)

**API Group**: `cache.jhouse.com`
**Kind**: `NFSProvisioner`
**Version**: `v1alpha1`

The primary custom resource managed by this operator.

#### Specification Fields (`NFSProvisionerSpec`)

| Field | Type | Required | Default | Description | Validation |
|-------|------|----------|---------|-------------|------------|
| `hostPathDir` | string | No | - | Directory on the node where NFS server will store data | Mutually exclusive with `pvc` and `scForNFSPvc` |
| `pvc` | string | No | - | Name of pre-existing PVC for NFS server storage | Mutually exclusive with `hostPathDir` and `scForNFSPvc` |
| `storageSize` | string | No | "10Gi" | Size of PVC to create for NFS server | Must be valid Kubernetes quantity (e.g., "10Gi", "1Ti") |
| `scForNFSPvc` | string | No | - | StorageClass name for dynamically provisioned NFS server PVC | Mutually exclusive with `hostPathDir` and `pvc` |
| `nodeSelector` | map[string]string | No | {} | Node selector for scheduling NFS server pod | Must match existing node labels |
| `scForNFS` | string | No | "nfs" | StorageClass name that NFS Provisioner will create for end users | Must be valid Kubernetes name |
| `nfsImageConfiguration` | ImageConfiguration | No | Default image | Container image configuration for NFS provisioner | See ImageConfiguration below |

**Storage Configuration Mutual Exclusivity**:
Exactly **one** of the following must be set:
- `hostPathDir`: Use host path on the node
- `pvc`: Use an existing PVC
- `scForNFSPvc`: Dynamically provision a new PVC using a StorageClass

**Current Validation Logic** (from `controllers/nfsprovisioner_controller.go:47-64`):
```go
func validate(m *cachev1alpha1.NFSProvisioner) error {
    pvc := m.Spec.Pvc
    sc := m.Spec.SCForNFSPvc
    hostPathDir := m.Spec.HostPathDir

    if pvc != "" && (sc != "" || hostPathDir != "") {
        return fmt.Errorf("scForPvc or hostPathDir can not set with Pvc")
    }
    if hostPathDir != "" && (sc != "" || pvc != "") {
        return fmt.Errorf("scForPvc or Pvc can not set with hostPathDir")
    }
    if sc != "" && (pvc != "" || hostPathDir != "") {
        return fmt.Errorf("Pvc or hostPathDir can not set with scForPvc")
    }
    return nil
}
```

**Refactoring Improvement**: This validation logic will be moved to a dedicated `pkg/validation` module with clearer error messages and CRD-level validation (kubebuilder markers).

#### Status Fields (`NFSProvisionerStatus`)

| Field | Type | Description | Update Trigger |
|-------|------|-------------|----------------|
| `nodes` | []string | Names of NFS server pods currently running | Updated during reconciliation when Deployment is created/updated |
| `error` | string | Brief error message if reconciliation fails | Updated when validation or resource creation fails |

**Refactoring Improvement**: The current status structure is minimal. The refactored implementation should add:
- **Conditions**: Kubernetes-standard status conditions (Ready, Progressing, Degraded)
- **Observed Generation**: Track which spec generation was last reconciled
- **Phase**: High-level state (Pending, Running, Failed, etc.)

**Proposed Enhanced Status** (post-refactor):
```go
type NFSProvisionerStatus struct {
    // Nodes are the names of the NFS server pods
    Nodes []string `json:"nodes"`

    // Conditions represent the latest available observations of the NFSProvisioner's state
    Conditions []metav1.Condition `json:"conditions,omitempty"`

    // ObservedGeneration is the most recent generation observed by the controller
    ObservedGeneration int64 `json:"observedGeneration,omitempty"`

    // Phase represents the current phase of the NFSProvisioner
    Phase string `json:"phase,omitempty"`
}
```

**Standard Conditions** (to be added):
- `Ready`: NFSProvisioner is fully operational
- `Progressing`: Reconciliation is in progress
- `Degraded`: NFSProvisioner is running but with issues
- `Available`: NFS server deployment has available replicas

#### ImageConfiguration

Nested object for container image configuration.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `image` | string | Yes | `k8s.gcr.io/sig-storage/nfs-provisioner@sha256:...` | Container image for NFS provisioner |
| `imagePullPolicy` | corev1.PullPolicy | Yes | `IfNotPresent` | Image pull policy (Always, IfNotPresent, Never) |

## Owned Kubernetes Resources

The operator creates and manages the following Kubernetes resources for each NFSProvisioner CR:

### Core NFS Server Resources

1. **Deployment** (`apps/v1`)
   - Name: `nfs-provisioner-<cr-name>`
   - Namespace: Same as CR
   - Purpose: Runs the NFS server and provisioner pod
   - Replicas: 1 (stateful, cannot scale)
   - Volume Mount: Based on `hostPathDir`, `pvc`, or dynamically created PVC

2. **Service** (`v1`)
   - Name: `nfs-provisioner-<cr-name>`
   - Type: ClusterIP
   - Purpose: Exposes NFS server endpoint (ports 2049/NFS, 20048/mountd, 111/rpcbind)

3. **PersistentVolumeClaim** (`v1`) - Conditional
   - Name: `nfs-server-<cr-name>`
   - Created only when `scForNFSPvc` is set
   - StorageClass: Value of `scForNFSPvc`
   - Capacity: Value of `storageSize`

4. **StorageClass** (`storage.k8s.io/v1`)
   - Name: Value of `scForNFS` (default: "nfs")
   - Provisioner: `<service-name>/<namespace>`
   - Purpose: Allows end users to request NFS-backed PVCs

### RBAC Resources

5. **ServiceAccount** (`v1`)
   - Name: `nfs-provisioner-<cr-name>`
   - Purpose: Identity for NFS provisioner pod

6. **ClusterRole** (`rbac.authorization.k8s.io/v1`)
   - Name: `nfs-provisioner-runner`
   - Permissions: Create/delete PVs, PVCs, endpoints; list/watch StorageClasses, etc.

7. **ClusterRoleBinding** (`rbac.authorization.k8s.io/v1`)
   - Name: `run-nfs-provisioner`
   - Binds ServiceAccount to ClusterRole

8. **Role** (`rbac.authorization.k8s.io/v1`) - Namespace-scoped
   - Name: `leader-locking-nfs-provisioner`
   - Purpose: Leader election for provisioner HA

9. **RoleBinding** (`rbac.authorization.k8s.io/v1`)
   - Name: `leader-locking-nfs-provisioner`
   - Binds ServiceAccount to leader election Role

### OpenShift-Specific Resources

10. **SecurityContextConstraints** (`security.openshift.io/v1`) - OpenShift only
    - Name: `nfs-provisioner-<cr-name>`
    - Purpose: Grant NFS pod permissions to run as privileged (required for NFS kernel modules)
    - Created conditionally based on SCC CRD availability detection

## Data Flow and State Transitions

### Reconciliation State Machine

```text
┌─────────────────────────────────────────────────────────────┐
│                     CR Created/Updated                      │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
                 ┌───────────────┐
                 │   Validation  │
                 │  (pkg/validation)│
                 └───────┬───────┘
                         │
                    Valid│Invalid
                         ├──────────► Error Status + Requeue
                         │
                         ▼
                ┌────────────────────┐
                │  Resource Creation │ (pkg/resources/manager)
                │  - ServiceAccount  │
                │  - RBAC            │
                │  - SCC (if OpenShift)│
                │  - PVC (if needed) │
                │  - Deployment      │
                │  - Service         │
                │  - StorageClass    │
                └────────┬───────────┘
                         │
                    Success│Failure
                         ├──────────► Error Status + Retry (exponential backoff)
                         │
                         ▼
                ┌────────────────────┐
                │  Status Update     │
                │  - Nodes list      │
                │  - Conditions      │ (post-refactor)
                │  - Phase           │ (post-refactor)
                └────────┬───────────┘
                         │
                         ▼
                   ┌──────────┐
                   │  Ready   │
                   └──────────┘
```

### Phase Transitions (Post-Refactor)

| From Phase | To Phase | Trigger | Condition Change |
|------------|----------|---------|------------------|
| - (nil) | Pending | CR created | Progressing=True, Ready=False |
| Pending | Progressing | Validation passed | Progressing=True, Ready=False |
| Progressing | Running | All resources created successfully | Progressing=False, Ready=True |
| Running | Degraded | Pod crashes, resource conflict | Degraded=True, Ready=True |
| Degraded | Running | Issue resolved automatically | Degraded=False, Ready=True |
| Running | Progressing | Spec updated | Progressing=True, Ready=True |
| Any | Failed | Validation error, permanent failure | Progressing=False, Ready=False |

## Data Validation Rules

### Current Validation (controllers/nfsprovisioner_controller.go)

- ✅ Storage option mutual exclusivity
- ❌ Missing: Field format validation (e.g., storageSize quantity format)
- ❌ Missing: NodeSelector label validation
- ❌ Missing: Image URL format validation
- ❌ Missing: StorageClass name format (RFC 1123 DNS subdomain)

### Proposed Validation (Post-Refactor)

**pkg/validation/validator.go** should implement:

1. **Structural Validation** (via CRD OpenAPI schema):
   - `storageSize`: Must match Kubernetes quantity regex
   - `scForNFS`, `scForNFSPvc`: Must be valid DNS subdomain names
   - `image`: Must be valid container image reference
   - `imagePullPolicy`: Enum (Always, IfNotPresent, Never)

2. **Semantic Validation** (in admission webhook or controller):
   - Storage option mutual exclusivity (existing logic)
   - NodeSelector labels match cluster nodes (optional warning)
   - StorageClass referenced in `scForNFSPvc` exists (optional warning)
   - PVC referenced in `pvc` exists (required check)

3. **Business Logic Validation**:
   - Default values application (`storageSize` = "10Gi", `scForNFS` = "nfs")
   - Image digest validation if specified
   - Conflict detection with existing StorageClass names

## Entity Relationships

```text
                    ┌──────────────────────┐
                    │  NFSProvisioner CR   │
                    └──────────┬───────────┘
                               │ owns
                 ┌─────────────┼─────────────┬────────────────┐
                 │             │             │                │
                 ▼             ▼             ▼                ▼
        ┌────────────┐  ┌──────────┐  ┌────────────┐  ┌────────────┐
        │Deployment  │  │ Service  │  │ PVC (opt)  │  │ServiceAcct │
        └─────┬──────┘  └────┬─────┘  └────────────┘  └──────┬─────┘
              │              │                                 │
              │              │                                 ▼
              │              │                         ┌────────────────┐
              │              │                         │ RBAC Resources │
              │              │                         │  - ClusterRole │
              │              │                         │  - Bindings    │
              │              │                         └────────────────┘
              │              │
              │              │ provides endpoint for
              │              │
              ▼              ▼
        ┌────────────────────────────┐
        │    NFS Server Pod          │
        │  (runs provisioner binary) │
        └────────────┬───────────────┘
                     │ creates PVs for
                     ▼
            ┌────────────────┐
            │ StorageClass   │  ─────► End User PVCs
            └────────────────┘
```

## Refactoring Impact on Data Model

### No Breaking Changes

The data model (CRD schema) **remains unchanged** during refactoring to maintain backward compatibility:
- Existing NFSProvisioner CRs continue to work
- No API version bump (stays v1alpha1)
- No field additions, removals, or renames in Spec

### Internal Representation Improvements

While the CRD schema is unchanged, internal handling improves:

1. **Type Safety**: Dedicated validation types in `pkg/validation`
2. **Default Handling**: Centralized default value application in `pkg/defaults`
3. **Status Management**: Richer status updates (Conditions, Phase) in new `pkg/reconciler`
4. **Error Representation**: Structured error types replacing string errors

### Migration Path

No user-facing migration required. Internal refactoring is transparent to CR authors.

**Operator Upgrade Process**:
1. Existing CRs remain functional during upgrade
2. New status fields (`conditions`, `observedGeneration`, `phase`) populate on next reconciliation
3. Old status fields (`nodes`, `error`) retained for compatibility
4. Improved validation provides better error messages for invalid CRs

## Testing Implications

### Unit Tests (Per Module)

- **pkg/validation**: Test all validation rules with valid and invalid CRs
- **pkg/defaults**: Test default value application for all optional fields
- **pkg/resources**: Test resource construction logic for each resource type

### Integration Tests (envtest)

- **CR Lifecycle**: Create, update, delete NFSProvisioner CRs
- **Storage Options**: Test all three storage configurations (hostPath, PVC, StorageClass)
- **Status Updates**: Verify conditions and phase transitions
- **Error Handling**: Trigger validation errors, API failures, resource conflicts

### E2E Tests (Real Cluster)

- **Full Workflow**: Deploy operator, create CR, verify NFS provisioner works, delete CR
- **Platform Variants**: Test on both Kubernetes (no SCC) and OpenShift (with SCC)
- **End-to-End Storage**: Create PVC using NFS StorageClass, mount in pod, write data

## References

- **CRD Source**: `api/v1alpha1/nfsprovisioner_types.go`
- **Current Validation**: `controllers/nfsprovisioner_controller.go:47-64`
- **Resource Creation**: `controllers/resources/` (manager, deployment, service, pvc, rbac, scc, storageclass)
- **Kubernetes API Conventions**: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md

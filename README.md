# NFS Provisioner Operator

Kubernetes operator that deploys an NFS server and provides a StorageClass for dynamic PV provisioning.

![](https://img.shields.io/badge/openshift%204.19+-supported-green) ![](https://img.shields.io/badge/kubernetes%201.30+-supported-green) ![](https://img.shields.io/badge/go%201.24-tested-blue) ![](https://img.shields.io/badge/coverage-80%25+-success)

## What it does

- Deploys an NFS server pod with flexible storage backend
- Creates a `nfs` StorageClass for dynamic PV provisioning (ReadWriteMany)
- Handles RBAC, ServiceAccount, and SCC (OpenShift) automatically

## Quick Start

```bash
# Install CRDs
kubectl apply -f config/crd/bases/

# Run operator
make run

# Create NFS server (uses cluster default StorageClass)
kubectl apply -f config/samples/cache_v1alpha1_nfsprovisioner_default_sc.yaml

# Create a PVC using NFS
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-nfs-claim
spec:
  storageClassName: nfs
  accessModes: [ReadWriteMany]
  resources:
    requests:
      storage: 1Gi
EOF
```

## Storage Modes

| Mode | CR Spec | Use Case |
|------|---------|----------|
| **Default SC** | `storageSize: "10Gi"` | Simplest, recommended |
| **Specific SC** | `scForNFSPvc: "gp3-csi"` | Choose a StorageClass |
| **Existing PVC** | `pvc: "my-pvc"` | Reuse existing storage |
| **HostPath** | `hostPathDir: "/data/nfs"` | Development only |

See [config/samples/](config/samples/) for example CRs.

## Installation

| Method | Guide |
|--------|-------|
| Local development | `make run` — [quickstart](docs/quickstart.md) |
| Cluster deployment | `make deploy` — [quickstart](docs/quickstart.md) |
| OLM / OperatorHub | [operational scripts](docs/operational_scripts.md) |

## Testing

```bash
make test           # Unit tests (80%+ coverage)
make test-e2e       # E2E on Kind
make test-e2e-ocp   # E2E on OpenShift
make lint           # Linter
```

## Documentation

| Doc | Description |
|-----|-------------|
| [Quick Start](docs/quickstart.md) | Install and run without OLM |
| [Architecture](docs/architecture.md) | Module structure and design |
| [Developer Workflow](docs/developer_workflow.md) | Skills, testing, release process |
| [Makefile Targets](docs/makefile_playbook.md) | Available make commands |
| [Testing Guide](docs/test_script.md) | Unit, integration, E2E testing |
| [Operational Scripts](docs/operational_scripts.md) | hack/scripts workflows |
| [Storage: LocalStorage](docs/storage_option_localStorage.md) | SC-backed storage setup |
| [Storage: HostPath](docs/storage_option_hostPath.md) | Node filesystem setup |

## Project Structure

```
pkg/
├── validation/    # CR validation
├── defaults/      # Default value application
├── builder/       # Stateless resource construction
├── resources/     # Resource lifecycle managers
└── reconciler/    # Reconciliation orchestration

controllers/       # Thin controller wrapper
catalog/           # File-Based Catalog (FBC) for OLM
test/
├── integration/   # envtest-based tests
└── e2e/           # Kind + OpenShift E2E tests
```

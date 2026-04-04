# Operational Scripts Guide

Scripts in `hack/scripts/` and templates in `hack/templates/` for deploying, testing, and managing the NFS Provisioner Operator on OpenShift.

## Prerequisites

All scripts source `env.sh` for shared variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `OP_NAME` | `nfs-provisioner-operator` | Operator name |
| `NAMESPACE` | `nfs-provisioner-operator` | Target namespace |
| `VERSION` | `0.0.8` | Operator version |
| `TAG` | `${VERSION}` | Image tag |

## Deployment Workflows

### Workflow 1: Local Development (make run)

Run the operator on your laptop against a cluster.

```bash
# 1. Install CRDs and deploy NFS CR
make deploy-nfs-cr

# 2. In another terminal, start the operator
make run

# 3. Verify
make test-pvc
make test-pod
make test-rw

# 4. Cleanup
make test-cleanup
```

**Script**: `hack/scripts/deploy-nfs-cr.sh`
- Cleans up previous NFS CR
- Applies `hack/templates/nfs-hostpath.yaml` with pinned image digests
- Creates the namespace if it doesn't exist

---

### Workflow 2: Cluster Deployment (make deploy)

Deploy the operator as a Deployment in the cluster.

```bash
# 1. Deploy operator + create NFS CR
make deploy-nfs-cluster

# 2. Wait for NFS server to be ready
hack/scripts/check-nfs-ready.sh

# 3. Test the NFS provisioner
make test-pvc     # Create PVC using NFS StorageClass
make test-pod     # Mount PVC in a test pod
make test-rw      # Write data and verify persistence

# 4. Cleanup
make test-cleanup
```

**Script**: `hack/scripts/deploy-nfs-cluster.sh`
- Runs `make deploy` to deploy the operator
- Creates NFS CR using `hack/templates/nfs-hostpath.yaml`

---

### Workflow 3: OLM Deployment (CatalogSource + Subscription)

Deploy via Operator Lifecycle Manager.

```bash
# 1. Build and push all images
make push-new-images
# This runs: podman-build → podman-push → bundle-build → bundle-push → index-build → index-push

# 2. Deploy via OLM
CUSTOM_OLD_VERSION=0.0.7 make deploy-nfs-cluster-olm
# This creates: CatalogSource → OperatorGroup → Subscription → NFS CR

# 3. Verify
make test-rw

# 4. Cleanup
make test-cleanup
```

**Script**: `hack/scripts/deploy-nfs-cluster-olm.sh`
- Requires `CUSTOM_OLD_VERSION` to be set
- Creates CatalogSource from `hack/templates/nfs-cs.yaml`
- Creates OperatorGroup from `hack/templates/nfs-og.yaml`
- Creates Subscription from `hack/templates/nfs-subs.yaml`
- Waits for CSV to be ready, then creates NFS CR

---

### Workflow 4: OLM Upgrade Testing

Test operator upgrade from one version to another.

```bash
# 1. Deploy old version first
CUSTOM_OLD_VERSION=0.0.7 make deploy-nfs-cluster-olm

# 2. Write test data
make test-rw

# 3. Upgrade to new version
make deploy-nfs-cluster-olm-upgrade
# Updates CatalogSource to point to the new index image

# 4. Verify data persists after upgrade
FILE_CHECK=TRUE make test-pod
# Checks /mnt/a still exists after upgrade
```

**Script**: `hack/scripts/deploy-nfs-cluster-olm-upgrade.sh`
- Updates CatalogSource with new index image version
- OLM automatically detects the new version and upgrades

---

## Test Scripts

### test-pvc.sh

Creates a PVC using the NFS StorageClass and verifies it becomes Bound.

```bash
make test-pvc
```

Uses `hack/templates/pvc.yaml`:
```yaml
kind: PersistentVolumeClaim
metadata:
  name: test-pvc
spec:
  storageClassName: nfs
  accessModes: [ReadWriteOnce]
  resources:
    requests:
      storage: 1M
```

---

### test-pod.sh

Creates a test pod that mounts the NFS PVC.

```bash
make test-pod

# With file existence check (for upgrade testing)
FILE_CHECK=TRUE make test-pod
```

Uses `hack/templates/pod.yaml` - an Alpine pod that mounts the PVC at `/mnt`.

---

### test-rw.sh

End-to-end read/write test. Creates a PVC, mounts it in a pod, writes a file, and verifies it exists.

```bash
make test-rw
```

Steps:
1. Creates PVC if not exists
2. Creates test pod
3. Writes file: `touch /mnt/a`
4. Reads file: `ls /mnt/a`
5. Reports pass/fail

---

### test-cleanup.sh

Removes all test resources and optionally undeploys the operator.

```bash
make test-cleanup
```

Cleans up:
- Test pod and PVC
- NFS CR
- Operator deployment (if running as cluster deployment)
- OLM resources (Subscription, OperatorGroup, CSV) if present

---

### check-nfs-ready.sh

Polls the NFS Deployment until it reaches desired replica count.

```bash
hack/scripts/check-nfs-ready.sh
```

- Checks every 10 seconds
- Gives up after 10 attempts (100 seconds)
- Reports "NFS is Ready!" or "NFS is not Ready"

---

## Utility Scripts

### push-new-images.sh

Builds and pushes all three images (operator, bundle, index) in one shot.

```bash
make push-new-images
# or
hack/scripts/push-new-images.sh
```

Runs: `podman-build` → `podman-push` → `bundle-build` → `bundle-push` → `index-build` → `index-push`

Prints next steps for upgrade testing after push.

---

### update_pinned_digests.sh

Updates pinned image digests in `env.sh` by pulling latest images and extracting SHA digests.

```bash
hack/scripts/update_pinned_digests.sh
```

Updates digests for:
- `kube-rbac-proxy`
- `nfs-provisioner` (NFS server)
- `nfs-provisioner-operator` (operator)

---

## Templates Reference

| Template | Used By | Description |
|----------|---------|-------------|
| `nfs.yaml` | deploy-nfs-cr.sh | NFS CR with `scForNFSPvc: gp2` (PVC storage) |
| `nfs-hostpath.yaml` | deploy-nfs-cluster.sh | NFS CR with `hostPathDir` (node storage) |
| `nfs-cs.yaml` | deploy-nfs-cluster-olm.sh | OLM CatalogSource |
| `nfs-og.yaml` | deploy-nfs-cluster-olm.sh | OLM OperatorGroup |
| `nfs-subs.yaml` | deploy-nfs-cluster-olm.sh | OLM Subscription (dev catalog) |
| `nfs-subs-prod.yaml` | - | OLM Subscription (community-operators) |
| `pvc.yaml` | test-pvc.sh, test-pod.sh | Test PVC (1M, NFS StorageClass) |
| `pod.yaml` | test-pod.sh, test-rw.sh | Test pod (Alpine, mounts PVC at /mnt) |

## Quick Reference

```bash
# Full cycle: build → deploy → test → cleanup
make push-new-images
CUSTOM_OLD_VERSION=0.0.7 make deploy-nfs-cluster-olm
make test-rw
make test-cleanup

# Local development cycle
make deploy-nfs-cr    # terminal 1
make run              # terminal 2
make test-rw          # terminal 1
make test-cleanup     # terminal 1
```

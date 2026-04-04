# Quick Start Guide

Get the NFS Provisioner Operator running on your cluster without OLM. This guide covers basic installation and usage.

## Prerequisites

- Kubernetes 1.30+ or OpenShift 4.19+ cluster
- kubectl or oc CLI
- Go 1.24+ (for building from source)

## Option 1: Quick Deploy from Source

Clone and install locally:

```bash
# Clone repository
git clone https://github.com/Jooho/nfs-provisioner-operator.git
cd nfs-provisioner-operator
git checkout 001-production-refactor

# Install CRDs
kubectl apply -f config/crd/bases/

# Run operator locally
export PATH=~/dev/lang/go/bin:$PATH
make run
```

The operator will run on your host and reconcile NFSProvisioner CRs in the cluster.

## Option 2: Deploy Operator to Cluster

Build and deploy the operator as a pod in the cluster:

```bash
# Build image
make docker-build IMG=quay.io/<your-org>/nfs-provisioner-operator:latest

# Push to registry (required for cluster deployment)
make docker-push IMG=quay.io/<your-org>/nfs-provisioner-operator:latest

# Install CRDs
kubectl apply -f config/crd/bases/

# Deploy operator to cluster
make deploy IMG=quay.io/<your-org>/nfs-provisioner-operator:latest

# Verify operator is running
kubectl get pods -n nfs-provisioner-operator-system
```

## Creating an NFS Provisioner Instance

Choose one of four storage modes based on your needs:

### 1. Default StorageClass (Recommended)

Simplest option. Operator creates a PVC using the default StorageClass:

```yaml
apiVersion: cache.jhouse.com/v1alpha1
kind: NFSProvisioner
metadata:
  name: nfs-server
spec:
  storageSize: "10Gi"
```

Deploy:
```bash
kubectl apply -f - <<'EOF'
apiVersion: cache.jhouse.com/v1alpha1
kind: NFSProvisioner
metadata:
  name: nfs-server
spec:
  storageSize: "10Gi"
EOF
```

### 2. Specific StorageClass

Use a particular StorageClass for NFS server storage:

```yaml
apiVersion: cache.jhouse.com/v1alpha1
kind: NFSProvisioner
metadata:
  name: nfs-server
spec:
  scForNFSPvc: "gp3-csi"
  storageSize: "10Gi"
```

Deploy:
```bash
kubectl apply -f config/samples/cache_v1alpha1_nfsprovisioner.yaml
```

### 3. Existing PVC

Use a pre-created PVC for NFS server storage:

```yaml
apiVersion: cache.jhouse.com/v1alpha1
kind: NFSProvisioner
metadata:
  name: nfs-server
spec:
  pvc: "my-existing-pvc"
```

Deploy:
```bash
kubectl apply -f config/samples/cache_v1alpha1_nfsprovisioner_pvc.yaml
```

### 4. HostPath (Development Only)

Store NFS data on a node's local filesystem. Requires nodeSelector:

```yaml
apiVersion: cache.jhouse.com/v1alpha1
kind: NFSProvisioner
metadata:
  name: nfs-server
spec:
  hostPathDir: "/data/nfs"
  nodeSelector:
    app: nfs-provisioner
```

Deploy:
```bash
# First label a node
kubectl label nodes <node-name> app=nfs-provisioner

# Then create the NFSProvisioner
kubectl apply -f config/samples/cache_v1alpha1_nfsprovisioner_hostPath.yaml
```

## Verify Installation

Check operator and NFS server status:

```bash
# Check operator is running
kubectl get pods -n nfs-provisioner-operator-system

# Check NFSProvisioner resource
kubectl get nfsprovisioner

# Check created resources
kubectl get deployment,service,pvc,storageclass -n nfs-provisioner-operator-system

# On OpenShift, verify SCC was created
oc get scc nfs-provisioner
```

## Test NFS Provisioning

Create a test PVC to verify dynamic provisioning works:

```bash
# Create test PVC
kubectl apply -f - <<'EOF'
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: test-nfs-pvc
spec:
  storageClassName: nfs
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 1Gi
EOF

# Verify PVC is Bound
kubectl get pvc test-nfs-pvc

# Expected output: test-nfs-pvc   Bound   pvc-xxxxx   1Gi   RWX   nfs   10s
```

### Test Read/Write

Create a pod using the test PVC:

```bash
# Create test pod
kubectl apply -f - <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: nfs-test-pod
spec:
  containers:
  - name: test
    image: busybox:latest
    command: ["sh", "-c", "echo 'Hello from NFS' > /mnt/data/test.txt && cat /mnt/data/test.txt && sleep 3600"]
    volumeMounts:
    - name: nfs-volume
      mountPath: /mnt/data
  volumes:
  - name: nfs-volume
    persistentVolumeClaim:
      claimName: test-nfs-pvc
EOF

# Wait for pod to start
kubectl wait --for=condition=ready pod/nfs-test-pod --timeout=60s

# Verify file was written
kubectl logs nfs-test-pod

# Expected output: Hello from NFS
```

## Cleanup

Remove test resources and operator:

```bash
# Remove test pod
kubectl delete pod nfs-test-pod

# Remove test PVC
kubectl delete pvc test-nfs-pvc

# Remove NFSProvisioner
kubectl delete nfsprovisioner nfs-server

# If deployed to cluster, undeploy operator
make undeploy

# Remove CRDs
kubectl delete -f config/crd/bases/
```

## Troubleshooting

### Check Operator Logs

```bash
# For local run
# Logs appear in terminal where you ran `make run`

# For cluster deployment
kubectl logs -n nfs-provisioner-operator-system deployment/nfs-provisioner-operator-controller-manager -f
```

### Check NFSProvisioner Status

```bash
# View full resource with status
kubectl get nfsprovisioner -o yaml

# Watch status changes
kubectl get nfsprovisioner -w

# Describe for events
kubectl describe nfsprovisioner nfs-server
```

### Common Issues

**PVC not Binding**: Check StorageClass exists and has provisioner
```bash
kubectl get storageclass
kubectl describe storageclass nfs
```

**NFSProvisioner in Failed state**: Check operator logs and CR spec for validation errors
```bash
kubectl describe nfsprovisioner nfs-server
kubectl logs -n nfs-provisioner-operator-system deployment/...
```

**SCC issues on OpenShift**: Verify SCC was created
```bash
oc get scc nfs-provisioner
oc describe scc nfs-provisioner
```

## Next Steps

- [Makefile Targets](./makefile_playbook.md) — Build, test, and deployment commands
- [Testing Guide](./test_script.md) — Unit, integration, and E2E testing
- [Architecture](./architecture.md) — Detailed design and module structure
- [Storage Options](./storage_option_hostPath.md) — Advanced storage configurations

## Support

For issues, questions, or contributions:
- [GitHub Issues](https://github.com/Jooho/nfs-provisioner-operator/issues)
- [Tutorial Docs](https://github.com/Jooho/jhouse_openshift/blob/master/test_cases/operator/go-operator/nfs-provisioner-tutorial-docs/Tutorial-1-Go-Operator-without-logic.md)

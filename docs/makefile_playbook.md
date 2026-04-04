# Makefile Targets Reference

## Environment Variables
- **CUSTOM_OLD_VERSION** — Set old operator version for upgrading test
- **UPGRADE_TEST** — Set TRUE for old index deployment during upgrade testing
- **IMG** — Docker image URL (default: controller:latest)
- **ENVTEST_K8S_VERSION** — Kubernetes version for envtest (default: 1.30.0)

## Build & Run Targets

- **build** — Build manager binary (runs: generate, fmt, vet)
- **run** — Run operator locally from host (requires CRDs installed)
  - Usage: `make run`
  - Runs: manifests, generate, fmt, vet

- **docker-build** — Build Docker image with manager (runs tests first)
  - Usage: `make docker-build IMG=quay.io/myrepo/nfs-provisioner:v1.0.0`

- **docker-push** — Push Docker image to registry
  - Usage: `make docker-push IMG=quay.io/myrepo/nfs-provisioner:v1.0.0`

## Test Targets

### Unit & Integration Tests

- **test** — Run all unit tests with 80% coverage enforcement
  - Runs: manifests, generate, fmt, vet, envtest
  - Usage: `make test`
  - Output: cover.out file with coverage analysis
  - Enforces minimum 80% code coverage threshold
  - Example output: `PASS: Coverage 80.0% meets 80% threshold`

- **coverage-report** — Generate HTML coverage report
  - Requires: prior `make test` run
  - Usage: `make coverage-report`
  - Output: coverage.html (opens in browser)
  - Shows: Line-by-line coverage per file

- **lint** — Run golangci-lint code quality checks
  - Checks: gofmt, govet, errcheck, staticcheck, etc.
  - Uses: project-specific .golangci.yml configuration
  - Usage: `make lint`
  - Auto-fix: `golangci-lint run --fix`

- **envtest** — Download and setup controller-runtime test environment
  - Installs: kubebuilder test binaries (etcd, kube-apiserver, kubectl)
  - Required for: integration tests
  - Usage: `make envtest`
  - Location: bin/setup-envtest

### E2E Tests

- **test-e2e** — Run end-to-end tests on current cluster
  - Creates PVCs using NFS StorageClass
  - Tests dynamic provisioning workflow
  - Usage: `make test-e2e`

- **test-e2e-ocp** — Run OpenShift-specific E2E tests
  - Tests SCC detection and creation
  - Verifies OpenShift resource compatibility
  - Usage: `make test-e2e-ocp`

- **test-pvc** — Create test PVC using NFS StorageClass
  - Quick validation of provisioning functionality
  - Usage: `make test-pvc`

- **test-pod** — Create test Pod using PVC
  - Validates PVC can be mounted by pods
  - Usage: `make test-pod`

- **test-cleanup** — Remove all test-related objects (PVCs, Pods)
  - Usage: `make test-cleanup`

## Deployment Targets

- **install** — Install CRDs into cluster
  - Usage: `make install`
  - Uses: kustomize from config/crd/bases/

- **uninstall** — Uninstall CRDs from cluster
  - Usage: `make uninstall`
  - Flag: `ignore-not-found=true` to ignore missing resources

- **deploy** — Deploy operator to cluster
  - Prerequisite: CRDs already installed
  - Usage: `make deploy IMG=quay.io/myrepo/nfs-provisioner:latest`
  - Uses: kustomize from config/manager/

- **undeploy** — Undeploy operator from cluster
  - Usage: `make undeploy`
  - Flag: `ignore-not-found=true` to ignore missing resources

## OLM & Bundle Targets

- **bundle** — Generate OLM bundle manifests and metadata
  - Creates: bundle/manifests and bundle/metadata
  - Usage: `make bundle`
  - Validates: generated bundle structure

- **bundle-build** — Build OLM bundle image
  - Usage: `make bundle-build`

- **bundle-push** — Push bundle image to registry
  - Usage: `make bundle-push`

- **catalog-build** — Build OLM catalog image
  - Usage: `make catalog-build`

- **catalog-push** — Push catalog image to registry
  - Usage: `make catalog-push`

- **index-build** — Build OLM index image
  - Usage: `make index-build`

- **index-push** — Push index image to registry
  - Usage: `make index-push`

- **push-new-images** — Build and push all images (operator, bundle, index)
  - Usage: `make push-new-images`

## Deployment Scenarios

### Local Development
```bash
make install      # Install CRDs
make run          # Run operator locally
```

### Cluster Deployment
```bash
make docker-build IMG=quay.io/myrepo/nfs-provisioner:latest
make docker-push IMG=quay.io/myrepo/nfs-provisioner:latest
make install      # Install CRDs
make deploy IMG=quay.io/myrepo/nfs-provisioner:latest
```

### OLM Installation
```bash
make bundle      # Generate bundle manifests
make push-new-images  # Build and push all images
```

## Testing Workflow

```bash
# Quick validation
make test
make coverage-report

# Integration tests with envtest
make envtest
KUBEBUILDER_ASSETS="$(./bin/setup-envtest use 1.30.0 -p path)" go test ./test/integration/ -v

# Run specific package tests
go test ./pkg/validation/ -v
go test ./pkg/resources/ -v -coverprofile=cover.out

# Race condition detection
go test -race ./...

# E2E tests on cluster
make install
make deploy IMG=quay.io/myrepo/nfs-provisioner:latest
make test-e2e
```
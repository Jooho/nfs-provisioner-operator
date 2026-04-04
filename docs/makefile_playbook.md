# Makefile targets

## Env
- **CUSTOM_OLD_VERSION**
  - Set old operator version for upgrading test
- **UPGRADE_TEST**
  - Set TRUE, if you want to deploy old index for upgrade test.


## Deploy
- **deploy-op-local**
  - Install a new operator locally
- **deploy-nfs-cr**
  - Create a NFSProvisioner CR to deploy NFS server
- **deploy-nfs-cluster**
  - Install a new operator on a openshift cluster using `operatorsdk deploy` cmd
- **deploy-nfs-cluster-olm**
  - Install a new operator on a openshift cluster using `OLM`
- **deploy-nfs-cluster-olm-upgrade**
  - Replace old index image tag to a new index image tag. It will upgrade old operator to new operator



## Test

### Unit and Integration Tests

- **test**
  - Run all unit tests in ./pkg/ and ./internal/ with coverage tracking
  - Enforces minimum 80% code coverage threshold
  - Generates cover.out file for coverage analysis
  - Automatically runs: manifests, generate, fmt, vet, envtest
  - Usage: `make test`
  - Example output:
    ```
    PASS: Coverage 80.0% meets 80% threshold
    ```

- **coverage-report**
  - Generate HTML coverage report from cover.out
  - Opens visual coverage analysis in browser
  - Shows line-by-line coverage per file
  - Usage: `make coverage-report` (requires prior `make test`)
  - Output file: coverage.html

- **lint**
  - Run golangci-lint to check code quality
  - Checks multiple linters: gofmt, govet, errcheck, staticcheck, etc.
  - Uses project-specific .golangci.yml configuration
  - Usage: `make lint`
  - Auto-fix: `golangci-lint run --fix`

- **envtest**
  - Download and setup controller-runtime test environment
  - Installs kubebuilder test binaries (etcd, kube-apiserver, kubectl)
  - Required for integration tests
  - Usage: `make envtest`
  - Location: bin/setup-envtest

### End-to-End Tests

- **test-pvc**
  - Create a PVC object using NFS StorageClass
  - Verifies dynamic provisioning works correctly
- **test-pod**
  - Create a test Pod to attach the PVC that is created by StorageClass
  - Validates PVC can be mounted by pods
- **test-rw**
  - Create a PVC and attach it to a test pod
  - Create a folder in the PVC and read it
  - Validates read-write operations work correctly
- **test-cleanup**
  - Cleanup all test-related objects (PVCs, Pods)

### Running Tests

**Quick test run**:
```bash
make test
```

**With coverage report**:
```bash
make test
make coverage-report
```

**Run integration tests**:
```bash
make envtest
KUBEBUILDER_ASSETS="$(make envtest use 1.30.0 -p path)" go test ./test/integration/ -v
```

**Run specific package tests**:
```bash
go test ./pkg/validation/ -v
go test ./pkg/resources/ -v -coverprofile=cover.out
```

**Run with race detection**:
```bash
go test -race ./...
```



## Images
- **podman-build**
  - Create a new operator image
- **podman-push**
  -  Push a new operator image
- **bundle-build**
  - Create a new bundle image
- **bundle-push**
  - Push a new bundle image
- **index-build**
  - Create a new index image
- **index-push**
  - Push a new index image
- **push-new-images**
  - All in one target(podman-build/podman-push/bundle-build/bundle-push/index-build/index-push)
# Developer Quickstart Guide

**Feature**: Production Quality Codebase Refactoring
**Date**: 2026-02-13

## Overview

This guide helps developers quickly set up their environment and contribute to the NFS Provisioner Operator refactoring effort. It covers building, testing, and running the operator locally.

**Target Audience**: Go developers contributing to the refactored codebase

## Prerequisites

### Required Tools

| Tool | Version | Purpose | Installation |
|------|---------|---------|--------------|
| Go | 1.24+ | Language runtime | [golang.org/dl](https://golang.org/dl/) |
| Docker | 20.10+ | Container builds | [docs.docker.com/get-docker](https://docs.docker.com/get-docker/) |
| kubectl | 1.30+ | Kubernetes CLI | [kubernetes.io/docs/tasks/tools](https://kubernetes.io/docs/tasks/tools/) |
| operator-sdk | 1.30+ | Operator framework | [sdk.operatorframework.io/docs/installation](https://sdk.operatorframework.io/docs/installation/) |
| kind | 0.20+ | Local Kubernetes | [kind.sigs.k8s.io/docs/user/quick-start](https://kind.sigs.k8s.io/docs/user/quick-start/) |

### Optional Tools

| Tool | Version | Purpose |
|------|---------|---------|
| golangci-lint | 1.55+ | Code linting (added during refactoring) |
| kustomize | 5.0+ | Kubernetes manifest management (bundled with kubectl) |

### Environment Setup

**CRITICAL**: Set the correct Go binary path to avoid version conflicts:

```bash
# Use custom Go installation (as per project constitution)
export PATH=~/dev/lang/go/bin:$PATH

# Verify Go version matches go.mod
go version  # Should show go1.24.x
```

**Why?**: The project requires Go 1.24 (see `go.mod`). System Go may be a different version, causing `compile: version "goX.Y" does not match go tool version "goX.Z"` errors.

## Quick Start (5 Minutes)

### 1. Clone and Build

```bash
# Clone the repository
git clone https://github.com/Jooho/nfs-provisioner-operator.git
cd nfs-provisioner-operator

# Checkout the refactoring branch
git checkout 001-production-refactor

# Download dependencies
go mod download
go mod tidy
go mod vendor

# Build the operator binary
make build
# Output: bin/manager
```

### 2. Run Unit Tests

```bash
# Set Go path (if not already in .bashrc/.zshrc)
export PATH=~/dev/lang/go/bin:$PATH

# Run all tests with coverage
make test

# Output shows:
# - Generated manifests (CRDs, RBAC)
# - go fmt results
# - go vet results
# - Test execution with coverage report (cover.out)
```

**Expected Output** (post-refactor):
```
PASS
coverage: 82.5% of statements
ok      github.com/jooho/nfs-provisioner-operator/pkg/reconciler   2.341s  coverage: 85.2% of statements
ok      github.com/jooho/nfs-provisioner-operator/pkg/validation   0.512s  coverage: 92.1% of statements
ok      github.com/jooho/nfs-provisioner-operator/pkg/resources    3.127s  coverage: 81.3% of statements
```

### 3. Run Operator Locally

```bash
# Install CRDs into your Kubernetes cluster
make install

# Run the operator locally (against the cluster in ~/.kube/config)
make run

# In another terminal, create a sample NFSProvisioner CR
kubectl apply -f config/samples/cache_v1alpha1_nfsprovisioner.yaml

# Watch logs in the `make run` terminal
# You should see reconciliation happening
```

**Cleanup**:
```bash
# Delete the sample CR
kubectl delete -f config/samples/cache_v1alpha1_nfsprovisioner.yaml

# Uninstall CRDs
make uninstall
```

## Development Workflows

### Workflow 1: Implementing a New Module

**Scenario**: You're refactoring the validation logic into `pkg/validation`.

**Steps**:

1. **Create the module structure**:
   ```bash
   mkdir -p pkg/validation
   touch pkg/validation/validator.go
   touch pkg/validation/validator_test.go
   ```

2. **Implement the interface** (see [contracts/module-interfaces.md](contracts/module-interfaces.md#contract-2-validator-interface)):
   ```go
   // pkg/validation/validator.go
   package validation

   import (
       cachev1alpha1 "github.com/jooho/nfs-provisioner-operator/api/v1alpha1"
   )

   type Validator interface {
       Validate(nfs *cachev1alpha1.NFSProvisioner) error
   }

   type validator struct{}

   func NewValidator() Validator {
       return &validator{}
   }

   func (v *validator) Validate(nfs *cachev1alpha1.NFSProvisioner) error {
       // Implementation (move logic from controllers/nfsprovisioner_controller.go:47-64)
       ...
   }
   ```

3. **Write tests FIRST** (TDD per constitution):
   ```go
   // pkg/validation/validator_test.go
   package validation_test

   import (
       . "github.com/onsi/ginkgo/v2"
       . "github.com/onsi/gomega"
   )

   var _ = Describe("Validator", func() {
       var validator Validator

       BeforeEach(func() {
           validator = NewValidator()
       })

       It("should accept valid hostPathDir configuration", func(ctx SpecContext) {
           nfs := &cachev1alpha1.NFSProvisioner{
               Spec: cachev1alpha1.NFSProvisionerSpec{
                   HostPathDir: "/mnt/nfs",
               },
           }
           Expect(validator.Validate(nfs)).To(Succeed())
       }, SpecTimeout(30*time.Second))

       // More test cases...
   })
   ```

4. **Run tests** (they should fail initially - Red phase):
   ```bash
   export PATH=~/dev/lang/go/bin:$PATH
   go test ./pkg/validation -v
   ```

5. **Implement the logic** until tests pass (Green phase)

6. **Refactor** for clarity and maintainability

7. **Run linting** (once golangci-lint is integrated):
   ```bash
   make lint
   ```

8. **Check coverage**:
   ```bash
   go test ./pkg/validation -coverprofile=coverage.out
   go tool cover -html=coverage.out
   # Ensure ≥80% coverage
   ```

### Workflow 2: Running Integration Tests

**Integration tests** use `envtest` to simulate a Kubernetes API server locally.

**Steps**:

1. **Ensure envtest is downloaded**:
   ```bash
   make test  # Downloads envtest binaries to bin/k8s/
   ```

2. **Run integration tests** (post-refactor):
   ```bash
   export PATH=~/dev/lang/go/bin:$PATH
   go test ./test/integration -v

   # Or run specific test
   go test ./test/integration -v -ginkgo.focus="should create all owned resources"
   ```

3. **Debug envtest issues**:
   ```bash
   # Check envtest version
   ls -la bin/k8s/
   # Should show directories like 1.30.0-linux-amd64/

   # Set KUBEBUILDER_ASSETS manually if needed
   export KUBEBUILDER_ASSETS="$(pwd)/bin/k8s/1.30.0-linux-amd64"
   go test ./test/integration -v
   ```

### Workflow 3: Running E2E Tests

**E2E tests** run against a real Kubernetes cluster (Kind or OpenShift).

**Prerequisites**: Kind cluster with OLM installed

**Steps**:

1. **Create a Kind cluster**:
   ```bash
   kind create cluster --name nfs-operator-test
   kubectl cluster-info --context kind-nfs-operator-test
   ```

2. **Install OLM** (Operator Lifecycle Manager):
   ```bash
   operator-sdk olm install
   kubectl get pods -n olm  # Verify OLM is running
   ```

3. **Build and load operator image**:
   ```bash
   make docker-build IMG=localhost:5000/nfs-provisioner-operator:test
   kind load docker-image localhost:5000/nfs-provisioner-operator:test --name nfs-operator-test
   ```

4. **Deploy operator via OLM** (post-refactor):
   ```bash
   # Create CatalogSource, Subscription, etc.
   kubectl apply -f test/e2e/manifests/

   # Wait for operator to be running
   kubectl wait --for=condition=ready pod -l control-plane=controller-manager -n nfs-provisioner-system --timeout=300s
   ```

5. **Run E2E tests**:
   ```bash
   export PATH=~/dev/lang/go/bin:$PATH
   go test ./test/e2e -v -timeout 30m
   ```

6. **Cleanup**:
   ```bash
   kubectl delete -f test/e2e/manifests/
   kind delete cluster --name nfs-operator-test
   ```

## Testing Strategy

### Test Pyramid

```text
        /\
       /E2E\        ← Slow, comprehensive (10-30 min)
      /------\
     /  Integ \     ← Moderate, envtest-based (2-10 min)
    /----------\
   /    Unit    \   ← Fast, mocked (< 2 min)
  /--------------\
```

**Coverage Targets** (per constitution):
- **Unit tests**: ≥80% code coverage
- **Integration tests**: ≥70% of reconciliation logic
- **E2E tests**: ≥60% of user workflows

### Running Tests Selectively

**Run tests for a specific package**:
```bash
go test ./pkg/validation -v
```

**Run a specific test case** (Ginkgo):
```bash
go test ./pkg/validation -v -ginkgo.focus="should reject multiple storage options"
```

**Run tests matching a pattern**:
```bash
go test ./... -run TestValidation
```

**Generate coverage HTML report**:
```bash
go test ./pkg/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

## Code Quality Checks

### Linting (Post-Refactor)

Once `golangci-lint` is integrated:

```bash
# Run linter
make lint

# Auto-fix issues where possible
golangci-lint run --fix

# Run specific linters
golangci-lint run --disable-all --enable=gofmt,govet
```

**Configuration**: `.golangci.yml` (will be added during refactoring)

### Formatting

```bash
# Format all Go code
make fmt

# Check formatting without modifying files
gofmt -l .
```

### Code Generation

After modifying CRD types (`api/v1alpha1/nfsprovisioner_types.go`):

```bash
# Regenerate DeepCopy, CRD manifests, RBAC
make manifests generate

# Verify changes
git diff config/crd/bases/cache.jhouse.com_nfsprovisioners.yaml
```

## Troubleshooting

### Issue: "compile: version mismatch"

**Error**:
```
compile: version "go1.24.2" does not match go tool version "go1.24.4"
```

**Solution**:
```bash
# Use custom Go installation
export PATH=~/dev/lang/go/bin:$PATH
go version  # Verify

# Add to ~/.bashrc or ~/.zshrc for persistence
echo 'export PATH=~/dev/lang/go/bin:$PATH' >> ~/.bashrc
```

### Issue: "envtest binaries not found"

**Error**:
```
unable to find kubebuilder assets
```

**Solution**:
```bash
# Run make test once to download envtest
make test

# Or manually download
make envtest
export KUBEBUILDER_ASSETS="$(pwd)/bin/k8s/1.30.0-linux-amd64"
```

### Issue: "Test timeouts"

**Error**:
```
Ginkgo timed out waiting for test to complete
```

**Solution**: Increase `SpecTimeout` in test files:
```go
It("long-running operation", func(ctx SpecContext) {
    // Test code
}, SpecTimeout(60*time.Second))  // ← Increase timeout
```

### Issue: "Kubernetes cluster not reachable"

**Error**:
```
The connection to the server localhost:8080 was refused
```

**Solution**:
```bash
# Verify KUBECONFIG
echo $KUBECONFIG  # Should point to valid kubeconfig
kubectl cluster-info

# Or set explicitly
export KUBECONFIG=~/.kube/config
```

### Issue: "Import cycle detected"

**Error**:
```
import cycle not allowed: pkg/reconciler → pkg/resources → pkg/reconciler
```

**Solution**: Refactor to break the cycle. Common patterns:
- Extract shared types to a separate package (e.g., `pkg/types`)
- Use dependency injection instead of direct imports
- Review module interfaces in [contracts/module-interfaces.md](contracts/module-interfaces.md)

## CI/CD Integration

### Pre-commit Checklist

Before pushing code:

```bash
# 1. Run tests
export PATH=~/dev/lang/go/bin:$PATH
make test

# 2. Run linting (post-refactor)
make lint

# 3. Verify code is formatted
make fmt
git diff --exit-code  # Should show no changes

# 4. Build successfully
make build

# 5. Run integration tests
go test ./test/integration -v
```

### Pull Request Workflow

1. **Create feature branch** from `001-production-refactor`:
   ```bash
   git checkout -b feature/validation-module
   ```

2. **Implement changes** following TDD (Tests → Implement → Refactor)

3. **Commit** with meaningful messages:
   ```bash
   git add pkg/validation/
   git commit -m "Add validation module with storage option checks

   - Implement Validator interface
   - Add comprehensive unit tests (92% coverage)
   - Move validation logic from controller
   - Improve error messages for users

   Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
   ```

4. **Push** and create pull request:
   ```bash
   git push origin feature/validation-module
   # Create PR on GitHub targeting 001-production-refactor branch
   ```

5. **CI checks** (automated):
   - ✅ Linting (golangci-lint)
   - ✅ Unit tests (≥80% coverage)
   - ✅ Integration tests
   - ✅ Build success

6. **Code review** and merge

## Additional Resources

### Documentation

- **Spec**: [spec.md](spec.md) - Feature requirements
- **Research**: [research.md](research.md) - Refactoring best practices
- **Data Model**: [data-model.md](data-model.md) - CRD structure and entities
- **Contracts**: [contracts/module-interfaces.md](contracts/module-interfaces.md) - Module interfaces
- **Constitution**: [../../.specify/memory/constitution.md](../../.specify/memory/constitution.md) - Project principles

### External Links

- **Ginkgo Testing Framework**: https://onsi.github.io/ginkgo/
  - **SpecContext Guide**: https://onsi.github.io/ginkgo/#interruptible-nodes-and-speccontext
- **Controller-Runtime**: https://github.com/kubernetes-sigs/controller-runtime
- **Operator SDK**: https://sdk.operatorframework.io/
- **Kubernetes API Conventions**: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md

### Project-Specific Resources

- **Makefile Playbook**: `docs/makefile_playbook.md` (existing documentation)
- **Build New Image**: `docs/new_image.md`
- **Test Scripts**: `docs/test_script.md`
- **Dev Env Setup**: `docs/setup_development.md`

## Next Steps

After completing this quickstart:

1. **Read the spec**: [spec.md](spec.md) - Understand feature requirements
2. **Review research**: [research.md](research.md) - Learn refactoring patterns
3. **Pick a user story**: Choose from P1 (Modular Architecture), P2 (Error Handling), P3 (Testing), or P4 (OpenShift 4.19+)
4. **Start coding**: Follow the TDD workflow (Tests → Implementation → Refactor)
5. **Contribute**: Submit PRs following the constitution's development workflow

**Welcome to the NFS Provisioner Operator refactoring effort! 🚀**

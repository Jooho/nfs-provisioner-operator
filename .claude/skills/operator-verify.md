---
name: operator-verify
description: |
  Run full verification suite after operator code changes.
  Unit tests → integration tests → E2E tests (Kind/OCP) → OLM upgrade test → code review.
argument-hint: "[--scope unit|integration|e2e|olm|all] [--platform kind|ocp]"
user-invocable: true
---

# Operator Verify Skill

Run verification checks after code changes. Catches issues before release.

## Arguments

| Argument | Description | Example |
|----------|-------------|---------|
| (none) | Run all checks | `/operator-verify` |
| `--scope unit` | Unit tests only | `/operator-verify --scope unit` |
| `--scope integration` | Integration tests (envtest) | `/operator-verify --scope integration` |
| `--scope e2e` | E2E tests on current cluster | `/operator-verify --scope e2e` |
| `--scope olm` | OLM deploy + upgrade test | `/operator-verify --scope olm` |
| `--platform kind` | E2E/OLM on Kind (default) | `/operator-verify --platform kind` |
| `--platform ocp` | E2E/OLM on OpenShift | `/operator-verify --platform ocp` |
| `--fix` | Auto-fix lint issues | `/operator-verify --fix` |

## Verification Pipeline

### Step 1: Build Check

```bash
export PATH=~/dev/lang/go/bin:$PATH
go vet ./...
go build ./...
```

If build fails → STOP, report errors.

### Step 2: Lint

```bash
make lint
```

If `--fix` is specified, run `golangci-lint run --fix` instead.

### Step 3: Unit Tests

```bash
go test ./pkg/... -timeout 2m -count=1
```

Report per-package pass/fail. If any fail → STOP.

### Step 4: Integration Tests

Requires envtest binaries. If not installed, run `make envtest` first.

```bash
export KUBEBUILDER_ASSETS=$(setup-envtest use 1.30.0 -p path)
go test ./test/integration/ -timeout 5m -count=1
```

Report test count and pass/fail. If any fail → STOP.

### Step 5: E2E Tests

Requires a running cluster (Kind or OpenShift).

**Detect platform automatically:**
1. Check if `oc whoami` succeeds → OCP
2. Check if `kind get clusters` succeeds → Kind
3. Override with `--platform` argument

**Kind:**
```bash
kubectl apply -f config/crd/bases/
make test-e2e
```

**OpenShift:**
```bash
oc apply -f config/crd/bases/
make test-e2e-ocp
```

Report test count and pass/fail. If any fail → report but continue.

### Step 6: OLM Deploy Test (optional, included in `all`)

Only runs if `--scope olm` or `--scope all` (default).

1. **Build operator image:**
   ```bash
   podman build --no-cache -t quay.io/jooholee/nfs-provisioner-operator:test .
   podman push quay.io/jooholee/nfs-provisioner-operator:test
   ```

2. **Build bundle and FBC catalog:**
   ```bash
   podman build -f bundle.Dockerfile -t quay.io/jooholee/nfs-provisioner-operator-bundle:test .
   podman push quay.io/jooholee/nfs-provisioner-operator-bundle:test

   # Render and build catalog
   mkdir -p /tmp/catalog-test/nfs-provisioner-operator
   opm render quay.io/jooholee/nfs-provisioner-operator-bundle:test \
     -o yaml > /tmp/catalog-test/nfs-provisioner-operator/bundle.yaml

   cat > /tmp/catalog-test/nfs-provisioner-operator/package.yaml <<EOF
   schema: olm.package
   name: nfs-provisioner-operator
   defaultChannel: alpha
   EOF

   cat > /tmp/catalog-test/nfs-provisioner-operator/channel.yaml <<EOF
   schema: olm.channel
   name: alpha
   package: nfs-provisioner-operator
   entries:
     - name: nfs-provisioner-operator.v{VERSION}
   EOF

   opm validate /tmp/catalog-test/
   ```

3. **Deploy via OLM:**
   ```bash
   # Create namespace, CatalogSource, OperatorGroup, Subscription
   # Wait for CSV Succeeded
   # Create NFSProvisioner CR with default SC
   # Verify: Status=Ready, Available=True, error count=0
   ```

4. **Cleanup:**
   ```bash
   # Delete CR, Subscription, CSV, CatalogSource, namespace
   # Delete cluster-scoped resources (ClusterRole, ClusterRoleBinding, StorageClass, SCC)
   ```

### Step 7: Code Review (optional)

If all tests pass, suggest running `/code-review` for quality check.

### Step 8: Report

```
Operator Verification Report
──────────────────────────────────────
Platform: Kind / OCP 4.19
──────────────────────────────────────
✅ Build          go vet + go build
✅ Lint           golangci-lint (0 issues)
✅ Unit Tests     5/5 packages passed
✅ Integration    10/10 tests passed (26s)
✅ E2E            8/8 tests passed (110s)
✅ OLM Deploy     CSV Succeeded, CR Ready
──────────────────────────────────────
Result: PASS — Ready for release
```

Or on failure:
```
──────────────────────────────────────
✅ Build          go vet + go build
✅ Lint           golangci-lint (0 issues)
✅ Unit Tests     5/5 packages passed
❌ Integration    9/10 tests passed (1 failed)
⏭️  E2E            Skipped (integration failed)
⏭️  OLM Deploy     Skipped
──────────────────────────────────────
Result: FAIL — Fix integration test before proceeding
```

## Scope Options

| Scope | Steps Run |
|-------|-----------|
| `unit` | 1-3 (build, lint, unit) |
| `integration` | 1-4 (+ integration) |
| `e2e` | 1-5 (+ e2e) |
| `olm` | 1-6 (+ olm deploy) |
| `all` (default) | 1-7 (everything) |

Each scope includes all previous steps. If a step fails, subsequent steps are skipped.

## Platform Auto-Detection

```
1. --platform argument provided? → Use it
2. oc whoami succeeds?           → OCP
3. kind get clusters succeeds?   → Kind
4. kubectl cluster-info succeeds? → Generic K8s (treat as Kind)
5. None available                → Skip E2E/OLM, run unit/integration only
```

## Usage Examples

```bash
# Full verification (default)
/operator-verify

# Quick check after small code change
/operator-verify --scope unit

# Test on OpenShift before release
/operator-verify --platform ocp

# Fix lint issues and run unit tests
/operator-verify --scope unit --fix

# Integration + E2E only
/operator-verify --scope e2e
```

## Integration with operator-release

Recommended flow:
```
/operator-verify                  # Verify everything passes
/operator-release 0.0.10          # Release if verify passed
```

The `operator-release` skill does NOT run tests — it assumes verification was done separately.

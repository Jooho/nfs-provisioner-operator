# Developer Workflow Guide

Day-to-day workflows for developing, testing, and releasing the NFS Provisioner Operator.

## Available Skills (Claude Code)

| Command | What it does |
|---------|-------------|
| `/operator-verify` | Run tests and checks after code changes |
| `/operator-release <version>` | Build images, create FBC catalog, submit PRs |

## Daily Development

### 1. Make code changes

```bash
# Work on your branch
git checkout 001-production-refactor

# Edit code in pkg/, controllers/, etc.
```

### 2. Quick verification

```bash
# Unit tests only (fast, ~5 seconds)
/operator-verify --scope unit

# Unit + integration tests (~30 seconds)
/operator-verify --scope integration
```

### 3. Full verification before PR

```bash
# Everything: build, lint, unit, integration, e2e, olm
/operator-verify

# On OpenShift specifically
/operator-verify --platform ocp
```

### 4. Fix lint issues

```bash
/operator-verify --scope unit --fix
```

## Release Process

### Step 1: Verify everything passes

```bash
/operator-verify
```

Wait for all checks to pass. Do NOT proceed if anything fails.

### Step 2: Release

```bash
# Full release: build images + FBC catalog + submit PRs
/operator-release 0.0.10

# Or just build without PRs
/operator-release 0.0.10 --skip-pr
```

This will:
1. Update version in `env`, CSV, bundle
2. Build and push operator, bundle, catalog images to quay.io
3. Update FBC catalog (`catalog/`)
4. Submit PRs to both:
   - [k8s-operatorhub/community-operators](https://github.com/k8s-operatorhub/community-operators) (OperatorHub.io)
   - [redhat-openshift-ecosystem/community-operators-prod](https://github.com/redhat-openshift-ecosystem/community-operators-prod) (OpenShift OperatorHub)

### Step 3: Monitor PRs

Both repos run automated CI. Watch for:
- Bundle validation
- Scorecard tests
- Operator installation test

## Manual Commands Reference

If you prefer running things manually instead of using skills:

### Build & Test

```bash
export PATH=~/dev/lang/go/bin:$PATH

# Build
go build ./...

# Unit tests
go test ./pkg/... -timeout 2m

# Integration tests (needs envtest)
export KUBEBUILDER_ASSETS=$(bin/setup-envtest use 1.30.0 -p path)
go test ./test/integration/ -timeout 5m

# E2E tests
make test-e2e          # Kind
make test-e2e-ocp      # OpenShift

# Lint
make lint
```

### Build Images

```bash
# Operator
podman build -t quay.io/jooholee/nfs-provisioner-operator:0.0.10 .
podman push quay.io/jooholee/nfs-provisioner-operator:0.0.10

# Bundle
podman build -f bundle.Dockerfile -t quay.io/jooholee/nfs-provisioner-operator-bundle:0.0.10 .
podman push quay.io/jooholee/nfs-provisioner-operator-bundle:0.0.10

# FBC Catalog
opm render quay.io/jooholee/nfs-provisioner-operator-bundle:0.0.10 \
  -o yaml > catalog/nfs-provisioner-operator/v0.0.10.yaml
# Update catalog/nfs-provisioner-operator/channel.yaml
opm validate catalog/
podman build -f catalog.Dockerfile -t quay.io/jooholee/nfs-provisioner-operator-catalog:0.0.10 .
podman push quay.io/jooholee/nfs-provisioner-operator-catalog:0.0.10
```

### OLM Deploy (manual)

```bash
# Install CRDs
oc apply -f config/crd/bases/

# Create CatalogSource
oc apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: nfs-provisioner-operator-catalog
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: quay.io/jooholee/nfs-provisioner-operator-catalog:0.0.10
  displayName: NFS Provisioner Operator
  publisher: Jooho Lee
EOF

# Create Subscription (see docs/operational_scripts.md for full details)
```

## Project Structure

```
.claude/skills/
├── operator-verify.md    # /operator-verify skill
└── operator-release.md   # /operator-release skill

docs/
├── quickstart.md           # Non-OLM install guide
├── architecture.md         # Module structure
├── operational_scripts.md  # hack/scripts guide
├── developer_workflow.md   # This file
├── makefile_playbook.md    # Makefile targets
└── test_script.md          # Testing guide

catalog/                    # File-Based Catalog (FBC)
├── catalog.Dockerfile
└── nfs-provisioner-operator/
    ├── package.yaml
    ├── channel.yaml
    ├── v0.0.8.yaml
    └── v0.0.9.yaml
```

## Related Docs

- [Quickstart (non-OLM install)](quickstart.md)
- [Operational Scripts](operational_scripts.md)
- [Makefile Playbook](makefile_playbook.md)
- [Testing Guide](test_script.md)
- [Architecture](architecture.md)

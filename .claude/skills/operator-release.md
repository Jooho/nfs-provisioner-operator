---
name: operator-release
description: |
  Build operator images, create FBC catalog, and submit PRs to community-operators repos.
  Full release pipeline: build → bundle → FBC catalog → push → PR to k8s-operatorhub + community-operators-prod.
argument-hint: "<version> [--skip-pr]"
user-invocable: true
---

# Operator Release Skill

Release the NFS Provisioner Operator: build images, create File-Based Catalog, and submit PRs to upstream operator catalogs.

## Arguments

| Argument | Description | Example |
|----------|-------------|---------|
| `<version>` | Version to release (required) | `/operator-release 0.0.10` |
| `--skip-pr` | Build and push only, skip PR creation | `/operator-release 0.0.10 --skip-pr` |
| `--dry-run` | Show what would be done without executing | `/operator-release 0.0.10 --dry-run` |

## Prerequisites

Before running this skill, verify:
1. All tests pass: `make test`, `make test-e2e`
2. Code is committed on the feature branch
3. `podman` logged into `quay.io` (`podman login quay.io`)
4. `gh` CLI authenticated (`gh auth status`)
5. Community operator repos cloned as forks:
   - `~/temp/20260213_SPECKIT/k8s-community-operators` (fork of k8s-operatorhub/community-operators)
   - `~/temp/20260213_SPECKIT/community-operators-prod` (fork of redhat-openshift-ecosystem/community-operators-prod)

## Release Pipeline

### Phase 1: Prepare Version

1. Read current version from `env` file (`VERSION` variable)
2. Read the `<version>` argument as the NEW_VERSION
3. Determine PREV_VERSION from the current VERSION
4. Confirm with user: "Release v{NEW_VERSION} (replaces v{PREV_VERSION})?"

### Phase 2: Update Version References

1. Update `env` file: `VERSION={NEW_VERSION}`, `TAG={NEW_VERSION}`
2. Update CSV in `bundle/manifests/nfs-provisioner-operator.clusterserviceversion.yaml`:
   - `metadata.name`: `nfs-provisioner-operator.v{NEW_VERSION}`
   - `spec.version`: `{NEW_VERSION}`
   - `spec.replaces`: `nfs-provisioner-operator.v{PREV_VERSION}`
   - `containerImage` annotation: `quay.io/jooholee/nfs-provisioner-operator:{NEW_VERSION}`
   - Deployment image: `quay.io/jooholee/nfs-provisioner-operator:{NEW_VERSION}`
3. Sync CRD to bundle: `cp config/crd/bases/*.yaml bundle/manifests/`

### Phase 3: Build & Push Images

Run these commands sequentially:

```bash
# 1. Operator image
podman build --no-cache -t quay.io/jooholee/nfs-provisioner-operator:{NEW_VERSION} .
podman push quay.io/jooholee/nfs-provisioner-operator:{NEW_VERSION}

# 2. Bundle image
podman build -f bundle.Dockerfile -t quay.io/jooholee/nfs-provisioner-operator-bundle:{NEW_VERSION} .
podman push quay.io/jooholee/nfs-provisioner-operator-bundle:{NEW_VERSION}
```

### Phase 4: Build File-Based Catalog (FBC)

1. Render the new bundle into FBC format:
   ```bash
   opm render quay.io/jooholee/nfs-provisioner-operator-bundle:{NEW_VERSION} \
     -o yaml > catalog/nfs-provisioner-operator/v{NEW_VERSION}.yaml
   ```

2. Update `catalog/nfs-provisioner-operator/channel.yaml`:
   - Add new entry with `replaces` pointing to PREV_VERSION
   ```yaml
   entries:
     - name: nfs-provisioner-operator.v{PREV_VERSION}
     - name: nfs-provisioner-operator.v{NEW_VERSION}
       replaces: nfs-provisioner-operator.v{PREV_VERSION}
   ```

3. Validate: `opm validate catalog/`

4. Build and push catalog image:
   ```bash
   podman build --no-cache -f catalog.Dockerfile \
     -t quay.io/jooholee/nfs-provisioner-operator-catalog:{NEW_VERSION} .
   podman push quay.io/jooholee/nfs-provisioner-operator-catalog:{NEW_VERSION}
   ```

### Phase 5: Commit Release

```bash
git add env bundle/ catalog/
git commit -S -s -m "release: v{NEW_VERSION}"
```

**IMPORTANT**: Always use `-S -s` flags for signed commits with sign-off.

### Phase 6: Submit PRs (skip if --skip-pr)

For each community-operators repo:

#### 6a. k8s-operatorhub/community-operators

```bash
REPO=~/temp/20260213_SPECKIT/k8s-community-operators
DEST=$REPO/operators/nfs-provisioner-operator/{NEW_VERSION}

# Create version directory
mkdir -p $DEST/manifests $DEST/metadata $DEST/tests

# Copy bundle
cp bundle/manifests/* $DEST/manifests/
cp bundle/metadata/annotations.yaml $DEST/metadata/
cp bundle/tests/scorecard/config.yaml $DEST/tests/ 2>/dev/null

# Branch, commit, push
cd $REPO
git checkout main && git pull
git checkout -b nfs-provisioner-operator-{NEW_VERSION}
git add operators/nfs-provisioner-operator/{NEW_VERSION}/
git commit -S -s -m "operators nfs-provisioner-operator ({NEW_VERSION})"
git push origin nfs-provisioner-operator-{NEW_VERSION}

# Create PR
gh pr create \
  --repo k8s-operatorhub/community-operators \
  --head Jooho:nfs-provisioner-operator-{NEW_VERSION} \
  --base main \
  --title "operators nfs-provisioner-operator ({NEW_VERSION})" \
  --body "Upgrade from {PREV_VERSION}. See CSV for details."
```

#### 6b. redhat-openshift-ecosystem/community-operators-prod (FBC format)

This repo uses **File-Based Catalog (FBC)** format, NOT the old registry+v1 bundle format.

```bash
REPO=~/temp/20260213_SPECKIT/community-operators-prod
OP_DIR=$REPO/operators/nfs-provisioner-operator

cd $REPO
git checkout main && git pull
git checkout -b nfs-provisioner-operator-{NEW_VERSION}

# 1. Copy bundle directory (same as 6a)
DEST=$OP_DIR/{NEW_VERSION}
mkdir -p $DEST/manifests $DEST/metadata $DEST/tests
cp bundle/manifests/* $DEST/manifests/
cp bundle/metadata/annotations.yaml $DEST/metadata/
cp bundle/tests/scorecard/config.yaml $DEST/tests/ 2>/dev/null

# 2. Update FBC basic-template.yaml - add new bundle entry and update channel
#    Edit $OP_DIR/catalog-templates/basic-template.yaml:
#    - Add new bundle image: quay.io/jooholee/nfs-provisioner-operator-bundle:{NEW_VERSION}
#    - Update channel entries: add v{NEW_VERSION} replaces v{PREV_VERSION}

# 3. Render catalog.yaml from local FBC data
#    Combine package.yaml + channel.yaml + all version yamls into catalog.yaml
#    Copy to ALL supported OCP version directories:
CATALOG_VERSIONS="v4.19 v4.20 v4.21 v4.22"
for ver in $CATALOG_VERSIONS; do
  mkdir -p $REPO/catalogs/${ver}/nfs-provisioner-operator
  cp rendered-catalog.yaml $REPO/catalogs/${ver}/nfs-provisioner-operator/catalog.yaml
done

# 4. If new OCP versions exist in catalogs/, add them to ci.yaml catalog_names

# 5. Commit and push
git add operators/nfs-provisioner-operator/ catalogs/
git commit -S -s -m "operators nfs-provisioner-operator ({NEW_VERSION})"
git push origin nfs-provisioner-operator-{NEW_VERSION}

# 6. Create PR
gh pr create \
  --repo redhat-openshift-ecosystem/community-operators-prod \
  --head Jooho:nfs-provisioner-operator-{NEW_VERSION} \
  --base main \
  --title "operators nfs-provisioner-operator ({NEW_VERSION})" \
  --body "Upgrade from {PREV_VERSION}. FBC migration included."
```

**Key differences from 6a:**
- Includes FBC files: `catalog-templates/`, `catalogs/v4.19~v4.22/`
- `ci.yaml` has `fbc.enabled: true` with catalog_mapping
- `Makefile` for FBC build/validation is already present

### Phase 7: Summary

Display:
```
Operator Release v{NEW_VERSION} Complete
──────────────────────────────────────
Images:
  ✅ quay.io/jooholee/nfs-provisioner-operator:{NEW_VERSION}
  ✅ quay.io/jooholee/nfs-provisioner-operator-bundle:{NEW_VERSION}
  ✅ quay.io/jooholee/nfs-provisioner-operator-catalog:{NEW_VERSION}

PRs:
  ✅ k8s-operatorhub/community-operators#XXXX
  ✅ community-operators-prod#XXXX

Next: Monitor PR CI checks
```

## Verification Checklist

Before submitting PRs, the skill verifies:
- [ ] `opm validate catalog/` passes
- [ ] Bundle image renders correctly (`opm render`)
- [ ] CSV has correct `version`, `replaces`, and `name` fields
- [ ] All images pushed successfully to quay.io
- [ ] Git commits use `-S -s` flags

## Rollback

If something goes wrong:
```bash
# Revert version changes
git revert HEAD

# Delete remote branches
git push origin --delete nfs-provisioner-operator-{NEW_VERSION}

# Close PRs via gh CLI
gh pr close <PR_NUMBER> --repo <repo>
```

## Usage Examples

```bash
# Full release with PRs
/operator-release 0.0.10

# Build and push only (no PRs)
/operator-release 0.0.10 --skip-pr

# Preview what would happen
/operator-release 0.0.10 --dry-run
```

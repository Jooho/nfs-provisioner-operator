---
name: operator-release
description: |
  Agentic release pipeline: verify → bump-version → build/push images → FBC catalog → commit → community-operators PRs.
  Full pipeline with dry-run support and 3 human approval gates.
argument-hint: "<version> [--skip-pr] [--dry-run]"
user-invocable: true
---

# Operator Release Skill

Release the NFS Provisioner Operator with an agentic pipeline: automated steps with human approval at critical gates.

## Arguments

| Argument | Description | Example |
|----------|-------------|---------|
| `<version>` | Version to release (required) | `/operator-release 0.0.10` |
| `--skip-pr` | Build and push only, skip PR creation | `/operator-release 0.0.10 --skip-pr` |
| `--dry-run` | Show what would be done without executing | `/operator-release 0.0.10 --dry-run` |

## Prerequisites

Before running this skill, verify with:
```bash
make validate-release
```

Required:
1. `podman` logged into `quay.io`
2. `gh` CLI authenticated
3. `opm`, `kustomize` available
4. Community operator repos cloned as forks:
   - `~/temp/20260213_SPECKIT/k8s-community-operators`
   - `~/temp/20260213_SPECKIT/community-operators-prod`

## Release Pipeline

### Phase 0: Prerequisites & Version Setup

1. Run `make validate-release` — abort if any check fails
2. Read current version from `env` file (`VERSION` variable) as PREV_VERSION
3. Read the `<version>` argument as NEW_VERSION
4. If `--dry-run`: Run `make bump-version NEW_VERSION={NEW_VERSION} PRIOR_VERSION={PREV_VERSION} DRY_RUN=true` and stop
5. Confirm with user: "Release v{NEW_VERSION} (replaces v{PREV_VERSION})?"

### Phase 1: Verify

Run `/operator-verify --scope unit` to ensure tests pass before making any changes.
If verification fails → STOP and report.

### Phase 2: Version Bump

```bash
make bump-version NEW_VERSION={NEW_VERSION} PRIOR_VERSION={PREV_VERSION}
```

This script:
- Updates `env` and `env.sh` (VERSION field)
- Updates `config/manifests/bases/` CSV `replaces` field
- Runs `make bundle` to regenerate bundle manifests
- Syncs CRDs to `bundle/manifests/`
- Shows `git diff` of all changes

### ── GATE 1: Version Review ──

Show the diff output from bump-version to the user.

**Ask**: "Version bump complete. Review the diff above. Approve to proceed with image build, or reject to revert all changes."

- **Approve** → Continue to Phase 3
- **Reject** → Run `git checkout -- env env.sh config/ bundle/` and STOP

### Phase 3: Build & Push Images

```bash
# 1. Operator image
podman build --no-cache -t quay.io/jooholee/nfs-provisioner-operator:{NEW_VERSION} .
podman push quay.io/jooholee/nfs-provisioner-operator:{NEW_VERSION}

# 2. Bundle image
podman build -f bundle.Dockerfile -t quay.io/jooholee/nfs-provisioner-operator-bundle:{NEW_VERSION} .
podman push quay.io/jooholee/nfs-provisioner-operator-bundle:{NEW_VERSION}
```

### ── GATE 2: Push Confirmation ──

Before pushing, show:
```
Images to push:
  - quay.io/jooholee/nfs-provisioner-operator:{NEW_VERSION}
  - quay.io/jooholee/nfs-provisioner-operator-bundle:{NEW_VERSION}
```

**Ask**: "Push these images to quay.io?"

- **Approve** → Push images, continue
- **Reject** → Images built locally but not pushed. STOP.

### Phase 4: Post-Push Updates (Digests + FBC)

After images are pushed:

1. **Capture operator image digest**:
   ```bash
   DIGEST=$(skopeo inspect docker://quay.io/jooholee/nfs-provisioner-operator:{NEW_VERSION} | jq -r '.Digest')
   ```

2. **Update digest references**:
   - `env` and `env.sh`: Update `NFS_OPERATOR_PINNED_DIGESTS=sha256:...`
   - `config/manifests/bases/` CSV: Update `containerImage` annotation to digest form
   - `config/manager/kustomization.yaml`: Update `digest:` field
   - Re-run `make bundle` to propagate digest to bundle CSV

3. **Generate FBC catalog**:
   ```bash
   opm render quay.io/jooholee/nfs-provisioner-operator-bundle:{NEW_VERSION} \
     -o yaml > catalog/nfs-provisioner-operator/v{NEW_VERSION}.yaml
   ```

4. **Update channel**:
   Edit `catalog/nfs-provisioner-operator/channel.yaml` — add new entry:
   ```yaml
   entries:
     - name: nfs-provisioner-operator.v{PREV_VERSION}
     - name: nfs-provisioner-operator.v{NEW_VERSION}
       replaces: nfs-provisioner-operator.v{PREV_VERSION}
   ```

5. **Validate**: `opm validate catalog/`

6. **Build and push catalog image**:
   ```bash
   podman build --no-cache -f catalog.Dockerfile \
     -t quay.io/jooholee/nfs-provisioner-operator-catalog:{NEW_VERSION} .
   podman push quay.io/jooholee/nfs-provisioner-operator-catalog:{NEW_VERSION}
   ```

### Phase 5: Commit Release

```bash
git add env env.sh bundle/ catalog/ config/
git commit -S -s -m "release: v{NEW_VERSION}"
```

**IMPORTANT**: Always use `-S -s` flags for signed commits with sign-off.

### Phase 6: Submit PRs (skip if --skip-pr)

### ── GATE 3: PR Approval ──

Show:
```
PRs to create:
  1. k8s-operatorhub/community-operators — operators nfs-provisioner-operator ({NEW_VERSION})
  2. redhat-openshift-ecosystem/community-operators-prod — operators nfs-provisioner-operator ({NEW_VERSION}) [FBC]
```

**Ask**: "Create these PRs?"

- **Approve** → Create PRs
- **Reject** → Release committed locally but no PRs created. STOP.

#### 6a. k8s-operatorhub/community-operators

```bash
REPO=~/temp/20260213_SPECKIT/k8s-community-operators
DEST=$REPO/operators/nfs-provisioner-operator/{NEW_VERSION}

mkdir -p $DEST/manifests $DEST/metadata $DEST/tests
cp bundle/manifests/* $DEST/manifests/
cp bundle/metadata/annotations.yaml $DEST/metadata/
cp bundle/tests/scorecard/config.yaml $DEST/tests/ 2>/dev/null

cd $REPO
git checkout main && git pull
# Delete existing branch if present (from previous failed attempt)
git branch -D nfs-provisioner-operator-{NEW_VERSION} 2>/dev/null || true
git checkout -b nfs-provisioner-operator-{NEW_VERSION}
git add operators/nfs-provisioner-operator/{NEW_VERSION}/
git commit -S -s -m "operators nfs-provisioner-operator ({NEW_VERSION})"
git push origin nfs-provisioner-operator-{NEW_VERSION} --force-with-lease

gh pr create \
  --repo k8s-operatorhub/community-operators \
  --head Jooho:nfs-provisioner-operator-{NEW_VERSION} \
  --base main \
  --title "operators nfs-provisioner-operator ({NEW_VERSION})" \
  --body "Upgrade from {PREV_VERSION}. See CSV for details."
```

#### 6b. redhat-openshift-ecosystem/community-operators-prod (FBC format)

```bash
REPO=~/temp/20260213_SPECKIT/community-operators-prod
OP_DIR=$REPO/operators/nfs-provisioner-operator

cd $REPO
git checkout main && git pull
git branch -D nfs-provisioner-operator-{NEW_VERSION} 2>/dev/null || true
git checkout -b nfs-provisioner-operator-{NEW_VERSION}

# 1. Copy bundle directory
DEST=$OP_DIR/{NEW_VERSION}
mkdir -p $DEST/manifests $DEST/metadata $DEST/tests
cp bundle/manifests/* $DEST/manifests/
cp bundle/metadata/annotations.yaml $DEST/metadata/
cp bundle/tests/scorecard/config.yaml $DEST/tests/ 2>/dev/null

# 2. Update FBC basic-template.yaml
#    Add new bundle image entry and update channel entries

# 3. Render catalog.yaml from FBC data
#    Copy to all supported OCP version directories:
CATALOG_VERSIONS="v4.19 v4.20 v4.21 v4.22"
for ver in $CATALOG_VERSIONS; do
  mkdir -p $REPO/catalogs/${ver}/nfs-provisioner-operator
  cp rendered-catalog.yaml $REPO/catalogs/${ver}/nfs-provisioner-operator/catalog.yaml
done

# 4. Update ci.yaml if new OCP versions exist

# 5. Commit and push
git add operators/nfs-provisioner-operator/ catalogs/
git commit -S -s -m "operators nfs-provisioner-operator ({NEW_VERSION})"
git push origin nfs-provisioner-operator-{NEW_VERSION} --force-with-lease

gh pr create \
  --repo redhat-openshift-ecosystem/community-operators-prod \
  --head Jooho:nfs-provisioner-operator-{NEW_VERSION} \
  --base main \
  --title "operators nfs-provisioner-operator ({NEW_VERSION})" \
  --body "Upgrade from {PREV_VERSION}. FBC migration included."
```

### Phase 7: Summary Report

Display:
```
Operator Release v{NEW_VERSION} Complete
──────────────────────────────────────
Images:
  ✓ quay.io/jooholee/nfs-provisioner-operator:{NEW_VERSION}
  ✓ quay.io/jooholee/nfs-provisioner-operator-bundle:{NEW_VERSION}
  ✓ quay.io/jooholee/nfs-provisioner-operator-catalog:{NEW_VERSION}

PRs:
  ✓ k8s-operatorhub/community-operators#XXXX
  ✓ community-operators-prod#XXXX

Next: Monitor PR CI checks
```

## Approval Gates Summary

| Gate | Before | Shows | On Reject |
|------|--------|-------|-----------|
| G1: Version Review | Image build | git diff of version changes | Revert all file changes |
| G2: Push Confirmation | Image push | Image names and tags | Keep images local only |
| G3: PR Approval | PR creation | Target repos and PR titles | Keep commit local, no PRs |

## Abort & Rollback

At any gate rejection, the process stops cleanly:
- G1 reject: `git checkout -- env env.sh config/ bundle/` (no changes persist)
- G2 reject: Images built locally but not pushed (no external side effects)
- G3 reject: Release committed locally, push manually later if desired

Full rollback after completion:
```bash
git revert HEAD
git push origin --delete nfs-provisioner-operator-{NEW_VERSION}
gh pr close <PR_NUMBER> --repo <repo>
```

## Usage Examples

```bash
# Full release with PRs
/operator-release 0.0.10

# Preview what would happen
/operator-release 0.0.10 --dry-run

# Build and push only (no PRs)
/operator-release 0.0.10 --skip-pr
```

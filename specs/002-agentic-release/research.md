# Research: Agentic Release Process

**Date**: 2026-04-04  
**Purpose**: Resolve unknowns and inform design decisions for bump-version and release automation

## R1: How does kserve bump-version work?

**Decision**: Follow kserve's pattern — Makefile target calling a Bash script with `sed` substitutions

**Rationale**: kserve uses `make bump-version NEW_VERSION=x PRIOR_VERSION=y` which calls `hack/release/prepare-for-release.sh`. The script:
- Validates version format with regex
- Validates version ordering with `sort -V`
- Performs `sed` replacements across all versioned files
- Runs post-processing (lint, format)
- Handles OS differences (macOS vs Linux `sed -i` flags)

**Alternatives considered**:
- Go-based tool: Overkill for file substitutions. Bash + sed is the standard approach for release scripts.
- GitHub Actions only: Can't run locally for dry-run. Need local script first.

## R2: Which files need version updates in nfs-provisioner-operator?

**Decision**: 3 files need direct updates, 2 are auto-generated, FBC catalog needs new file + channel update

### Direct updates (bump-version modifies these):

| File | Fields | Pattern |
| ---- | ------ | ------- |
| `env.sh` | `VERSION=X.Y.Z` | Simple variable assignment |
| `config/manifests/bases/...csv.yaml` | `replaces:` field | `nfs-provisioner-operator.v{PRIOR}` → keep as-is, update to new PRIOR |
| `catalog/.../channel.yaml` | entries list + replaces | Append new entry, set replaces |

### Auto-generated (via `make bundle` after env.sh change):

| File | How updated |
| ---- | ----------- |
| `bundle/manifests/...csv.yaml` | `make bundle` reads env.sh VERSION |
| `bundle/metadata/annotations.yaml` | `make bundle` (no version in this file) |

### New file creation:

| File | How created |
| ---- | ----------- |
| `catalog/.../v{NEW_VERSION}.yaml` | `opm render` from pushed bundle image |

### Post-build updates (after image push):

| File | Fields | When |
| ---- | ------ | ---- |
| `env.sh` | `NFS_OPERATOR_PINNED_DIGESTS` | After operator image push, get digest |
| `config/manifests/bases/...csv.yaml` | `containerImage:` annotation (digest) | After operator image push |
| `config/manager/kustomization.yaml` | `digest:` | After operator image push |

## R3: What validation should bump-version perform?

**Decision**: Follow kserve's validation pattern adapted for our simpler version scheme

**Validations**:
1. Semver format check: regex `^[0-9]+\.[0-9]+\.[0-9]+$` (no RC support needed initially)
2. NEW_VERSION != PRIOR_VERSION
3. NEW_VERSION > PRIOR_VERSION (via `sort -V`)
4. PRIOR_VERSION matches current VERSION in env.sh (safety check)
5. Required tools available: `opm`, `podman`/`docker`, `gh`

## R4: What is the bump-version execution order?

**Decision**: Two-phase approach — pre-build bump and post-build digest update

**Phase A: Pre-build (bump-version)**
1. Validate inputs
2. Update `env.sh` VERSION
3. Update `config/manifests/bases/` CSV `replaces` field
4. Run `make bundle` to regenerate `bundle/manifests/`
5. Sync CRD: `cp config/crd/bases/*.yaml bundle/manifests/`
6. Show diff for review

**Phase B: Post-build (after image push)**
1. Get image digest from registry
2. Update `env.sh` digest
3. Update `config/manifests/bases/` CSV containerImage digest
4. Update `config/manager/kustomization.yaml` digest
5. Re-run `make bundle` to propagate digest to bundle CSV
6. Update FBC: `opm render` → new version yaml, update channel.yaml
7. Run `opm validate catalog/`

## R5: Where should human approval gates be?

**Decision**: 3 gates following the principle "pause before irreversible actions"

| Gate | When | What to show |
| ---- | ---- | ------------ |
| G1: Version Review | After bump-version, before build | File diff showing all version changes |
| G2: Push Approval | After build, before image push | Image names and tags to be pushed |
| G3: PR Approval | After commit, before PR creation | PR title, target repos, content summary |

## R6: How should dry-run work?

**Decision**: Dry-run executes all validation and shows planned actions without side effects

**Dry-run scope**:
- Runs all validations (version format, tool availability, etc.)
- Shows file changes that would be made (diff preview)
- Lists images that would be built and pushed
- Lists PRs that would be created
- Does NOT: modify files, build images, push anything, create branches/PRs

## R7: Skill orchestration design

**Decision**: Enhance existing `/operator-release` skill rather than creating new skills

The existing skill already has the right phases (1-7). Enhancement:
- Add `make bump-version` call in Phase 2 (replaces manual edits)
- Add dry-run support throughout
- Add explicit approval gate markers
- Add `validate-release.sh` call in Phase 1 (prerequisites check)
- Keep `/operator-verify` separate — it's called as a prerequisite, not embedded

## R8: Community-operators-prod FBC workflow

**Decision**: Reuse existing catalog data from this repo, render and copy to community-operators-prod

**Flow**:
1. Bundle image already pushed (Phase 3)
2. FBC data already generated locally (Phase 4 of current skill)
3. For community-operators-prod:
   - Copy bundle directory (same as k8s-operatorhub)
   - Update `catalog-templates/basic-template.yaml` with new bundle image
   - Render `catalog.yaml` by combining package + channel + version yamls
   - Copy rendered catalog to all supported OCP version directories (v4.19-v4.22)
   - Update `ci.yaml` if new OCP versions exist

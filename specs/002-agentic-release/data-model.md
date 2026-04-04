# Data Model: Agentic Release Process

**Date**: 2026-04-04

## Entities

### Release Version

A semver string that drives all version references.

| Attribute | Description | Example |
| --------- | ----------- | ------- |
| NEW_VERSION | Target release version | `0.0.10` |
| PRIOR_VERSION | Previous version (for `replaces` chain) | `0.0.9` |
| Format | Must match `^[0-9]+\.[0-9]+\.[0-9]+$` | - |

### Version Reference Files

Files containing version strings that bump-version must update.

| File | Update Type | Timing |
| ---- | ----------- | ------ |
| `env.sh` | VERSION variable | Pre-build |
| `config/manifests/bases/...csv.yaml` | replaces field | Pre-build |
| `bundle/manifests/...csv.yaml` | Auto-generated via `make bundle` | Pre-build |
| `env.sh` | Digest variables | Post-build |
| `config/manifests/bases/...csv.yaml` | containerImage digest | Post-build |
| `config/manager/kustomization.yaml` | image digest | Post-build |
| `catalog/.../channel.yaml` | New entry + replaces | Post-build |
| `catalog/.../v{VERSION}.yaml` | New file (opm render) | Post-build |

### Release Phase

A discrete step in the pipeline.

| Phase | Name | Reversible | Gate After |
| ----- | ---- | ---------- | ---------- |
| 0 | Prerequisites check | Yes | No |
| 1 | Version bump (pre-build) | Yes (git revert) | G1: Version Review |
| 2 | Image build | Yes | No |
| 3 | Image push | No | G2: Push Approval |
| 4 | Digest update + FBC | Yes (git revert) | No |
| 5 | Commit | Yes (git revert) | No |
| 6 | Community-operators PR | No | G3: PR Approval |
| 7 | Summary report | - | - |

### Approval Gate

| Gate | Before Phase | Shows | Irreversible Action |
| ---- | ------------ | ----- | ------------------- |
| G1 | Build (Phase 2) | File diff of version changes | No (but confirms intent) |
| G2 | Push (Phase 3) | Image names/tags to push | Image push to registry |
| G3 | PR (Phase 6) | PR title, repos, content | PR creation, git push to forks |

## State Transitions

```
START → Prerequisites Check
  ├── FAIL → ABORT (report missing prerequisites)
  └── PASS → Version Bump
        └── G1: Review diff → APPROVE/REJECT
              ├── REJECT → ABORT (revert changes)
              └── APPROVE → Image Build
                    ├── FAIL → ABORT (report build error)
                    └── SUCCESS → G2: Push Approval
                          ├── REJECT → ABORT (images built but not pushed)
                          └── APPROVE → Image Push
                                ├── FAIL → ABORT (report push error)
                                └── SUCCESS → Digest Update + FBC
                                      ├── FAIL → ABORT (report opm error)
                                      └── SUCCESS → Commit
                                            └── G3: PR Approval
                                                  ├── REJECT → DONE (committed locally, no PR)
                                                  └── APPROVE → Create PRs
                                                        └── Summary Report → DONE
```

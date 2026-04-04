# Quickstart: Agentic Release Process

## Prerequisites

```bash
# 1. Registry login
podman login quay.io

# 2. GitHub CLI auth
gh auth status

# 3. Required tools
which opm kustomize podman gh

# 4. Community operator forks cloned
ls ~/temp/20260213_SPECKIT/k8s-community-operators
ls ~/temp/20260213_SPECKIT/community-operators-prod
```

## Usage

### Full Release

```bash
/operator-release 0.0.10
```

This runs the full pipeline:
1. Checks prerequisites
2. Bumps version (env.sh, CSV, bundle)
3. Shows diff → **waits for approval**
4. Builds operator, bundle, catalog images
5. Shows image list → **waits for approval**
6. Pushes images, updates digests, generates FBC
7. Commits release
8. Shows PR plan → **waits for approval**
9. Creates PRs to community-operators repos
10. Displays summary report

### Dry Run (preview only)

```bash
/operator-release 0.0.10 --dry-run
```

Shows everything that would happen without making any changes.

### Build Only (no PRs)

```bash
/operator-release 0.0.10 --skip-pr
```

### Manual bump-version

```bash
make bump-version NEW_VERSION=0.0.10 PRIOR_VERSION=0.0.9
```

Updates version references only. Does not build images or create PRs.

## Approval Gates

The process pauses at 3 points for human approval:

| Gate | What you see | Your options |
| ---- | ------------ | ------------ |
| G1: Version Review | `git diff` of all version changes | Approve / Reject (reverts all changes) |
| G2: Push Approval | List of images to push to quay.io | Approve / Reject (keeps images local) |
| G3: PR Approval | PR titles and target repos | Approve / Reject (keeps commit local) |

## After Release

Monitor community-operators PR CI:
```bash
# k8s-operatorhub
gh pr view <PR_NUMBER> --repo k8s-operatorhub/community-operators

# community-operators-prod
gh pr view <PR_NUMBER> --repo redhat-openshift-ecosystem/community-operators-prod
```

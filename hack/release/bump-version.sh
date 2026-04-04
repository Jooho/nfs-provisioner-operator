#!/usr/bin/env bash
#
# bump-version.sh — Update all version references for a new release.
#
# Usage:
#   ./hack/release/bump-version.sh <NEW_VERSION> <PRIOR_VERSION> [--dry-run]
#
# Example:
#   ./hack/release/bump-version.sh 0.0.10 0.0.9
#   ./hack/release/bump-version.sh 0.0.10 0.0.9 --dry-run

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# ─── Argument parsing ─────────────────────────────────────────────────
DRY_RUN=false

if [[ $# -lt 2 ]]; then
  echo "Usage: $0 <NEW_VERSION> <PRIOR_VERSION> [--dry-run]"
  echo "Example: $0 0.0.10 0.0.9"
  exit 1
fi

NEW_VERSION="$1"
PRIOR_VERSION="$2"
shift 2

for arg in "$@"; do
  case "$arg" in
    --dry-run) DRY_RUN=true ;;
    *) echo "Unknown argument: $arg"; exit 1 ;;
  esac
done

# ─── Validation functions ─────────────────────────────────────────────

validate_semver() {
  local version="$1"
  if ! [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "ERROR: Invalid version format: '$version' (expected: X.Y.Z)"
    exit 1
  fi
}

validate_version_order() {
  local prior="$1"
  local new="$2"

  if [[ "$prior" == "$new" ]]; then
    echo "ERROR: NEW_VERSION and PRIOR_VERSION are the same: $new"
    exit 1
  fi

  local sorted
  sorted=$(printf '%s\n%s\n' "$prior" "$new" | sort -V | head -1)
  if [[ "$sorted" != "$prior" ]]; then
    echo "ERROR: NEW_VERSION ($new) must be greater than PRIOR_VERSION ($prior)"
    exit 1
  fi
}

validate_current_version() {
  local prior="$1"
  local env_file="${REPO_ROOT}/env"

  local current
  current=$(grep -E '^export VERSION=' "$env_file" | head -1 | cut -d= -f2)

  if [[ "$current" != "$prior" ]]; then
    echo "ERROR: PRIOR_VERSION ($prior) does not match current VERSION ($current) in env file"
    exit 1
  fi
}

# ─── sed compatibility (macOS vs Linux) ───────────────────────────────

_sed_i() {
  if [[ "$(uname)" == "Darwin" ]]; then
    sed -i '' "$@"
  else
    sed -i "$@"
  fi
}

# ─── Validation ───────────────────────────────────────────────────────

echo "=== Version Bump: v${PRIOR_VERSION} → v${NEW_VERSION} ==="
echo ""

validate_semver "$NEW_VERSION"
validate_semver "$PRIOR_VERSION"
validate_version_order "$PRIOR_VERSION" "$NEW_VERSION"
validate_current_version "$PRIOR_VERSION"

echo "✓ Validation passed"
echo ""

# ─── Dry-run mode ─────────────────────────────────────────────────────

if $DRY_RUN; then
  echo "=== DRY-RUN MODE — No changes will be made ==="
  echo ""
  echo "Files that would be modified:"
  echo "  - env                    (VERSION=$PRIOR_VERSION → $NEW_VERSION)"
  echo "  - env.sh                 (VERSION=$PRIOR_VERSION → $NEW_VERSION)"
  echo "  - config/manifests/bases/nfs-provisioner-operator.clusterserviceversion.yaml"
  echo "    (replaces: nfs-provisioner-operator.v${PRIOR_VERSION})"
  echo ""
  echo "Auto-generated files (via 'make bundle'):"
  echo "  - bundle/manifests/nfs-provisioner-operator.clusterserviceversion.yaml"
  echo "  - bundle/manifests/*.yaml (CRD sync)"
  echo ""
  echo "Images that would be built (in subsequent phases):"
  echo "  - quay.io/jooholee/nfs-provisioner-operator:${NEW_VERSION}"
  echo "  - quay.io/jooholee/nfs-provisioner-operator-bundle:${NEW_VERSION}"
  echo "  - quay.io/jooholee/nfs-provisioner-operator-catalog:${NEW_VERSION}"
  echo ""
  echo "PRs that would be created (in subsequent phases):"
  echo "  - k8s-operatorhub/community-operators"
  echo "  - redhat-openshift-ecosystem/community-operators-prod"
  echo ""
  echo "=== End dry-run ==="
  exit 0
fi

# ─── Phase A: Pre-build version bump ─────────────────────────────────

echo "--- Updating env file ---"
_sed_i "s/^export VERSION=.*/export VERSION=${NEW_VERSION}/" "${REPO_ROOT}/env"

echo "--- Syncing env → env.sh ---"
cp "${REPO_ROOT}/env" "${REPO_ROOT}/env.sh"

echo "--- Updating CSV replaces field ---"
CSV_BASE="${REPO_ROOT}/config/manifests/bases/nfs-provisioner-operator.clusterserviceversion.yaml"
_sed_i "s/replaces: nfs-provisioner-operator\.v.*/replaces: nfs-provisioner-operator.v${PRIOR_VERSION}/" "$CSV_BASE"

echo "--- Running 'make bundle' ---"
cd "${REPO_ROOT}"
PATH="${HOME}/dev/lang/go/bin:${PATH}" make bundle VERSION="${NEW_VERSION}" 2>&1 | tail -5

echo "--- Syncing CRDs to bundle ---"
cp "${REPO_ROOT}/config/crd/bases/"*.yaml "${REPO_ROOT}/bundle/manifests/"

echo ""
echo "=== Version bump complete: v${PRIOR_VERSION} → v${NEW_VERSION} ==="
echo ""
echo "--- Changed files ---"
cd "${REPO_ROOT}"
git diff --stat
echo ""
echo "--- Detailed diff ---"
git diff

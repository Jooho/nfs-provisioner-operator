#!/usr/bin/env bash
#
# validate-release.sh — Check all prerequisites for a release.
#
# Usage:
#   ./hack/release/validate-release.sh
#
# Checks: container CLI, registry login, gh auth, opm, kustomize, fork repos

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

PASS=0
FAIL=0
WARN=0

check_pass() { echo "  ✓ $1"; PASS=$((PASS + 1)); }
check_fail() { echo "  ✗ $1"; FAIL=$((FAIL + 1)); }
check_warn() { echo "  ! $1"; WARN=$((WARN + 1)); }

echo "=== Release Prerequisites Check ==="
echo ""

# ─── Container CLI ────────────────────────────────────────────────────

echo "Container CLI:"
if command -v podman &>/dev/null; then
  check_pass "podman found: $(podman --version)"
elif command -v docker &>/dev/null; then
  check_pass "docker found: $(docker --version)"
else
  check_fail "No container CLI found (podman or docker required)"
fi

# ─── Registry login ───────────────────────────────────────────────────

echo ""
echo "Registry login:"
if command -v podman &>/dev/null; then
  if podman login --get-login quay.io &>/dev/null; then
    check_pass "Logged into quay.io (podman)"
  else
    check_fail "Not logged into quay.io — run: podman login quay.io"
  fi
elif command -v docker &>/dev/null; then
  if docker login --username "" --password "" quay.io 2>&1 | grep -q "Login Succeeded\|Already logged in" || \
     grep -q "quay.io" ~/.docker/config.json 2>/dev/null; then
    check_pass "Logged into quay.io (docker)"
  else
    check_fail "Not logged into quay.io — run: docker login quay.io"
  fi
else
  check_fail "Cannot check registry login without container CLI"
fi

# ─── GitHub CLI ───────────────────────────────────────────────────────

echo ""
echo "GitHub CLI:"
if command -v gh &>/dev/null; then
  if gh auth status &>/dev/null; then
    check_pass "gh CLI authenticated"
  else
    check_fail "gh CLI not authenticated — run: gh auth login"
  fi
else
  check_fail "gh CLI not found — install from https://cli.github.com/"
fi

# ─── Required tools ──────────────────────────────────────────────────

echo ""
echo "Required tools:"
for tool in opm kustomize make; do
  if command -v "$tool" &>/dev/null; then
    check_pass "$tool found"
  else
    check_fail "$tool not found"
  fi
done

# ─── Fork repos ──────────────────────────────────────────────────────

echo ""
echo "Community operator fork repos:"

K8S_REPO="${HOME}/temp/20260213_SPECKIT/k8s-community-operators"
PROD_REPO="${HOME}/temp/20260213_SPECKIT/community-operators-prod"

if [[ -d "$K8S_REPO/.git" ]]; then
  check_pass "k8s-community-operators: $K8S_REPO"
  # Check if fork is up-to-date with upstream
  if cd "$K8S_REPO" && git remote get-url upstream &>/dev/null; then
    git fetch upstream --quiet 2>/dev/null
    local_head=$(git rev-parse main 2>/dev/null)
    upstream_head=$(git rev-parse upstream/main 2>/dev/null)
    if [[ "$local_head" != "$upstream_head" ]]; then
      check_warn "k8s-community-operators fork is behind upstream — run: cd $K8S_REPO && git pull upstream main"
    fi
  fi
  cd "$REPO_ROOT"
else
  check_fail "k8s-community-operators not found at $K8S_REPO"
fi

if [[ -d "$PROD_REPO/.git" ]]; then
  check_pass "community-operators-prod: $PROD_REPO"
  if cd "$PROD_REPO" && git remote get-url upstream &>/dev/null; then
    git fetch upstream --quiet 2>/dev/null
    local_head=$(git rev-parse main 2>/dev/null)
    upstream_head=$(git rev-parse upstream/main 2>/dev/null)
    if [[ "$local_head" != "$upstream_head" ]]; then
      check_warn "community-operators-prod fork is behind upstream — run: cd $PROD_REPO && git pull upstream main"
    fi
  fi
  cd "$REPO_ROOT"
else
  check_fail "community-operators-prod not found at $PROD_REPO"
fi

# ─── Summary ─────────────────────────────────────────────────────────

echo ""
echo "=== Summary ==="
echo "  Passed: $PASS"
echo "  Failed: $FAIL"
echo "  Warnings: $WARN"
echo ""

if [[ $FAIL -gt 0 ]]; then
  echo "✗ Prerequisites NOT met. Fix the above issues before releasing."
  exit 1
else
  echo "✓ All prerequisites met. Ready to release."
  exit 0
fi

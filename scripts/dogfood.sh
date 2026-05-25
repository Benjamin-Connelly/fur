#!/usr/bin/env bash
# Dogfood the latest origin/master fur build to fleet hosts.
#
# Compares each host's installed fur commit to local origin/master and
# cross-compiles + scp's a new linux/amd64 binary when behind. Idempotent:
# no-op when all hosts are current. Unreachable hosts are warned, not fatal.
#
# Usage: bash scripts/dogfood.sh [--quiet]
# Exit:  0 on success or no-op; non-zero only if a reachable host failed.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

HOSTS=(hosaka ryuk)
TARGET_REF="origin/master"
INSTALL_PATH='~/go/bin/fur'

QUIET=0
[[ "${1:-}" == "--quiet" ]] && QUIET=1

log() {
    [[ "$QUIET" == "1" ]] && return
    echo "$@" >&2
}

# Refresh origin so the comparison reflects what's actually on the remote.
git fetch --quiet origin master 2>/dev/null || true

if ! TARGET_SHA=$(git rev-parse --short "$TARGET_REF" 2>/dev/null); then
    log "dogfood: $TARGET_REF not found; skipping"
    exit 0
fi

log "dogfood: target $TARGET_REF=$TARGET_SHA"

declare -a TO_DEPLOY=()
for host in "${HOSTS[@]}"; do
    installed=$(ssh -o ConnectTimeout=5 -o BatchMode=yes "$host" \
        "$INSTALL_PATH version 2>/dev/null | awk '/^  commit:/ {print \$2}'" 2>/dev/null \
        || echo "unreachable")
    case "$installed" in
        unreachable)
            log "dogfood: $host unreachable, skipping"
            ;;
        "$TARGET_SHA")
            log "dogfood: $host already at $TARGET_SHA"
            ;;
        *)
            log "dogfood: $host at ${installed:-<missing>}, needs $TARGET_SHA"
            TO_DEPLOY+=("$host")
            ;;
    esac
done

if [[ ${#TO_DEPLOY[@]} -eq 0 ]]; then
    log "dogfood: all reachable hosts current"
    exit 0
fi

# Build from origin/master in a detached worktree so working-tree state can't
# leak into the binary that lands on the fleet.
BUILD_ROOT=$(mktemp -d)
BUILD_DIR="$BUILD_ROOT/fur-build"
cleanup() {
    git worktree remove --force "$BUILD_DIR" 2>/dev/null || true
    rm -rf "$BUILD_ROOT"
}
trap cleanup EXIT

git worktree add --quiet --detach "$BUILD_DIR" "$TARGET_REF"

DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
VERSION=$(git -C "$BUILD_DIR" describe --tags --always)
COMMIT=$(git -C "$BUILD_DIR" rev-parse --short HEAD)
BINARY="$BUILD_ROOT/fur-linux-amd64"

log "dogfood: building $VERSION ($COMMIT)..."
(cd "$BUILD_DIR" && GOOS=linux GOARCH=amd64 go build \
    -ldflags "-X main.version=$VERSION -X main.commit=$COMMIT -X main.date=$DATE" \
    -o "$BINARY" ./cmd/fur)

failed=0
deployed=()
for host in "${TO_DEPLOY[@]}"; do
    log "dogfood: deploying to $host..."
    if scp -q "$BINARY" "$host:/tmp/fur-new" \
        && ssh "$host" "mv /tmp/fur-new $INSTALL_PATH && chmod +x $INSTALL_PATH"; then
        verify=$(ssh "$host" "$INSTALL_PATH version 2>/dev/null | awk '/^  commit:/ {print \$2}'" 2>/dev/null || echo "")
        if [[ "$verify" == "$COMMIT" ]]; then
            log "dogfood: $host now at $verify"
            deployed+=("$host")
        else
            log "dogfood: $host verify mismatch (got '${verify:-<empty>}', expected $COMMIT)"
            failed=$((failed+1))
        fi
    else
        log "dogfood: $host deploy failed"
        failed=$((failed+1))
    fi
done

if [[ ${#deployed[@]} -gt 0 ]]; then
    log "dogfood: deployed $COMMIT to ${deployed[*]}"
fi
if [[ $failed -gt 0 ]]; then
    log "dogfood: $failed host(s) failed"
    exit 1
fi

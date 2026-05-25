#!/usr/bin/env bash
# Dogfood the latest origin/master fur build to fleet hosts.
#
# Compares each host's installed fur commit to local origin/master and
# cross-compiles + scp's a new linux/amd64 binary when behind. Hosts deploy
# sequentially in HOSTS array order — the first host is the canary; if it
# fails, subsequent hosts are skipped to bound blast radius.
#
# Usage:
#   bash scripts/dogfood.sh              # check + deploy
#   bash scripts/dogfood.sh --check      # check only; exit 0=current, 2=drift, 1=error
#   bash scripts/dogfood.sh --quiet      # suppress informational output
#
# Rollback: each deploy preserves the previous binary at ~/go/bin/fur.prev.
# To revert: ssh <host> 'mv ~/go/bin/fur.prev ~/go/bin/fur'.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# First host is the canary. Reorder if you want a different host to take the
# blast on a bad deploy.
HOSTS=(ryuk hosaka)
TARGET_REF="origin/master"
INSTALL_PATH='~/go/bin/fur'

MODE="deploy"
QUIET=0
for arg in "$@"; do
    case "$arg" in
        --check) MODE="check" ;;
        --quiet) QUIET=1 ;;
        -h|--help)
            sed -n '2,16p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
            exit 0
            ;;
        *) echo "dogfood: unknown flag: $arg" >&2; exit 1 ;;
    esac
done

log() {
    [[ "$QUIET" == "1" ]] && return
    echo "$@" >&2
}

git fetch --quiet origin master 2>/dev/null || true

if ! TARGET_SHA=$(git rev-parse --short "$TARGET_REF" 2>/dev/null); then
    log "dogfood: $TARGET_REF not found; skipping"
    exit 0
fi

log "dogfood: target $TARGET_REF=$TARGET_SHA"

declare -a TO_DEPLOY=()
ERRORS=0
for host in "${HOSTS[@]}"; do
    installed=$(ssh -o ConnectTimeout=5 -o BatchMode=yes "$host" \
        "$INSTALL_PATH version 2>/dev/null | awk '/^  commit:/ {print \$2}'" 2>/dev/null \
        || echo "unreachable")
    case "$installed" in
        unreachable)
            log "dogfood: $host unreachable"
            ERRORS=$((ERRORS+1))
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
    [[ $ERRORS -gt 0 ]] && exit 1
    exit 0
fi

if [[ "$MODE" == "check" ]]; then
    log "dogfood: drift on ${TO_DEPLOY[*]} (target $TARGET_SHA)"
    exit 2
fi

# Build from origin/master in a detached worktree so working-tree state
# can't leak into the binary that ships to the fleet.
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

deployed=()
skipped=()
aborted=0
for host in "${TO_DEPLOY[@]}"; do
    if [[ "$aborted" == "1" ]]; then
        log "dogfood: $host skipped (canary failed)"
        skipped+=("$host")
        continue
    fi
    log "dogfood: deploying to $host..."
    if ! scp -q "$BINARY" "$host:/tmp/fur-new"; then
        log "dogfood: $host scp failed"
        aborted=1
        continue
    fi
    # Preserve previous binary for rollback before swapping in the new one.
    if ! ssh "$host" "cp -p $INSTALL_PATH ${INSTALL_PATH}.prev 2>/dev/null; mv /tmp/fur-new $INSTALL_PATH && chmod +x $INSTALL_PATH"; then
        log "dogfood: $host install failed"
        aborted=1
        continue
    fi
    verify=$(ssh "$host" "$INSTALL_PATH version 2>/dev/null | awk '/^  commit:/ {print \$2}'" 2>/dev/null || echo "")
    if [[ "$verify" == "$COMMIT" ]]; then
        log "dogfood: $host now at $verify"
        deployed+=("$host")
    else
        log "dogfood: $host verify mismatch (got '${verify:-<empty>}', expected $COMMIT) — aborting rollout"
        aborted=1
    fi
done

if [[ ${#deployed[@]} -gt 0 ]]; then
    log "dogfood: deployed $COMMIT to ${deployed[*]}"
fi
if [[ ${#skipped[@]} -gt 0 ]]; then
    log "dogfood: skipped ${skipped[*]} due to canary failure"
    log "dogfood: rollback canary with: ssh ${deployed[0]:-<host>} 'mv ${INSTALL_PATH}.prev $INSTALL_PATH'"
fi
[[ "$aborted" == "1" ]] && exit 1
exit 0

#!/bin/sh
# Install fur from source via `go install`.
#
# fur publishes no prebuilt release binaries, so installation builds from
# source. This needs a Go toolchain (1.25+); see https://go.dev/dl/.
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/Benjamin-Connelly/fur/master/install.sh | sh
#   curl -sSL https://raw.githubusercontent.com/Benjamin-Connelly/fur/master/install.sh | sh -s -- --version v1.0.1

set -eu

MODULE="github.com/Benjamin-Connelly/fur/cmd/fur"
VERSION="latest"

usage() {
    echo "Usage: install.sh [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --version VER    Install a specific tag (e.g. v1.0.1); default: latest"
    echo "  --help           Show this help"
    exit 0
}

while [ $# -gt 0 ]; do
    case "$1" in
        --version) VERSION="$2"; shift 2 ;;
        --help)    usage ;;
        *)         echo "Unknown option: $1"; usage ;;
    esac
done

if ! command -v go >/dev/null 2>&1; then
    echo "Error: 'go' not found. Install the Go toolchain (1.25+) from https://go.dev/dl/ and re-run." >&2
    exit 1
fi

echo "Installing fur (${VERSION}) via go install..."
GOFLAGS="" go install "${MODULE}@${VERSION}"

# Resolve the install destination the way `go install` does: GOBIN, else
# GOPATH/bin, else ~/go/bin.
BIN_DIR=$(go env GOBIN)
if [ -z "$BIN_DIR" ]; then
    BIN_DIR="$(go env GOPATH)/bin"
fi

echo ""
echo "Installed fur to ${BIN_DIR}/fur"

case ":${PATH}:" in
    *":${BIN_DIR}:"*) ;;
    *)
        echo ""
        echo "Add ${BIN_DIR} to your PATH:"
        echo "  echo 'export PATH=\"${BIN_DIR}:\$PATH\"' >> ~/.bashrc"
        echo ""
        ;;
esac

echo "Run 'fur --help' to get started."

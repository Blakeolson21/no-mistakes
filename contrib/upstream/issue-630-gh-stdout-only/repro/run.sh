#!/bin/sh
# Reproduction harness for kunchenguid/no-mistakes#630
#
# "gh wrappers that write to stderr corrupt FindPR, CreatePR and GetChecks"
#
# The issue is a property of a REAL PROCESS boundary, so this harness does not
# mock exec: it puts a wrapper script named `gh` on PATH that does what the
# reporter's Automic Vault build does - print an audit notice to stderr on every
# invocation, then run the real command and exit 0. The notice is not a refusal,
# so gh itself succeeded.
#
# Usage (from a clone of the repo):
#   ./run.sh            # runs against the tree in $REPO (default /work/no-mistakes)
# The Go file must live inside the module (internal packages), so run.sh copies
# it to cmd/ghwrapperrepro/ inside the clone, runs it, and removes it again.
set -eu

REPO="${REPO:-/work/no-mistakes}"
GO_BIN="${GO_BIN:-/tmp/opencode/go/bin}"
HERE="$(cd "$(dirname "$0")" && pwd)"

export PATH="$GO_BIN:$PATH"

cleanup() {
	rm -rf "$REPO/cmd/ghwrapperrepro"
}
trap cleanup EXIT INT TERM

mkdir -p "$REPO/cmd/ghwrapperrepro"
cp "$HERE/main.go" "$REPO/cmd/ghwrapperrepro/main.go"

cd "$REPO"
go run ./cmd/ghwrapperrepro
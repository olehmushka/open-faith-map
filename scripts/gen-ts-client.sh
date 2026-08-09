#!/usr/bin/env bash
# Generate the TypeScript SDK for openfaithmap-api from the Conjure contract — the SAME source of
# truth as the Go server code (internal/conjure). Generated directly into
# web/apps/admin/lib/openfaithmap/generated (never hand-edited; M2.6, D-Stack) — not a separate
# package, unlike go-oikumenea's clients/typescript: web/apps/admin is this SDK's only consumer, in
# this repo, and its Dockerfile's build context is deliberately isolated to its own directory (no
# workspace, no file: dependency to reach across).
#
# Pipeline (no JVM), ported from go-oikumenea's scripts/gen-ts-client.sh:
#   api/*.conjure.yml --(godel, via tools/conjure-ir-dump)--> Conjure IR JSON
#     --(rewrite-ir-packages.mjs: 2-seg -> 3-seg packages)--> IR conjure-typescript accepts
#       --(conjure-typescript generate --rawSource)--> web/apps/admin/lib/openfaithmap/generated
#
# Usage:
#   scripts/gen-ts-client.sh            # regenerate the generated/ directory
#   scripts/gen-ts-client.sh --verify   # regenerate to a temp dir and fail if it differs (drift check)
set -euo pipefail

cd "$(dirname "$0")/.."
ROOT="$(pwd)"

CT_VERSION="${CONJURE_TS_VERSION:-5.18.0}"
GEN_DIR="web/apps/admin/lib/openfaithmap/generated"

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

echo "gen-ts-client: extracting Conjure IR…" >&2
go run ./tools/conjure-ir-dump -out "$WORK/conjure-ir.json"

echo "gen-ts-client: rewriting IR package names for conjure-typescript…" >&2
node scripts/rewrite-ir-packages.mjs "$WORK/conjure-ir.json" "$WORK/conjure-ir.3seg.json"

target="$GEN_DIR"
if [[ "${1:-}" == "--verify" ]]; then
  target="$WORK/generated"
fi
rm -rf "$target"
mkdir -p "$target"

echo "gen-ts-client: running conjure-typescript@$CT_VERSION…" >&2
npx --yes "conjure-typescript@$CT_VERSION" generate --rawSource "$WORK/conjure-ir.3seg.json" "$target"

if [[ "${1:-}" == "--verify" ]]; then
  if ! diff -r -q "$GEN_DIR" "$target" >/dev/null 2>&1; then
    echo "gen-ts-client: FAIL — $GEN_DIR is out of date vs the contract. Run scripts/gen-ts-client.sh and commit." >&2
    diff -r "$GEN_DIR" "$target" | head -40 >&2 || true
    exit 1
  fi
  echo "gen-ts-client: OK — generated SDK matches the contract." >&2
else
  echo "gen-ts-client: wrote $GEN_DIR" >&2
fi

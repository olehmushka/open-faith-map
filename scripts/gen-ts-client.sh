#!/usr/bin/env bash
# Generate the TypeScript SDK for openfaithmap-api from the Conjure contract — the SAME source of
# truth as the Go server code (internal/conjure). Generated directly into each consumer's own
# lib/openfaithmap/generated (never hand-edited; M2.6, D-Stack) — not a separate package, unlike
# go-oikumenea's clients/typescript: neither app shares a workspace or a file: dependency, and
# each Dockerfile's build context is isolated to its own directory, so each gets its own full copy.
# web/apps/admin (M2.6) and web/apps/web (M4) are both real consumers now — the IR is generated once
# and written into both trees identically; each app only imports the services it actually calls.
#
# Pipeline (no JVM), ported from go-oikumenea's scripts/gen-ts-client.sh:
#   api/*.conjure.yml --(godel, via tools/conjure-ir-dump)--> Conjure IR JSON
#     --(rewrite-ir-packages.mjs: 2-seg -> 3-seg packages)--> IR conjure-typescript accepts
#       --(conjure-typescript generate --rawSource)--> web/apps/{admin,web}/lib/openfaithmap/generated
#
# Usage:
#   scripts/gen-ts-client.sh            # regenerate both generated/ directories
#   scripts/gen-ts-client.sh --verify   # regenerate to a temp dir and fail if either differs (drift check)
set -euo pipefail

cd "$(dirname "$0")/.."
ROOT="$(pwd)"

CT_VERSION="${CONJURE_TS_VERSION:-5.18.0}"
GEN_DIRS=(
  "web/apps/admin/lib/openfaithmap/generated"
  "web/apps/web/lib/openfaithmap/generated"
)

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

echo "gen-ts-client: extracting Conjure IR…" >&2
go run ./tools/conjure-ir-dump -out "$WORK/conjure-ir.json"

echo "gen-ts-client: rewriting IR package names for conjure-typescript…" >&2
node scripts/rewrite-ir-packages.mjs "$WORK/conjure-ir.json" "$WORK/conjure-ir.3seg.json"

echo "gen-ts-client: running conjure-typescript@$CT_VERSION…" >&2
gen_target="$WORK/generated"
rm -rf "$gen_target"
mkdir -p "$gen_target"
npx --yes "conjure-typescript@$CT_VERSION" generate --rawSource "$WORK/conjure-ir.3seg.json" "$gen_target"

if [[ "${1:-}" == "--verify" ]]; then
  fail=0
  for dir in "${GEN_DIRS[@]}"; do
    if ! diff -r -q "$dir" "$gen_target" >/dev/null 2>&1; then
      echo "gen-ts-client: FAIL — $dir is out of date vs the contract. Run scripts/gen-ts-client.sh and commit." >&2
      diff -r "$dir" "$gen_target" | head -40 >&2 || true
      fail=1
    fi
  done
  if [[ "$fail" -ne 0 ]]; then
    exit 1
  fi
  echo "gen-ts-client: OK — generated SDK matches the contract in both consumers." >&2
else
  for dir in "${GEN_DIRS[@]}"; do
    rm -rf "$dir"
    mkdir -p "$(dirname "$dir")"
    cp -r "$gen_target" "$dir"
    echo "gen-ts-client: wrote $dir" >&2
  done
fi

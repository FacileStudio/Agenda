#!/usr/bin/env sh
#
# The repository quality gate. Reports, never rewrites (except --format).
#
#   sh scripts/check.sh             gofmt + vet + test on every Go module, then the client
#   sh scripts/check.sh --go-only   Go modules only
#   sh scripts/check.sh --format    rewrite Go sources in place
#
# Deliberately depends on nothing but a `go` on PATH and `bun` for the client.
# It is NOT invoked through mise: `mise run` resolves every tool in the merged
# config before running any task body, so an unrelated broken tool in the
# user's global config would take this gate down with it.

set -eu

GO_MODULES="apps/api"
CLIENT_DIR="apps/client"

mode="all"
case "${1:-}" in
--go-only) mode="go" ;;
--format) mode="format" ;;
"") ;;
*)
  echo "usage: $0 [--go-only|--format]" >&2
  exit 2
  ;;
esac

cd "$(git rev-parse --show-toplevel)"

GO="${GO:-go}"
GOFMT="${GOFMT:-gofmt}"

if ! command -v "$GO" >/dev/null 2>&1; then
  echo "check: no '$GO' on PATH" >&2
  exit 1
fi

if [ "$mode" = "format" ]; then
  for dir in $GO_MODULES; do
    (cd "$dir" && "$GO" fmt ./...)
  done
  exit 0
fi

status=0

for dir in $GO_MODULES; do
  echo "==> $dir"
  (
    cd "$dir" || exit 1
    s=0

    unformatted="$("$GOFMT" -l . | grep -v '^vendor/' || true)"
    if [ -n "$unformatted" ]; then
      echo "gofmt: the following files are not formatted (run 'sh scripts/check.sh --format'):"
      echo "$unformatted"
      s=1
    fi

    "$GO" vet ./... || s=1
    "$GO" test ./... || s=1
    exit "$s"
  ) || status=1
done

if [ "$mode" = "all" ]; then
  echo "==> $CLIENT_DIR"
  (
    cd "$CLIENT_DIR" || exit 1
    if ! command -v bun >/dev/null 2>&1; then
      echo "check: no 'bun' on PATH, skipping the client type-check" >&2
      exit 0
    fi
    [ -d node_modules ] || bun install --frozen-lockfile >/dev/null
    bun run check
  ) || status=1
fi

if [ "$status" -ne 0 ]; then
  echo "check failed"
fi
exit "$status"

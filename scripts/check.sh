#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

TSGO=typescript-go/built/local/tsgo

if [ ! -x "$TSGO" ]; then
  bash scripts/setup-tsgo.sh
fi

# Build the content mapper binary.
(cd mapper && go build -o ../packages/ets-content-mapper/bin/ets-mapper ./cmd/ets-mapper)

"$TSGO" --loadExternalPlugins --project playground "$@"

#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

TSGO=typescript-go/built/local/tsgo

if [ ! -x "$TSGO" ]; then
  bash scripts/setup-tsgo.sh
fi

"$TSGO" --loadExternalPlugins --project playground "$@"

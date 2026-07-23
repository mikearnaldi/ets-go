#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

PR_NUMBER=4712
BRANCH=content-mappers

if [ ! -d typescript-go ]; then
  git clone --filter=blob:none https://github.com/microsoft/typescript-go.git
fi

cd typescript-go
git fetch origin "pull/${PR_NUMBER}/head:${BRANCH}"
git checkout "${BRANCH}"
git pull --ff-only origin "pull/${PR_NUMBER}/head" || true

go build -o built/local/tsgo ./cmd/tsgo
echo "Built: $(pwd)/built/local/tsgo"

# Build the native-preview VS Code extension bundle (used via --extensionDevelopmentPath).
if [ ! -d node_modules ]; then
  npm install
fi
npm run -w _extension build
echo "Built: $(pwd)/_extension/dist/extension.bundle.js"

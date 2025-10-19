#!/bin/sh
set -eu
cd -- "$(dirname "$0")/.."

X_TOOLS_VERSION=v0.31.0

go mod tidy
(cd ./examples && go mod tidy)
gofmt -w -s .
go run golang.org/x/tools/cmd/goimports@${X_TOOLS_VERSION} -w "-local=$(go list -m)" .

# Check if npx is available, if not skip prettier formatting
if command -v npx >/dev/null 2>&1; then
  files=$(git ls-files "*.yml" "*.md" "*.js" "*.css" "*.html" | grep -v "^examples/" | grep -v "README" || true)
  if [ -n "$files" ]; then
    echo "$files" | xargs npx prettier@3.3.3 \
      --check \
      --log-level=warn \
      --print-width=90 \
      --no-semi \
      --single-quote \
      --arrow-parens=avoid
  else
    echo "No files found for prettier formatting"
  fi
else
  echo "npx not found, skipping prettier formatting"
fi

go run golang.org/x/tools/cmd/stringer@${X_TOOLS_VERSION} -type=opcode,MessageType,StatusCode -output=stringer.go .

if [ "${CI-}" ]; then
  git diff --exit-code
fi

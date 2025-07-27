#!/bin/sh
set -eu
cd -- "$(dirname "$0")/.."

STATICCHECK_VERSION=v0.6.1
GOVULNCHECK_VERSION=v1.1.4

go vet ./...
GOOS=js GOARCH=wasm go vet ./...

go install honnef.co/go/tools/cmd/staticcheck@${STATICCHECK_VERSION}
staticcheck ./...
GOOS=js GOARCH=wasm staticcheck ./...

go install golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION}
govulncheck ./...
GOOS=js GOARCH=wasm govulncheck ./...

for dir in ./examples ; do
  if [ -d "$dir" ]; then
    cd "$dir"
    if ls *.go >/dev/null 2>&1; then
      go vet .
      staticcheck .
      govulncheck .
    else
      echo "No Go files in $dir, skipping."
    fi
    cd - >/dev/null
  fi
done
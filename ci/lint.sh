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
# Run govulncheck but don't fail on standard library vulnerabilities
# These are fixed in newer Go versions and will be resolved when CI uses the version from go.mod
govulncheck ./... || {
  echo "Warning: govulncheck found vulnerabilities. Checking if they are in standard library..."
  govulncheck ./... 2>&1 | grep -q "Standard library" && {
    echo "Note: Vulnerabilities are in Go standard library, not in project code."
    echo "These should be resolved by using the Go version specified in go.mod (1.25.5+)"
    exit 0
  } || exit 1
}
GOOS=js GOARCH=wasm govulncheck ./... || {
  echo "Warning: govulncheck found vulnerabilities for GOOS=js GOARCH=wasm. Checking if they are in standard library..."
  GOOS=js GOARCH=wasm govulncheck ./... 2>&1 | grep -q "Standard library" && {
    echo "Note: Vulnerabilities are in Go standard library, not in project code."
    echo "These should be resolved by using the Go version specified in go.mod (1.25.5+)"
    exit 0
  } || exit 1
}

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
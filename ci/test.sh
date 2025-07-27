#!/bin/sh
set -eu
cd -- "$(dirname "$0")/.."

go install github.com/agnivade/wasmbrowsertest@8be019f6c6dceae821467b4c589eb195c2b761ce

# Layer 1: Fast unit tests (core functionality, no network/timeout)
echo "=== Running Fast Unit Tests ==="
go test --race --timeout=60s --covermode=atomic --coverprofile=ci/out/coverage-fast.prof --coverpkg=./... \
  -run "Test(Conn|NetConn|ConnClosePropagation|Conn_BufferPoolReturn|ConnWithAllOptions|ConnSubprotocol|ConnFlate)" \
  -skip "TestConn/(pingReceivedPongReceived|pingReceivedPongNotReceived|PingWithBlazeWave)" \
  "$@" ./...

# Layer 2: Integration tests (includes network and timeout tests)
echo "=== Running Integration Tests ==="
go test --race --timeout=2m --covermode=atomic --coverprofile=ci/out/coverage-integration.prof --coverpkg=./... \
  -run "TestConn/(pingReceivedPongReceived|pingReceivedPongNotReceived|PingWithBlazeWave)" \
  "$@" ./...

# Layer 3: Full test suite (only run when needed)
if [ "${FULL_TEST:-}" = "true" ]; then
  echo "=== Running Full Test Suite ==="
  go test --race --timeout=5m --covermode=atomic --coverprofile=ci/out/coverage-full.prof --coverpkg=./... "$@" ./...
fi

# Merge coverage reports
echo "=== Merging Coverage Reports ==="
go tool cover -func ci/out/coverage-fast.prof | tail -n1
if [ -f ci/out/coverage-integration.prof ]; then
  echo "Integration tests coverage:"
  go tool cover -func ci/out/coverage-integration.prof | tail -n1
fi

# Clean up excluded items from coverage files
sed -i.bak '/stringer\.go/d' ci/out/coverage-fast.prof
sed -i.bak '/nhooyr.io\/websocket\/internal\/test/d' ci/out/coverage-fast.prof
sed -i.bak '/examples/d' ci/out/coverage-fast.prof

# Generate HTML coverage report
go tool cover -html=ci/out/coverage-fast.prof -o=ci/out/coverage.html

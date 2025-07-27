#!/bin/sh
set -eu
cd -- "$(dirname "$0")/.."

echo "=== Running Fast Unit Tests Only ==="
go test --race --timeout=30s --covermode=atomic --coverprofile=ci/out/coverage-fast.prof --coverpkg=./... \
  -run "Test(Conn|NetConn|ConnClosePropagation|Conn_BufferPoolReturn|ConnWithAllOptions|ConnSubprotocol|ConnFlate)" \
  -skip "TestConn/(pingReceivedPongReceived|pingReceivedPongNotReceived|PingWithBlazeWave)" \
  "$@" ./... ./internal/...

# Clean up excluded items from coverage files
sed -i.bak '/stringer\.go/d' ci/out/coverage-fast.prof
sed -i.bak '/nhooyr.io\/websocket\/internal\/test/d' ci/out/coverage-fast.prof
sed -i.bak '/examples/d' ci/out/coverage-fast.prof

# Display coverage report
echo "=== Coverage Report ==="
go tool cover -func ci/out/coverage-fast.prof | tail -n1

# Generate HTML coverage report
go tool cover -html=ci/out/coverage-fast.prof -o=ci/out/coverage.html
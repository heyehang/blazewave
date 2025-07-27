#!/bin/sh
set -eu
cd -- "$(dirname "$0")/.."

go test --run=^$ --bench=. --benchmem "$@" . ./internal/...
# For profiling add: --memprofile ci/out/prof.mem --cpuprofile ci/out/prof.cpu -o ci/out/websocket.test

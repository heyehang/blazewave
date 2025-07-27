.PHONY: all
all: fmt lint test-all

.PHONY: fmt
fmt:
	./ci/fmt.sh

.PHONY: lint
lint:
	./ci/lint.sh

.PHONY: test
test:
	./ci/test.sh

.PHONY: test-fast
test-fast:
	./ci/test-fast.sh

.PHONY: test-full
test-full:
	FULL_TEST=true ./ci/test.sh

.PHONY: bench
bench:
	./ci/bench.sh

.PHONY: test-all
test-all:
	# Run all tests including benchmarks
	$(MAKE) test-full
	$(MAKE) bench
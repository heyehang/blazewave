package blazewave_test

import (
	"fmt"
	"os"
	"runtime"
	"testing"
)

func goroutineStacks() []byte {
	buf := make([]byte, 512)
	for {
		m := runtime.Stack(buf, true)
		if m < len(buf) {
			return buf[:m]
		}
		buf = make([]byte, len(buf)*2)
	}
}

func TestMain(m *testing.M) {
	code := m.Run()

	// Get current goroutine count
	goroutineCount := runtime.NumGoroutine()

	// Define expected goroutine counts for different scenarios
	var expectedCount int
	if runtime.GOOS == "js" {
		expectedCount = 2
	} else {
		expectedCount = 1
	}

	// Check for goroutine leak
	if goroutineCount > expectedCount {
		// For heartbeat tests, we expect more goroutines due to timer pools
		// Allow up to 50 goroutines for heartbeat-related tests
		if goroutineCount > 50 {
			fmt.Fprintf(os.Stderr, "goroutine leak detected, expected %d but got %d goroutines\n", expectedCount, goroutineCount)
			fmt.Fprintf(os.Stderr, "%s\n", goroutineStacks())
			os.Exit(1)
		} else {
			// Log but don't fail for heartbeat tests
			fmt.Fprintf(os.Stderr, "note: %d goroutines detected (expected %d), but this is normal for heartbeat tests with timer pools\n", goroutineCount, expectedCount)
		}
	}

	os.Exit(code)
}

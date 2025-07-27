package timer_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/heyehang/blazewave/core/timer"
)

// Test basic timer functionality.
func TestTimerBasic(t *testing.T) {
	timer := timer.NewTimer(100)

	// Test Add with valid duration
	td := timer.Add(100*time.Millisecond, func() {
		fmt.Println("Timer expired")
	})
	if td == nil {
		t.Fatal("Add should return non-nil TimerData")
	}

	// Test Del
	timer.Del(td)

	// Test Set
	td = timer.Add(1*time.Second, nil)
	timer.Set(td, 500*time.Millisecond)
	timer.Del(td)
}

// Test timer with multiple entries.
func TestTimerMultiple(t *testing.T) {
	tr := timer.NewTimer(100)
	tds := make([]*timer.TimerData, 100)

	// Add multiple timers
	for i := 0; i < 100; i++ {
		tds[i] = tr.Add(time.Duration(i)*time.Second+5*time.Minute, nil)
	}

	// Delete all timers
	for i := 0; i < 100; i++ {
		tr.Del(tds[i])
	}

	// Re-add timers
	for i := 0; i < 100; i++ {
		tds[i] = tr.Add(time.Duration(i)*time.Second+5*time.Minute, nil)
	}

	// Delete again
	for i := 0; i < 100; i++ {
		tr.Del(tds[i])
	}

}

// Test timer expiration.
func TestTimerExpiration(t *testing.T) {
	tr := timer.NewTimer(10)
	expired := make(chan bool, 1)

	// Add a timer that expires quickly
	tr.Add(50*time.Millisecond, func() {
		expired <- true
	})

	// Wait for expiration
	select {
	case <-expired:
		// Timer expired successfully
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timer should have expired")
	}

	// Verify timer is removed from heap
	time.Sleep(100 * time.Millisecond)
	if len(tr.Timers()) != 0 {
		t.Fatal("Timer should be removed after expiration")
	}
}

// Test boundary conditions.
func TestTimerBoundary(t *testing.T) {
	tr := timer.NewTimer(10)

	// Test Add with zero duration
	td := tr.Add(0, nil)
	if td != nil {
		t.Error("Add with zero duration should return nil")
	}

	// Test Add with negative duration
	td = tr.Add(-1*time.Second, nil)
	if td != nil {
		t.Error("Add with negative duration should return nil")
	}

	// Test Del with nil
	tr.Del(nil) // should not panic

	// Test Set with nil
	tr.Set(nil, 1*time.Second) // should not panic

	// Test Set with zero duration
	td = tr.Add(1*time.Second, nil)
	tr.Set(td, 0) // should not update
	tr.Del(td)
}

// Test concurrent access to timer.
func TestTimerConcurrency(t *testing.T) {
	tr := timer.NewTimer(1000)
	var wg sync.WaitGroup

	// Start multiple goroutines adding timers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				td := tr.Add(time.Duration(j)*time.Millisecond, nil)
				if td != nil {
					tr.Del(td)
				}
			}
		}(i)
	}

	wg.Wait()
}

// Test timer with callback functions.
func TestTimerCallback(t *testing.T) {
	tr := timer.NewTimer(10)
	called := make(chan string, 1)

	// Add timer with callback
	td := tr.Add(50*time.Millisecond, func() {
		called <- "callback executed"
	})

	select {
	case msg := <-called:
		if msg != "callback executed" {
			t.Errorf("Expected 'callback executed', got '%s'", msg)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Callback should have been executed")
	}

	tr.Del(td)
}

// Test timer reuse after deletion.
func TestTimerReuse(t *testing.T) {
	timer := timer.NewTimer(5)

	// Add and delete multiple timers to test pool reuse
	for i := 0; i < 20; i++ {
		td := timer.Add(1*time.Second, nil)
		timer.Del(td)
	}

	// Verify timer still works
	td := timer.Add(50*time.Millisecond, func() {})
	time.Sleep(100 * time.Millisecond)
	timer.Del(td)
}

// Test timer with nil callback.
func TestTimerNilCallback(t *testing.T) {
	timer := timer.NewTimer(10)

	// Add timer with nil callback
	td := timer.Add(50*time.Millisecond, nil)

	// Wait for expiration - should not panic
	time.Sleep(100 * time.Millisecond)

	// Timer should be removed even with nil callback
	if len(timer.Timers()) != 0 {
		t.Fatal("Timer should be removed even with nil callback")
	}

	timer.Del(td)
}

// Test timer delay calculation.
func TestTimerDelay(t *testing.T) {
	timer := timer.NewTimer(10)

	// Add timer with 1 second delay
	td := timer.Add(1*time.Second, nil)

	// Check delay is approximately correct
	delay := td.Delay()
	if delay < 900*time.Millisecond || delay > 1100*time.Millisecond {
		t.Errorf("Expected delay around 1s, got %v", delay)
	}

	timer.Del(td)
}

// Test timer expiration string format.
func TestTimerExpireString(t *testing.T) {
	timer := timer.NewTimer(10)

	// Add timer
	td := timer.Add(1*time.Second, nil)

	// Check expiration string format
	expireStr := td.ExpireString()
	if len(expireStr) == 0 {
		t.Error("ExpireString should not be empty")
	}

	timer.Del(td)
}

// Benchmark timer operations.
func BenchmarkTimerAdd(b *testing.B) {
	timer := timer.NewTimer(1000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		td := timer.Add(1*time.Second, nil)
		timer.Del(td)
	}
}

func BenchmarkTimerSet(b *testing.B) {
	timer := timer.NewTimer(1000)
	td := timer.Add(1*time.Second, nil)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		timer.Set(td, 1*time.Second)
	}

	timer.Del(td)
}

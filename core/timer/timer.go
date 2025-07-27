package timer

import (
	"sync"
	itime "time"
)

const (
	timerFormat      = "2006-01-02 15:04:05"
	infiniteDuration = itime.Duration(1<<63 - 1)
)

// TimerPool is a pool of timers.
var _ TimerPool = (*Timer)(nil)

// TimerPool is a pool of timers.
type TimerPool interface {
	Add(expire itime.Duration, fn func()) *TimerData
	Del(td *TimerData)
	Set(td *TimerData, expire itime.Duration)
	Stop()
}

// TimerData represents a timer entry with expiration time and callback function.
type TimerData struct {
	Key    string     // unique identifier for the timer
	expire itime.Time // expiration time
	fn     func()     // callback function to execute when timer expires
	index  int        // position in the heap
	next   *TimerData // next free timer data in the pool
}

// Delay returns the remaining time until expiration.
func (td *TimerData) Delay() itime.Duration {
	return itime.Until(td.expire)
}

// ExpireString returns the expiration time as a formatted string.
func (td *TimerData) ExpireString() string {
	return td.expire.Format(timerFormat)
}

// Timer implements a high-performance timer using a min-heap.
type Timer struct {
	mu       sync.Mutex   // protects timers, free, and signal
	free     *TimerData   // free timer data linked list
	timers   []*TimerData // min-heap of active timers
	signal   *itime.Timer // system timer for next expiration
	capacity int          // initial capacity for timer pool
}

// NewTimer creates a new timer with the specified initial capacity.
func NewTimer(capacity int) *Timer {
	if capacity <= 0 {
		capacity = 100 // default capacity
	}
	t := &Timer{}
	t.init(capacity)
	return t
}

// Init initializes the timer with the specified capacity.
func (t *Timer) Init(capacity int) {
	if capacity <= 0 {
		capacity = 100
	}
	t.init(capacity)
}

// init initializes the timer with internal structures.
func (t *Timer) init(capacity int) {
	t.signal = itime.NewTimer(infiniteDuration)
	t.timers = make([]*TimerData, 0, capacity)
	t.capacity = capacity
	t.grow()
	go t.start()
}

// grow allocates a new batch of TimerData objects and adds them to the free list.
func (t *Timer) grow() {
	timerData := make([]TimerData, t.capacity)
	t.free = &timerData[0]

	// Link all TimerData objects in a linked list
	for i := 0; i < t.capacity-1; i++ {
		timerData[i].next = &timerData[i+1]
	}
	timerData[t.capacity-1].next = nil
}

// get retrieves a free TimerData from the pool, growing if necessary.
func (t *Timer) get() *TimerData {
	if t.free == nil {
		t.grow()
	}
	td := t.free
	t.free = td.next
	td.next = nil
	return td
}

// put returns a TimerData to the free pool for reuse.
func (t *Timer) put(td *TimerData) {
	if td == nil {
		return
	}
	td.fn = nil
	td.next = t.free
	t.free = td
}

// Add creates a new timer that expires after the specified duration.
// Returns a TimerData that can be used to modify or cancel the timer.
func (t *Timer) Add(expire itime.Duration, fn func()) *TimerData {
	if expire <= 0 {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	td := t.get()
	td.expire = itime.Now().Add(expire)
	td.fn = fn
	t.add(td)
	return td
}

// Del removes and cancels a timer.
func (t *Timer) Del(td *TimerData) {
	if td == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.del(td)
	t.put(td)
}

// add adds a TimerData to the min-heap.
func (t *Timer) add(td *TimerData) {
	td.index = len(t.timers)
	t.timers = append(t.timers, td)
	t.up(td.index)

	// If this is the first timer, reset the signal timer
	if td.index == 0 {
		delay := td.Delay()
		t.signal.Reset(delay)

	}

}

// del removes a TimerData from the min-heap.
func (t *Timer) del(td *TimerData) {
	i := td.index
	last := len(t.timers) - 1

	// Check if the timer is already removed or invalid
	if i < 0 || i > last || t.timers[i] != td {

		return
	}

	// If not the last element, swap with last and re-heapify
	if i != last {
		t.swap(i, last)
		t.down(i, last)
		t.up(i)
	}

	// Remove the last element
	t.timers[last].index = -1 // mark as removed
	t.timers = t.timers[:last]

}

// Set updates the expiration time of an existing timer.
func (t *Timer) Set(td *TimerData, expire itime.Duration) {
	if td == nil || expire <= 0 {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.del(td)
	td.expire = itime.Now().Add(expire)
	t.add(td)
}

// start runs the main timer loop in a separate goroutine.
func (t *Timer) start() {
	for {
		t.expire()
		<-t.signal.C
	}
}

// expire processes expired timers and resets the signal timer.
func (t *Timer) expire() {
	t.mu.Lock()

	var delay itime.Duration
	for {
		if len(t.timers) == 0 {
			delay = infiniteDuration

			break
		}

		td := t.timers[0]
		if delay = td.Delay(); delay > 0 {
			break
		}

		// Timer has expired, execute callback
		fn := td.fn
		t.del(td)
		t.mu.Unlock()

		if fn != nil {

			fn()
		}

		t.mu.Lock()
	}

	t.signal.Reset(delay)

	t.mu.Unlock()
}

// up performs heap up operation to maintain min-heap property.
func (t *Timer) up(j int) {
	for {
		i := (j - 1) / 2 // parent index
		if i >= j || !t.less(j, i) {
			break
		}
		t.swap(i, j)
		j = i
	}
}

// down performs heap down operation to maintain min-heap property.
func (t *Timer) down(i, n int) {
	for {
		j1 := 2*i + 1          // left child
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}

		j := j1 // left child
		if j2 := j1 + 1; j2 < n && !t.less(j1, j2) {
			j = j2 // right child
		}

		if !t.less(j, i) {
			break
		}

		t.swap(i, j)
		i = j
	}
}

// less compares two timers by expiration time for min-heap ordering.
func (t *Timer) less(i, j int) bool {
	return t.timers[i].expire.Before(t.timers[j].expire)
}

// swap exchanges two timers in the heap and updates their indices.
func (t *Timer) swap(i, j int) {
	t.timers[i], t.timers[j] = t.timers[j], t.timers[i]
	t.timers[i].index = i
	t.timers[j].index = j
}

// Timers returns the active timers.
func (t *Timer) Timers() []*TimerData {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]*TimerData{}, t.timers...)
}

// Stop stops the timer and cleans up resources.
// This should be called when the timer is no longer needed to prevent goroutine leaks.
func (t *Timer) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Stop the signal timer
	if t.signal != nil {
		t.signal.Stop()
	}

	// Clear all timers
	t.timers = t.timers[:0]
}

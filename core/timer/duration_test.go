package timer_test

import (
	"testing"
	"time"

	"github.com/heyehang/blazewave/core/timer"
)

// Test basic duration unmarshaling.
func TestDurationUnmarshalText(t *testing.T) {
	var d timer.Duration

	// Test valid duration strings
	testCases := []struct {
		input    string
		expected time.Duration
	}{
		{"1s", time.Second},
		{"500ms", 500 * time.Millisecond},
		{"2h", 2 * time.Hour},
		{"30m", 30 * time.Minute},
		{"1h30m", 90 * time.Minute},
		{"1h30m45s", 90*time.Minute + 45*time.Second},
		{"1.5s", 1500 * time.Millisecond},
		{"0s", 0},
	}

	for _, tc := range testCases {
		err := d.UnmarshalText([]byte(tc.input))
		if err != nil {
			t.Errorf("UnmarshalText(%s) failed: %v", tc.input, err)
		}
		if d.Duration() != tc.expected {
			t.Errorf("UnmarshalText(%s) = %v, want %v", tc.input, d.Duration(), tc.expected)
		}
	}
}

// Test duration unmarshaling with empty input.
func TestDurationUnmarshalEmpty(t *testing.T) {
	var d timer.Duration

	// Test empty string
	err := d.UnmarshalText([]byte(""))
	if err != nil {
		t.Errorf("UnmarshalText(\"\") failed: %v", err)
	}

	// Test nil input
	err = d.UnmarshalText(nil)
	if err != nil {
		t.Errorf("UnmarshalText(nil) failed: %v", err)
	}
}

// Test duration unmarshaling with invalid input.
func TestDurationUnmarshalInvalid(t *testing.T) {
	var d timer.Duration

	// Test invalid duration strings
	invalidInputs := []string{
		"invalid",
		"1x",
		"1.5.5s",
		"1h2",
		"abc123",
		// Note: "1s2s" is actually valid in Go's time.ParseDuration (parses as 2s)
	}

	for _, input := range invalidInputs {
		err := d.UnmarshalText([]byte(input))
		if err == nil {
			t.Errorf("UnmarshalText(%s) should have failed", input)
		}
	}
}

// Test duration string representation.
func TestDurationString(t *testing.T) {
	testCases := []struct {
		duration timer.Duration
		expected string
	}{
		{timer.Duration(time.Second), "1s"},
		{timer.Duration(500 * time.Millisecond), "500ms"},
		{timer.Duration(2 * time.Hour), "2h0m0s"},
		{timer.Duration(90 * time.Minute), "1h30m0s"},
		{0, "0s"},
	}

	for _, tc := range testCases {
		result := tc.duration.String()
		if result != tc.expected {
			t.Errorf("Duration(%v).String() = %s, want %s", tc.duration.Duration(), result, tc.expected)
		}
	}
}

// Test duration conversion to time.Duration.
func TestDurationDuration(t *testing.T) {
	testCases := []struct {
		duration timer.Duration
		expected time.Duration
	}{
		{timer.Duration(time.Second), time.Second},
		{timer.Duration(500 * time.Millisecond), 500 * time.Millisecond},
		{timer.Duration(2 * time.Hour), 2 * time.Hour},
		{timer.Duration(0), 0},
	}

	for _, tc := range testCases {
		result := tc.duration.Duration()
		if result != tc.expected {
			t.Errorf("Duration(%v).Duration() = %v, want %v", tc.duration.Duration(), result, tc.expected)
		}
	}
}

// Test duration with complex time formats.
func TestDurationComplexFormats(t *testing.T) {
	var d timer.Duration

	complexFormats := []struct {
		input    string
		expected time.Duration
	}{
		{"1h2m3s", time.Hour + 2*time.Minute + 3*time.Second},
		{"1h2m3s4ms", time.Hour + 2*time.Minute + 3*time.Second + 4*time.Millisecond},
		{"1h2m3s4ms5us", time.Hour + 2*time.Minute + 3*time.Second + 4*time.Millisecond + 5*time.Microsecond},
		{"1h2m3s4ms5us6ns", time.Hour + 2*time.Minute + 3*time.Second + 4*time.Millisecond + 5*time.Microsecond + 6*time.Nanosecond},
	}

	for _, tc := range complexFormats {
		err := d.UnmarshalText([]byte(tc.input))
		if err != nil {
			t.Errorf("UnmarshalText(%s) failed: %v", tc.input, err)
		}
		if d.Duration() != tc.expected {
			t.Errorf("UnmarshalText(%s) = %v, want %v", tc.input, d.Duration(), tc.input)
		}
	}
}

// Test duration with decimal values.
func TestDurationDecimal(t *testing.T) {
	var d timer.Duration

	decimalFormats := []struct {
		input    string
		expected time.Duration
	}{
		{"1.5s", 1500 * time.Millisecond},
		{"0.5s", 500 * time.Millisecond},
		{"1.5h", 90 * time.Minute},
		{"0.5h", 30 * time.Minute},
		{"1.5m", 90 * time.Second},
		{"0.5m", 30 * time.Second},
	}

	for _, tc := range decimalFormats {
		err := d.UnmarshalText([]byte(tc.input))
		if err != nil {
			t.Errorf("UnmarshalText(%s) failed: %v", tc.input, err)
		}
		if d.Duration() != tc.expected {
			t.Errorf("UnmarshalText(%s) = %v, want %v", tc.input, d.Duration(), tc.expected)
		}
	}
}

// Benchmark duration unmarshaling.
func BenchmarkDurationUnmarshal(b *testing.B) {
	var d timer.Duration
	input := []byte("1h30m45s")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d.UnmarshalText(input)
	}
}

func BenchmarkDurationString(b *testing.B) {
	d := timer.Duration(time.Hour + 30*time.Minute + 45*time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.String()
	}
}

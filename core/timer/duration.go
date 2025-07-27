package timer

import (
	xtime "time"
)

// Duration is a custom type that can be unmarshaled from TOML string format.
// It supports time duration strings like "1s", "500ms", "2h30m", etc.
type Duration xtime.Duration

// UnmarshalText unmarshals text representation of duration into Duration type.
// It parses strings like "1s", "500ms", "2h30m" into time.Duration.
func (d *Duration) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return nil // empty text is valid
	}

	duration, err := xtime.ParseDuration(string(text))
	if err != nil {
		return err
	}

	*d = Duration(duration)
	return nil
}

// String returns the string representation of the duration.
func (d Duration) String() string {
	return xtime.Duration(d).String()
}

// Duration returns the underlying time.Duration value.
func (d Duration) Duration() xtime.Duration {
	return xtime.Duration(d)
}

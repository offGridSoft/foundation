package temporal

import (
	"encoding/json"
	"math"
	"strconv"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

// Duration is non-negative elapsed time bounded by signed 64-bit nanoseconds.
type Duration struct {
	nanoseconds int64
}

// Validate enforces the non-negative duration domain.
func (d Duration) Validate() error {
	if d.nanoseconds < 0 {
		return contractError(errFmtDuration)
	}
	return nil
}

// Nanoseconds returns exact elapsed nanoseconds.
func (d Duration) Nanoseconds() int64 {
	return d.nanoseconds
}

// Stdlib projects to the standard library duration.
func (d Duration) Stdlib() (time.Duration, error) {
	if err := d.Validate(); err != nil {
		return 0, err
	}
	return time.Duration(d.nanoseconds), nil
}

// IsZero reports whether no time elapsed.
func (d Duration) IsZero() bool {
	return d.nanoseconds == 0
}

// Add adds two durations with checked overflow.
func (d Duration) Add(other Duration) (Duration, error) {
	if maxSignedNanoseconds-d.nanoseconds < other.nanoseconds {
		return Duration{}, contractError(errFmtDuration, core.ErrNumericOverflow)
	}
	return DurationFromNanoseconds(d.nanoseconds + other.nanoseconds)
}

// Multiply scales a duration by an unsigned scalar with checked overflow.
func (d Duration) Multiply(multiplier uint64) (Duration, error) {
	if d.nanoseconds == 0 || multiplier == 0 {
		return Duration{}, nil
	}
	if multiplier > math.MaxInt64 {
		return Duration{}, contractError(errFmtDuration, core.ErrNumericOverflow)
	}
	// #nosec G115 -- the immediately preceding bound proves the scalar fits.
	scalar := int64(multiplier)
	if d.nanoseconds > maxSignedNanoseconds/scalar {
		return Duration{}, contractError(errFmtDuration, core.ErrNumericOverflow)
	}
	return DurationFromNanoseconds(d.nanoseconds * scalar)
}

// Compare orders two proof-carrying durations. It cannot fail because Duration
// has no publicly reachable invalid representation.
func (d Duration) Compare(other Duration) Order {
	return compareInt64(d.nanoseconds, other.nanoseconds)
}

// Aggregate widens a duration without loss.
func (d Duration) Aggregate() AggregateDuration {
	return AggregateDurationFromDuration(d)
}

// MarshalJSON emits exact nanoseconds as a canonical JSON string.
func (d Duration) MarshalJSON() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(strconv.FormatInt(d.nanoseconds, 10))
}

// UnmarshalJSON accepts exact canonical nanoseconds without mutating on error.
func (d *Duration) UnmarshalJSON(data []byte) error {
	if d == nil {
		return contractError(errFmtDuration)
	}
	decimal, err := decodeJSONString(data)
	if err != nil {
		return contractError(errFmtDuration, core.ErrJSONContract, err)
	}
	nanoseconds, err := parseSignedNanoseconds(decimal, errFmtDuration)
	if err != nil {
		return err
	}
	parsed, err := DurationFromNanoseconds(nanoseconds)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

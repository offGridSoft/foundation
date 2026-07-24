package temporal

import (
	"encoding/json"
	"math"
	"strconv"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	nanosecondsPerSecond = int64(time.Second)
	maxSignedNanoseconds = int64(math.MaxInt64)
)

// Instant is a set, non-negative UTC Unix instant with nanosecond precision.
type Instant struct {
	nanoseconds int64
	set         bool
}

func instantFromTime(value time.Time) (Instant, error) {
	if value.IsZero() {
		return Instant{}, contractError(errFmtInstant)
	}
	seconds := value.Unix()
	nanosecond := int64(value.Nanosecond())
	if seconds < 0 || seconds > maxSignedNanoseconds/nanosecondsPerSecond {
		return Instant{}, contractError(errFmtInstant, core.ErrNumericOverflow)
	}
	if seconds == maxSignedNanoseconds/nanosecondsPerSecond &&
		nanosecond > maxSignedNanoseconds%nanosecondsPerSecond {
		return Instant{}, contractError(errFmtInstant, core.ErrNumericOverflow)
	}
	return Instant{
		nanoseconds: seconds*nanosecondsPerSecond + nanosecond,
		set:         true,
	}, nil
}

// IsSet reports whether the instant crossed a constructor or decode boundary.
func (i Instant) IsSet() bool {
	return i.set
}

// Validate rejects the unavoidable unset Go zero value.
func (i Instant) Validate() error {
	if !i.set || i.nanoseconds < 0 {
		return contractError(errFmtInstant)
	}
	return nil
}

// Nanoseconds returns exact Unix nanoseconds.
func (i Instant) Nanoseconds() (int64, error) {
	if err := i.Validate(); err != nil {
		return 0, err
	}
	return i.nanoseconds, nil
}

// Time projects the instant to canonical UTC.
func (i Instant) Time() (time.Time, error) {
	if err := i.Validate(); err != nil {
		return time.Time{}, err
	}
	return time.Unix(0, i.nanoseconds).UTC(), nil
}

// Add adds a non-negative duration with checked overflow.
func (i Instant) Add(duration Duration) (Instant, error) {
	if err := i.Validate(); err != nil {
		return Instant{}, err
	}
	if maxSignedNanoseconds-i.nanoseconds < duration.nanoseconds {
		return Instant{}, contractError(errFmtInstant, core.ErrNumericOverflow)
	}
	return InstantFromNanoseconds(i.nanoseconds + duration.nanoseconds)
}

// Subtract subtracts a non-negative duration with checked underflow.
func (i Instant) Subtract(duration Duration) (Instant, error) {
	if err := i.Validate(); err != nil {
		return Instant{}, err
	}
	if duration.nanoseconds > i.nanoseconds {
		return Instant{}, contractError(errFmtInstant, core.ErrNumericOverflow)
	}
	return InstantFromNanoseconds(i.nanoseconds - duration.nanoseconds)
}

// Since returns elapsed time since an earlier instant.
func (i Instant) Since(earlier Instant) (Duration, error) {
	if err := i.Validate(); err != nil {
		return Duration{}, err
	}
	if err := earlier.Validate(); err != nil {
		return Duration{}, err
	}
	if i.nanoseconds < earlier.nanoseconds {
		return Duration{}, contractError(errFmtInstant)
	}
	return DurationFromNanoseconds(i.nanoseconds - earlier.nanoseconds)
}

// Compare orders two set instants.
func (i Instant) Compare(other Instant) (Order, error) {
	if err := i.Validate(); err != nil {
		return OrderUnknown, err
	}
	if err := other.Validate(); err != nil {
		return OrderUnknown, err
	}
	return compareInt64(i.nanoseconds, other.nanoseconds), nil
}

// MarshalJSON emits exact nanoseconds as a canonical JSON string.
func (i Instant) MarshalJSON() ([]byte, error) {
	nanoseconds, err := i.Nanoseconds()
	if err != nil {
		return nil, err
	}
	return json.Marshal(strconv.FormatInt(nanoseconds, 10))
}

// UnmarshalJSON accepts exact canonical nanoseconds without mutating on error.
func (i *Instant) UnmarshalJSON(data []byte) error {
	if i == nil {
		return contractError(errFmtInstant)
	}
	decimal, err := decodeJSONString(data)
	if err != nil {
		return contractError(errFmtInstant, core.ErrJSONContract, err)
	}
	nanoseconds, err := parseSignedNanoseconds(decimal, errFmtInstant)
	if err != nil {
		return err
	}
	parsed, err := InstantFromNanoseconds(nanoseconds)
	if err != nil {
		return err
	}
	*i = parsed
	return nil
}

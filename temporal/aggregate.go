package temporal

import (
	"encoding/json"
	"math/bits"

	"github.com/offGridSoft/foundation/v2026/core"
)

// AggregateDuration is an unsigned 128-bit nanosecond accumulator.
type AggregateDuration struct {
	high uint64
	low  uint64
}

func parseAggregateDuration(decimal string) (AggregateDuration, error) {
	if !canonicalUnsignedDecimal(decimal, maxUint128DecimalDigits) {
		return AggregateDuration{}, contractError(errFmtAggregate, core.ErrInvalidDecimal)
	}
	var value AggregateDuration
	for index := range len(decimal) {
		next, err := value.multiplyAddDecimal(decimal[index] - '0')
		if err != nil {
			return AggregateDuration{}, err
		}
		value = next
	}
	return value, nil
}

func (a AggregateDuration) multiplyAddDecimal(digit byte) (AggregateDuration, error) {
	scaled, err := a.Multiply(10)
	if err != nil {
		return AggregateDuration{}, contractError(errFmtAggregate, core.ErrInvalidDecimal, core.ErrNumericOverflow)
	}
	low, carry := bits.Add64(scaled.low, uint64(digit), 0)
	high, overflow := bits.Add64(scaled.high, 0, carry)
	if overflow != 0 {
		return AggregateDuration{}, contractError(errFmtAggregate, core.ErrInvalidDecimal, core.ErrNumericOverflow)
	}
	return AggregateDuration{high: high, low: low}, nil
}

// Validate accepts every bit pattern because all unsigned 128-bit states are
// valid aggregate durations.
func (a AggregateDuration) Validate() error {
	return nil
}

// IsZero reports whether the accumulator is empty.
func (a AggregateDuration) IsZero() bool {
	return a.high == 0 && a.low == 0
}

// Decimal returns canonical unsigned base-10 nanoseconds.
func (a AggregateDuration) Decimal() string {
	if a.IsZero() {
		return "0"
	}
	var buffer [maxUint128DecimalDigits]byte
	index := len(buffer)
	current := a
	for !current.IsZero() {
		var remainder uint64
		current, remainder = current.divide(10)
		index--
		buffer[index] = decimalDigits[remainder]
	}
	return string(buffer[index:])
}

// Add adds two aggregate durations with checked overflow.
func (a AggregateDuration) Add(other AggregateDuration) (AggregateDuration, error) {
	low, carry := bits.Add64(a.low, other.low, 0)
	high, overflow := bits.Add64(a.high, other.high, carry)
	if overflow != 0 {
		return AggregateDuration{}, contractError(errFmtAggregate, core.ErrNumericOverflow)
	}
	return AggregateDuration{high: high, low: low}, nil
}

// AddDuration adds an ordinary duration with checked overflow.
func (a AggregateDuration) AddDuration(duration Duration) (AggregateDuration, error) {
	return a.Add(AggregateDurationFromDuration(duration))
}

// Multiply scales an aggregate duration by an unsigned scalar.
func (a AggregateDuration) Multiply(multiplier uint64) (AggregateDuration, error) {
	if multiplier == 0 || a.IsZero() {
		return AggregateDuration{}, nil
	}
	highCarry, low := bits.Mul64(a.low, multiplier)
	overflow, highBase := bits.Mul64(a.high, multiplier)
	high, carry := bits.Add64(highBase, highCarry, 0)
	if overflow != 0 || carry != 0 {
		return AggregateDuration{}, contractError(errFmtAggregate, core.ErrNumericOverflow)
	}
	return AggregateDuration{high: high, low: low}, nil
}

// Compare orders two aggregate durations. Every unsigned 128-bit state is
// valid, so comparison cannot fail.
func (a AggregateDuration) Compare(other AggregateDuration) Order {
	switch {
	case a.high < other.high:
		return OrderBefore
	case a.high > other.high:
		return OrderAfter
	case a.low < other.low:
		return OrderBefore
	case a.low > other.low:
		return OrderAfter
	default:
		return OrderEqual
	}
}

// MarshalJSON emits canonical decimal nanoseconds as a JSON string.
func (a AggregateDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Decimal())
}

// UnmarshalJSON accepts canonical decimal nanoseconds without mutating on
// failure.
func (a *AggregateDuration) UnmarshalJSON(data []byte) error {
	if a == nil {
		return contractError(errFmtAggregate)
	}
	decimal, err := decodeJSONString(data)
	if err != nil {
		return contractError(errFmtAggregate, core.ErrJSONContract, err)
	}
	parsed, err := ParseAggregateDuration(decimal)
	if err != nil {
		return err
	}
	*a = parsed
	return nil
}

func (a AggregateDuration) divide(divisor uint64) (AggregateDuration, uint64) {
	high, remainder := bits.Div64(0, a.high, divisor)
	low, remainder := bits.Div64(remainder, a.low, divisor)
	return AggregateDuration{high: high, low: low}, remainder
}

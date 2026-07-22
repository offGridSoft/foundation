package peachfuzz

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"strconv"

	"github.com/offGridSoft/foundation/v2026/core"
)

// EffortNanoseconds is the fixed-width fleet aggregate for CPU effort. Two
// limbs preserve exact nanoseconds beyond the int64 lifetime ceiling while
// keeping every operation bounded and allocation-free except presentation.
type EffortNanoseconds struct {
	High uint64 `json:"high"`
	Low  uint64 `json:"low"`
}

func NewEffortNanoseconds(value core.NanosecondsDuration) (EffortNanoseconds, error) {
	if err := value.Validate(); err != nil {
		return EffortNanoseconds{}, effortNanosecondsError(err)
	}
	nanoseconds := value.Nanoseconds()
	if nanoseconds < 0 {
		return EffortNanoseconds{}, effortNanosecondsError(core.ErrNumericOverflow)
	}
	return EffortNanoseconds{Low: uint64(nanoseconds)}, nil
}

func NewEffortNanosecondsParts(high, low uint64) EffortNanoseconds {
	return EffortNanoseconds{High: high, Low: low}
}

func (e EffortNanoseconds) Validate() error { return nil }

func (e EffortNanoseconds) IsZero() bool { return e.High == 0 && e.Low == 0 }

func (e EffortNanoseconds) Add(other EffortNanoseconds) (EffortNanoseconds, error) {
	if err := other.Validate(); err != nil {
		return EffortNanoseconds{}, effortNanosecondsError(err)
	}
	low, carry := bits.Add64(e.Low, other.Low, 0)
	high, overflow := bits.Add64(e.High, other.High, carry)
	if overflow != 0 {
		return EffortNanoseconds{}, effortNanosecondsError(core.ErrNumericOverflow)
	}
	return EffortNanoseconds{High: high, Low: low}, nil
}

func (e EffortNanoseconds) Decimal() string {
	if e.High == 0 {
		return strconv.FormatUint(e.Low, 10)
	}
	digits := [39]byte{}
	index := len(digits)
	high, low := e.High, e.Low
	for high != 0 || low != 0 {
		quotientHigh, remainderHigh := bits.Div64(0, high, 10)
		quotientLow, remainder := bits.Div64(remainderHigh, low, 10)
		index--
		digits[index] = "0123456789"[remainder]
		high, low = quotientHigh, quotientLow
	}
	return string(digits[index:])
}

func ParseEffortNanosecondsDecimal(value string) (EffortNanoseconds, error) {
	if value == "" || len(value) > 39 || len(value) > 1 && value[0] == '0' {
		return EffortNanoseconds{}, effortNanosecondsError(core.ErrInvalidDecimal)
	}
	var result EffortNanoseconds
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return EffortNanoseconds{}, effortNanosecondsError(core.ErrInvalidDecimal)
		}
		high, low, overflow := multiplyAdd128(result.High, result.Low, 10, uint64(digit-'0'))
		if overflow {
			return EffortNanoseconds{}, effortNanosecondsError(core.ErrNumericOverflow)
		}
		result = EffortNanoseconds{High: high, Low: low}
	}
	return result, nil
}

func multiplyAdd128(high, low, multiplier, addend uint64) (uint64, uint64, bool) {
	upperLow, lower := bits.Mul64(low, multiplier)
	upperHigh, middle := bits.Mul64(high, multiplier)
	middle, carry := bits.Add64(middle, upperLow, 0)
	lower, addCarry := bits.Add64(lower, addend, 0)
	middle, carry2 := bits.Add64(middle, 0, addCarry)
	return middle, lower, upperHigh != 0 || carry != 0 || carry2 != 0
}

func (e EffortNanoseconds) quotientRemainder(divisor uint64) (EffortNanoseconds, uint64, error) {
	if divisor == 0 {
		return EffortNanoseconds{}, 0, effortNanosecondsError(core.ErrInvalidDecimal)
	}
	quotientHigh := e.High / divisor
	remainderHigh := e.High % divisor
	quotientLow, remainder := bits.Div64(remainderHigh, e.Low, divisor)
	return EffortNanoseconds{High: quotientHigh, Low: quotientLow}, remainder, nil
}

func (e EffortNanoseconds) Float64() float64 {
	return float64(e.High)*math.Exp2(64) + float64(e.Low)
}

func (e EffortNanoseconds) MarshalJSON() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	type wire EffortNanoseconds
	return json.Marshal(wire(e))
}

func effortNanosecondsError(err error) error {
	return fmt.Errorf(ErrFmtEffortNanoseconds, errors.Join(ErrContract, err))
}

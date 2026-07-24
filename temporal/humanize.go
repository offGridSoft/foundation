package temporal

import (
	"encoding/json"
	"strings"
)

const maximumFractionDigits = uint8(9)

// HumanUnit is a closed mechanical duration unit.
type HumanUnit uint8

const (
	HumanUnitUnknown HumanUnit = iota
	HumanUnitAutomatic
	HumanUnitNanoseconds
	HumanUnitMicroseconds
	HumanUnitMilliseconds
	HumanUnitSeconds
	HumanUnitMinutes
	HumanUnitHours
	HumanUnitDays
	HumanUnitYears
)

// HumanizeStyle is a closed output-token style.
type HumanizeStyle uint8

const (
	HumanizeStyleUnknown HumanizeStyle = iota
	HumanizeStyleLong
	HumanizeStyleCompact
)

const (
	humanUnitTokenAutomatic    = "automatic"
	humanUnitTokenNanoseconds  = "nanoseconds"
	humanUnitTokenMicroseconds = "microseconds"
	humanUnitTokenMilliseconds = "milliseconds"
	humanUnitTokenSeconds      = "seconds"
	humanUnitTokenMinutes      = "minutes"
	humanUnitTokenHours        = "hours"
	humanUnitTokenDays         = "days"
	humanUnitTokenYears        = "years"
	humanizeStyleTokenLong     = "long"
	humanizeStyleTokenCompact  = "compact"
)

const (
	compactNanoseconds  = "ns"
	compactMicroseconds = "us"
	compactMilliseconds = "ms"
	compactSeconds      = "s"
	compactMinutes      = "m"
	compactHours        = "h"
	compactDays         = "d"
	compactYears        = "y"
)

const (
	nanosecondsPerNanosecond  = uint64(1)
	nanosecondsPerMicrosecond = uint64(1_000)
	nanosecondsPerMillisecond = uint64(1_000_000)
	nanosecondsPerMinute      = uint64(60) * uint64(nanosecondsPerSecond)
	nanosecondsPerHour        = uint64(60) * nanosecondsPerMinute
	nanosecondsPerDay         = uint64(24) * nanosecondsPerHour
	nanosecondsPerYear        = uint64(36525) * nanosecondsPerDay / 100
)

// HumanizePolicy explicitly controls deterministic duration presentation.
type HumanizePolicy struct {
	Unit           HumanUnit
	Style          HumanizeStyle
	FractionDigits uint8
}

// Validate rejects unknown states and invented sub-nanosecond precision.
func (p HumanizePolicy) Validate() error {
	if err := p.Unit.Validate(); err != nil {
		return err
	}
	if err := p.Style.Validate(); err != nil {
		return err
	}
	if p.FractionDigits > maximumFractionDigits {
		return contractError(errFmtHumanize)
	}
	return nil
}

// Humanized is a validated, deterministic presentation value.
type Humanized struct {
	number string
	unit   HumanUnit
	style  HumanizeStyle
}

// Validate enforces the closed presentation result.
func (h Humanized) Validate() error {
	if !humanizedNumberValid(h.number) || h.unit <= HumanUnitAutomatic {
		return contractError(errFmtHumanize)
	}
	if err := h.unit.Validate(); err != nil {
		return err
	}
	return h.style.Validate()
}

// Number returns the canonical decimal number without a unit token.
func (h Humanized) Number() string {
	return h.number
}

// Unit returns the selected explicit unit.
func (h Humanized) Unit() HumanUnit {
	return h.unit
}

// Text returns the complete stable presentation.
func (h Humanized) Text() string {
	if h.style == HumanizeStyleCompact {
		return h.number + h.unit.compactToken()
	}
	return h.number + " " + h.unit.String()
}

// Humanize projects an ordinary duration deterministically.
func (d Duration) Humanize(policy HumanizePolicy) (Humanized, error) {
	return AggregateDurationFromDuration(d).Humanize(policy)
}

// Humanize projects a full-range aggregate duration deterministically.
func (a AggregateDuration) Humanize(policy HumanizePolicy) (Humanized, error) {
	if err := policy.Validate(); err != nil {
		return Humanized{}, err
	}
	unit := policy.Unit
	if unit == HumanUnitAutomatic {
		unit = automaticHumanUnit(a)
	}
	divisor := unit.divisor()
	integer, remainder := a.divide(divisor)
	number := humanizedNumber(integer.Decimal(), remainder, divisor, policy.FractionDigits)
	result := Humanized{number: number, unit: unit, style: policy.Style}
	if err := result.Validate(); err != nil {
		return Humanized{}, err
	}
	return result, nil
}

// IsValid reports whether the unit belongs to the closed enum.
func (u HumanUnit) IsValid() bool {
	return u > HumanUnitUnknown && u <= HumanUnitYears
}

// Validate enforces the closed unit enum.
func (u HumanUnit) Validate() error {
	if !u.IsValid() {
		return contractError(errFmtHumanize)
	}
	return nil
}

// String returns the canonical unit token or an empty string.
func (u HumanUnit) String() string {
	if !u.IsValid() {
		return ""
	}
	return humanUnitTokens()[u]
}

// MarshalJSON emits the canonical unit token.
func (u HumanUnit) MarshalJSON() ([]byte, error) {
	if err := u.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(u.String())
}

// UnmarshalJSON accepts only a canonical unit token without mutating on error.
func (u *HumanUnit) UnmarshalJSON(data []byte) error {
	if u == nil {
		return contractError(errFmtHumanize)
	}
	token, err := decodeJSONString(data)
	if err != nil {
		return contractError(errFmtHumanize, err)
	}
	parsed, err := ParseHumanUnit(token)
	if err != nil {
		return err
	}
	*u = parsed
	return nil
}

// IsValid reports whether the style belongs to the closed enum.
func (s HumanizeStyle) IsValid() bool {
	return s > HumanizeStyleUnknown && s <= HumanizeStyleCompact
}

// Validate enforces the closed style enum.
func (s HumanizeStyle) Validate() error {
	if !s.IsValid() {
		return contractError(errFmtHumanize)
	}
	return nil
}

// String returns the canonical style token or an empty string.
func (s HumanizeStyle) String() string {
	if !s.IsValid() {
		return ""
	}
	return humanizeStyleTokens()[s]
}

// MarshalJSON emits the canonical style token.
func (s HumanizeStyle) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(s.String())
}

// UnmarshalJSON accepts only a canonical style token without mutating on
// error.
func (s *HumanizeStyle) UnmarshalJSON(data []byte) error {
	if s == nil {
		return contractError(errFmtHumanize)
	}
	token, err := decodeJSONString(data)
	if err != nil {
		return contractError(errFmtHumanize, err)
	}
	parsed, err := ParseHumanizeStyle(token)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

func automaticHumanUnit(value AggregateDuration) HumanUnit {
	for unit := HumanUnitYears; unit >= HumanUnitMicroseconds; unit-- {
		if value.atLeast(unit.divisor()) {
			return unit
		}
	}
	return HumanUnitNanoseconds
}

func (a AggregateDuration) atLeast(value uint64) bool {
	return a.high != 0 || a.low >= value
}

func humanizedNumber(integer string, remainder, divisor uint64, digits uint8) string {
	if digits == 0 {
		return integer
	}
	var fraction [maximumFractionDigits]byte
	for index := range int(digits) {
		remainder *= 10
		fraction[index] = decimalDigits[remainder/divisor]
		remainder %= divisor
	}
	return integer + "." + string(fraction[:digits])
}

func humanizedNumberValid(number string) bool {
	if len(number) == 0 || len(number) > maxUint128DecimalDigits+1+int(maximumFractionDigits) {
		return false
	}
	parts := strings.Split(number, ".")
	if len(parts) > 2 || !canonicalUnsignedDecimal(parts[0], maxUint128DecimalDigits) {
		return false
	}
	return len(parts) == 1 || validFraction(parts[1])
}

func validFraction(fraction string) bool {
	if len(fraction) == 0 || len(fraction) > int(maximumFractionDigits) {
		return false
	}
	for index := range len(fraction) {
		if fraction[index] < '0' || fraction[index] > '9' {
			return false
		}
	}
	return true
}

func humanUnitTokens() [HumanUnitYears + 1]string {
	return [...]string{
		HumanUnitAutomatic:    humanUnitTokenAutomatic,
		HumanUnitNanoseconds:  humanUnitTokenNanoseconds,
		HumanUnitMicroseconds: humanUnitTokenMicroseconds,
		HumanUnitMilliseconds: humanUnitTokenMilliseconds,
		HumanUnitSeconds:      humanUnitTokenSeconds,
		HumanUnitMinutes:      humanUnitTokenMinutes,
		HumanUnitHours:        humanUnitTokenHours,
		HumanUnitDays:         humanUnitTokenDays,
		HumanUnitYears:        humanUnitTokenYears,
	}
}

func humanizeStyleTokens() [HumanizeStyleCompact + 1]string {
	return [...]string{
		HumanizeStyleLong:    humanizeStyleTokenLong,
		HumanizeStyleCompact: humanizeStyleTokenCompact,
	}
}

func (u HumanUnit) divisor() uint64 {
	return humanUnitDivisors()[u]
}

func humanUnitDivisors() [HumanUnitYears + 1]uint64 {
	return [...]uint64{
		HumanUnitNanoseconds:  nanosecondsPerNanosecond,
		HumanUnitMicroseconds: nanosecondsPerMicrosecond,
		HumanUnitMilliseconds: nanosecondsPerMillisecond,
		HumanUnitSeconds:      uint64(nanosecondsPerSecond),
		HumanUnitMinutes:      nanosecondsPerMinute,
		HumanUnitHours:        nanosecondsPerHour,
		HumanUnitDays:         nanosecondsPerDay,
		HumanUnitYears:        nanosecondsPerYear,
	}
}

func (u HumanUnit) compactToken() string {
	return humanUnitCompactTokens()[u]
}

func humanUnitCompactTokens() [HumanUnitYears + 1]string {
	return [...]string{
		HumanUnitNanoseconds:  compactNanoseconds,
		HumanUnitMicroseconds: compactMicroseconds,
		HumanUnitMilliseconds: compactMilliseconds,
		HumanUnitSeconds:      compactSeconds,
		HumanUnitMinutes:      compactMinutes,
		HumanUnitHours:        compactHours,
		HumanUnitDays:         compactDays,
		HumanUnitYears:        compactYears,
	}
}

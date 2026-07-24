package currency

import (
	"encoding/json"
	"strconv"
)

const humanizeFractionDigitsMaximum = 6

// DisplayUnit is the closed monetary presentation scale.
type DisplayUnit uint8

const (
	DisplayUnitUnknown DisplayUnit = iota
	DisplayUnitAutomatic
	DisplayUnitMinor
	DisplayUnitMajor
	DisplayUnitHundreds
	DisplayUnitThousands
	DisplayUnitMillions
	DisplayUnitBillions
)

func displayUnitTokens() [DisplayUnitBillions + 1]string {
	return [...]string{
		DisplayUnitAutomatic: "automatic",
		DisplayUnitMinor:     "minor",
		DisplayUnitMajor:     "major",
		DisplayUnitHundreds:  "hundreds",
		DisplayUnitThousands: "thousands",
		DisplayUnitMillions:  "millions",
		DisplayUnitBillions:  "billions",
	}
}

// IsValid reports membership in the display-unit domain.
func (u DisplayUnit) IsValid() bool {
	return u > DisplayUnitUnknown &&
		int(u) < len(displayUnitTokens()) &&
		displayUnitTokens()[u] != ""
}

// Validate enforces the closed display-unit domain.
func (u DisplayUnit) Validate() error {
	if !u.IsValid() {
		return contractError(errLabelHumanize)
	}
	return nil
}

// String returns the canonical token or an empty string when invalid.
func (u DisplayUnit) String() string {
	if !u.IsValid() {
		return ""
	}
	return displayUnitTokens()[u]
}

// MarshalJSON emits the canonical display-unit token.
func (u DisplayUnit) MarshalJSON() ([]byte, error) {
	if err := u.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(u.String())
}

// UnmarshalJSON accepts only a canonical display-unit token and preserves the
// receiver on failure.
func (u *DisplayUnit) UnmarshalJSON(data []byte) error {
	if u == nil {
		return contractError(errLabelHumanize)
	}
	if len(data) > canonicalEnumJSONMaximumBytes {
		return contractError(errLabelHumanize)
	}
	var token string
	if err := json.Unmarshal(data, &token); err != nil {
		return contractError(errLabelHumanize, err)
	}
	if string(data) != strconv.Quote(token) {
		return contractError(errLabelHumanize)
	}
	parsed, err := ParseDisplayUnit(token)
	if err != nil {
		return err
	}
	*u = parsed
	return nil
}

// HumanizePolicy owns deterministic monetary presentation choices.
type HumanizePolicy struct {
	Unit           DisplayUnit
	FractionDigits uint8
}

// Validate rejects unknown units, excessive precision, and fractional minor
// units.
func (p HumanizePolicy) Validate() error {
	if err := p.Unit.Validate(); err != nil {
		return err
	}
	if p.FractionDigits > humanizeFractionDigitsMaximum {
		return contractError(errLabelHumanize)
	}
	if p.Unit == DisplayUnitMinor && p.FractionDigits != 0 {
		return contractError(errLabelHumanize)
	}
	return nil
}

// Humanized is a validated display projection, never stored truth.
type Humanized struct {
	number         string
	code           Code
	unit           DisplayUnit
	fractionDigits uint8
}

// Validate enforces a closed, canonical display projection.
func (h Humanized) Validate() error {
	if err := h.code.Validate(); err != nil {
		return contractError(errLabelHumanize, err)
	}
	if err := validateConcreteDisplayUnit(h.unit); err != nil {
		return err
	}
	if h.fractionDigits > humanizeFractionDigitsMaximum {
		return contractError(errLabelHumanize)
	}
	if h.unit == DisplayUnitMinor && h.fractionDigits != 0 {
		return contractError(errLabelHumanize)
	}
	if !validHumanizedNumber(h.number, h.fractionDigits) {
		return contractError(errLabelHumanize)
	}
	return nil
}

// Code returns the projection's currency.
func (h Humanized) Code() (Code, error) {
	if err := h.Validate(); err != nil {
		return CodeUnknown, err
	}
	return h.code, nil
}

// Number returns the deterministic signed decimal.
func (h Humanized) Number() (string, error) {
	if err := h.Validate(); err != nil {
		return "", err
	}
	return h.number, nil
}

// Unit returns the concrete selected display unit.
func (h Humanized) Unit() (DisplayUnit, error) {
	if err := h.Validate(); err != nil {
		return DisplayUnitUnknown, err
	}
	return h.unit, nil
}

// Humanize projects exact minor units through a deterministic integer scale.
func (a Amount) Humanize(policy HumanizePolicy) (Humanized, error) {
	if err := a.Validate(); err != nil {
		return Humanized{}, err
	}
	if err := policy.Validate(); err != nil {
		return Humanized{}, err
	}
	unit, err := selectDisplayUnit(a, policy.Unit)
	if err != nil {
		return Humanized{}, err
	}
	fractionDigits := policy.FractionDigits
	if unit == DisplayUnitMinor {
		fractionDigits = 0
	}
	divisor, err := displayDivisor(a.code, unit)
	if err != nil {
		return Humanized{}, err
	}
	result := Humanized{
		number:         scaledNumber(a.minorUnits, divisor, fractionDigits),
		code:           a.code,
		unit:           unit,
		fractionDigits: fractionDigits,
	}
	if err := result.Validate(); err != nil {
		return Humanized{}, err
	}
	return result, nil
}

func validateConcreteDisplayUnit(unit DisplayUnit) error {
	if err := unit.Validate(); err != nil {
		return err
	}
	if unit == DisplayUnitAutomatic {
		return contractError(errLabelHumanize)
	}
	return nil
}

func validHumanizedNumber(number string, fractionDigits uint8) bool {
	unsigned, negative, ok := splitHumanizedSign(number)
	if !ok {
		return false
	}
	if fractionDigits == 0 {
		return validHumanizedWhole(unsigned, negative)
	}
	return validHumanizedFraction(unsigned, negative, fractionDigits)
}

func splitHumanizedSign(number string) (string, bool, bool) {
	if number == "" {
		return "", false, false
	}
	if number[0] != '-' {
		return number, false, true
	}
	if len(number) == 1 {
		return "", false, false
	}
	return number[1:], true, true
}

func validHumanizedWhole(unsigned string, negative bool) bool {
	return asciiDigits(unsigned) &&
		(len(unsigned) == 1 || unsigned[0] != '0') &&
		(!negative || unsigned != "0")
}

func validHumanizedFraction(unsigned string, negative bool, fractionDigits uint8) bool {
	point := len(unsigned) - int(fractionDigits) - 1
	return point > 0 &&
		unsigned[point] == '.' &&
		asciiDigits(unsigned[:point]) &&
		asciiDigits(unsigned[point+1:]) &&
		(len(unsigned[:point]) == 1 || unsigned[0] != '0') &&
		(!negative || !zeroDecimal(unsigned))
}

func zeroDecimal(unsigned string) bool {
	for index := range len(unsigned) {
		if unsigned[index] != '0' && unsigned[index] != '.' {
			return false
		}
	}
	return true
}

func selectDisplayUnit(amount Amount, requested DisplayUnit) (DisplayUnit, error) {
	if requested != DisplayUnitAutomatic {
		return requested, nil
	}
	major, err := displayDivisor(amount.code, DisplayUnitMajor)
	if err != nil {
		return DisplayUnitUnknown, err
	}
	magnitude := signedMagnitude(amount.minorUnits)
	switch {
	case magnitude == 0:
		return DisplayUnitMajor, nil
	case magnitude < major:
		return DisplayUnitMinor, nil
	case magnitude < major*100:
		return DisplayUnitMajor, nil
	case magnitude < major*1_000:
		return DisplayUnitHundreds, nil
	case magnitude < major*1_000_000:
		return DisplayUnitThousands, nil
	case magnitude < major*1_000_000_000:
		return DisplayUnitMillions, nil
	default:
		return DisplayUnitBillions, nil
	}
}

func displayDivisor(code Code, unit DisplayUnit) (uint64, error) {
	exponent := code.fractionDigits()
	major := powerOfTen(exponent)
	switch unit {
	case DisplayUnitMinor:
		return 1, nil
	case DisplayUnitMajor:
		return major, nil
	case DisplayUnitHundreds:
		return major * 100, nil
	case DisplayUnitThousands:
		return major * 1_000, nil
	case DisplayUnitMillions:
		return major * 1_000_000, nil
	case DisplayUnitBillions:
		return major * 1_000_000_000, nil
	case DisplayUnitUnknown, DisplayUnitAutomatic:
		return 0, contractError(errLabelHumanize)
	default:
		return 0, contractError(errLabelHumanize)
	}
}

func powerOfTen(exponent uint8) uint64 {
	value := uint64(1)
	for range exponent {
		value *= 10
	}
	return value
}

func scaledNumber(value int64, divisor uint64, fractionDigits uint8) string {
	magnitude := signedMagnitude(value)
	whole := magnitude / divisor
	remainder := magnitude % divisor
	number := strconv.FormatUint(whole, 10)
	nonZero := whole != 0
	if fractionDigits > 0 {
		fraction, fractionNonZero := fractionalDigits(remainder, divisor, fractionDigits)
		number += "." + fraction
		nonZero = nonZero || fractionNonZero
	}
	if value < 0 && nonZero {
		return "-" + number
	}
	return number
}

func fractionalDigits(remainder, divisor uint64, count uint8) (string, bool) {
	var digits [humanizeFractionDigitsMaximum]byte
	nonZero := false
	for index := range count {
		remainder *= 10
		// #nosec G115 -- remainder is strictly below divisor before the
		// multiplication, so the quotient is one decimal digit in 0..9.
		digit := byte(remainder / divisor)
		digits[index] = digit + '0'
		nonZero = nonZero || digit != 0
		remainder %= divisor
	}
	return string(digits[:count]), nonZero
}

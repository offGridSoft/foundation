package currency

import (
	"math"
	"strconv"
	"strings"
)

const decimalMaximumBytes = 32

func parseDecimal(code Code, raw string) (int64, error) {
	digits, negative, err := decimalParts(code, raw)
	if err != nil {
		return 0, err
	}
	limit := uint64(math.MaxInt64)
	if negative {
		limit++
	}
	magnitude, err := accumulateDecimal(digits, limit)
	if err != nil || negative && magnitude == 0 {
		return 0, decimalError(errLabelDecimal, raw)
	}
	return signedValue(negative, magnitude), nil
}

func decimalParts(code Code, raw string) (string, bool, error) {
	if raw == "" || len(raw) > decimalMaximumBytes {
		return "", false, decimalError(errLabelDecimal, raw)
	}
	negative := raw[0] == '-'
	unsigned := raw
	if negative {
		unsigned = raw[1:]
	}
	if unsigned == "" || raw[0] == '+' {
		return "", false, decimalError(errLabelDecimal, raw)
	}
	whole, fraction, hasFraction, err := splitDecimal(unsigned)
	if err != nil || whole == "" || !asciiDigits(whole) {
		return "", false, decimalError(errLabelDecimal, raw)
	}
	exponent := code.fractionDigits()
	if err := validateFraction(fraction, hasFraction, exponent); err != nil {
		return "", false, decimalError(errLabelDecimal, raw)
	}
	return whole + fraction + strings.Repeat("0", int(exponent)-len(fraction)), negative, nil
}

func splitDecimal(raw string) (string, string, bool, error) {
	before, after, ok := strings.Cut(raw, ".")
	if !ok {
		return raw, "", false, nil
	}
	if strings.IndexByte(after, '.') >= 0 {
		return "", "", false, decimalError(errLabelDecimal, raw)
	}
	return before, after, true, nil
}

func validateFraction(fraction string, hasFraction bool, exponent uint8) error {
	if !hasFraction {
		return nil
	}
	if exponent == 0 || fraction == "" || len(fraction) > int(exponent) || !asciiDigits(fraction) {
		return decimalError(errLabelDecimal, fraction)
	}
	return nil
}

func asciiDigits(value string) bool {
	for index := range len(value) {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}

func accumulateDecimal(digits string, limit uint64) (uint64, error) {
	var value uint64
	for index := range len(digits) {
		digit := uint64(digits[index] - '0')
		if value > (limit-digit)/10 {
			return 0, decimalError(errLabelDecimal, digits)
		}
		value = value*10 + digit
	}
	return value, nil
}

// Decimal returns the exact fixed-exponent numeric representation.
func (a Amount) Decimal() (string, error) {
	if err := a.Validate(); err != nil {
		return "", err
	}
	exponent := a.code.fractionDigits()
	magnitude := strconv.FormatUint(signedMagnitude(a.minorUnits), 10)
	if exponent > 0 {
		magnitude = fixedExponentDecimal(magnitude, int(exponent))
	}
	if a.minorUnits < 0 {
		return "-" + magnitude, nil
	}
	return magnitude, nil
}

func fixedExponentDecimal(digits string, exponent int) string {
	if len(digits) <= exponent {
		digits = strings.Repeat("0", exponent-len(digits)+1) + digits
	}
	split := len(digits) - exponent
	return digits[:split] + "." + digits[split:]
}

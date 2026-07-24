package temporal

import (
	"encoding/json"
	"strconv"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	maxInt64DecimalDigits   = 19
	maxUint128DecimalDigits = 39
	decimalDigits           = "0123456789"
)

func decodeJSONString(data []byte) (string, error) {
	var decoded string
	if err := json.Unmarshal(data, &decoded); err != nil {
		return "", err
	}
	if string(data) != strconv.Quote(decoded) {
		return "", core.ErrInvalidDecimal
	}
	return decoded, nil
}

func parseSignedNanoseconds(decimal, format string) (int64, error) {
	if !canonicalUnsignedDecimal(decimal, maxInt64DecimalDigits) {
		return 0, contractError(format, core.ErrInvalidDecimal)
	}
	var value int64
	for index := range len(decimal) {
		digit := int64(decimal[index] - '0')
		if value > (maxSignedNanoseconds-digit)/10 {
			return 0, contractError(format, core.ErrInvalidDecimal, core.ErrNumericOverflow)
		}
		value = value*10 + digit
	}
	return value, nil
}

func canonicalUnsignedDecimal(decimal string, maximumDigits int) bool {
	if len(decimal) == 0 || len(decimal) > maximumDigits {
		return false
	}
	if len(decimal) > 1 && decimal[0] == '0' {
		return false
	}
	for index := range len(decimal) {
		if decimal[index] < '0' || decimal[index] > '9' {
			return false
		}
	}
	return true
}

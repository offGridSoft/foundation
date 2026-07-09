package core

import (
	"bytes"
	"strconv"
)

func parseStrictInt64JSON(data []byte) (int64, error) {
	trimmed := bytes.TrimSpace(data)
	if !validJSONInteger(trimmed) {
		return 0, ErrFoundationContract
	}
	value, err := strconv.ParseInt(string(trimmed), 10, 64)
	if err != nil {
		return 0, ErrFoundationContract
	}
	return value, nil
}

func parseStrictUint64JSON(data []byte) (uint64, error) {
	trimmed := bytes.TrimSpace(data)
	if !validJSONUnsignedInteger(trimmed) {
		return 0, ErrFoundationContract
	}
	value, err := strconv.ParseUint(string(trimmed), 10, 64)
	if err != nil {
		return 0, ErrFoundationContract
	}
	return value, nil
}

func appendInt64JSON(value int64) []byte {
	return strconv.AppendInt(nil, value, 10)
}

func appendUint64JSON(value uint64) []byte {
	return strconv.AppendUint(nil, value, 10)
}

func validJSONInteger(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	if data[0] == '-' {
		if len(data) == 2 && data[1] == '0' {
			return false
		}
		return validJSONUnsignedInteger(data[1:])
	}
	return validJSONUnsignedInteger(data)
}

func validJSONUnsignedInteger(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	if data[0] == '0' {
		return len(data) == 1
	}
	if data[0] < '1' || data[0] > '9' {
		return false
	}
	for _, b := range data[1:] {
		if b < '0' || b > '9' {
			return false
		}
	}
	return true
}

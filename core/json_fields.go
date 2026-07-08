package core

import (
	"encoding/json"
	"strconv"
)

const JSONFieldSchema = "schema"

func AppendJSONField[T any](dst []byte, name string, value T) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	dst = strconv.AppendQuote(dst, name)
	dst = append(dst, ':')
	dst = append(dst, encoded...)
	return dst, nil
}

func AppendJSONFieldAfterComma[T any](dst []byte, prior error, name string, value T) ([]byte, error) {
	if prior != nil {
		return nil, prior
	}
	dst = append(dst, ',')
	return AppendJSONField(dst, name, value)
}

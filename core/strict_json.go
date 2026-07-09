package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

func DecodeStrictJSON[T any](data []byte) (T, error) {
	var value T
	if err := rejectDuplicateJSONFields(data); err != nil {
		return value, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, jsonContractError(err)
	}
	if err := rejectTrailingJSON(decoder, data); err != nil {
		return value, err
	}
	return value, nil
}

func rejectTrailingJSON(decoder *json.Decoder, data []byte) error {
	if onlyJSONWhitespaceAfter(data, decoder.InputOffset()) {
		return nil
	}
	return fmt.Errorf(ErrFmtJSONTrailingValue, ErrJSONContract)
}

type strictJSONContainerKind uint8

const (
	strictJSONContainerObject strictJSONContainerKind = iota + 1
	strictJSONContainerArray
)

type strictJSONContainer struct {
	keys      map[string]struct{}
	kind      strictJSONContainerKind
	expectKey bool
}

func rejectDuplicateJSONFields(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	stack := make([]strictJSONContainer, 0)
	for {
		token, err := decoder.Token()
		if err != nil && onlyJSONWhitespaceAfter(data, decoder.InputOffset()) {
			return nil
		}
		if err != nil {
			return jsonContractError(err)
		}
		next, scanErr := scanStrictJSONToken(stack, token)
		if scanErr != nil {
			return scanErr
		}
		stack = next
	}
}

func jsonContractError(err error) error {
	return errors.Join(ErrJSONContract, fmt.Errorf(ErrFmtJSONDecode, err))
}

func onlyJSONWhitespaceAfter(data []byte, offset int64) bool {
	if offset < 0 || offset > int64(len(data)) {
		return false
	}
	return onlyJSONWhitespace(data[offset:])
}

func onlyJSONWhitespace(data []byte) bool {
	for _, b := range data {
		if b != ' ' && b != '\n' && b != '\r' && b != '\t' {
			return false
		}
	}
	return true
}

func scanStrictJSONToken(stack []strictJSONContainer, token json.Token) ([]strictJSONContainer, error) {
	if len(stack) == 0 && token == nil {
		return nil, fmt.Errorf(ErrFmtJSONUnexpectedValue, ErrJSONContract)
	}
	if delim, ok := token.(json.Delim); ok {
		return scanStrictJSONDelim(stack, delim)
	}
	if key, ok := token.(string); ok && strictJSONTopExpectsKey(stack) {
		return scanStrictJSONKey(stack, key)
	}
	return completeStrictJSONValue(stack), nil
}

func scanStrictJSONDelim(stack []strictJSONContainer, delim json.Delim) ([]strictJSONContainer, error) {
	switch delim {
	case '{':
		return append(stack, strictJSONContainer{
			keys:      make(map[string]struct{}),
			kind:      strictJSONContainerObject,
			expectKey: true,
		}), nil
	case '[':
		return append(stack, strictJSONContainer{kind: strictJSONContainerArray}), nil
	case '}', ']':
		if len(stack) == 0 {
			return nil, fmt.Errorf(ErrFmtJSONUnexpectedDelim, ErrJSONContract)
		}
		return completeStrictJSONValue(stack[:len(stack)-1]), nil
	default:
		return stack, nil
	}
}

func scanStrictJSONKey(stack []strictJSONContainer, key string) ([]strictJSONContainer, error) {
	if len(stack) == 0 {
		return nil, fmt.Errorf(ErrFmtJSONUnexpectedField, ErrJSONContract)
	}
	top := &stack[len(stack)-1]
	if _, exists := top.keys[key]; exists {
		return nil, fmt.Errorf(ErrFmtJSONDuplicateField, ErrJSONContract)
	}
	top.keys[key] = struct{}{}
	top.expectKey = false
	return stack, nil
}

func strictJSONTopExpectsKey(stack []strictJSONContainer) bool {
	if len(stack) == 0 {
		return false
	}
	top := stack[len(stack)-1]
	return top.kind == strictJSONContainerObject && top.expectKey
}

func completeStrictJSONValue(stack []strictJSONContainer) []strictJSONContainer {
	if len(stack) == 0 {
		return stack
	}
	top := &stack[len(stack)-1]
	if top.kind == strictJSONContainerObject && !top.expectKey {
		top.expectKey = true
	}
	return stack
}

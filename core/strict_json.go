package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	StrictJSONMaxBytes        = 1 << 20
	StrictJSONMaxDepth        = 64
	StrictJSONMaxObjectFields = 256
)

func DecodeStrictJSON[T Validatable](data []byte) (T, error) {
	value, err := DecodeStrictJSONStructure[T](data)
	if err != nil {
		return value, err
	}
	if err := value.Validate(); err != nil {
		return value, errors.Join(ErrJSONContract, err)
	}
	return value, nil
}

// DecodeStrictJSONStructure enforces the complete Foundation JSON grammar
// without invoking the decoded domain type's Validate method. Transports use
// this when request path, query, or header fields must be projected into the
// structure before the owning type can validate the complete boundary value.
func DecodeStrictJSONStructure[T any](data []byte) (T, error) {
	var value T
	if len(data) == 0 || len(data) > StrictJSONMaxBytes || !utf8.Valid(data) {
		return value, fmt.Errorf(ErrFmtJSONUnexpectedValue, ErrJSONContract)
	}
	if err := rejectDuplicateJSONFields[T](data); err != nil {
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
	keys      []string
	itemCount uint32
	kind      strictJSONContainerKind
	expectKey bool
}

func rejectDuplicateJSONFields[T any](data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	stack := make([]strictJSONContainer, 0)
	fields := jsonFieldNamesForType(reflect.TypeFor[T]())
	for range StrictJSONMaxBytes {
		token, err := decoder.Token()
		if err != nil && onlyJSONWhitespaceAfter(data, decoder.InputOffset()) {
			return nil
		}
		if err != nil {
			return jsonContractError(err)
		}
		next, scanErr := scanStrictJSONToken(stack, token, fields)
		if scanErr != nil {
			return scanErr
		}
		stack = next
	}
	return fmt.Errorf(ErrFmtJSONUnexpectedValue, ErrJSONContract)
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

func scanStrictJSONToken(stack []strictJSONContainer, token json.Token, fields []string) ([]strictJSONContainer, error) {
	if len(stack) == 0 && token == nil {
		return nil, fmt.Errorf(ErrFmtJSONUnexpectedValue, ErrJSONContract)
	}
	if delim, ok := token.(json.Delim); ok {
		return scanStrictJSONDelim(stack, delim)
	}
	if key, ok := token.(string); ok && strictJSONTopExpectsKey(stack) {
		return scanStrictJSONKey(stack, key, fields)
	}
	return completeStrictJSONValue(stack)
}

func scanStrictJSONDelim(stack []strictJSONContainer, delim json.Delim) ([]strictJSONContainer, error) {
	switch delim {
	case '{':
		if len(stack) >= StrictJSONMaxDepth {
			return nil, fmt.Errorf(ErrFmtJSONUnexpectedValue, ErrJSONContract)
		}
		return append(stack, strictJSONContainer{
			kind:      strictJSONContainerObject,
			expectKey: true,
		}), nil
	case '[':
		if len(stack) >= StrictJSONMaxDepth {
			return nil, fmt.Errorf(ErrFmtJSONUnexpectedValue, ErrJSONContract)
		}
		return append(stack, strictJSONContainer{kind: strictJSONContainerArray}), nil
	case '}', ']':
		if len(stack) == 0 {
			return nil, fmt.Errorf(ErrFmtJSONUnexpectedDelim, ErrJSONContract)
		}
		return completeStrictJSONValue(stack[:len(stack)-1])
	default:
		return stack, nil
	}
}

func scanStrictJSONKey(stack []strictJSONContainer, key string, fields []string) ([]strictJSONContainer, error) {
	if len(stack) == 0 {
		return nil, fmt.Errorf(ErrFmtJSONUnexpectedField, ErrJSONContract)
	}
	top := &stack[len(stack)-1]
	if len(top.keys) >= StrictJSONMaxObjectFields {
		return nil, fmt.Errorf(ErrFmtJSONUnexpectedValue, ErrJSONContract)
	}
	if slices.ContainsFunc(top.keys, func(existing string) bool {
		return strings.EqualFold(existing, key)
	}) {
		return nil, fmt.Errorf(ErrFmtJSONDuplicateField, ErrJSONContract)
	}
	if _, declared := slices.BinarySearch(fields, key); !declared {
		if hasCaseFoldJSONField(fields, key) {
			return nil, fmt.Errorf(ErrFmtJSONUnexpectedField, ErrJSONContract)
		}
		if err := ValidateJSONFieldName(key); err != nil {
			return nil, fmt.Errorf(ErrFmtJSONUnexpectedField, ErrJSONContract)
		}
	}
	top.keys = append(top.keys, key)
	top.expectKey = false
	return stack, nil
}

func hasCaseFoldJSONField(fields []string, key string) bool {
	for _, declared := range fields {
		if strings.EqualFold(declared, key) {
			return true
		}
	}
	return false
}

func jsonFieldNamesForType(root reflect.Type) []string {
	fields := make([]string, 0)
	visited := make(map[reflect.Type]struct{})
	pending := []reflect.Type{root}
	for len(pending) > 0 {
		last := len(pending) - 1
		valueType := indirectJSONFieldType(pending[last])
		pending = pending[:last]
		fields, pending = collectJSONFieldType(valueType, fields, pending, visited)
	}
	slices.Sort(fields)
	return slices.Compact(fields)
}

func indirectJSONFieldType(valueType reflect.Type) reflect.Type {
	for valueType.Kind() == reflect.Pointer || valueType.Kind() == reflect.Slice || valueType.Kind() == reflect.Array {
		valueType = valueType.Elem()
	}
	return valueType
}

func jsonFieldTypeOwnsContract(valueType reflect.Type) bool {
	unmarshaler := reflect.TypeFor[json.Unmarshaler]()
	return valueType.Implements(unmarshaler) || reflect.PointerTo(valueType).Implements(unmarshaler)
}

func collectJSONFieldType(
	valueType reflect.Type,
	fields []string,
	pending []reflect.Type,
	visited map[reflect.Type]struct{},
) ([]string, []reflect.Type) {
	if jsonFieldTypeOwnsContract(valueType) {
		return fields, pending
	}
	if valueType.Kind() == reflect.Map {
		return fields, append(pending, valueType.Elem())
	}
	if valueType.Kind() != reflect.Struct {
		return fields, pending
	}
	if _, found := visited[valueType]; found {
		return fields, pending
	}
	visited[valueType] = struct{}{}
	for field := range valueType.Fields() {
		if !field.IsExported() {
			continue
		}
		name, included := reflectedJSONFieldName(field)
		if included && name != "" {
			fields = append(fields, name)
		}
		pending = append(pending, field.Type)
	}
	return fields, pending
}

func reflectedJSONFieldName(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("json")
	name, _, _ := strings.Cut(tag, ",")
	if name == "-" {
		return "", false
	}
	if name == "" {
		return field.Name, true
	}
	return name, true
}

func strictJSONTopExpectsKey(stack []strictJSONContainer) bool {
	if len(stack) == 0 {
		return false
	}
	top := stack[len(stack)-1]
	return top.kind == strictJSONContainerObject && top.expectKey
}

func completeStrictJSONValue(stack []strictJSONContainer) ([]strictJSONContainer, error) {
	if len(stack) == 0 {
		return stack, nil
	}
	top := &stack[len(stack)-1]
	if top.kind == strictJSONContainerObject && !top.expectKey {
		top.expectKey = true
		return stack, nil
	}
	if top.kind == strictJSONContainerArray {
		if top.itemCount >= CollectionMaximumDefault {
			return nil, fmt.Errorf(ErrFmtJSONUnexpectedValue, ErrJSONContract)
		}
		top.itemCount++
	}
	return stack, nil
}

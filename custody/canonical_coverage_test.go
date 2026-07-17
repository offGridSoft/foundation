package custody

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestSignedBodyCanonicalProjectionCoversEveryTaggedField(t *testing.T) {
	t.Parallel()
	requireCanonicalJSONFieldCoverage(t, validReceipt(t))
	requireCanonicalJSONFieldCoverage(t, validReceipt(t).Timestamp)
}

func requireCanonicalJSONFieldCoverage(t *testing.T, body core.CanonicalValue) {
	t.Helper()
	canonical, err := body.Canonical(nil)
	if err != nil {
		t.Fatalf("Canonical() error = %v", err)
	}
	want := taggedJSONFieldNames(reflect.TypeOf(body))
	got, err := topLevelJSONFieldNames(canonical)
	if err != nil {
		t.Fatalf("canonical field scan error = %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("canonical fields = %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("canonical fields = %v, want %v", got, want)
		}
	}
}

func taggedJSONFieldNames(valueType reflect.Type) []string {
	fields := make([]string, 0, valueType.NumField())
	for field := range valueType.Fields() {
		tag := strings.Split(field.Tag.Get("json"), ",")[0]
		if tag != "" && tag != "-" {
			fields = append(fields, tag)
		}
	}
	sort.Strings(fields)
	return fields
}

func topLevelJSONFieldNames(data []byte) ([]string, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	open, err := decoder.Token()
	if err != nil || open != json.Delim('{') {
		return nil, fmt.Errorf("open object: %w", err)
	}
	fields := make([]string, 0)
	for decoder.More() {
		token, tokenErr := decoder.Token()
		if tokenErr != nil {
			return nil, tokenErr
		}
		field, ok := token.(string)
		if !ok {
			return nil, fmt.Errorf("field token contract")
		}
		fields = append(fields, field)
		if err := skipJSONValue(decoder); err != nil {
			return nil, err
		}
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("trailing JSON contract")
	}
	sort.Strings(fields)
	return fields, nil
}

func skipJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, nested := token.(json.Delim)
	if !nested {
		return nil
	}
	for decoder.More() {
		if delim == '{' {
			if _, err := decoder.Token(); err != nil {
				return err
			}
		}
		if err := skipJSONValue(decoder); err != nil {
			return err
		}
	}
	_, err = decoder.Token()
	return err
}

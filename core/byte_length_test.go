package core

import (
	"encoding/json"
	"errors"
	"math"
	"testing"
)

func TestByteLengthExactExtentBoundaryTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     uint64
		wantInt64 int64
		wantError bool
	}{
		{name: "empty extent", value: 0},
		{name: "one byte", value: 1, wantInt64: 1},
		{name: "largest signed filesystem extent", value: math.MaxInt64, wantInt64: math.MaxInt64},
		{name: "one beyond signed filesystem extent", value: uint64(math.MaxInt64) + 1, wantError: true},
		{name: "largest wire extent", value: math.MaxUint64, wantError: true},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			length := NewByteLength(testCase.value)
			if err := length.Validate(); err != nil {
				t.Fatalf("ByteLength(%d).Validate() error = %v, want nil", testCase.value, err)
			}
			if length.Uint64() != testCase.value {
				t.Fatalf("ByteLength(%d).Uint64() = %d, want exact value", testCase.value, length.Uint64())
			}
			got, err := length.Int64()
			if testCase.wantError {
				if !errors.Is(err, ErrFoundationContract) || got != 0 {
					t.Fatalf("ByteLength(%d).Int64() = (%d,%v), want zero and foundation contract", testCase.value, got, err)
				}
				return
			}
			if err != nil || got != testCase.wantInt64 {
				t.Fatalf("ByteLength(%d).Int64() = (%d,%v), want (%d,nil)", testCase.value, got, err, testCase.wantInt64)
			}
		})
	}
}

func TestByteLengthStrictJSONBoundaryTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		want      uint64
		wantError bool
		wantOwn   bool
	}{
		{name: "zero", input: "0"},
		{name: "one", input: "1", want: 1},
		{name: "maximum", input: "18446744073709551615", want: math.MaxUint64},
		{name: "negative", input: "-1", wantError: true, wantOwn: true},
		{name: "fraction", input: "1.0", wantError: true, wantOwn: true},
		{name: "exponent", input: "1e3", wantError: true, wantOwn: true},
		{name: "quoted", input: `"1"`, wantError: true, wantOwn: true},
		{name: "null", input: "null", wantError: true, wantOwn: true},
		{name: "empty", input: "", wantError: true},
		{name: "overflow", input: "18446744073709551616", wantError: true, wantOwn: true},
		{name: "trailing token", input: "1 2", wantError: true},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			receiver := NewByteLength(73)
			err := json.Unmarshal([]byte(testCase.input), &receiver)
			if testCase.wantError {
				if err == nil || testCase.wantOwn && !errors.Is(err, ErrFoundationContract) || receiver.Uint64() != 73 {
					t.Fatalf("Unmarshal(%q) = (%d,%v), want unchanged receiver and typed rejection=%t", testCase.input, receiver.Uint64(), err, testCase.wantOwn)
				}
				return
			}
			if err != nil || receiver.Uint64() != testCase.want {
				t.Fatalf("Unmarshal(%q) = (%d,%v), want (%d,nil)", testCase.input, receiver.Uint64(), err, testCase.want)
			}
			encoded, marshalErr := json.Marshal(receiver)
			if marshalErr != nil || string(encoded) != testCase.input {
				t.Fatalf("Marshal(%d) = (%q,%v), want %q", receiver.Uint64(), encoded, marshalErr, testCase.input)
			}
		})
	}
}

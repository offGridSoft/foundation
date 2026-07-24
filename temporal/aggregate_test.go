package temporal

import (
	"encoding/json"
	"errors"
	"math"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

const (
	maxUint64Decimal   = "18446744073709551615"
	uint64CarryDecimal = "18446744073709551616"
	maxUint128Decimal  = "340282366920938463463374607431768211455"
)

func TestAggregateDurationCanonicalDecimalExtremeTable(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		text string
	}{
		{name: "zero", text: "0"},
		{name: "one", text: "1"},
		{name: "javascript unsafe boundary", text: "9007199254740993"},
		{name: "maximum unsigned 64", text: maxUint64Decimal},
		{name: "unsigned 64 carry", text: uint64CarryDecimal},
		{name: "maximum unsigned 128", text: maxUint128Decimal},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseAggregateDuration(test.text)
			if err != nil || got.Validate() != nil || got.Decimal() != test.text {
				t.Fatalf("ParseAggregateDuration() = (%q, %v), want (%q, nil)", got.Decimal(), err, test.text)
			}
			wire, err := json.Marshal(got)
			wantWire := `"` + test.text + `"`
			if err != nil || string(wire) != wantWire {
				t.Fatalf("AggregateDuration JSON = (%s, %v), want (%s, nil)", wire, err, wantWire)
			}
		})
	}
}

func TestAggregateDurationRejectsMalformedOverflowWithoutMutationTable(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name               string
		raw                string
		wantInvalidDecimal bool
	}{
		{name: "empty", raw: `""`, wantInvalidDecimal: true},
		{name: "leading zero", raw: `"00"`, wantInvalidDecimal: true},
		{name: "positive sign", raw: `"+1"`, wantInvalidDecimal: true},
		{name: "negative sign", raw: `"-1"`, wantInvalidDecimal: true},
		{name: "fraction", raw: `"1.0"`, wantInvalidDecimal: true},
		{name: "exponent", raw: `"1e0"`, wantInvalidDecimal: true},
		{name: "whitespace", raw: `"1 "`, wantInvalidDecimal: true},
		{name: "bare number", raw: `1`},
		{name: "null", raw: `null`},
		{name: "object", raw: `{}`},
		{name: "maximum plus one", raw: `"340282366920938463463374607431768211456"`, wantInvalidDecimal: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			value, err := ParseAggregateDuration("5")
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(test.raw), &value); !errors.Is(err, core.ErrTemporalContract) {
				t.Fatalf("AggregateDuration.UnmarshalJSON(%s) error = %v, want %v", test.raw, err, core.ErrTemporalContract)
			}
			if test.wantInvalidDecimal {
				var probe AggregateDuration
				if err := json.Unmarshal([]byte(test.raw), &probe); !errors.Is(err, core.ErrInvalidDecimal) {
					t.Fatalf("AggregateDuration.UnmarshalJSON(%s) error = %v, want %v", test.raw, err, core.ErrInvalidDecimal)
				}
			}
			if value.Decimal() != "5" {
				t.Fatalf("failed decode mutated AggregateDuration = %q, want 5", value.Decimal())
			}
		})
	}
}

func TestAggregateDurationAdditionCarryAndOverflowTable(t *testing.T) {
	t.Parallel()

	maxLow, err := ParseAggregateDuration(maxUint64Decimal)
	if err != nil {
		t.Fatal(err)
	}
	one, err := ParseAggregateDuration("1")
	if err != nil {
		t.Fatal(err)
	}
	carried, err := maxLow.Add(one)
	if err != nil || carried.Decimal() != uint64CarryDecimal {
		t.Fatalf("Add carry = (%q, %v), want (%q, nil)", carried.Decimal(), err, uint64CarryDecimal)
	}

	maximum, err := ParseAggregateDuration(maxUint128Decimal)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := maximum.Add(one); !errors.Is(err, core.ErrNumericOverflow) {
		t.Fatalf("Add overflow error = %v, want %v", err, core.ErrNumericOverflow)
	}

	duration, err := DurationFromNanoseconds(math.MaxInt64)
	if err != nil {
		t.Fatal(err)
	}
	aggregate := AggregateDurationFromDuration(duration)
	if aggregate.Decimal() != "9223372036854775807" {
		t.Fatalf("AggregateDurationFromDuration() = %q", aggregate.Decimal())
	}
}

func TestAggregateDurationAddDurationBoundary(t *testing.T) {
	t.Parallel()

	duration, err := DurationFromNanoseconds(math.MaxInt64)
	if err != nil {
		t.Fatal(err)
	}
	aggregate := AggregateDurationFromDuration(duration)
	withDuration, err := aggregate.AddDuration(duration)
	if err != nil || withDuration.Decimal() != "18446744073709551614" {
		t.Fatalf("AddDuration() = (%q, %v), want (18446744073709551614, nil)", withDuration.Decimal(), err)
	}
}

func TestAggregateDurationMultiplyExtremeTable(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		wantErr    error
		input      string
		name       string
		want       string
		multiplier uint64
	}{
		{name: "zero times maximum scalar", input: "0", multiplier: math.MaxUint64, want: "0"},
		{name: "one times maximum scalar", input: "1", multiplier: math.MaxUint64, want: maxUint64Decimal},
		{name: "high limb times maximum scalar", input: uint64CarryDecimal, multiplier: math.MaxUint64, want: "340282366920938463444927863358058659840"},
		{name: "both limbs propagate carry", input: "18446744073709551617", multiplier: 2, want: "36893488147419103234"},
		{name: "maximum times zero", input: maxUint128Decimal, want: "0"},
		{name: "maximum times one", input: maxUint128Decimal, multiplier: 1, want: maxUint128Decimal},
		{name: "maximum times two overflows", input: maxUint128Decimal, multiplier: 2, wantErr: core.ErrNumericOverflow},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			input, err := ParseAggregateDuration(test.input)
			if err != nil {
				t.Fatal(err)
			}
			got, err := input.Multiply(test.multiplier)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("Multiply() error = %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil || got.Decimal() != test.want {
				t.Fatalf("Multiply() = (%q, %v), want (%q, nil)", got.Decimal(), err, test.want)
			}
		})
	}
}

func FuzzAggregateDurationCanonicalRoundTrip(f *testing.F) {
	for _, seed := range []string{"0", "1", "9007199254740993", maxUint64Decimal, uint64CarryDecimal, maxUint128Decimal, "", "00", "-1"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		value, err := ParseAggregateDuration(input)
		if err != nil {
			return
		}
		if value.Decimal() != input {
			t.Fatalf("accepted non-canonical input %q as %q", input, value.Decimal())
		}
		wire, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("MarshalJSON(%q): %v", input, err)
		}
		var decoded AggregateDuration
		if err := json.Unmarshal(wire, &decoded); err != nil {
			t.Fatalf("UnmarshalJSON(%s): %v", wire, err)
		}
		if decoded != value {
			t.Fatalf("round trip = %q, want %q", decoded.Decimal(), value.Decimal())
		}
	})
}

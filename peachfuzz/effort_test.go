package peachfuzz

import (
	"errors"
	"math"
	"testing"

	foundationcore "github.com/offGridSoft/foundation/v2026/core"
)

func TestEffortNanosecondsDecimalBoundaries(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		text string
		want EffortNanoseconds
	}{
		{name: "zero", text: "0", want: EffortNanoseconds{}},
		{name: "low limb", text: "18446744073709551615", want: EffortNanoseconds{Low: math.MaxUint64}},
		{name: "carry", text: "18446744073709551616", want: EffortNanoseconds{High: 1}},
		{name: "maximum", text: "340282366920938463463374607431768211455", want: EffortNanoseconds{High: math.MaxUint64, Low: math.MaxUint64}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseEffortNanosecondsDecimal(tc.text)
			if err != nil {
				t.Fatalf("ParseEffortNanosecondsDecimal() error = %v", err)
			}
			if got != tc.want || got.Decimal() != tc.text {
				t.Fatalf("effort round trip = %+v %q, want %+v %q", got, got.Decimal(), tc.want, tc.text)
			}
		})
	}
}

func TestEffortNanosecondsRejectsNonCanonicalAndOverflow(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"", "00", "01", "-1", "+1", "1.0", "340282366920938463463374607431768211456"} {
		if _, err := ParseEffortNanosecondsDecimal(value); !errors.Is(err, ErrContract) {
			t.Fatalf("ParseEffortNanosecondsDecimal(%q) error = %v, want %v", value, err, ErrContract)
		}
	}
}

func TestEffortNanosecondsAddCarriesAndRejectsOverflow(t *testing.T) {
	t.Parallel()
	got, err := (EffortNanoseconds{Low: math.MaxUint64}).Add(EffortNanoseconds{Low: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got != (EffortNanoseconds{High: 1}) {
		t.Fatalf("Add() = %+v, want high-limb carry", got)
	}
	if _, err := (EffortNanoseconds{High: math.MaxUint64, Low: math.MaxUint64}).Add(EffortNanoseconds{Low: 1}); !errors.Is(err, foundationcore.ErrNumericOverflow) {
		t.Fatalf("overflow Add() error = %v, want %v", err, foundationcore.ErrNumericOverflow)
	}
}

func TestHumanizeEffortBeyondInt64Lifetime(t *testing.T) {
	t.Parallel()
	effort, err := ParseEffortNanosecondsDecimal("3155760000000000000000000")
	if err != nil {
		t.Fatal(err)
	}
	got, err := HumanizeEffort(effort)
	if err != nil {
		t.Fatal(err)
	}
	if got != "100000000.00 core-years" {
		t.Fatalf("HumanizeEffort() = %q, want 100000000.00 core-years", got)
	}
}

package currency

import (
	"errors"
	"math"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestParseDecimalHostileValidBoundaryTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want int64
		code Code
	}{
		{name: "usd zero without decimal", code: CodeUSD, raw: "0", want: 0},
		{name: "usd one minor unit exact", code: CodeUSD, raw: "0.01", want: 1},
		{name: "usd short fraction pads right", code: CodeUSD, raw: "1.2", want: 120},
		{name: "usd exact two digit fraction", code: CodeUSD, raw: "1.23", want: 123},
		{name: "usd omitted fraction becomes zero", code: CodeUSD, raw: "42", want: 4_200},
		{name: "usd leading zeros canonicalize", code: CodeUSD, raw: "00000000010.05", want: 1_005},
		{name: "usd negative one minor unit", code: CodeUSD, raw: "-0.01", want: -1},
		{name: "usd positive signed maximum", code: CodeUSD, raw: "92233720368547758.07", want: math.MaxInt64},
		{name: "usd negative signed minimum", code: CodeUSD, raw: "-92233720368547758.08", want: math.MinInt64},
		{name: "jpy zero exponent zero", code: CodeJPY, raw: "0", want: 0},
		{name: "jpy positive one", code: CodeJPY, raw: "1", want: 1},
		{name: "jpy negative one", code: CodeJPY, raw: "-1", want: -1},
		{name: "jpy positive signed maximum", code: CodeJPY, raw: "9223372036854775807", want: math.MaxInt64},
		{name: "jpy negative signed minimum", code: CodeJPY, raw: "-9223372036854775808", want: math.MinInt64},
		{name: "bhd short fraction pads two zeros", code: CodeBHD, raw: "1.2", want: 1_200},
		{name: "bhd exact three digit fraction", code: CodeBHD, raw: "1.234", want: 1_234},
		{name: "bhd negative exact three digit fraction", code: CodeBHD, raw: "-9.876", want: -9_876},
		{name: "clf shortest fraction pads three zeros", code: CodeCLF, raw: "1.2", want: 12_000},
		{name: "clf exact four digit fraction", code: CodeCLF, raw: "1.2345", want: 12_345},
		{name: "clf positive signed maximum", code: CodeCLF, raw: "922337203685477.5807", want: math.MaxInt64},
		{name: "clf negative signed minimum", code: CodeCLF, raw: "-922337203685477.5808", want: math.MinInt64},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := Parse(test.code, test.raw)
			if err != nil {
				t.Fatalf("Parse(%s, %q) error = %v, want nil", test.code, test.raw, err)
			}
			gotMinor, err := got.MinorUnits()
			if err != nil || gotMinor != test.want {
				t.Fatalf("Parse(%s, %q).MinorUnits() = (%d, %v), want (%d, nil)", test.code, test.raw, gotMinor, err, test.want)
			}
		})
	}
}

func TestParseDecimalHostileRejectionTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		code Code
	}{
		{name: "empty input", code: CodeUSD, raw: ""},
		{name: "plus sign", code: CodeUSD, raw: "+1.00"},
		{name: "minus sign without digits", code: CodeUSD, raw: "-"},
		{name: "leading ascii space", code: CodeUSD, raw: " 1.00"},
		{name: "trailing ascii newline", code: CodeUSD, raw: "1.00\n"},
		{name: "missing whole before point", code: CodeUSD, raw: ".99"},
		{name: "missing fraction after point", code: CodeUSD, raw: "1."},
		{name: "multiple decimal points", code: CodeUSD, raw: "1.0.0"},
		{name: "scientific notation", code: CodeUSD, raw: "1e3"},
		{name: "negative scientific notation", code: CodeUSD, raw: "-1E3"},
		{name: "comma grouping separator", code: CodeUSD, raw: "1,000.00"},
		{name: "underscore separator", code: CodeUSD, raw: "1_000.00"},
		{name: "embedded nul", code: CodeUSD, raw: "1\x00.00"},
		{name: "fullwidth digits", code: CodeUSD, raw: "１.００"},
		{name: "arabic indic digits", code: CodeUSD, raw: "١.٠٠"},
		{name: "usd third fractional digit", code: CodeUSD, raw: "1.001"},
		{name: "usd positive one beyond maximum", code: CodeUSD, raw: "92233720368547758.08"},
		{name: "usd negative one beyond minimum", code: CodeUSD, raw: "-92233720368547758.09"},
		{name: "usd huge whole overflow", code: CodeUSD, raw: "99999999999999999999999999999"},
		{name: "usd negative zero whole", code: CodeUSD, raw: "-0"},
		{name: "usd negative zero fixed", code: CodeUSD, raw: "-0.00"},
		{name: "usd negative zero leading zeros", code: CodeUSD, raw: "-000.0"},
		{name: "jpy rejects decimal point", code: CodeJPY, raw: "1.0"},
		{name: "jpy positive one beyond maximum", code: CodeJPY, raw: "9223372036854775808"},
		{name: "jpy negative one beyond minimum", code: CodeJPY, raw: "-9223372036854775809"},
		{name: "bhd fourth fractional digit", code: CodeBHD, raw: "1.0001"},
		{name: "clf fifth fractional digit", code: CodeCLF, raw: "1.00001"},
		{name: "one byte beyond input ceiling", code: CodeUSD, raw: "000000000000000000000000000000001"},
		{name: "unsupported currency zero value", code: CodeUnknown, raw: "1.00"},
		{name: "future currency enum", code: CodeCLF + 1, raw: "1.00"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := Parse(test.code, test.raw)
			if got != (Amount{}) {
				t.Fatalf("Parse(%d, %q) amount = %+v, want zero Amount", test.code, test.raw, got)
			}
			if !errors.Is(err, core.ErrCurrencyContract) {
				t.Fatalf("Parse(%d, %q) error = %v, want ErrCurrencyContract", test.code, test.raw, err)
			}
			if test.code.IsValid() {
				if !errors.Is(err, core.ErrCurrencyDecimal) || !errors.Is(err, core.ErrInvalidDecimal) {
					t.Fatalf("Parse(%s, %q) error = %v, want decimal identities", test.code, test.raw, err)
				}
			}
		})
	}
}

func TestDecimalExactRoundTripExtremeTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		want  string
		value int64
		code  Code
	}{
		{name: "usd negative signed minimum", code: CodeUSD, value: math.MinInt64, want: "-92233720368547758.08"},
		{name: "usd minimum plus one", code: CodeUSD, value: math.MinInt64 + 1, want: "-92233720368547758.07"},
		{name: "usd negative one", code: CodeUSD, value: -1, want: "-0.01"},
		{name: "usd zero", code: CodeUSD, value: 0, want: "0.00"},
		{name: "usd positive one", code: CodeUSD, value: 1, want: "0.01"},
		{name: "usd maximum minus one", code: CodeUSD, value: math.MaxInt64 - 1, want: "92233720368547758.06"},
		{name: "usd positive signed maximum", code: CodeUSD, value: math.MaxInt64, want: "92233720368547758.07"},
		{name: "jpy negative signed minimum", code: CodeJPY, value: math.MinInt64, want: "-9223372036854775808"},
		{name: "jpy zero", code: CodeJPY, value: 0, want: "0"},
		{name: "jpy positive signed maximum", code: CodeJPY, value: math.MaxInt64, want: "9223372036854775807"},
		{name: "bhd one major", code: CodeBHD, value: 1_000, want: "1.000"},
		{name: "clf one minor", code: CodeCLF, value: 1, want: "0.0001"},
		{name: "clf negative signed minimum", code: CodeCLF, value: math.MinInt64, want: "-922337203685477.5808"},
		{name: "clf positive signed maximum", code: CodeCLF, value: math.MaxInt64, want: "922337203685477.5807"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			amount, err := New(test.code, test.value)
			if err != nil {
				t.Fatalf("New(%s, %d) error = %v, want nil", test.code, test.value, err)
			}
			got, err := amount.Decimal()
			if err != nil || got != test.want {
				t.Fatalf("Amount(%s,%d).Decimal() = (%q, %v), want (%q, nil)", test.code, test.value, got, err, test.want)
			}
			roundTrip, err := Parse(test.code, got)
			if err != nil || roundTrip != amount {
				t.Fatalf("Parse(%s, Decimal()) = (%+v, %v), want (%+v, nil)", test.code, roundTrip, err, amount)
			}
		})
	}
}

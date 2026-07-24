package currency

import (
	"encoding/json"
	"errors"
	"math"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestHumanizeExplicitUnitsAndExponentClassesTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		want   string
		value  int64
		policy HumanizePolicy
		code   Code
	}{
		{name: "minor units preserve exact positive integer", code: CodeUSD, value: 12_345, policy: HumanizePolicy{Unit: DisplayUnitMinor}, want: "12345"},
		{name: "minor units preserve exact negative integer", code: CodeUSD, value: -12_345, policy: HumanizePolicy{Unit: DisplayUnitMinor}, want: "-12345"},
		{name: "major units use currency exponent two", code: CodeUSD, value: 12_345, policy: HumanizePolicy{Unit: DisplayUnitMajor, FractionDigits: 2}, want: "123.45"},
		{name: "hundreds scale major units", code: CodeUSD, value: 12_345, policy: HumanizePolicy{Unit: DisplayUnitHundreds, FractionDigits: 2}, want: "1.23"},
		{name: "thousands scale major units", code: CodeUSD, value: 12_345, policy: HumanizePolicy{Unit: DisplayUnitThousands, FractionDigits: 2}, want: "0.12"},
		{name: "millions scale major units", code: CodeUSD, value: 123_456_789, policy: HumanizePolicy{Unit: DisplayUnitMillions, FractionDigits: 6}, want: "1.234567"},
		{name: "billions scale major units", code: CodeUSD, value: 123_456_789_012, policy: HumanizePolicy{Unit: DisplayUnitBillions, FractionDigits: 6}, want: "1.234567"},
		{name: "zero exponent major remains exact", code: CodeJPY, value: 123, policy: HumanizePolicy{Unit: DisplayUnitMajor, FractionDigits: 2}, want: "123.00"},
		{name: "three exponent major remains exact", code: CodeBHD, value: 1_234, policy: HumanizePolicy{Unit: DisplayUnitMajor, FractionDigits: 3}, want: "1.234"},
		{name: "four exponent major remains exact", code: CodeCLF, value: 12_345, policy: HumanizePolicy{Unit: DisplayUnitMajor, FractionDigits: 4}, want: "1.2345"},
		{name: "fraction truncates toward zero positive", code: CodeUSD, value: 199, policy: HumanizePolicy{Unit: DisplayUnitMajor, FractionDigits: 1}, want: "1.9"},
		{name: "fraction truncates toward zero negative", code: CodeUSD, value: -199, policy: HumanizePolicy{Unit: DisplayUnitMajor, FractionDigits: 1}, want: "-1.9"},
		{name: "negative minor below visible major precision becomes signless zero", code: CodeUSD, value: -1, policy: HumanizePolicy{Unit: DisplayUnitMajor, FractionDigits: 1}, want: "0.0"},
		{name: "negative amount below visible thousand precision becomes signless zero", code: CodeUSD, value: -999, policy: HumanizePolicy{Unit: DisplayUnitThousands, FractionDigits: 2}, want: "0.00"},
		{name: "negative amount at first visible major precision retains sign", code: CodeUSD, value: -10, policy: HumanizePolicy{Unit: DisplayUnitMajor, FractionDigits: 1}, want: "-0.1"},
		{name: "negative amount one below first visible major precision becomes signless zero", code: CodeUSD, value: -9, policy: HumanizePolicy{Unit: DisplayUnitMajor, FractionDigits: 1}, want: "0.0"},
		{name: "negative amount at first visible billion precision retains sign", code: CodeCLF, value: -10_000_000, policy: HumanizePolicy{Unit: DisplayUnitBillions, FractionDigits: 6}, want: "-0.000001"},
		{name: "negative amount one below first visible billion precision becomes signless zero", code: CodeCLF, value: -9_999_999, policy: HumanizePolicy{Unit: DisplayUnitBillions, FractionDigits: 6}, want: "0.000000"},
		{name: "signed minimum formats without negation overflow", code: CodeJPY, value: math.MinInt64, policy: HumanizePolicy{Unit: DisplayUnitMajor}, want: "-9223372036854775808"},
		{name: "signed maximum formats exact major integer", code: CodeJPY, value: math.MaxInt64, policy: HumanizePolicy{Unit: DisplayUnitMajor}, want: "9223372036854775807"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			amount, err := New(test.code, test.value)
			if err != nil {
				t.Fatalf("New(%s,%d) error = %v, want nil", test.code, test.value, err)
			}
			got, err := amount.Humanize(test.policy)
			if err != nil {
				t.Fatalf("Humanize(%+v) error = %v, want nil", test.policy, err)
			}
			gotNumber, numberErr := got.Number()
			gotCode, codeErr := got.Code()
			gotUnit, unitErr := got.Unit()
			if numberErr != nil || codeErr != nil || unitErr != nil {
				t.Fatalf("Humanized accessors errors = (%v,%v,%v), want nil", numberErr, codeErr, unitErr)
			}
			if gotNumber != test.want || gotCode != test.code || gotUnit != test.policy.Unit {
				t.Fatalf("Humanize() = (%q,%s,%s), want (%q,%s,%s)", gotNumber, gotCode, gotUnit, test.want, test.code, test.policy.Unit)
			}
		})
	}
}

func TestHumanizeAutomaticThresholdExtremeTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		want     string
		value    int64
		code     Code
		wantUnit DisplayUnit
	}{
		{name: "zero selects major", code: CodeUSD, value: 0, wantUnit: DisplayUnitMajor, want: "0.00"},
		{name: "one minor below one major selects minor", code: CodeUSD, value: 1, wantUnit: DisplayUnitMinor, want: "1"},
		{name: "one below one major selects minor", code: CodeUSD, value: 99, wantUnit: DisplayUnitMinor, want: "99"},
		{name: "exact one major selects major", code: CodeUSD, value: 100, wantUnit: DisplayUnitMajor, want: "1.00"},
		{name: "one above one major selects major", code: CodeUSD, value: 101, wantUnit: DisplayUnitMajor, want: "1.01"},
		{name: "one below hundred major selects major", code: CodeUSD, value: 9_999, wantUnit: DisplayUnitMajor, want: "99.99"},
		{name: "exact hundred major selects hundreds", code: CodeUSD, value: 10_000, wantUnit: DisplayUnitHundreds, want: "1.00"},
		{name: "one above hundred major selects hundreds", code: CodeUSD, value: 10_001, wantUnit: DisplayUnitHundreds, want: "1.00"},
		{name: "one below thousand major selects hundreds", code: CodeUSD, value: 99_999, wantUnit: DisplayUnitHundreds, want: "9.99"},
		{name: "exact thousand major selects thousands", code: CodeUSD, value: 100_000, wantUnit: DisplayUnitThousands, want: "1.00"},
		{name: "one above thousand major selects thousands", code: CodeUSD, value: 100_001, wantUnit: DisplayUnitThousands, want: "1.00"},
		{name: "one below million major selects thousands", code: CodeUSD, value: 99_999_999, wantUnit: DisplayUnitThousands, want: "999.99"},
		{name: "exact million major selects millions", code: CodeUSD, value: 100_000_000, wantUnit: DisplayUnitMillions, want: "1.00"},
		{name: "one above million major selects millions", code: CodeUSD, value: 100_000_001, wantUnit: DisplayUnitMillions, want: "1.00"},
		{name: "one below billion major selects millions", code: CodeUSD, value: 99_999_999_999, wantUnit: DisplayUnitMillions, want: "999.99"},
		{name: "exact billion major selects billions", code: CodeUSD, value: 100_000_000_000, wantUnit: DisplayUnitBillions, want: "1.00"},
		{name: "one above billion major selects billions", code: CodeUSD, value: 100_000_000_001, wantUnit: DisplayUnitBillions, want: "1.00"},
		{name: "negative one minor uses absolute threshold", code: CodeUSD, value: -1, wantUnit: DisplayUnitMinor, want: "-1"},
		{name: "negative exact billion uses absolute threshold", code: CodeUSD, value: -100_000_000_000, wantUnit: DisplayUnitBillions, want: "-1.00"},
		{name: "four exponent one below major selects minor", code: CodeCLF, value: 9_999, wantUnit: DisplayUnitMinor, want: "9999"},
		{name: "four exponent exact major selects major", code: CodeCLF, value: 10_000, wantUnit: DisplayUnitMajor, want: "1.00"},
		{name: "zero exponent one is already major", code: CodeJPY, value: 1, wantUnit: DisplayUnitMajor, want: "1.00"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			amount, err := New(test.code, test.value)
			if err != nil {
				t.Fatalf("New(%s,%d) error = %v, want nil", test.code, test.value, err)
			}
			got, err := amount.Humanize(HumanizePolicy{Unit: DisplayUnitAutomatic, FractionDigits: 2})
			if err != nil {
				t.Fatalf("Humanize(automatic) error = %v, want nil", err)
			}
			gotNumber, numberErr := got.Number()
			gotUnit, unitErr := got.Unit()
			if numberErr != nil || unitErr != nil || gotNumber != test.want || gotUnit != test.wantUnit {
				t.Fatalf("Humanize(automatic) = (%q,%s,%v,%v), want (%q,%s,nil,nil)", gotNumber, gotUnit, numberErr, unitErr, test.want, test.wantUnit)
			}
		})
	}
}

func TestHumanizePolicyAndProjectionHostileTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		policy HumanizePolicy
	}{
		{name: "unknown unit zero", policy: HumanizePolicy{}},
		{name: "future unit one beyond domain", policy: HumanizePolicy{Unit: DisplayUnitBillions + 1}},
		{name: "maximum underlying unit", policy: HumanizePolicy{Unit: DisplayUnit(math.MaxUint8)}},
		{name: "fraction digits one above maximum", policy: HumanizePolicy{Unit: DisplayUnitMajor, FractionDigits: humanizeFractionDigitsMaximum + 1}},
		{name: "fraction digits maximum uint", policy: HumanizePolicy{Unit: DisplayUnitMajor, FractionDigits: math.MaxUint8}},
		{name: "minor unit rejects one fraction digit", policy: HumanizePolicy{Unit: DisplayUnitMinor, FractionDigits: 1}},
		{name: "minor unit rejects maximum fraction digits", policy: HumanizePolicy{Unit: DisplayUnitMinor, FractionDigits: humanizeFractionDigitsMaximum}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := test.policy.Validate(); !errors.Is(err, core.ErrCurrencyContract) {
				t.Fatalf("HumanizePolicy(%+v).Validate() error = %v, want ErrCurrencyContract", test.policy, err)
			}
		})
	}

	invalidProjections := []Humanized{
		{},
		{code: CodeUSD, unit: DisplayUnitAutomatic, number: "1.00", fractionDigits: 2},
		{code: CodeUSD, unit: DisplayUnitMajor, number: "-0.00", fractionDigits: 2},
		{code: CodeUSD, unit: DisplayUnitMajor, number: "01.00", fractionDigits: 2},
		{code: CodeUSD, unit: DisplayUnitMajor, number: "1.0", fractionDigits: 2},
		{code: CodeUSD, unit: DisplayUnitMinor, number: "1.0", fractionDigits: 1},
	}
	for index, projection := range invalidProjections {
		if err := projection.Validate(); !errors.Is(err, core.ErrCurrencyContract) {
			t.Fatalf("Humanized hostile row %d Validate() error = %v, want ErrCurrencyContract", index, err)
		}
	}
}

func TestDisplayUnitDomainAndJSONTable(t *testing.T) {
	t.Parallel()

	valid := []DisplayUnit{
		DisplayUnitAutomatic,
		DisplayUnitMinor,
		DisplayUnitMajor,
		DisplayUnitHundreds,
		DisplayUnitThousands,
		DisplayUnitMillions,
		DisplayUnitBillions,
	}
	for _, unit := range valid {
		proveDisplayUnitRoundTrip(t, unit)
	}

	for _, raw := range []string{"", "AUTO", "minor ", "BILLIONS", "trillions", "automatic\n"} {
		proveDisplayUnitRejection(t, raw)
	}
}

func proveDisplayUnitRoundTrip(t *testing.T, unit DisplayUnit) {
	t.Helper()

	if !unit.IsValid() {
		t.Fatalf("DisplayUnit(%d).IsValid() = false, want true", unit)
	}
	parsed, err := ParseDisplayUnit(unit.String())
	if err != nil || parsed != unit {
		t.Fatalf("ParseDisplayUnit(%q) = (%d,%v), want (%d,nil)", unit.String(), parsed, err, unit)
	}
	encoded, err := json.Marshal(unit)
	if err != nil {
		t.Fatalf("Marshal(DisplayUnit(%d)) error = %v, want nil", unit, err)
	}
	var decoded DisplayUnit
	if err := json.Unmarshal(encoded, &decoded); err != nil || decoded != unit {
		t.Fatalf("Unmarshal(%q) = (%d,%v), want (%d,nil)", encoded, decoded, err, unit)
	}
}

func proveDisplayUnitRejection(t *testing.T, raw string) {
	t.Helper()

	got, err := ParseDisplayUnit(raw)
	if got != DisplayUnitUnknown || !errors.Is(err, core.ErrCurrencyContract) {
		t.Fatalf("ParseDisplayUnit(%q) = (%d,%v), want (unknown,ErrCurrencyContract)", raw, got, err)
	}
}

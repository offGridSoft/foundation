package temporal

import (
	"errors"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestHumanizeAutomaticThresholdAndTruncationExtremeTable(t *testing.T) {
	t.Parallel()

	const (
		microsecond = int64(1_000)
		millisecond = int64(1_000_000)
		second      = int64(1_000_000_000)
		minute      = 60 * second
		hour        = 60 * minute
		day         = 24 * hour
		year        = 36525 * day / 100
	)
	policy := HumanizePolicy{Unit: HumanUnitAutomatic, FractionDigits: 1, Style: HumanizeStyleLong}
	for _, test := range []struct {
		name  string
		want  string
		nanos int64
	}{
		{name: "zero", nanos: 0, want: "0.0 nanoseconds"},
		{name: "one nanosecond", nanos: 1, want: "1.0 nanoseconds"},
		{name: "one below microsecond", nanos: microsecond - 1, want: "999.0 nanoseconds"},
		{name: "microsecond", nanos: microsecond, want: "1.0 microseconds"},
		{name: "one below millisecond", nanos: millisecond - 1, want: "999.9 microseconds"},
		{name: "millisecond", nanos: millisecond, want: "1.0 milliseconds"},
		{name: "one below second", nanos: second - 1, want: "999.9 milliseconds"},
		{name: "second", nanos: second, want: "1.0 seconds"},
		{name: "one below minute", nanos: minute - 1, want: "59.9 seconds"},
		{name: "minute", nanos: minute, want: "1.0 minutes"},
		{name: "one below hour", nanos: hour - 1, want: "59.9 minutes"},
		{name: "hour", nanos: hour, want: "1.0 hours"},
		{name: "one below day", nanos: day - 1, want: "23.9 hours"},
		{name: "day", nanos: day, want: "1.0 days"},
		{name: "one below year", nanos: year - 1, want: "365.2 days"},
		{name: "year", nanos: year, want: "1.0 years"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			duration, err := DurationFromNanoseconds(test.nanos)
			if err != nil {
				t.Fatal(err)
			}
			got, err := duration.Humanize(policy)
			if err != nil || got.Text() != test.want {
				t.Fatalf("Humanize() = (%q, %v), want (%q, nil)", got.Text(), err, test.want)
			}
		})
	}
}

func TestHumanizeOrdinaryAggregateParityAndMaximum(t *testing.T) {
	t.Parallel()

	policies := []HumanizePolicy{
		{Unit: HumanUnitAutomatic, FractionDigits: 0, Style: HumanizeStyleLong},
		{Unit: HumanUnitAutomatic, FractionDigits: 3, Style: HumanizeStyleCompact},
		{Unit: HumanUnitHours, FractionDigits: 9, Style: HumanizeStyleLong},
	}
	for _, nanos := range []int64{0, 1, 999, 1_000, 1_000_000_000, 31_557_600_000_000_000} {
		duration, err := DurationFromNanoseconds(nanos)
		if err != nil {
			t.Fatal(err)
		}
		aggregate := AggregateDurationFromDuration(duration)
		for _, policy := range policies {
			fromDuration, durationErr := duration.Humanize(policy)
			fromAggregate, aggregateErr := aggregate.Humanize(policy)
			if durationErr != nil || aggregateErr != nil || fromDuration != fromAggregate {
				t.Fatalf("humanize parity nanos=%d policy=%+v got=(%+v,%v)/(%+v,%v)", nanos, policy, fromDuration, durationErr, fromAggregate, aggregateErr)
			}
		}
	}

	maximum, err := ParseAggregateDuration(maxUint128Decimal)
	if err != nil {
		t.Fatal(err)
	}
	got, err := maximum.Humanize(HumanizePolicy{Unit: HumanUnitYears, FractionDigits: 2, Style: HumanizeStyleLong})
	if err != nil || got.Text() != "10782897524556318080696.07 years" {
		t.Fatalf("maximum Humanize() = (%q, %v)", got.Text(), err)
	}
}

func TestHumanizePolicyAndEnumsHostileTable(t *testing.T) {
	t.Parallel()

	for _, policy := range []HumanizePolicy{
		{},
		{Unit: HumanUnitUnknown, Style: HumanizeStyleLong},
		{Unit: HumanUnitAutomatic, Style: HumanizeStyleUnknown},
		{Unit: HumanUnitAutomatic, FractionDigits: 10, Style: HumanizeStyleLong},
		{Unit: HumanUnit(255), Style: HumanizeStyleLong},
	} {
		duration, err := DurationFromNanoseconds(1)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := duration.Humanize(policy); !errors.Is(err, core.ErrTemporalContract) {
			t.Fatalf("Humanize(%+v) error = %v, want %v", policy, err, core.ErrTemporalContract)
		}
	}

	for _, unit := range []HumanUnit{
		HumanUnitAutomatic,
		HumanUnitNanoseconds,
		HumanUnitMicroseconds,
		HumanUnitMilliseconds,
		HumanUnitSeconds,
		HumanUnitMinutes,
		HumanUnitHours,
		HumanUnitDays,
		HumanUnitYears,
	} {
		if err := unit.Validate(); err != nil {
			t.Fatalf("HumanUnit(%d).Validate() = %v", unit, err)
		}
	}
	for _, style := range []HumanizeStyle{HumanizeStyleLong, HumanizeStyleCompact} {
		if err := style.Validate(); err != nil {
			t.Fatalf("HumanizeStyle(%d).Validate() = %v", style, err)
		}
	}
}

func TestHumanizeExactFractionAndStyleBytesTable(t *testing.T) {
	t.Parallel()

	duration, err := DurationFromNanoseconds(1_500_000_000)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		want   string
		digits uint8
		style  HumanizeStyle
	}{
		{name: "long integer", style: HumanizeStyleLong, want: "1 seconds"},
		{name: "long one digit", digits: 1, style: HumanizeStyleLong, want: "1.5 seconds"},
		{name: "long exact trailing zeros", digits: 9, style: HumanizeStyleLong, want: "1.500000000 seconds"},
		{name: "compact integer no separator", style: HumanizeStyleCompact, want: "1s"},
		{name: "compact exact trailing zeros", digits: 9, style: HumanizeStyleCompact, want: "1.500000000s"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := duration.Humanize(HumanizePolicy{
				Unit:           HumanUnitSeconds,
				Style:          test.style,
				FractionDigits: test.digits,
			})
			if err != nil || got.Text() != test.want {
				t.Fatalf("Humanize() = (%q, %v), want (%q, nil)", got.Text(), err, test.want)
			}
			if got.Unit() != HumanUnitSeconds {
				t.Fatalf("Humanize().Unit() = %v, want %v", got.Unit(), HumanUnitSeconds)
			}
		})
	}
}

func TestHumanizeEveryPermittedFractionPrecisionTable(t *testing.T) {
	t.Parallel()

	duration, err := DurationFromNanoseconds(1)
	if err != nil {
		t.Fatal(err)
	}
	for digits := uint8(0); digits <= maximumFractionDigits; digits++ {
		got, err := duration.Humanize(HumanizePolicy{
			Unit:           HumanUnitSeconds,
			Style:          HumanizeStyleCompact,
			FractionDigits: digits,
		})
		if err != nil {
			t.Fatalf("Humanize(FractionDigits=%d) error = %v", digits, err)
		}
		wantLength := 1
		if digits > 0 {
			wantLength += 1 + int(digits)
		}
		if len(got.Number()) != wantLength {
			t.Fatalf("Humanize(FractionDigits=%d).Number() length = %d, want %d", digits, len(got.Number()), wantLength)
		}
	}
}

func TestHumanizedRejectsImpossibleStateTable(t *testing.T) {
	t.Parallel()

	for _, value := range []Humanized{
		{},
		{number: "1", unit: HumanUnitAutomatic, style: HumanizeStyleLong},
		{number: "1", unit: HumanUnitSeconds, style: HumanizeStyleUnknown},
		{number: "", unit: HumanUnitSeconds, style: HumanizeStyleLong},
		{number: "01", unit: HumanUnitSeconds, style: HumanizeStyleLong},
		{number: "1.", unit: HumanUnitSeconds, style: HumanizeStyleLong},
		{number: "1.0000000000", unit: HumanUnitSeconds, style: HumanizeStyleLong},
		{number: "1.x", unit: HumanUnitSeconds, style: HumanizeStyleLong},
	} {
		if err := value.Validate(); !errors.Is(err, core.ErrTemporalContract) {
			t.Fatalf("Humanized%+v.Validate() error = %v, want %v", value, err, core.ErrTemporalContract)
		}
	}
}

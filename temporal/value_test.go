package temporal

import (
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestInstantConstructionExtremeTable(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		input   time.Time
		wantErr error
		name    string
		want    int64
	}{
		{name: "epoch is set", input: time.Unix(0, 0).UTC()},
		{name: "one nanosecond", input: time.Unix(0, 1).UTC(), want: 1},
		{name: "non UTC normalizes", input: time.Unix(0, 7).In(time.FixedZone("hostile", -11*60*60)), want: 7},
		{name: "maximum signed nanoseconds", input: time.Unix(0, math.MaxInt64).UTC(), want: math.MaxInt64},
		{name: "zero time is unset", wantErr: core.ErrTemporalContract},
		{name: "one before epoch", input: time.Unix(-1, 999_999_999).UTC(), wantErr: core.ErrTemporalContract},
		{name: "one beyond signed nanosecond range", input: time.Unix(0, math.MaxInt64).Add(time.Nanosecond), wantErr: core.ErrNumericOverflow},
		{name: "far future outside signed nanosecond range", input: time.Date(3000, time.January, 1, 0, 0, 0, 0, time.UTC), wantErr: core.ErrNumericOverflow},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewInstant(test.input)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("NewInstant() error = %v, want %v", err, test.wantErr)
				}
				return
			}
			nanos, nanosErr := got.Nanoseconds()
			if err != nil || nanosErr != nil || nanos != test.want || !got.IsSet() {
				t.Fatalf("NewInstant() = (%d, %v, %v, set=%t), want (%d, nil, nil, true)", nanos, err, nanosErr, got.IsSet(), test.want)
			}
		})
	}
}

func TestUnsetInstantFailsEveryProjectionAndOperationTable(t *testing.T) {
	t.Parallel()

	epoch, err := InstantFromNanoseconds(0)
	if err != nil {
		t.Fatal(err)
	}
	zeroDuration, err := DurationFromNanoseconds(0)
	if err != nil {
		t.Fatal(err)
	}
	var unset Instant
	for _, test := range []struct {
		run  func() error
		name string
	}{
		{name: "validate", run: unset.Validate},
		{name: "nanoseconds", run: func() error { _, callErr := unset.Nanoseconds(); return callErr }},
		{name: "time", run: func() error { _, callErr := unset.Time(); return callErr }},
		{name: "add receiver", run: func() error { _, callErr := unset.Add(zeroDuration); return callErr }},
		{name: "subtract receiver", run: func() error { _, callErr := unset.Subtract(zeroDuration); return callErr }},
		{name: "since receiver", run: func() error { _, callErr := unset.Since(epoch); return callErr }},
		{name: "since operand", run: func() error { _, callErr := epoch.Since(unset); return callErr }},
		{name: "compare receiver", run: func() error { order, callErr := unset.Compare(epoch); requireUnknownOrder(t, order); return callErr }},
		{name: "compare operand", run: func() error { order, callErr := epoch.Compare(unset); requireUnknownOrder(t, order); return callErr }},
		{name: "firestore", run: func() error { _, callErr := unset.Firestore(); return callErr }},
		{name: "postgresql", run: func() error { _, callErr := unset.PostgreSQL(); return callErr }},
		{name: "marshal json", run: func() error { _, callErr := json.Marshal(unset); return callErr }},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if callErr := test.run(); !errors.Is(callErr, core.ErrTemporalContract) {
				t.Fatalf("%s error = %v, want %v", test.name, callErr, core.ErrTemporalContract)
			}
		})
	}
	if unset.IsSet() {
		t.Fatal("zero Instant.IsSet() = true, want false")
	}
}

func TestInstantArithmeticBoundaryTable(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		wantErr  error
		name     string
		start    int64
		duration int64
		want     int64
		add      bool
	}{
		{name: "add zero", start: 0, add: true},
		{name: "add ordinary", start: 40, duration: 2, want: 42, add: true},
		{name: "add exact maximum", start: math.MaxInt64 - 1, duration: 1, want: math.MaxInt64, add: true},
		{name: "add one past maximum", start: math.MaxInt64, duration: 1, add: true, wantErr: core.ErrNumericOverflow},
		{name: "subtract zero", start: 0},
		{name: "subtract ordinary", start: 42, duration: 2, want: 40},
		{name: "subtract exact epoch", start: 1, duration: 1},
		{name: "subtract one before epoch", start: 0, duration: 1, wantErr: core.ErrNumericOverflow},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			start, startErr := InstantFromNanoseconds(test.start)
			duration, durationErr := DurationFromNanoseconds(test.duration)
			if startErr != nil || durationErr != nil {
				t.Fatalf("construct errors = (%v, %v)", startErr, durationErr)
			}
			var (
				got Instant
				err error
			)
			if test.add {
				got, err = start.Add(duration)
			} else {
				got, err = start.Subtract(duration)
			}
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("arithmetic error = %v, want %v", err, test.wantErr)
				}
				return
			}
			nanos, nanosErr := got.Nanoseconds()
			if err != nil || nanosErr != nil || nanos != test.want {
				t.Fatalf("arithmetic = (%d, %v, %v), want (%d, nil, nil)", nanos, err, nanosErr, test.want)
			}
		})
	}
}

func TestTemporalOrderingExtremeTable(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name  string
		left  int64
		right int64
		want  Order
	}{
		{name: "before at zero edge", left: 0, right: 1, want: OrderBefore},
		{name: "equal at maximum", left: math.MaxInt64, right: math.MaxInt64, want: OrderEqual},
		{name: "after at maximum edge", left: math.MaxInt64, right: math.MaxInt64 - 1, want: OrderAfter},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			leftInstant, _ := InstantFromNanoseconds(test.left)
			rightInstant, _ := InstantFromNanoseconds(test.right)
			leftDuration, _ := DurationFromNanoseconds(test.left)
			rightDuration, _ := DurationFromNanoseconds(test.right)
			leftAggregate := AggregateDurationFromDuration(leftDuration)
			rightAggregate := AggregateDurationFromDuration(rightDuration)

			instantOrder, err := leftInstant.Compare(rightInstant)
			if err != nil || instantOrder != test.want {
				t.Fatalf("Instant.Compare() = (%v, %v), want (%v, nil)", instantOrder, err, test.want)
			}
			if order := leftDuration.Compare(rightDuration); order != test.want {
				t.Fatalf("Duration.Compare() = %v, want %v", order, test.want)
			}
			if order := leftAggregate.Compare(rightAggregate); order != test.want {
				t.Fatalf("AggregateDuration.Compare() = %v, want %v", order, test.want)
			}
		})
	}
}

func TestDurationArithmeticExtremeTable(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		wantErr    error
		name       string
		nanos      int64
		multiplier uint64
		want       int64
	}{
		{name: "zero times maximum", multiplier: math.MaxUint64},
		{name: "one times one", nanos: 1, multiplier: 1, want: 1},
		{name: "ordinary multiply", nanos: 21, multiplier: 2, want: 42},
		{name: "exact maximum", nanos: 1, multiplier: math.MaxInt64, want: math.MaxInt64},
		{name: "one past maximum", nanos: math.MaxInt64, multiplier: 2, wantErr: core.ErrNumericOverflow},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			duration, err := DurationFromNanoseconds(test.nanos)
			if err != nil {
				t.Fatal(err)
			}
			got, err := duration.Multiply(test.multiplier)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("Duration.Multiply() error = %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil || got.Nanoseconds() != test.want {
				t.Fatalf("Duration.Multiply() = (%d, %v), want (%d, nil)", got.Nanoseconds(), err, test.want)
			}
		})
	}
}

func TestDurationAddExtremeTable(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		wantErr error
		name    string
		left    int64
		right   int64
		want    int64
	}{
		{name: "zero plus zero"},
		{name: "ordinary", left: 40, right: 2, want: 42},
		{name: "exact maximum", left: math.MaxInt64 - 1, right: 1, want: math.MaxInt64},
		{name: "one beyond maximum", left: math.MaxInt64, right: 1, wantErr: core.ErrNumericOverflow},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			left, leftErr := DurationFromNanoseconds(test.left)
			right, rightErr := DurationFromNanoseconds(test.right)
			if leftErr != nil || rightErr != nil {
				t.Fatalf("construct errors = (%v, %v)", leftErr, rightErr)
			}
			got, err := left.Add(right)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("Duration.Add() error = %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil || got.Nanoseconds() != test.want {
				t.Fatalf("Duration.Add() = (%d, %v), want (%d, nil)", got.Nanoseconds(), err, test.want)
			}
		})
	}
}

func TestDurationConstructionAndStdlibExtremeTable(t *testing.T) {
	t.Parallel()

	for _, input := range []time.Duration{0, 1, time.Duration(math.MaxInt64)} {
		got, err := NewDuration(input)
		stdlib, stdlibErr := got.Stdlib()
		if err != nil || stdlibErr != nil || stdlib != input {
			t.Fatalf("duration %d = (%d, %v, %v), want (%d, nil, nil)", input, stdlib, err, stdlibErr, input)
		}
	}
	for _, constructor := range []struct {
		run  func() error
		name string
	}{
		{name: "stdlib duration", run: func() error { _, err := NewDuration(-1); return err }},
		{name: "nanoseconds", run: func() error { _, err := DurationFromNanoseconds(-1); return err }},
		{name: "instant nanoseconds", run: func() error { _, err := InstantFromNanoseconds(-1); return err }},
	} {
		if err := constructor.run(); !errors.Is(err, core.ErrTemporalContract) {
			t.Fatalf("%s negative error = %v, want %v", constructor.name, err, core.ErrTemporalContract)
		}
	}

	zero, err := DurationFromNanoseconds(0)
	if err != nil || !zero.IsZero() || zero.Aggregate().Decimal() != "0" {
		t.Fatalf("zero duration = (%+v, %v), want valid zero aggregate", zero, err)
	}
}

func TestSinceExtremeTable(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		wantErr error
		name    string
		earlier int64
		later   int64
		want    int64
	}{
		{name: "same instant", earlier: 7, later: 7},
		{name: "one nanosecond", earlier: 7, later: 8, want: 1},
		{name: "full signed range", later: math.MaxInt64, want: math.MaxInt64},
		{name: "reversed", earlier: 8, later: 7, wantErr: core.ErrTemporalContract},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			earlier, _ := InstantFromNanoseconds(test.earlier)
			later, _ := InstantFromNanoseconds(test.later)
			got, err := later.Since(earlier)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("Since() error = %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil || got.Nanoseconds() != test.want {
				t.Fatalf("Since() = (%d, %v), want (%d, nil)", got.Nanoseconds(), err, test.want)
			}
		})
	}
}

func TestTemporalErrorIdentityHierarchyTable(t *testing.T) {
	t.Parallel()

	maximum, err := InstantFromNanoseconds(math.MaxInt64)
	if err != nil {
		t.Fatal(err)
	}
	one, err := DurationFromNanoseconds(1)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		run        func() error
		additional error
		name       string
	}{
		{name: "unset instant", run: func() error { var value Instant; return value.Validate() }},
		{name: "negative duration", run: func() error { _, callErr := DurationFromNanoseconds(-1); return callErr }},
		{name: "malformed decimal", additional: core.ErrInvalidDecimal, run: func() error { _, callErr := ParseAggregateDuration("00"); return callErr }},
		{name: "numeric overflow", additional: core.ErrNumericOverflow, run: func() error { _, callErr := maximum.Add(one); return callErr }},
		{name: "unknown enum", run: OrderUnknown.Validate},
		{name: "contradictory persistence", run: func() error {
			return FirestoreInstant{Nanoseconds: 1, QueryTimestamp: time.Unix(0, 1).UTC()}.Validate()
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := test.run()
			if !errors.Is(got, core.ErrTemporalContract) || !errors.Is(got, core.ErrFoundationContract) {
				t.Fatalf("error = %v, want temporal and foundation identities", got)
			}
			if test.additional != nil && !errors.Is(got, test.additional) {
				t.Fatalf("error = %v, want additional identity %v", got, test.additional)
			}
		})
	}
}

func TestTemporalScalarJSONRejectsWithoutMutationTable(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		raw  string
	}{
		{name: "bare number", raw: `5`},
		{name: "empty string", raw: `""`},
		{name: "leading zero", raw: `"05"`},
		{name: "positive sign", raw: `"+5"`},
		{name: "negative", raw: `"-1"`},
		{name: "fraction", raw: `"5.0"`},
		{name: "exponent", raw: `"5e0"`},
		{name: "inside whitespace", raw: `" 5"`},
		{name: "non ascii digit", raw: `"５"`},
		{name: "null", raw: `null`},
		{name: "object", raw: `{}`},
		{name: "overflow", raw: `"9223372036854775808"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			instant, _ := InstantFromNanoseconds(7)
			duration, _ := DurationFromNanoseconds(7)
			if err := json.Unmarshal([]byte(test.raw), &instant); !errors.Is(err, core.ErrTemporalContract) {
				t.Fatalf("Instant.UnmarshalJSON(%s) error = %v, want %v", test.raw, err, core.ErrTemporalContract)
			}
			if err := json.Unmarshal([]byte(test.raw), &duration); !errors.Is(err, core.ErrTemporalContract) {
				t.Fatalf("Duration.UnmarshalJSON(%s) error = %v, want %v", test.raw, err, core.ErrTemporalContract)
			}
			instantNanos, instantErr := instant.Nanoseconds()
			if instantErr != nil || instantNanos != 7 || duration.Nanoseconds() != 7 {
				t.Fatalf("failed decode mutated values = (%d, %d, %v), want (7, 7, nil)", instantNanos, duration.Nanoseconds(), instantErr)
			}
		})
	}

}

func TestTemporalScalarJSONPreservesEpochAndUnsafeIntegersTable(t *testing.T) {
	t.Parallel()

	epoch, err := InstantFromNanoseconds(0)
	if err != nil {
		t.Fatal(err)
	}
	var decodedEpoch Instant
	if err := json.Unmarshal([]byte(`"0"`), &decodedEpoch); err != nil || decodedEpoch != epoch || !decodedEpoch.IsSet() {
		t.Fatalf("epoch JSON decode = (%+v, %v, set=%t), want (%+v, nil, true)", decodedEpoch, err, decodedEpoch.IsSet(), epoch)
	}

	instant, _ := InstantFromNanoseconds(9_007_199_254_740_993)
	duration, _ := DurationFromNanoseconds(math.MaxInt64)
	for _, test := range []struct {
		value any
		name  string
		want  string
	}{
		{name: "javascript unsafe instant remains exact", value: instant, want: `"9007199254740993"`},
		{name: "maximum duration remains exact", value: duration, want: `"9223372036854775807"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, marshalErr := json.Marshal(test.value)
			if marshalErr != nil || string(got) != test.want {
				t.Fatalf("json.Marshal() = (%s, %v), want (%s, nil)", got, marshalErr, test.want)
			}
		})
	}
}

func requireUnknownOrder(t *testing.T, got Order) {
	t.Helper()
	if got != OrderUnknown {
		t.Fatalf("failed Compare order = %v, want %v", got, OrderUnknown)
	}
}

func FuzzSignedTemporalCanonicalRoundTrip(f *testing.F) {
	for _, seed := range []string{"0", "1", "9007199254740993", "9223372036854775807", "", "00", "-1", "1e0", "５"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		wire := []byte(strconv.Quote(input))
		var duration Duration
		durationErr := json.Unmarshal(wire, &duration)
		var instant Instant
		instantErr := json.Unmarshal(wire, &instant)
		if durationErr != nil || instantErr != nil {
			return
		}
		requireSignedTemporalCanonical(t, input, wire, duration, instant)
	})
}

func requireSignedTemporalCanonical(t *testing.T, input string, wire []byte, duration Duration, instant Instant) {
	t.Helper()

	if strconv.FormatInt(duration.Nanoseconds(), 10) != input {
		t.Fatalf("Duration accepted non-canonical input %q", input)
	}
	nanoseconds, err := instant.Nanoseconds()
	if err != nil || strconv.FormatInt(nanoseconds, 10) != input {
		t.Fatalf("Instant accepted non-canonical input %q as %d with error %v", input, nanoseconds, err)
	}
	durationWire, durationErr := json.Marshal(duration)
	instantWire, instantErr := json.Marshal(instant)
	if durationErr != nil || instantErr != nil ||
		string(durationWire) != string(wire) || string(instantWire) != string(wire) {
		t.Fatalf("round trip %q = duration(%s,%v) instant(%s,%v)", input, durationWire, durationErr, instantWire, instantErr)
	}
}

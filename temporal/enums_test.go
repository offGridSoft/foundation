package temporal

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func TestOrderClosedEnumAndJSONExtremeTable(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name  string
		token string
		value Order
	}{
		{name: "before", token: "before", value: OrderBefore},
		{name: "equal", token: "equal", value: OrderEqual},
		{name: "after", token: "after", value: OrderAfter},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			parsed, err := ParseOrder(test.token)
			wire, marshalErr := json.Marshal(test.value)
			if err != nil || marshalErr != nil || parsed != test.value || string(wire) != `"`+test.token+`"` {
				t.Fatalf("Order %q = (parsed=%v, wire=%s, errors=%v/%v)", test.token, parsed, wire, err, marshalErr)
			}
		})
	}

}

func TestOrderRejectsUnknownAndOutOfRangeTable(t *testing.T) {
	t.Parallel()

	for _, value := range []Order{OrderUnknown, Order(4), Order(255)} {
		if value.IsValid() || !errors.Is(value.Validate(), core.ErrTemporalContract) {
			t.Fatalf("Order(%d) accepted outside closed domain", value)
		}
		if value.String() != "" {
			t.Fatalf("Order(%d).String() = %q, want empty", value, value.String())
		}
		if _, err := json.Marshal(value); !errors.Is(err, core.ErrTemporalContract) {
			t.Fatalf("json.Marshal(Order(%d)) error = %v, want %v", value, err, core.ErrTemporalContract)
		}
	}
}

func TestHumanUnitClosedEnumAndJSONExtremeTable(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name  string
		token string
		value HumanUnit
	}{
		{name: "automatic", token: "automatic", value: HumanUnitAutomatic},
		{name: "nanoseconds", token: "nanoseconds", value: HumanUnitNanoseconds},
		{name: "microseconds", token: "microseconds", value: HumanUnitMicroseconds},
		{name: "milliseconds", token: "milliseconds", value: HumanUnitMilliseconds},
		{name: "seconds", token: "seconds", value: HumanUnitSeconds},
		{name: "minutes", token: "minutes", value: HumanUnitMinutes},
		{name: "hours", token: "hours", value: HumanUnitHours},
		{name: "days", token: "days", value: HumanUnitDays},
		{name: "years", token: "years", value: HumanUnitYears},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			parsed, err := ParseHumanUnit(test.token)
			wire, marshalErr := json.Marshal(test.value)
			if err != nil || marshalErr != nil || parsed != test.value || string(wire) != `"`+test.token+`"` {
				t.Fatalf("HumanUnit %q = (parsed=%v, wire=%s, errors=%v/%v)", test.token, parsed, wire, err, marshalErr)
			}
		})
	}
	for _, value := range []HumanUnit{HumanUnitUnknown, HumanUnit(10), HumanUnit(255)} {
		if value.String() != "" || value.IsValid() {
			t.Fatalf("HumanUnit(%d) escaped closed domain", value)
		}
		if _, err := json.Marshal(value); !errors.Is(err, core.ErrTemporalContract) {
			t.Fatalf("json.Marshal(HumanUnit(%d)) error = %v, want %v", value, err, core.ErrTemporalContract)
		}
	}
}

func TestHumanizeStyleClosedEnumAndJSONExtremeTable(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name  string
		token string
		value HumanizeStyle
	}{
		{name: "long", token: "long", value: HumanizeStyleLong},
		{name: "compact", token: "compact", value: HumanizeStyleCompact},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			parsed, err := ParseHumanizeStyle(test.token)
			wire, marshalErr := json.Marshal(test.value)
			if err != nil || marshalErr != nil || parsed != test.value || string(wire) != `"`+test.token+`"` {
				t.Fatalf("HumanizeStyle %q = (parsed=%v, wire=%s, errors=%v/%v)", test.token, parsed, wire, err, marshalErr)
			}
		})
	}
	for _, value := range []HumanizeStyle{HumanizeStyleUnknown, HumanizeStyle(3), HumanizeStyle(255)} {
		if value.String() != "" || value.IsValid() {
			t.Fatalf("HumanizeStyle(%d) escaped closed domain", value)
		}
		if _, err := json.Marshal(value); !errors.Is(err, core.ErrTemporalContract) {
			t.Fatalf("json.Marshal(HumanizeStyle(%d)) error = %v, want %v", value, err, core.ErrTemporalContract)
		}
	}
}

func TestTemporalEnumsRejectHostileJSONWithoutMutationTable(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{`""`, `"unknown"`, `"Before"`, `"before "`, `"\u0062efore"`, `0`, `null`, `{}`} {
		order := OrderAfter
		unit := HumanUnitYears
		style := HumanizeStyleCompact
		orderErr := json.Unmarshal([]byte(raw), &order)
		unitErr := json.Unmarshal([]byte(raw), &unit)
		styleErr := json.Unmarshal([]byte(raw), &style)
		if !errors.Is(orderErr, core.ErrTemporalContract) || order != OrderAfter {
			t.Fatalf("Order.UnmarshalJSON(%s) = (%v, %v), want temporal error and unchanged", raw, order, orderErr)
		}
		if !errors.Is(unitErr, core.ErrTemporalContract) || unit != HumanUnitYears {
			t.Fatalf("HumanUnit.UnmarshalJSON(%s) = (%v, %v), want temporal error and unchanged", raw, unit, unitErr)
		}
		if !errors.Is(styleErr, core.ErrTemporalContract) || style != HumanizeStyleCompact {
			t.Fatalf("HumanizeStyle.UnmarshalJSON(%s) = (%v, %v), want temporal error and unchanged", raw, style, styleErr)
		}
	}

	var order *Order
	var unit *HumanUnit
	var style *HumanizeStyle
	for _, test := range []struct {
		run  func() error
		name string
	}{
		{name: "nil order receiver", run: func() error { return order.UnmarshalJSON([]byte(`"before"`)) }},
		{name: "nil unit receiver", run: func() error { return unit.UnmarshalJSON([]byte(`"seconds"`)) }},
		{name: "nil style receiver", run: func() error { return style.UnmarshalJSON([]byte(`"long"`)) }},
	} {
		if err := test.run(); !errors.Is(err, core.ErrTemporalContract) {
			t.Fatalf("%s error = %v, want %v", test.name, err, core.ErrTemporalContract)
		}
	}
}

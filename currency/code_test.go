package currency

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

type codeDomainCase struct {
	name           string
	wantToken      string
	code           Code
	wantFractional uint8
}

func TestCodeDomainExhaustiveTable(t *testing.T) {
	t.Parallel()

	tests := []codeDomainCase{
		{name: "usd owns two fractional digits", code: CodeUSD, wantToken: "USD", wantFractional: 2},
		{name: "eur owns two fractional digits", code: CodeEUR, wantToken: "EUR", wantFractional: 2},
		{name: "gbp owns two fractional digits", code: CodeGBP, wantToken: "GBP", wantFractional: 2},
		{name: "cad owns two fractional digits", code: CodeCAD, wantToken: "CAD", wantFractional: 2},
		{name: "aud owns two fractional digits", code: CodeAUD, wantToken: "AUD", wantFractional: 2},
		{name: "jpy owns zero fractional digits", code: CodeJPY, wantToken: "JPY", wantFractional: 0},
		{name: "chf owns two fractional digits", code: CodeCHF, wantToken: "CHF", wantFractional: 2},
		{name: "nzd owns two fractional digits", code: CodeNZD, wantToken: "NZD", wantFractional: 2},
		{name: "sgd owns two fractional digits", code: CodeSGD, wantToken: "SGD", wantFractional: 2},
		{name: "hkd owns two fractional digits", code: CodeHKD, wantToken: "HKD", wantFractional: 2},
		{name: "bhd owns three fractional digits", code: CodeBHD, wantToken: "BHD", wantFractional: 3},
		{name: "clf owns four fractional digits", code: CodeCLF, wantToken: "CLF", wantFractional: 4},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			proveCodeDomainRow(t, test)
		})
	}
}

func proveCodeDomainRow(t *testing.T, test codeDomainCase) {
	t.Helper()

	if !test.code.IsValid() {
		t.Fatalf("Code(%d).IsValid() = false, want true", test.code)
	}
	if err := test.code.Validate(); err != nil {
		t.Fatalf("Code(%d).Validate() error = %v, want nil", test.code, err)
	}
	if got := test.code.String(); got != test.wantToken {
		t.Fatalf("Code(%d).String() = %q, want %q", test.code, got, test.wantToken)
	}
	gotDigits, err := test.code.FractionDigits()
	if err != nil || gotDigits != test.wantFractional {
		t.Fatalf("Code(%d).FractionDigits() = (%d, %v), want (%d, nil)", test.code, gotDigits, err, test.wantFractional)
	}
	gotCode, err := ParseCode(test.wantToken)
	if err != nil || gotCode != test.code {
		t.Fatalf("ParseCode(%q) = (%d, %v), want (%d, nil)", test.wantToken, gotCode, err, test.code)
	}
	proveCodeJSONRoundTrip(t, test)
}

func proveCodeJSONRoundTrip(t *testing.T, test codeDomainCase) {
	t.Helper()

	encoded, err := json.Marshal(test.code)
	if err != nil || string(encoded) != `"`+test.wantToken+`"` {
		t.Fatalf("Marshal(Code(%d)) = (%q, %v), want canonical token", test.code, encoded, err)
	}
	var decoded Code
	if err := json.Unmarshal(encoded, &decoded); err != nil || decoded != test.code {
		t.Fatalf("Unmarshal(%q) = (%d, %v), want (%d, nil)", encoded, decoded, err, test.code)
	}
}

func TestCodeHostileUnknownAndWireTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		code Code
		enum bool
	}{
		{name: "unknown zero enum", code: CodeUnknown, enum: true},
		{name: "one beyond declared enum", code: CodeCLF + 1, enum: true},
		{name: "future enum gap fourteen", code: 14, enum: true},
		{name: "future enum forty two", code: 42, enum: true},
		{name: "maximum underlying enum", code: Code(math.MaxUint8), enum: true},
		{name: "empty token", raw: ""},
		{name: "lowercase canonical letters", raw: "usd"},
		{name: "mixed case canonical letters", raw: "Usd"},
		{name: "leading space", raw: " USD"},
		{name: "trailing newline", raw: "USD\n"},
		{name: "four ascii letters", raw: "USDX"},
		{name: "two ascii letters", raw: "US"},
		{name: "fullwidth confusable letters", raw: "ＵＳＤ"},
		{name: "embedded nul", raw: "US\x00D"},
		{name: "unsupported iso token", raw: "BRL"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if test.enum {
				if test.code.IsValid() {
					t.Fatalf("Code(%d).IsValid() = true, want false", test.code)
				}
				if err := test.code.Validate(); !errors.Is(err, core.ErrCurrencyContract) {
					t.Fatalf("Code(%d).Validate() error = %v, want ErrCurrencyContract", test.code, err)
				}
				if got := test.code.String(); got != "" {
					t.Fatalf("Code(%d).String() = %q, want empty", test.code, got)
				}
				return
			}
			got, err := ParseCode(test.raw)
			if got != CodeUnknown || !errors.Is(err, core.ErrCurrencyContract) {
				t.Fatalf("ParseCode(%q) = (%d, %v), want (CodeUnknown, ErrCurrencyContract)", test.raw, got, err)
			}
		})
	}
}

func TestCodeUnmarshalHostileReceiverNonMutationTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
	}{
		{name: "empty json", raw: ""},
		{name: "json null", raw: "null"},
		{name: "json boolean", raw: "true"},
		{name: "json number", raw: "1"},
		{name: "json array", raw: `["USD"]`},
		{name: "json object", raw: `{"currency":"USD"}`},
		{name: "lowercase token", raw: `"usd"`},
		{name: "escaped canonical token", raw: `"\u0055SD"`},
		{name: "leading json whitespace", raw: ` "USD"`},
		{name: "trailing json whitespace", raw: `"USD" `},
		{name: "unknown uppercase token", raw: `"ZZZ"`},
		{name: "trailing json value", raw: `"USD""CAD"`},
		{name: "one byte beyond enum json ceiling", raw: `"` + strings.Repeat("U", canonicalEnumJSONMaximumBytes-1) + `"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			receiver := CodeCAD
			err := receiver.UnmarshalJSON([]byte(test.raw))
			if !errors.Is(err, core.ErrCurrencyContract) {
				t.Fatalf("Unmarshal(%q) error = %v, want ErrCurrencyContract", test.raw, err)
			}
			if receiver != CodeCAD {
				t.Fatalf("Unmarshal(%q) receiver = %d, want preserved CodeCAD", test.raw, receiver)
			}
		})
	}
}

func TestOrderDomainExhaustiveAndHostileTable(t *testing.T) {
	t.Parallel()

	valid := []struct {
		name  string
		token string
		order Order
	}{
		{name: "less order", order: OrderLess, token: "less"},
		{name: "equal order", order: OrderEqual, token: "equal"},
		{name: "greater order", order: OrderGreater, token: "greater"},
	}
	for _, test := range valid {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			proveOrderDomainRow(t, test.order, test.token)
		})
	}

	for _, order := range []Order{OrderUnknown, OrderGreater + 1, 42, Order(math.MaxUint8)} {
		proveInvalidOrder(t, order)
	}
}

func proveOrderDomainRow(t *testing.T, order Order, token string) {
	t.Helper()

	if !order.IsValid() || order.String() != token {
		t.Fatalf("Order(%d) = valid %t token %q, want true and %q", order, order.IsValid(), order.String(), token)
	}
	encoded, err := json.Marshal(order)
	if err != nil || string(encoded) != `"`+token+`"` {
		t.Fatalf("Marshal(Order(%d)) = (%q, %v), want canonical", order, encoded, err)
	}
	var decoded Order
	if err := json.Unmarshal(encoded, &decoded); err != nil || decoded != order {
		t.Fatalf("Unmarshal(%q) = (%d, %v), want (%d, nil)", encoded, decoded, err, order)
	}
}

func proveInvalidOrder(t *testing.T, order Order) {
	t.Helper()

	if order.IsValid() {
		t.Fatalf("Order(%d).IsValid() = true, want false", order)
	}
	if err := order.Validate(); !errors.Is(err, core.ErrCurrencyContract) {
		t.Fatalf("Order(%d).Validate() error = %v, want ErrCurrencyContract", order, err)
	}
}

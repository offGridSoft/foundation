package currency

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

func FuzzParseDecimalSemanticRoundTrip(f *testing.F) {
	f.Add(uint8(CodeUSD), "0.01")
	f.Add(uint8(CodeUSD), "-92233720368547758.08")
	f.Add(uint8(CodeJPY), "9223372036854775807")
	f.Add(uint8(CodeBHD), "1.234")
	f.Add(uint8(CodeCLF), "1.2345")
	f.Add(uint8(CodeUnknown), "1.00")
	f.Add(uint8(CodeCLF+1), "1.00")
	f.Add(uint8(CodeUSD), "1e3")
	f.Add(uint8(CodeUSD), "-0.00")

	f.Fuzz(func(t *testing.T, rawCode uint8, raw string) {
		code := Code(rawCode)
		got, err := Parse(code, raw)
		if err != nil {
			proveRejectedFuzzDecimal(t, code, raw, got, err)
			return
		}
		proveAcceptedFuzzDecimal(t, code, raw, got)
	})
}

func proveRejectedFuzzDecimal(t *testing.T, code Code, raw string, got Amount, err error) {
	t.Helper()

	if got != (Amount{}) {
		t.Fatalf("Parse(%d,%q) rejected with nonzero amount %+v", code, raw, got)
	}
	if !errors.Is(err, core.ErrCurrencyContract) {
		t.Fatalf("Parse(%d,%q) error = %v, want ErrCurrencyContract", code, raw, err)
	}
	if code.IsValid() && (!errors.Is(err, core.ErrCurrencyDecimal) || !errors.Is(err, core.ErrInvalidDecimal)) {
		t.Fatalf("Parse(%s,%q) error = %v, want decimal identities", code, raw, err)
	}
}

func proveAcceptedFuzzDecimal(t *testing.T, code Code, raw string, got Amount) {
	t.Helper()

	if err := got.Validate(); err != nil {
		t.Fatalf("Parse(%s,%q) accepted invalid amount: %v", code, raw, err)
	}
	canonical, err := got.Decimal()
	if err != nil {
		t.Fatalf("Amount.Decimal() error = %v, want nil", err)
	}
	roundTrip, err := Parse(code, canonical)
	if err != nil || roundTrip != got {
		t.Fatalf("Parse(%s, Decimal()) = (%+v,%v), want (%+v,nil)", code, roundTrip, err, got)
	}
}

func FuzzAmountJSONSemanticCanonicalRoundTrip(f *testing.F) {
	f.Add([]byte(`{"currency":"USD","minor_units":"0"}`))
	f.Add([]byte(`{"currency":"CAD","minor_units":"-1"}`))
	f.Add([]byte(`{"currency":"JPY","minor_units":"9223372036854775807"}`))
	f.Add([]byte(`{"currency":"BHD","minor_units":"-9223372036854775808"}`))
	f.Add([]byte(`{"currency":"usd","minor_units":"1"}`))
	f.Add([]byte(`{"currency":"USD","minor_units":"\u0031"}`))
	f.Add([]byte(`{"currency":"USD","minor_units":"-0"}`))
	f.Add([]byte(`{"currency":"USD","minor_units":"1","extra":true}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		receiver, err := New(CodeCAD, 73)
		if err != nil {
			t.Fatal(err)
		}
		prior := receiver
		err = receiver.UnmarshalJSON(data)
		if err != nil {
			proveRejectedFuzzJSON(t, receiver, prior, err)
			return
		}
		proveAcceptedFuzzJSON(t, receiver)
	})
}

func proveRejectedFuzzJSON(t *testing.T, got, want Amount, err error) {
	t.Helper()

	if got != want {
		t.Fatalf("UnmarshalJSON(rejected) receiver = %+v, want preserved %+v", got, want)
	}
	if !errors.Is(err, core.ErrCurrencyContract) {
		t.Fatalf("UnmarshalJSON(rejected) error = %v, want ErrCurrencyContract", err)
	}
}

func proveAcceptedFuzzJSON(t *testing.T, receiver Amount) {
	t.Helper()

	if err := receiver.Validate(); err != nil {
		t.Fatalf("UnmarshalJSON accepted invalid amount: %v", err)
	}
	canonical, err := json.Marshal(receiver)
	if err != nil {
		t.Fatalf("Marshal(accepted amount) error = %v, want nil", err)
	}
	if len(canonical) > core.StrictJSONMaxBytes {
		t.Fatalf("canonical bytes = %d, want <= %d", len(canonical), core.StrictJSONMaxBytes)
	}
	proveCanonicalFuzzJSON(t, receiver, canonical)
}

func proveCanonicalFuzzJSON(t *testing.T, receiver Amount, canonical []byte) {
	t.Helper()

	var roundTrip Amount
	if err := roundTrip.UnmarshalJSON(canonical); err != nil || roundTrip != receiver {
		t.Fatalf("canonical round trip = (%+v,%v), want (%+v,nil)", roundTrip, err, receiver)
	}
	second, err := json.Marshal(roundTrip)
	if err != nil || !bytes.Equal(second, canonical) {
		t.Fatalf("second canonical encoding = (%q,%v), want (%q,nil)", second, err, canonical)
	}
}

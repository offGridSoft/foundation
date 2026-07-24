package currency

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/offGridSoft/foundation/v2026/core"
)

type persistenceRoundTripCase struct {
	name  string
	value int64
	code  Code
}

func TestAmountJSONCanonicalSignedBoundaryTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		want  string
		value int64
		code  Code
	}{
		{name: "negative signed minimum preserved as string", code: CodeUSD, value: math.MinInt64, want: `{"currency":"USD","minor_units":"-9223372036854775808"}`},
		{name: "negative one preserved as string", code: CodeCAD, value: -1, want: `{"currency":"CAD","minor_units":"-1"}`},
		{name: "zero preserved as canonical string", code: CodeJPY, value: 0, want: `{"currency":"JPY","minor_units":"0"}`},
		{name: "positive one preserved as string", code: CodeBHD, value: 1, want: `{"currency":"BHD","minor_units":"1"}`},
		{name: "positive signed maximum preserved as string", code: CodeCLF, value: math.MaxInt64, want: `{"currency":"CLF","minor_units":"9223372036854775807"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			amount, err := New(test.code, test.value)
			if err != nil {
				t.Fatalf("New(%s,%d) error = %v, want nil", test.code, test.value, err)
			}
			encoded, err := json.Marshal(amount)
			if err != nil || string(encoded) != test.want {
				t.Fatalf("Marshal(%s,%d) = (%q,%v), want (%q,nil)", test.code, test.value, encoded, err, test.want)
			}
			var decoded Amount
			if err := json.Unmarshal(encoded, &decoded); err != nil || decoded != amount {
				t.Fatalf("Unmarshal(%q) = (%+v,%v), want (%+v,nil)", encoded, decoded, err, amount)
			}
			reencoded, err := json.Marshal(decoded)
			if err != nil || !bytes.Equal(reencoded, encoded) {
				t.Fatalf("second canonical encoding = (%q,%v), want (%q,nil)", reencoded, err, encoded)
			}
		})
	}
}

func TestAmountJSONHostileClosedBoundaryTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
	}{
		{name: "empty input", raw: ""},
		{name: "json null", raw: "null"},
		{name: "json array", raw: `["USD","1"]`},
		{name: "json string", raw: `"USD 1"`},
		{name: "missing currency", raw: `{"minor_units":"1"}`},
		{name: "missing minor units", raw: `{"currency":"USD"}`},
		{name: "unknown field", raw: `{"currency":"USD","minor_units":"1","amount":"1"}`},
		{name: "duplicate currency exact", raw: `{"currency":"USD","currency":"CAD","minor_units":"1"}`},
		{name: "duplicate currency case variant", raw: `{"currency":"USD","Currency":"CAD","minor_units":"1"}`},
		{name: "duplicate minor units", raw: `{"currency":"USD","minor_units":"1","minor_units":"2"}`},
		{name: "lowercase currency", raw: `{"currency":"usd","minor_units":"1"}`},
		{name: "unsupported currency", raw: `{"currency":"BRL","minor_units":"1"}`},
		{name: "escaped currency canonical bytes", raw: `{"currency":"\u0055SD","minor_units":"1"}`},
		{name: "numeric minor units", raw: `{"currency":"USD","minor_units":1}`},
		{name: "null minor units", raw: `{"currency":"USD","minor_units":null}`},
		{name: "empty minor units", raw: `{"currency":"USD","minor_units":""}`},
		{name: "positive sign minor units", raw: `{"currency":"USD","minor_units":"+1"}`},
		{name: "negative zero minor units", raw: `{"currency":"USD","minor_units":"-0"}`},
		{name: "leading zero minor units", raw: `{"currency":"USD","minor_units":"01"}`},
		{name: "negative leading zero minor units", raw: `{"currency":"USD","minor_units":"-01"}`},
		{name: "fractional minor units", raw: `{"currency":"USD","minor_units":"1.0"}`},
		{name: "scientific minor units", raw: `{"currency":"USD","minor_units":"1e3"}`},
		{name: "positive overflow minor units", raw: `{"currency":"USD","minor_units":"9223372036854775808"}`},
		{name: "negative overflow minor units", raw: `{"currency":"USD","minor_units":"-9223372036854775809"}`},
		{name: "minor units one byte above scalar ceiling", raw: `{"currency":"USD","minor_units":"` + strings.Repeat("1", canonicalMinorUnitsJSONMaximumBytes-1) + `"}`},
		{name: "escaped digit minor units", raw: `{"currency":"USD","minor_units":"\u0031"}`},
		{name: "trailing json value", raw: `{"currency":"USD","minor_units":"1"}{}`},
		{name: "invalid utf8 currency byte", raw: "{\"currency\":\"US\xff\",\"minor_units\":\"1\"}"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			receiver, err := New(CodeCAD, 73)
			if err != nil {
				t.Fatal(err)
			}
			want := receiver
			err = receiver.UnmarshalJSON([]byte(test.raw))
			if !errors.Is(err, core.ErrCurrencyContract) {
				t.Fatalf("Unmarshal(%q) error = %v, want ErrCurrencyContract", test.raw, err)
			}
			if receiver != want {
				t.Fatalf("Unmarshal(%q) receiver = %+v, want preserved %+v", test.raw, receiver, want)
			}
		})
	}
}

func TestAmountJSONResourceCeilingAndNilReceiver(t *testing.T) {
	t.Parallel()

	oversized := strings.Repeat(" ", core.StrictJSONMaxBytes+1)
	receiver, err := New(CodeUSD, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := receiver
	if err := receiver.UnmarshalJSON([]byte(oversized)); !errors.Is(err, core.ErrCurrencyContract) {
		t.Fatalf("Unmarshal(over byte ceiling) error = %v, want ErrCurrencyContract", err)
	}
	if receiver != want {
		t.Fatalf("Unmarshal(over byte ceiling) receiver = %+v, want preserved %+v", receiver, want)
	}

	var nilAmount *Amount
	if err := nilAmount.UnmarshalJSON([]byte(`{"currency":"USD","minor_units":"1"}`)); !errors.Is(err, core.ErrCurrencyContract) {
		t.Fatalf("nil Amount.UnmarshalJSON() error = %v, want ErrCurrencyContract", err)
	}
}

func TestPersistenceProjectionExactSignedRoundTripTable(t *testing.T) {
	t.Parallel()

	tests := []persistenceRoundTripCase{
		{name: "usd negative signed minimum", code: CodeUSD, value: math.MinInt64},
		{name: "cad negative one", code: CodeCAD, value: -1},
		{name: "jpy zero", code: CodeJPY, value: 0},
		{name: "bhd positive one", code: CodeBHD, value: 1},
		{name: "clf positive signed maximum", code: CodeCLF, value: math.MaxInt64},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			provePersistenceRoundTrip(t, test)
		})
	}
}

func provePersistenceRoundTrip(t *testing.T, test persistenceRoundTripCase) {
	t.Helper()

	amount, err := New(test.code, test.value)
	if err != nil {
		t.Fatalf("New(%s,%d) error = %v, want nil", test.code, test.value, err)
	}
	proveFirestoreRoundTrip(t, test, amount)
	provePostgreSQLRoundTrip(t, test, amount)
}

func proveFirestoreRoundTrip(t *testing.T, test persistenceRoundTripCase, amount Amount) {
	t.Helper()

	firestore, err := amount.Firestore()
	if err != nil {
		t.Fatalf("Firestore() error = %v, want nil", err)
	}
	if firestore.Currency != test.code.String() || firestore.MinorUnits != test.value {
		t.Fatalf("Firestore() = %+v, want currency %s minor %d", firestore, test.code, test.value)
	}
	got, err := FromFirestore(firestore)
	if err != nil || got != amount {
		t.Fatalf("FromFirestore(%+v) = (%+v,%v), want (%+v,nil)", firestore, got, err, amount)
	}
}

func provePostgreSQLRoundTrip(t *testing.T, test persistenceRoundTripCase, amount Amount) {
	t.Helper()

	postgresql, err := amount.PostgreSQL()
	if err != nil {
		t.Fatalf("PostgreSQL() error = %v, want nil", err)
	}
	if postgresql.Currency != test.code.String() || postgresql.MinorUnits != test.value {
		t.Fatalf("PostgreSQL() = %+v, want currency %s minor %d", postgresql, test.code, test.value)
	}
	got, err := FromPostgreSQL(postgresql)
	if err != nil || got != amount {
		t.Fatalf("FromPostgreSQL(%+v) = (%+v,%v), want (%+v,nil)", postgresql, got, err, amount)
	}
}

func TestPersistenceProjectionHostileCurrencyTable(t *testing.T) {
	t.Parallel()

	tokens := []string{"", "usd", "Usd", " USD", "USD ", "US", "USDX", "BRL", "ＵＳＤ", "US\x00D"}
	for _, token := range tokens {
		firestore := FirestoreAmount{Currency: token, MinorUnits: math.MinInt64}
		if got, err := FromFirestore(firestore); got != (Amount{}) || !errors.Is(err, core.ErrCurrencyContract) {
			t.Fatalf("FromFirestore(%q) = (%+v,%v), want (zero,ErrCurrencyContract)", token, got, err)
		}
		postgresql := PostgreSQLAmount{Currency: token, MinorUnits: math.MaxInt64}
		if got, err := FromPostgreSQL(postgresql); got != (Amount{}) || !errors.Is(err, core.ErrCurrencyContract) {
			t.Fatalf("FromPostgreSQL(%q) = (%+v,%v), want (zero,ErrCurrencyContract)", token, got, err)
		}
	}
}

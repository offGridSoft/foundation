package core

import (
	"crypto/sha256"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	json "github.com/goccy/go-json"
)

func TestUnixNanoTimeJSONIsBareNanoseconds(t *testing.T) {
	t.Parallel()
	now := time.Unix(0, 1782302400000000000).UTC()
	got, err := json.Marshal(NewUnixNanoTime(now))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `1782302400000000000` {
		t.Fatalf("UnixNanoTime JSON = %s", got)
	}
}

func TestUnixNanoTimeHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		raw string
	}{
		{raw: `"1782302400000000000"`},
		{raw: `1782302400000000000.0`},
		{raw: `+1782302400000000000`},
		{raw: `01782302400000000000`},
		{raw: `-1`},
		{raw: `9223372036854775808`},
		{raw: `{}`},
		{raw: `null`},
		{raw: ``},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			t.Parallel()
			value := UnixNanoTimeFromInt64(5)
			if err := value.UnmarshalJSON([]byte(tc.raw)); !errors.Is(err, ErrFoundationContract) {
				t.Fatalf("UnixNanoTime.UnmarshalJSON(%s) error = %v, want ErrFoundationContract", tc.raw, err)
			}
			if value.UnixNano() != 5 {
				t.Fatalf("failed unmarshal mutated UnixNanoTime = %d, want 5", value.UnixNano())
			}
		})
	}

	value := UnixNanoTimeFromInt64(1782302400000000000)
	if got := value.Time().UnixNano(); got != value.UnixNano() {
		t.Fatalf("UnixNanoTime.Time().UnixNano() = %d, want %d", got, value.UnixNano())
	}
	if _, err := json.Marshal(UnixNanoTimeFromInt64(-1)); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("UnixNanoTime.MarshalJSON(negative) error = %v, want ErrFoundationContract", err)
	}
}

func TestNanosecondsDurationJSONIsBareNanoseconds(t *testing.T) {
	t.Parallel()
	got, err := json.Marshal(NewNanosecondsDuration(3 * time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `3000000000` {
		t.Fatalf("NanosecondsDuration JSON = %s", got)
	}
}

func TestNanosecondsDurationHostileTable(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{`"3000000000"`, `3.5`, `+3`, `03`, `-1`, `{}`, `null`, ``} {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			value := NanosecondsDurationFromInt64(5)
			if err := value.UnmarshalJSON([]byte(raw)); !errors.Is(err, ErrFoundationContract) {
				t.Fatalf("NanosecondsDuration.UnmarshalJSON(%s) error = %v, want ErrFoundationContract", raw, err)
			}
			if value.Nanoseconds() != 5 {
				t.Fatalf("failed unmarshal mutated NanosecondsDuration = %d, want 5", value.Nanoseconds())
			}
		})
	}
	if _, err := json.Marshal(NanosecondsDurationFromInt64(-1)); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("NanosecondsDuration.MarshalJSON(negative) error = %v, want ErrFoundationContract", err)
	}
}

func TestMoneyPenniesHostileTable(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{`"100"`, `1.5`, `+1`, `01`, `-1`, `{}`, `null`, ``} {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			value := NewMoneyPennies(5)
			if err := value.UnmarshalJSON([]byte(raw)); !errors.Is(err, ErrFoundationContract) {
				t.Fatalf("MoneyPennies.UnmarshalJSON(%s) error = %v, want ErrFoundationContract", raw, err)
			}
			if value.Uint64() != 5 {
				t.Fatalf("failed unmarshal mutated MoneyPennies = %d, want 5", value.Uint64())
			}
		})
	}

	for _, amount := range []MoneyPennies{NewMoneyPennies(0), NewMoneyPennies(1), NewMoneyPennies(100)} {
		if err := amount.Validate(); err != nil {
			t.Fatalf("MoneyPennies(%d).Validate() = %v, want nil", amount.Uint64(), err)
		}
		data, err := json.Marshal(amount)
		if err != nil {
			t.Fatalf("MoneyPennies(%d).MarshalJSON() = %v", amount.Uint64(), err)
		}
		var decoded MoneyPennies
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("MoneyPennies round trip %s: %v", data, err)
		}
		if decoded != amount {
			t.Fatalf("MoneyPennies round trip = %d, want %d", decoded.Uint64(), amount.Uint64())
		}
	}

}

func TestMoneyPenniesArithmeticHostileTable(t *testing.T) {
	t.Parallel()
	sum, err := NewMoneyPennies(40).Add(NewMoneyPennies(2))
	if err != nil {
		t.Fatalf("MoneyPennies.Add valid: %v", err)
	}
	if sum.Uint64() != 42 {
		t.Fatalf("MoneyPennies.Add = %d, want 42", sum.Uint64())
	}
	diff, err := NewMoneyPennies(40).Sub(NewMoneyPennies(2))
	if err != nil {
		t.Fatalf("MoneyPennies.Sub valid: %v", err)
	}
	if diff.Uint64() != 38 {
		t.Fatalf("MoneyPennies.Sub = %d, want 38", diff.Uint64())
	}
	product, err := NewMoneyPennies(25).MulQuantity(4)
	if err != nil {
		t.Fatalf("MoneyPennies.MulQuantity valid: %v", err)
	}
	if product.Uint64() != 100 {
		t.Fatalf("MoneyPennies.MulQuantity = %d, want 100", product.Uint64())
	}

	for _, tc := range []struct {
		run  func() error
		name string
	}{
		{name: "add overflow", run: func() error {
			_, err := NewMoneyPennies(math.MaxUint64).Add(NewMoneyPennies(1))
			return err
		}},
		{name: "sub underflow", run: func() error {
			_, err := NewMoneyPennies(1).Sub(NewMoneyPennies(2))
			return err
		}},
		{name: "mul overflow", run: func() error {
			_, err := NewMoneyPennies(math.MaxUint64).MulQuantity(2)
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.run(); !errors.Is(err, ErrFoundationContract) {
				t.Fatalf("%s error = %v, want ErrFoundationContract", tc.name, err)
			}
		})
	}
}

func TestSHA256HexHostileTable(t *testing.T) {
	t.Parallel()
	constructed := NewSHA256Hex(sha256.Sum256([]byte("foundation")))
	if err := constructed.Validate(); err != nil {
		t.Fatalf("NewSHA256Hex produced invalid digest: %v", err)
	}
	for _, tc := range []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "lowercase fixed hex accepted", value: strings.Repeat("a", 64)},
		{name: "uppercase rejected", value: strings.Repeat("A", 64), wantErr: true},
		{name: "short rejected", value: strings.Repeat("a", 63), wantErr: true},
		{name: "non hex rejected", value: strings.Repeat("g", 64), wantErr: true},
		{name: "empty rejected", value: "", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseSHA256Hex(tc.value)
			if tc.wantErr {
				if !errors.Is(err, ErrFoundationContract) {
					t.Fatalf("ParseSHA256Hex error = %v, want ErrFoundationContract", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestHTTPContractsHostileTable(t *testing.T) {
	t.Parallel()
	for _, outcome := range []HTTPOutcome{HTTPOutcomeSuccess, HTTPOutcomeRetryable, HTTPOutcomeTerminal} {
		t.Run(outcome.String(), func(t *testing.T) {
			t.Parallel()
			if !outcome.IsValid() {
				t.Fatalf("%s should be valid", outcome)
			}
			data, err := outcome.MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}
			var roundTrip HTTPOutcome
			if err := json.Unmarshal(data, &roundTrip); err != nil {
				t.Fatal(err)
			}
			if roundTrip != outcome {
				t.Fatalf("roundTrip = %s, want %s", roundTrip, outcome)
			}
		})
	}
	for _, raw := range []string{`"unknown"`, `0`, `""`} {
		var outcome HTTPOutcome
		if err := json.Unmarshal([]byte(raw), &outcome); !errors.Is(err, ErrFoundationContract) {
			t.Fatalf("HTTPOutcome(%s) error = %v, want ErrFoundationContract", raw, err)
		}
	}
}

func TestOpaqueTokenBoundariesRejectControlsAndEdges(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		run  func() error
		name string
	}{
		{name: "api request id rejects crlf", run: func() error {
			return NewAPIRequestID("evil\r\nX-Injected: 1").Validate()
		}},
		{name: "api request id rejects leading space", run: func() error {
			return NewAPIRequestID(" req-1").Validate()
		}},
		{name: "signing key id rejects nul", run: func() error {
			_, err := ParseSigningKeyID("key\x001")
			return err
		}},
		{name: "product version rejects newline", run: func() error {
			_, err := ParseProductVersion("1.0\n")
			return err
		}},
		{name: "lease id rejects tab", run: func() error {
			_, err := ParseLeaseID("lease\t1")
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.run(); !errors.Is(err, ErrFoundationContract) {
				t.Fatalf("%s error = %v, want ErrFoundationContract", tc.name, err)
			}
		})
	}
}

func TestHTTPHeaderValidationHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		run  func() error
		name string
	}{
		{name: "header name rejects edge spaces", run: func() error { return ValidateHTTPHeaderName(" X-Foo ") }},
		{name: "header name rejects colon", run: func() error { return ValidateHTTPHeaderName("X-Foo: bar") }},
		{name: "header name rejects newline", run: func() error { return ValidateHTTPHeaderName("X-Foo\n") }},
		{name: "header value rejects nul", run: func() error { return ValidateHTTPHeaderValue("a\x00b") }},
		{name: "header value rejects newline", run: func() error { return ValidateHTTPHeaderValue("a\nb") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.run(); !errors.Is(err, ErrFoundationContract) {
				t.Fatalf("%s error = %v, want ErrFoundationContract", tc.name, err)
			}
		})
	}
	if err := ValidateHTTPHeaderValue("cache control\tok"); err != nil {
		t.Fatalf("ValidateHTTPHeaderValue tab = %v, want nil", err)
	}
}

func TestBackoffPolicyValidateHostileTable(t *testing.T) {
	t.Parallel()
	valid := BackoffPolicy{
		Base:        time.Second,
		Max:         time.Minute,
		MaxAttempts: 3,
	}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, policy := range []BackoffPolicy{
		{Base: time.Second, Max: time.Minute},
		{Base: 0, Max: time.Minute, MaxAttempts: 1},
		{Base: time.Minute, Max: time.Second, MaxAttempts: 1},
	} {
		if err := policy.Validate(); !errors.Is(err, ErrFoundationContract) {
			t.Fatalf("BackoffPolicy.Validate error = %v, want ErrFoundationContract", err)
		}
	}
}

func TestBLAKE3HexHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "lowercase fixed hex accepted", value: strings.Repeat("b", 64)},
		{name: "uppercase rejected", value: strings.Repeat("B", 64), wantErr: true},
		{name: "short rejected", value: strings.Repeat("b", 63), wantErr: true},
		{name: "non hex rejected", value: strings.Repeat("x", 64), wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseBLAKE3Hex(tc.value)
			if tc.wantErr {
				if !errors.Is(err, ErrFoundationContract) {
					t.Fatalf("ParseBLAKE3Hex error = %v, want ErrFoundationContract", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestByteCountRejectsZero(t *testing.T) {
	t.Parallel()
	var count ByteCount
	if err := json.Unmarshal([]byte(`0`), &count); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("ByteCount zero error = %v, want ErrFoundationContract", err)
	}
}

func TestDecodeStrictJSONHostileTable(t *testing.T) {
	t.Parallel()
	type payload struct {
		Name string `json:"name"`
	}
	for _, tc := range []struct {
		name string
		raw  string
	}{
		{name: "duplicate field", raw: `{"name":"a","name":"b"}`},
		{name: "unknown field", raw: `{"name":"a","extra":"b"}`},
		{name: "trailing object", raw: `{"name":"a"}{"name":"b"}`},
		{name: "array instead of object", raw: `[]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := DecodeStrictJSON[payload]([]byte(tc.raw)); !errors.Is(err, ErrJSONContract) {
				t.Fatalf("DecodeStrictJSON error = %v, want ErrJSONContract", err)
			}
		})
	}
}

func TestAPIEnvelopeHostileTable(t *testing.T) {
	t.Parallel()
	value := "ok"
	errBody := APIErrorBody{Code: APICodeForbidden, Message: "no"}
	for _, tc := range []struct {
		wantError error
		envelope  APIEnvelope[string]
		name      string
		success   bool
	}{
		{
			name: "success envelope accepts data only",
			envelope: APIEnvelope[string]{
				RequestID: NewAPIRequestID("req-1"),
				Data:      &value,
			},
			success: true,
		},
		{
			name: "success rejects error arm",
			envelope: APIEnvelope[string]{
				RequestID: NewAPIRequestID("req-1"),
				Data:      &value,
				Error:     &errBody,
			},
			success:   true,
			wantError: ErrFoundationContract,
		},
		{
			name: "failure envelope accepts error only",
			envelope: APIEnvelope[string]{
				RequestID: NewAPIRequestID("req-1"),
				Error:     &errBody,
			},
		},
		{
			name: "failure rejects data arm",
			envelope: APIEnvelope[string]{
				RequestID: NewAPIRequestID("req-1"),
				Data:      &value,
				Error:     &errBody,
			},
			wantError: ErrFoundationContract,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.envelope.ValidateFailure()
			if tc.success {
				err = tc.envelope.ValidateSuccess()
			}
			if tc.wantError == nil && err != nil {
				t.Fatal(err)
			}
			if tc.wantError != nil && !errors.Is(err, tc.wantError) {
				t.Fatalf("envelope validation error = %v, want %v", err, tc.wantError)
			}
		})
	}
}

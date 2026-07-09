package core

import (
	"crypto/ed25519"
	"crypto/sha256"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"encoding/json"
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
		{raw: `-0`},
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
	epoch := UnixNanoTimeFromInt64(0)
	if err := epoch.Validate(); err != nil {
		t.Fatalf("UnixNanoTime epoch Validate() = %v", err)
	}
	if got := epoch.Add(time.Nanosecond).UnixNano(); got != 1 {
		t.Fatalf("UnixNanoTime epoch Add(1ns) = %d, want 1", got)
	}
	if _, err := json.Marshal(UnixNanoTime{}); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("UnixNanoTime unset MarshalJSON error = %v, want ErrFoundationContract", err)
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

func TestEd25519SignatureHexHostileTable(t *testing.T) {
	t.Parallel()

	constructed, err := NewEd25519SignatureHex(make([]byte, ed25519.SignatureSize))
	if err != nil {
		t.Fatalf("NewEd25519SignatureHex(valid) error = %v", err)
	}
	if err := constructed.Validate(); err != nil {
		t.Fatalf("NewEd25519SignatureHex(valid).Validate() error = %v", err)
	}
	raw, err := constructed.Bytes()
	if err != nil {
		t.Fatalf("Ed25519SignatureHex.Bytes() error = %v", err)
	}
	if len(raw) != ed25519.SignatureSize {
		t.Fatalf("Ed25519SignatureHex.Bytes() len = %d, want %d", len(raw), ed25519.SignatureSize)
	}
	if _, err := NewEd25519SignatureHex(make([]byte, ed25519.SignatureSize-1)); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("NewEd25519SignatureHex(short) error = %v, want ErrFoundationContract", err)
	}

	for _, tc := range []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "lowercase fixed hex accepted", value: strings.Repeat("a", ed25519.SignatureSize*2)},
		{name: "uppercase rejected", value: strings.Repeat("A", ed25519.SignatureSize*2), wantErr: true},
		{name: "short rejected", value: strings.Repeat("a", ed25519.SignatureSize*2-1), wantErr: true},
		{name: "long rejected", value: strings.Repeat("a", ed25519.SignatureSize*2+1), wantErr: true},
		{name: "non hex rejected", value: strings.Repeat("g", ed25519.SignatureSize*2), wantErr: true},
		{name: "empty rejected", value: "", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseEd25519SignatureHex(tc.value)
			if tc.wantErr {
				if !errors.Is(err, ErrFoundationContract) {
					t.Fatalf("ParseEd25519SignatureHex(%q) error = %v, want ErrFoundationContract", tc.value, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseEd25519SignatureHex(%q) error = %v", tc.value, err)
			}
		})
	}

	for _, rawJSON := range []string{`1`, `true`, `null`, `{}`, `[]`, `"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"`} {
		t.Run("json "+rawJSON, func(t *testing.T) {
			t.Parallel()
			value := constructed
			if err := value.UnmarshalJSON([]byte(rawJSON)); !errors.Is(err, ErrFoundationContract) {
				t.Fatalf("Ed25519SignatureHex.UnmarshalJSON(%s) error = %v, want ErrFoundationContract", rawJSON, err)
			}
			if value != constructed {
				t.Fatalf("failed unmarshal mutated signature = %q, want %q", value.String(), constructed.String())
			}
		})
	}
}

func TestFixedHexJSONRejectsNullAndEmptyHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		raw  string
		name string
		kind fixedHexJSONKind
	}{
		{name: "sha256 rejects null", kind: fixedHexJSONKindSHA256, raw: `null`},
		{name: "sha256 rejects empty string", kind: fixedHexJSONKindSHA256, raw: `""`},
		{name: "blake3 rejects null", kind: fixedHexJSONKindBLAKE3, raw: `null`},
		{name: "blake3 rejects empty string", kind: fixedHexJSONKindBLAKE3, raw: `""`},
		{name: "ed25519 public key rejects null", kind: fixedHexJSONKindEd25519PublicKey, raw: `null`},
		{name: "ed25519 public key rejects empty string", kind: fixedHexJSONKindEd25519PublicKey, raw: `""`},
		{name: "ed25519 signature rejects null", kind: fixedHexJSONKindEd25519Signature, raw: `null`},
		{name: "ed25519 signature rejects empty string", kind: fixedHexJSONKindEd25519Signature, raw: `""`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := unmarshalFixedHexForTest(tc.kind, []byte(tc.raw)); !errors.Is(err, ErrFoundationContract) {
				t.Fatalf("%s error = %v, want ErrFoundationContract", tc.name, err)
			}
		})
	}
}

func TestAppendJSONFieldRejectsInvalidFieldNameHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		fieldName string
	}{
		{name: "control byte rejected", fieldName: "bad\x01field"},
		{name: "newline rejected", fieldName: "bad\nfield"},
		{name: "edge space rejected", fieldName: " field"},
		{name: "empty rejected", fieldName: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := AppendJSONField(nil, tc.fieldName, "value"); !errors.Is(err, ErrFoundationContract) {
				t.Fatalf("AppendJSONField(%q) error = %v, want ErrFoundationContract", tc.fieldName, err)
			}
		})
	}
}

type fixedHexJSONKind uint8

const (
	fixedHexJSONKindSHA256 fixedHexJSONKind = iota + 1
	fixedHexJSONKindBLAKE3
	fixedHexJSONKindEd25519PublicKey
	fixedHexJSONKindEd25519Signature
)

func unmarshalFixedHexForTest(kind fixedHexJSONKind, data []byte) error {
	switch kind {
	case fixedHexJSONKindSHA256:
		value := NewSHA256Hex(sha256.Sum256([]byte("foundation")))
		return value.UnmarshalJSON(data)
	case fixedHexJSONKindBLAKE3:
		value, err := ParseBLAKE3Hex(strings.Repeat("b", BLAKE3DigestBytes*2))
		if err != nil {
			return err
		}
		return value.UnmarshalJSON(data)
	case fixedHexJSONKindEd25519PublicKey:
		public, err := NewEd25519PublicKeyHex(make([]byte, ed25519.PublicKeySize))
		if err != nil {
			return err
		}
		return public.UnmarshalJSON(data)
	case fixedHexJSONKindEd25519Signature:
		value, err := NewEd25519SignatureHex(make([]byte, ed25519.SignatureSize))
		if err != nil {
			return err
		}
		return value.UnmarshalJSON(data)
	default:
		return ErrFoundationContract
	}
}

func TestBuildCommitHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "git sha1 accepted", value: strings.Repeat("a", 40)},
		{name: "git sha256 accepted", value: strings.Repeat("b", 64)},
		{name: "short rejected", value: strings.Repeat("a", 39), wantErr: true},
		{name: "upper rejected", value: strings.Repeat("A", 40), wantErr: true},
		{name: "non hex rejected", value: strings.Repeat("g", 40), wantErr: true},
		{name: "empty rejected", value: "", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseBuildCommit(tc.value)
			if tc.wantErr {
				if !errors.Is(err, ErrFoundationContract) {
					t.Fatalf("ParseBuildCommit() error = %v, want ErrFoundationContract", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestStorageTransferEnumsHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		run  func(*testing.T)
		name string
	}{
		{name: "storage provider gcs wire token", run: func(t *testing.T) {
			requireEnumJSON(t, StorageProviderGCS, storageProviderTokenGCS)
		}},
		{name: "storage provider unknown rejects", run: func(t *testing.T) {
			if _, err := StorageProviderUnknown.MarshalJSON(); !errors.Is(err, ErrFoundationContract) {
				t.Fatalf("StorageProviderUnknown.MarshalJSON() error = %v, want ErrFoundationContract", err)
			}
		}},
		{name: "upload method signed put wire token", run: func(t *testing.T) {
			requireEnumJSON(t, UploadMethodSignedPUT, uploadMethodTokenSignedPUT)
		}},
		{name: "upload method unknown rejects", run: func(t *testing.T) {
			if _, err := UploadMethodUnknown.MarshalJSON(); !errors.Is(err, ErrFoundationContract) {
				t.Fatalf("UploadMethodUnknown.MarshalJSON() error = %v, want ErrFoundationContract", err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}

func requireEnumJSON[T json.Marshaler](t *testing.T, value T, token string) {
	t.Helper()
	data, err := value.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	want := `"` + token + `"`
	if string(data) != want {
		t.Fatalf("MarshalJSON() = %s, want %s", data, want)
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
			_, err := ParseAPIRequestID("evil\r\nX-Injected: 1")
			return err
		}},
		{name: "api request id rejects leading space", run: func() error {
			_, err := ParseAPIRequestID(" req-1")
			return err
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
		{name: "lease id rejects empty string", run: func() error {
			var id LeaseID
			return id.UnmarshalJSON([]byte(`""`))
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

func TestLeaseIDNullRoundTripHostileTable(t *testing.T) {
	t.Parallel()
	var id LeaseID
	if err := id.UnmarshalJSON([]byte(`null`)); err != nil {
		t.Fatalf("LeaseID.UnmarshalJSON(null) error = %v", err)
	}
	data, err := id.MarshalJSON()
	if err != nil {
		t.Fatalf("LeaseID.MarshalJSON(zero) error = %v", err)
	}
	if string(data) != `null` {
		t.Fatalf("LeaseID zero JSON = %s, want null", data)
	}
}

func TestAPIRequestIDConstructorSanitizesHostileHeaderTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input string
		want  string
		name  string
	}{
		{name: "blank header becomes missing", input: "", want: APIRequestIDMissing},
		{name: "edge spaces trimmed", input: " req-1 ", want: "req-1"},
		{name: "crlf injection loses controls", input: "evil\r\nX-Injected: 1", want: "evilX-Injected: 1"},
		{name: "only controls becomes missing", input: "\r\n\t", want: APIRequestIDMissing},
		{name: "oversized header truncates to contract", input: strings.Repeat("a", APIRequestIDMaxRunes+1), want: strings.Repeat("a", APIRequestIDMaxRunes)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			requestID := NewAPIRequestID(tc.input)
			if requestID.String() != tc.want {
				t.Fatalf("NewAPIRequestID() = %q, want %q", requestID.String(), tc.want)
			}
			envelope := APIEnvelope[string]{
				RequestID: requestID,
				Error:     &APIErrorBody{Code: APICodeInvalidInput, Message: "bad request"},
			}
			if _, err := json.Marshal(envelope); err != nil {
				t.Fatalf("json.Marshal(APIEnvelope) error = %v", err)
			}
		})
	}
}

func TestAPIRequestIDUnmarshalRejectsWithoutMutationHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		raw  string
		name string
	}{
		{name: "crlf rejected", raw: `"evil\r\nX-Injected: 1"`},
		{name: "edge space rejected", raw: `" req-1"`},
		{name: "number rejected", raw: `1`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			requestID := NewAPIRequestID("original")
			if err := json.Unmarshal([]byte(tc.raw), &requestID); !errors.Is(err, ErrFoundationContract) {
				t.Fatalf("APIRequestID.UnmarshalJSON() error = %v, want ErrFoundationContract", err)
			}
			if requestID.String() != "original" {
				t.Fatalf("APIRequestID after failed unmarshal = %q, want original", requestID.String())
			}
		})
	}
}

func TestUniqueStringSetZeroValueHostileTable(t *testing.T) {
	t.Parallel()
	var set UniqueStringSet
	if err := set.Add("first"); err != nil {
		t.Fatalf("UniqueStringSet zero Add(first) error = %v", err)
	}
	if err := set.Add("first"); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("UniqueStringSet duplicate error = %v, want ErrFoundationContract", err)
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
		{Base: time.Second, Max: BackoffMaxDuration + time.Nanosecond, MaxAttempts: 1},
	} {
		if err := policy.Validate(); !errors.Is(err, ErrFoundationContract) {
			t.Fatalf("BackoffPolicy.Validate error = %v, want ErrFoundationContract", err)
		}
		if _, err := policy.Delay(0, 1); !errors.Is(err, ErrFoundationContract) {
			t.Fatalf("BackoffPolicy.Delay error = %v, want ErrFoundationContract", err)
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

func TestCryptoHexNonStringJSONWrapsFoundationContract(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		run  func() error
		name string
	}{
		{name: "sha256 number", run: func() error {
			var value SHA256Hex
			return value.UnmarshalJSON([]byte(`1`))
		}},
		{name: "blake3 object", run: func() error {
			var value BLAKE3Hex
			return value.UnmarshalJSON([]byte(`{}`))
		}},
		{name: "ed25519 bool", run: func() error {
			var value Ed25519PublicKeyHex
			return value.UnmarshalJSON([]byte(`true`))
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
		{name: "top-level null rejected before zero struct fabrication", raw: `null`},
		{name: "whitespace padded top-level null rejected before zero struct fabrication", raw: "\n\t null \r\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := DecodeStrictJSON[payload]([]byte(tc.raw)); !errors.Is(err, ErrJSONContract) {
				t.Fatalf("DecodeStrictJSON error = %v, want ErrJSONContract", err)
			}
		})
	}
}

func TestDecodeStrictJSONValidLineTable(t *testing.T) {
	t.Parallel()
	type payload struct {
		Name string `json:"name"`
		At   int64  `json:"at"`
		OK   bool   `json:"ok"`
	}
	for _, tc := range []struct {
		name string
		raw  string
		want payload
	}{
		{
			name: "compact object without newline",
			raw:  `{"name":"ok","at":1783484211688677000,"ok":true}`,
			want: payload{Name: "ok", At: 1783484211688677000, OK: true},
		},
		{
			name: "escaped angle brackets from operation log line",
			raw:  `{"name":"Ase Deliri \u003cdeliri.ase@gmail.com\u003e","at":1783484211688677000,"ok":true}`,
			want: payload{Name: "Ase Deliri <deliri.ase@gmail.com>", At: 1783484211688677000, OK: true},
		},
		{
			name: "valid object with trailing json whitespace",
			raw:  "{\"name\":\"ok\",\"at\":1783484211688677000,\"ok\":true}\n\t ",
			want: payload{Name: "ok", At: 1783484211688677000, OK: true},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := DecodeStrictJSON[payload]([]byte(tc.raw))
			if err != nil {
				t.Fatalf("DecodeStrictJSON error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("DecodeStrictJSON = %+v, want %+v", got, tc.want)
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
		{
			name: "failure rejects whitespace message",
			envelope: APIEnvelope[string]{
				RequestID: NewAPIRequestID("req-1"),
				Error:     &APIErrorBody{Code: APICodeForbidden, Message: " \t "},
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

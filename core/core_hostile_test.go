package core

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"
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

func TestAddUnixNanoDurationRejectsRangeEscape(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		base     UnixNanoTime
		duration time.Duration
		want     int64
		wantErr  bool
	}{
		{name: "ordinary addition", base: UnixNanoTimeFromInt64(5), duration: 3, want: 8},
		{name: "positive overflow", base: UnixNanoTimeFromInt64(math.MaxInt64), duration: 1, wantErr: true},
		{name: "negative range escape", base: UnixNanoTimeFromInt64(0), duration: -1, wantErr: true},
		{name: "minimum duration range escape", base: UnixNanoTimeFromInt64(0), duration: time.Duration(math.MinInt64), wantErr: true},
		{name: "unset base", duration: 1, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := AddUnixNanoDuration(tc.base, tc.duration)
			if tc.wantErr {
				if !errors.Is(err, ErrFoundationContract) {
					t.Fatalf("AddUnixNanoDuration() error = %v, want ErrFoundationContract", err)
				}
				return
			}
			if err != nil || got.UnixNano() != tc.want {
				t.Fatalf("AddUnixNanoDuration() = (%d, %v), want (%d, nil)", got.UnixNano(), err, tc.want)
			}
		})
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

func TestAppendJSONFieldNameHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		fieldName string
		wantError bool
	}{
		{name: "single lowercase letter accepted", fieldName: "a"},
		{name: "lowercase word accepted", fieldName: "message"},
		{name: "snake case accepted", fieldName: "device_fingerprint"},
		{name: "digit suffix accepted", fieldName: "sha256"},
		{name: "maximum length lowercase accepted", fieldName: strings.Repeat("a", JSONFieldNameMaxRunes)},
		{name: "empty rejected", fieldName: "", wantError: true},
		{name: "one over maximum rejected", fieldName: strings.Repeat("a", JSONFieldNameMaxRunes+1), wantError: true},
		{name: "leading uppercase rejected", fieldName: "Message", wantError: true},
		{name: "mixed uppercase rejected", fieldName: "MeSsAgE", wantError: true},
		{name: "all uppercase rejected", fieldName: "MESSAGE", wantError: true},
		{name: "digit prefix rejected", fieldName: "2026_schema", wantError: true},
		{name: "leading separator rejected", fieldName: "_schema", wantError: true},
		{name: "trailing separator rejected", fieldName: "schema_", wantError: true},
		{name: "repeated separator rejected", fieldName: "request__id", wantError: true},
		{name: "hyphen rejected", fieldName: "request-id", wantError: true},
		{name: "period rejected", fieldName: "request.id", wantError: true},
		{name: "slash rejected", fieldName: "request/id", wantError: true},
		{name: "backslash rejected", fieldName: `request\id`, wantError: true},
		{name: "leading space rejected", fieldName: " field", wantError: true},
		{name: "trailing space rejected", fieldName: "field ", wantError: true},
		{name: "newline rejected", fieldName: "bad\nfield", wantError: true},
		{name: "control byte rejected", fieldName: "bad\x01field", wantError: true},
		{name: "unicode case fold rejected", fieldName: "meſſage", wantError: true},
		{name: "non-ascii letter rejected", fieldName: "mésage", wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := AppendJSONField(nil, tc.fieldName, "value")
			if tc.wantError && !errors.Is(err, ErrJSONContract) {
				t.Fatalf("AppendJSONField(%q) error = %v, want ErrJSONContract", tc.fieldName, err)
			}
			if !tc.wantError && err != nil {
				t.Fatalf("AppendJSONField(%q) unexpected error = %v", tc.fieldName, err)
			}
		})
	}
}

func TestCrossPackageJSONFieldOwnershipTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		got  string
		want string
	}{
		{name: "device fingerprint", got: JSONFieldDeviceFingerprint, want: "device_fingerprint"},
		{name: "object count", got: JSONFieldObjectCount, want: "object_count"},
		{name: "product", got: JSONFieldProduct, want: "product"},
		{name: "total bytes", got: JSONFieldTotalBytes, want: "total_bytes"},
		{name: "writer key id", got: JSONFieldWriterKeyID, want: "writer_key_id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.got != tc.want {
				t.Fatalf("field = %q, want %q", tc.got, tc.want)
			}
			if err := ValidateJSONFieldName(tc.got); err != nil {
				t.Fatalf("ValidateJSONFieldName(%q) error = %v", tc.got, err)
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
			envelope := APIEnvelope[apiEnvelopeHostileData]{
				RequestID: requestID,
				Error:     &APIErrorBody{Code: APICodeInvalidInput, Message: "bad request"},
			}
			if _, err := json.Marshal(envelope); err != nil {
				t.Fatalf("json.Marshal(APIEnvelope) error = %v", err)
			}
		})
	}
}

type apiEnvelopeHostileData struct {
	Value string
}

func (d apiEnvelopeHostileData) Validate() error {
	return ValidateOpaqueToken(d.Value, OpaqueTokenDefaultMaxRunes)
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

func TestCollectionCardinalityHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		contract CollectionCardinality
		wantErr  bool
	}{
		{name: "bounded length", contract: CollectionCardinality{Length: 1, Minimum: 1, Maximum: 2}},
		{name: "declared count matches", contract: CollectionCardinality{Length: 2, DeclaredCount: 2, Maximum: 2, RequireDeclared: true}},
		{name: "negative length", contract: CollectionCardinality{Length: -1, Maximum: 1}, wantErr: true},
		{name: "zero maximum", contract: CollectionCardinality{Length: 0}, wantErr: true},
		{name: "above maximum", contract: CollectionCardinality{Length: 2, Maximum: 1}, wantErr: true},
		{name: "below minimum", contract: CollectionCardinality{Length: 0, Minimum: 1, Maximum: 1}, wantErr: true},
		{name: "declared mismatch", contract: CollectionCardinality{Length: 1, DeclaredCount: 2, Maximum: 2, RequireDeclared: true}, wantErr: true},
		{name: "undeclared ghost count", contract: CollectionCardinality{Length: 1, DeclaredCount: 1, Maximum: 2}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.contract.Validate()
			if tc.wantErr {
				if !errors.Is(err, ErrFoundationContract) {
					t.Fatalf("CollectionCardinality.Validate() error = %v, want ErrFoundationContract", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("CollectionCardinality.Validate() error = %v", err)
			}
		})
	}
}

type artifactSetHostileItem struct {
	name string
	size ByteCount
}

func (i artifactSetHostileItem) Validate() error            { return i.size.Validate() }
func (i artifactSetHostileItem) ArtifactSetName() string    { return i.name }
func (i artifactSetHostileItem) ArtifactSetSize() ByteCount { return i.size }

func TestArtifactSetRejectsUnboundedBytes(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		item  uint64
		total uint64
	}{
		{name: "item above maximum", item: ArtifactMaximumBytes + 1, total: ArtifactMaximumBytes + 1},
		{name: "total above maximum", item: ArtifactMaximumBytes, total: ArtifactSetMaximumBytes + 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			set := ArtifactSet[artifactSetHostileItem]{
				Items:      []artifactSetHostileItem{{name: tc.name, size: NewByteCount(tc.item)}},
				TotalBytes: NewByteCount(tc.total),
				Count:      1,
			}
			if err := ValidateArtifactSet(set); !errors.Is(err, ErrFoundationContract) {
				t.Fatalf("ValidateArtifactSet() error = %v, want ErrFoundationContract", err)
			}
		})
	}
}

func TestArtifactSetBuilderOwnsCanonicalOrder(t *testing.T) {
	t.Parallel()
	input := []artifactSetHostileItem{
		{name: "zeta", size: NewByteCount(2)},
		{name: "alpha", size: NewByteCount(1)},
	}
	set, err := BuildArtifactSet(input)
	if err != nil {
		t.Fatalf("BuildArtifactSet() error = %v", err)
	}
	if set.Items[0].name != "alpha" || set.Items[1].name != "zeta" {
		t.Fatalf("BuildArtifactSet() order = %q, %q", set.Items[0].name, set.Items[1].name)
	}
	input[0].name = "mutated"
	if set.Items[1].name != "zeta" {
		t.Fatalf("BuildArtifactSet() retained caller alias")
	}
	set.Items[0], set.Items[1] = set.Items[1], set.Items[0]
	if err := ValidateArtifactSet(set); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("ValidateArtifactSet(unsorted) error = %v, want ErrFoundationContract", err)
	}
}

func TestDeriveCollectionCountHostileTable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		length  int
		minimum uint32
		maximum uint32
		want    uint32
		wantErr bool
	}{
		{name: "single", length: 1, minimum: 1, maximum: 1, want: 1},
		{name: "middle", length: 5, minimum: 1, maximum: 10, want: 5},
		{name: "at maximum", length: 10, minimum: 1, maximum: 10, want: 10},
		{name: "zero allowed", length: 0, minimum: 0, maximum: 1, want: 0},
		{name: "negative length", length: -1, minimum: 0, maximum: 1, wantErr: true},
		{name: "zero refused", length: 0, minimum: 1, maximum: 1, wantErr: true},
		{name: "below minimum", length: 1, minimum: 2, maximum: 3, wantErr: true},
		{name: "above maximum", length: 4, minimum: 1, maximum: 3, wantErr: true},
		{name: "zero maximum", length: 0, minimum: 0, maximum: 0, wantErr: true},
		{name: "inverted bounds", length: 2, minimum: 3, maximum: 2, wantErr: true},
		{name: "default maximum", length: int(CollectionMaximumDefault), minimum: 1, maximum: CollectionMaximumDefault, want: CollectionMaximumDefault},
		{name: "past default maximum", length: int(CollectionMaximumDefault) + 1, minimum: 1, maximum: CollectionMaximumDefault, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := DeriveCollectionCount(tc.length, tc.minimum, tc.maximum)
			if tc.wantErr {
				if !errors.Is(err, ErrFoundationContract) {
					t.Fatalf("DeriveCollectionCount() error = %v, want %v", err, ErrFoundationContract)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("DeriveCollectionCount() = %d, %v; want %d, nil", got, err, tc.want)
			}
		})
	}
}

func TestDecodeStrictJSONRejectsUnboundedObjectFields(t *testing.T) {
	t.Parallel()
	var body strings.Builder
	body.WriteString("{")
	for index := range StrictJSONMaxObjectFields + 1 {
		if index > 0 {
			body.WriteString(",")
		}
		body.WriteString("\"field_" + strconv.Itoa(index) + "\":0")
	}
	body.WriteString("}")
	if _, err := DecodeStrictJSON[strictJSONHostilePayload]([]byte(body.String())); !errors.Is(err, ErrJSONContract) {
		t.Fatalf("DecodeStrictJSON oversized object error = %v, want ErrJSONContract", err)
	}
}

func TestDecodeStrictJSONRejectsUnboundedArrayItems(t *testing.T) {
	t.Parallel()
	var body strings.Builder
	body.WriteString("[")
	for index := range CollectionMaximumDefault + 1 {
		if index > 0 {
			body.WriteString(",")
		}
		body.WriteString("0")
	}
	body.WriteString("]")
	if _, err := DecodeStrictJSON[strictJSONHostilePayload]([]byte(body.String())); !errors.Is(err, ErrJSONContract) {
		t.Fatalf("DecodeStrictJSON oversized array error = %v, want ErrJSONContract", err)
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
		{name: "header name rejects oversized value", run: func() error { return ValidateHTTPHeaderName(strings.Repeat("a", HTTPHeaderNameMaxRunes+1)) }},
		{name: "header name rejects invalid utf8", run: func() error { return ValidateHTTPHeaderName(string([]byte{0xff})) }},
		{name: "header value rejects nul", run: func() error { return ValidateHTTPHeaderValue("a\x00b") }},
		{name: "header value rejects newline", run: func() error { return ValidateHTTPHeaderValue("a\nb") }},
		{name: "header value rejects oversized value", run: func() error { return ValidateHTTPHeaderValue(strings.Repeat("a", HTTPHeaderValueMaxRunes+1)) }},
		{name: "header value rejects invalid utf8", run: func() error { return ValidateHTTPHeaderValue(string([]byte{0xff})) }},
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
	cases := []struct {
		wantValidateErr error
		wantDelayErr    error
		name            string
		policy          BackoffPolicy
		attempt         uint64
		fraction        float64
		wantDelay       NanosecondsDuration
	}{
		{name: "one nanosecond floor accepts zero jitter", policy: deliveryBackoffPolicy(time.Nanosecond, time.Nanosecond, 1), fraction: 0, wantDelay: NewNanosecondsDuration(0)},
		{name: "one nanosecond floor accepts full jitter", policy: deliveryBackoffPolicy(time.Nanosecond, time.Nanosecond, 1), fraction: 1, wantDelay: NewNanosecondsDuration(time.Nanosecond)},
		{name: "equal base and maximum accept first attempt", policy: deliveryBackoffPolicy(time.Second, time.Second, 1), fraction: 1, wantDelay: NewNanosecondsDuration(time.Second)},
		{name: "zero attempt selects base", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), fraction: 1, wantDelay: NewNanosecondsDuration(time.Second)},
		{name: "middle attempt doubles base", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), attempt: 1, fraction: 1, wantDelay: NewNanosecondsDuration(2 * time.Second)},
		{name: "last permitted attempt reaches maximum", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), attempt: 2, fraction: 1, wantDelay: NewNanosecondsDuration(4 * time.Second)},
		{name: "non power of two window clamps to maximum", policy: deliveryBackoffPolicy(3*time.Second, 5*time.Second, 2), attempt: 1, fraction: 1, wantDelay: NewNanosecondsDuration(5 * time.Second)},
		{name: "deep permitted attempt remains saturated", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 12), attempt: 10, fraction: 1, wantDelay: NewNanosecondsDuration(4 * time.Second)},
		{name: "quarter jitter scales within ceiling", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), fraction: 0.25, wantDelay: NewNanosecondsDuration(250 * time.Millisecond)},
		{name: "half jitter scales within ceiling", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), fraction: 0.5, wantDelay: NewNanosecondsDuration(500 * time.Millisecond)},
		{name: "three quarter jitter scales within ceiling", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), fraction: 0.75, wantDelay: NewNanosecondsDuration(750 * time.Millisecond)},
		{name: "smallest positive jitter rounds down safely", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, 3), fraction: math.SmallestNonzeroFloat64, wantDelay: NewNanosecondsDuration(0)},
		{name: "maximum legal window accepts exact boundary", policy: deliveryBackoffPolicy(BackoffMaxDuration, BackoffMaxDuration, 1), fraction: 1, wantDelay: NewNanosecondsDuration(BackoffMaxDuration)},
		{name: "half maximum base doubles to exact maximum", policy: deliveryBackoffPolicy(BackoffMaxDuration/2, BackoffMaxDuration, 2), attempt: 1, fraction: 1, wantDelay: NewNanosecondsDuration(BackoffMaxDuration)},
		{name: "one below maximum duration remains legal", policy: deliveryBackoffPolicy(BackoffMaxDuration-time.Nanosecond, BackoffMaxDuration, 1), fraction: 1, wantDelay: NewNanosecondsDuration(BackoffMaxDuration - time.Nanosecond)},
		{name: "maximum attempt domain accepts one below", policy: deliveryBackoffPolicy(time.Second, 4*time.Second, math.MaxUint64), attempt: math.MaxUint64 - 1, fraction: 1, wantDelay: NewNanosecondsDuration(4 * time.Second)},
		{name: "huge attempt stays constant when base equals maximum", policy: deliveryBackoffPolicy(time.Second, time.Second, math.MaxUint64), attempt: math.MaxUint64 - 2, fraction: 1, wantDelay: NewNanosecondsDuration(time.Second)},
		{name: "two nanosecond base preserves half jitter", policy: deliveryBackoffPolicy(2*time.Nanosecond, 2*time.Nanosecond, 1), fraction: 0.5, wantDelay: NewNanosecondsDuration(time.Nanosecond)},
		{name: "one third jitter truncates inside three nanoseconds", policy: deliveryBackoffPolicy(3*time.Nanosecond, 3*time.Nanosecond, 1), fraction: 1.0 / 3.0, wantDelay: NewNanosecondsDuration(time.Nanosecond)},
		{name: "odd maximum clamps doubled base", policy: deliveryBackoffPolicy(2*time.Second, 3*time.Second, 2), attempt: 1, fraction: 1, wantDelay: NewNanosecondsDuration(3 * time.Second)},
		{name: "zero value policy rejects validation and delay", policy: BackoffPolicy{}, fraction: 1, wantValidateErr: ErrFoundationContract, wantDelayErr: ErrFoundationContract},
		{name: "zero attempts rejects otherwise legal window", policy: deliveryBackoffPolicy(time.Second, time.Minute, 0), fraction: 1, wantValidateErr: ErrFoundationContract, wantDelayErr: ErrFoundationContract},
		{name: "zero base rejects exact lower boundary", policy: deliveryBackoffPolicy(0, time.Second, 1), fraction: 1, wantValidateErr: ErrFoundationContract, wantDelayErr: ErrFoundationContract},
		{name: "negative base rejects one below lower boundary", policy: BackoffPolicy{Base: NanosecondsDurationFromInt64(-1), Max: NewNanosecondsDuration(time.Second), MaxAttempts: 1}, fraction: 1, wantValidateErr: ErrFoundationContract, wantDelayErr: ErrFoundationContract},
		{name: "zero maximum rejects required window", policy: deliveryBackoffPolicy(time.Second, 0, 1), fraction: 1, wantValidateErr: ErrFoundationContract, wantDelayErr: ErrFoundationContract},
		{name: "negative maximum rejects hostile duration", policy: BackoffPolicy{Base: NewNanosecondsDuration(time.Nanosecond), Max: NanosecondsDurationFromInt64(-1), MaxAttempts: 1}, fraction: 1, wantValidateErr: ErrFoundationContract, wantDelayErr: ErrFoundationContract},
		{name: "maximum one nanosecond below base rejects inversion", policy: deliveryBackoffPolicy(time.Second, time.Second-time.Nanosecond, 1), fraction: 1, wantValidateErr: ErrFoundationContract, wantDelayErr: ErrFoundationContract},
		{name: "maximum one nanosecond above global boundary rejects", policy: deliveryBackoffPolicy(time.Second, BackoffMaxDuration+time.Nanosecond, 1), fraction: 1, wantValidateErr: ErrFoundationContract, wantDelayErr: ErrFoundationContract},
		{name: "base one nanosecond above global boundary rejects", policy: deliveryBackoffPolicy(BackoffMaxDuration+time.Nanosecond, BackoffMaxDuration+time.Nanosecond, 1), fraction: 1, wantValidateErr: ErrFoundationContract, wantDelayErr: ErrFoundationContract},
		{name: "maximum duration value rejects global upper boundary", policy: deliveryBackoffPolicy(time.Second, time.Duration(math.MaxInt64), 1), fraction: 1, wantValidateErr: ErrFoundationContract, wantDelayErr: ErrFoundationContract},
		{name: "attempt exact limit rejects", policy: deliveryBackoffPolicy(time.Second, time.Minute, 3), attempt: 3, fraction: 1, wantDelayErr: ErrFoundationContract},
		{name: "attempt one above limit rejects", policy: deliveryBackoffPolicy(time.Second, time.Minute, 3), attempt: 4, fraction: 1, wantDelayErr: ErrFoundationContract},
		{name: "attempt maximum rejects finite budget", policy: deliveryBackoffPolicy(time.Second, time.Minute, 3), attempt: math.MaxUint64, fraction: 1, wantDelayErr: ErrFoundationContract},
		{name: "smallest negative jitter rejects one below boundary", policy: deliveryBackoffPolicy(time.Second, time.Minute, 3), fraction: -math.SmallestNonzeroFloat64, wantDelayErr: ErrFoundationContract},
		{name: "negative one jitter rejects", policy: deliveryBackoffPolicy(time.Second, time.Minute, 3), fraction: -1, wantDelayErr: ErrFoundationContract},
		{name: "jitter one ulp above boundary rejects", policy: deliveryBackoffPolicy(time.Second, time.Minute, 3), fraction: math.Nextafter(1, 2), wantDelayErr: ErrFoundationContract},
		{name: "jitter two rejects far above boundary", policy: deliveryBackoffPolicy(time.Second, time.Minute, 3), fraction: 2, wantDelayErr: ErrFoundationContract},
		{name: "negative infinity jitter rejects", policy: deliveryBackoffPolicy(time.Second, time.Minute, 3), fraction: math.Inf(-1), wantDelayErr: ErrFoundationContract},
		{name: "positive infinity jitter rejects", policy: deliveryBackoffPolicy(time.Second, time.Minute, 3), fraction: math.Inf(1), wantDelayErr: ErrFoundationContract},
		{name: "not a number jitter rejects unordered comparison", policy: deliveryBackoffPolicy(time.Second, time.Minute, 3), fraction: math.NaN(), wantDelayErr: ErrFoundationContract},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotValidateErr := tc.policy.Validate()
			if !errors.Is(gotValidateErr, tc.wantValidateErr) {
				t.Fatalf("BackoffPolicy.Validate() error = %v, want %v", gotValidateErr, tc.wantValidateErr)
			}
			gotDelay, gotDelayErr := tc.policy.Delay(tc.attempt, tc.fraction)
			if !errors.Is(gotDelayErr, tc.wantDelayErr) {
				t.Fatalf("BackoffPolicy.Delay() error = %v, want %v", gotDelayErr, tc.wantDelayErr)
			}
			if tc.wantDelayErr == nil && gotDelay != tc.wantDelay {
				t.Fatalf("BackoffPolicy.Delay() = %v, want %v", gotDelay, tc.wantDelay)
			}
		})
	}
}

func TestHTTPRetryPolicyValidateHostileBoundaryTable(t *testing.T) {
	t.Parallel()

	baseline := DefaultHTTPRetryPolicy()
	cases := []struct {
		mutate  func(*HTTPRetryPolicy)
		wantErr error
		name    string
	}{
		{name: "canonical default is valid"},
		{name: "one nanosecond retry hint floor is valid", mutate: func(p *HTTPRetryPolicy) {
			p.MaximumRetryAfter = NewNanosecondsDuration(time.Nanosecond)
			p.RetryWaitLimit = NewNanosecondsDuration(time.Nanosecond)
		}},
		{name: "exact global retry hint ceiling is valid", mutate: func(p *HTTPRetryPolicy) { p.MaximumRetryAfter = NewNanosecondsDuration(BackoffMaxDuration) }},
		{name: "foreground wait may equal retry hint ceiling", mutate: func(p *HTTPRetryPolicy) { p.RetryWaitLimit = p.MaximumRetryAfter }},
		{name: "zero policy rejects every missing requirement", mutate: func(p *HTTPRetryPolicy) { *p = HTTPRetryPolicy{} }, wantErr: ErrExchangeContract},
		{name: "zero retry hint rejects missing ceiling", mutate: func(p *HTTPRetryPolicy) { p.MaximumRetryAfter = NanosecondsDuration{} }, wantErr: ErrExchangeContract},
		{name: "negative retry hint rejects one below floor", mutate: func(p *HTTPRetryPolicy) { p.MaximumRetryAfter = NanosecondsDurationFromInt64(-1) }, wantErr: ErrExchangeContract},
		{name: "retry hint one nanosecond above global ceiling rejects", mutate: func(p *HTTPRetryPolicy) {
			p.MaximumRetryAfter = NewNanosecondsDuration(BackoffMaxDuration + time.Nanosecond)
		}, wantErr: ErrExchangeContract},
		{name: "maximum duration retry hint rejects hostile upper end", mutate: func(p *HTTPRetryPolicy) { p.MaximumRetryAfter = NewNanosecondsDuration(time.Duration(math.MaxInt64)) }, wantErr: ErrExchangeContract},
		{name: "zero foreground wait rejects missing bound", mutate: func(p *HTTPRetryPolicy) { p.RetryWaitLimit = NanosecondsDuration{} }, wantErr: ErrExchangeContract},
		{name: "negative foreground wait rejects one below floor", mutate: func(p *HTTPRetryPolicy) { p.RetryWaitLimit = NanosecondsDurationFromInt64(-1) }, wantErr: ErrExchangeContract},
		{name: "foreground wait one above retry hint rejects inversion", mutate: func(p *HTTPRetryPolicy) {
			p.RetryWaitLimit = NewNanosecondsDuration(p.MaximumRetryAfter.Duration() + time.Nanosecond)
		}, wantErr: ErrExchangeContract},
		{name: "zero backoff rejects missing immediate retry policy", mutate: func(p *HTTPRetryPolicy) { p.Backoff = BackoffPolicy{} }, wantErr: ErrExchangeContract},
		{name: "zero backoff attempts rejects no execution budget", mutate: func(p *HTTPRetryPolicy) { p.Backoff.MaxAttempts = 0 }, wantErr: ErrExchangeContract},
		{name: "backoff above retry hint remains independently invalid", mutate: func(p *HTTPRetryPolicy) {
			p.Backoff.Max = NewNanosecondsDuration(BackoffMaxDuration + time.Nanosecond)
		}, wantErr: ErrExchangeContract},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			policy := baseline
			if tc.mutate != nil {
				tc.mutate(&policy)
			}
			gotErr := policy.Validate()
			if !errors.Is(gotErr, tc.wantErr) {
				t.Fatalf("HTTPRetryPolicy.Validate() error = %v, want %v", gotErr, tc.wantErr)
			}
		})
	}

	mutated := DefaultHTTPRetryPolicy()
	mutated.Backoff.MaxAttempts = 0
	if gotErr := DefaultHTTPRetryPolicy().Validate(); gotErr != nil {
		t.Fatalf("DefaultHTTPRetryPolicy() after caller mutation error = %v, want nil", gotErr)
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

func TestSignedUploadURLRejectsHostileInputTable(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{
		"http://storage.example/upload",
		"https://storage.example",
		"https://trusted.example@evil.example/upload",
		"https://storage.example/upload\nx",
	} {
		if _, err := ParseSignedUploadURL(raw); !errors.Is(err, ErrFoundationContract) {
			t.Fatalf("ParseSignedUploadURL(%q) error = %v, want ErrFoundationContract", raw, err)
		}
	}
}

func TestByteCountRejectsZero(t *testing.T) {
	t.Parallel()
	var count ByteCount
	if err := json.Unmarshal([]byte(`0`), &count); !errors.Is(err, ErrFoundationContract) {
		t.Fatalf("ByteCount zero error = %v, want ErrFoundationContract", err)
	}
}

type strictJSONHostilePayload struct {
	Name      string `json:"name"`
	FirstName string `json:"firstName"`
	At        int64  `json:"at"`
	OK        bool   `json:"ok"`
}

const strictJSONHostileUnexpectedField = "unexpected"

type strictJSONHostileEncoder struct {
	Value string `json:"value"`
}

func (e strictJSONHostileEncoder) Validate() error {
	if e.Value == "" {
		return ErrFoundationContract
	}
	return nil
}

func (e strictJSONHostileEncoder) MarshalJSON() ([]byte, error) {
	encoded := []byte{'{'}
	encoded, err := AppendJSONField(encoded, strictJSONHostileUnexpectedField, e.Value)
	if err != nil {
		return nil, err
	}
	return append(encoded, '}'), nil
}

func (p strictJSONHostilePayload) Validate() error {
	if p.Name == "" || p.At < 0 {
		return ErrFoundationContract
	}
	return nil
}

func TestEncodeValidatedJSONRoundTripTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantErr error
		name    string
		value   strictJSONHostilePayload
	}{
		{
			name:  "valid structure round trips",
			value: strictJSONHostilePayload{Name: "Ada", FirstName: "Lovelace", At: 1, OK: true},
		},
		{
			name:    "owner validation rejects output",
			value:   strictJSONHostilePayload{},
			wantErr: ErrJSONContract,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			encoded, err := EncodeValidatedJSON(tc.value)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("EncodeValidatedJSON() error = %v, want errors.Is(..., %v)", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("EncodeValidatedJSON() error = %v, want nil", err)
			}
			decoded, err := DecodeStrictJSON[strictJSONHostilePayload](encoded)
			if err != nil {
				t.Fatalf("DecodeStrictJSON() error = %v, want nil", err)
			}
			if decoded != tc.value {
				t.Fatalf("DecodeStrictJSON() = %+v, want %+v", decoded, tc.value)
			}
		})
	}
}

func TestEncodeValidatedJSONRejectsEncoderContractDrift(t *testing.T) {
	t.Parallel()

	if _, err := EncodeValidatedJSON(strictJSONHostileEncoder{Value: "valid-owner-state"}); !errors.Is(err, ErrJSONContract) {
		t.Fatalf("EncodeValidatedJSON(drifting encoder) error = %v, want errors.Is(..., %v)", err, ErrJSONContract)
	}
}

func TestDecodeStrictJSONHostileTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		raw  string
	}{
		{name: "duplicate field", raw: `{"name":"a","name":"b"}`},
		{name: "case-variant duplicate field", raw: `{"name":"a","Name":"b"}`},
		{name: "case-variant field", raw: `{"Name":"a","at":1,"ok":true}`},
		{name: "unknown field", raw: `{"name":"a","extra":"b"}`},
		{name: "trailing object", raw: `{"name":"a"}{"name":"b"}`},
		{name: "array instead of object", raw: `[]`},
		{name: "top-level null rejected before zero struct fabrication", raw: `null`},
		{name: "whitespace padded top-level null rejected before zero struct fabrication", raw: "\n\t null \r\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := DecodeStrictJSON[strictJSONHostilePayload]([]byte(tc.raw)); !errors.Is(err, ErrJSONContract) {
				t.Fatalf("DecodeStrictJSON error = %v, want ErrJSONContract", err)
			}
		})
	}
}

func TestDecodeStrictJSONStructureDefersDomainValidation(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"name":"","at":0,"ok":true}`)
	decoded, err := DecodeStrictJSONStructure[strictJSONHostilePayload](raw)
	if err != nil {
		t.Fatalf("DecodeStrictJSONStructure() error = %v, want nil", err)
	}
	if decoded.OK != true || decoded.Name != "" || decoded.At != 0 {
		t.Fatalf("DecodeStrictJSONStructure() = %+v, want decoded wire structure", decoded)
	}
	if _, err := DecodeStrictJSON[strictJSONHostilePayload](raw); !errors.Is(err, ErrJSONContract) {
		t.Fatalf("DecodeStrictJSON() error = %v, want errors.Is(..., %v)", err, ErrJSONContract)
	}
}

func TestDecodeStrictJSONStructureUsesCompilerDeclaredFieldNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantErr error
		name    string
		raw     string
	}{
		{name: "lower camel tag accepted", raw: `{"name":"Ada","firstName":"Ada","at":1,"ok":true}`},
		{name: "flattened case rejected", raw: `{"name":"Ada","firstname":"Ada","at":1,"ok":true}`, wantErr: ErrJSONContract},
		{name: "title case rejected", raw: `{"name":"Ada","FirstName":"Ada","at":1,"ok":true}`, wantErr: ErrJSONContract},
		{name: "upper snake invention rejected", raw: `{"name":"Ada","first_name":"Ada","at":1,"ok":true}`, wantErr: ErrJSONContract},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := DecodeStrictJSONStructure[strictJSONHostilePayload]([]byte(tc.raw))
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("DecodeStrictJSONStructure() error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("DecodeStrictJSONStructure() error = %v, want errors.Is(..., %v)", err, tc.wantErr)
			}
		})
	}
}

func TestDecodeStrictJSONStructureHostileTable(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		raw  string
	}{
		{name: "duplicate", raw: `{"name":"a","name":"b"}`},
		{name: "case folded duplicate", raw: `{"name":"a","Name":"b"}`},
		{name: "unknown", raw: `{"name":"a","extra":1}`},
		{name: "trailing", raw: `{"name":"a"}{}`},
		{name: "null", raw: `null`},
		{name: "excess depth", raw: strings.Repeat("[", StrictJSONMaxDepth+1) + strings.Repeat("]", StrictJSONMaxDepth+1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := DecodeStrictJSONStructure[strictJSONHostilePayload]([]byte(tc.raw)); !errors.Is(err, ErrJSONContract) {
				t.Fatalf("DecodeStrictJSONStructure() error = %v, want errors.Is(..., %v)", err, ErrJSONContract)
			}
		})
	}
	if _, err := DecodeStrictJSONStructure[strictJSONHostilePayload]([]byte{'{', '"', 'n', 'a', 'm', 'e', '"', ':', '"', 0xff, '"', '}'}); !errors.Is(err, ErrJSONContract) {
		t.Fatalf("DecodeStrictJSONStructure(invalid UTF-8) error = %v, want errors.Is(..., %v)", err, ErrJSONContract)
	}
}

func TestDecodeStrictJSONRejectsNonCanonicalAPIFieldCase(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		raw  string
	}{
		{name: "mixed-case message rejected", raw: `{"MeSsAgE":"smuggled","code":"forbidden"}`},
		{name: "title-case message rejected", raw: `{"Message":"smuggled","code":"forbidden"}`},
		{name: "uppercase message rejected", raw: `{"MESSAGE":"smuggled","code":"forbidden"}`},
		{name: "mixed-case code rejected", raw: `{"message":"denied","CoDe":"forbidden"}`},
		{name: "title-case tip rejected", raw: `{"message":"denied","Tip":"retry","code":"forbidden"}`},
		{name: "canonical then title-case duplicate rejected", raw: `{"message":"legit","Message":"smuggled","code":"forbidden"}`},
		{name: "title-case then canonical duplicate rejected", raw: `{"Message":"smuggled","message":"legit","code":"forbidden"}`},
		{name: "uppercase then canonical duplicate rejected", raw: `{"MESSAGE":"smuggled","message":"legit","code":"forbidden"}`},
		{name: "unicode long-s field rejected", raw: `{"meſſage":"smuggled","code":"forbidden"}`},
		{name: "non-ascii field rejected", raw: `{"méssage":"smuggled","code":"forbidden"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := DecodeStrictJSON[APIErrorBody]([]byte(tc.raw)); !errors.Is(err, ErrJSONContract) {
				t.Fatalf("DecodeStrictJSON(%s) error = %v, want ErrJSONContract", tc.raw, err)
			}
		})
	}
}

func TestDecodeStrictJSONRejectsResourceExhaustionShapes(t *testing.T) {
	t.Parallel()

	t.Run("byte cap", func(t *testing.T) {
		t.Parallel()
		body := make([]byte, StrictJSONMaxBytes+1)
		for i := range body {
			body[i] = ' '
		}
		if _, err := DecodeStrictJSON[strictJSONHostilePayload](body); !errors.Is(err, ErrJSONContract) {
			t.Fatalf("DecodeStrictJSON(over byte cap) error = %v, want ErrJSONContract", err)
		}
	})

	t.Run("depth cap", func(t *testing.T) {
		t.Parallel()
		body := strings.Repeat("[", StrictJSONMaxDepth+1) + strings.Repeat("]", StrictJSONMaxDepth+1)
		if _, err := DecodeStrictJSON[strictJSONHostilePayload]([]byte(body)); !errors.Is(err, ErrJSONContract) {
			t.Fatalf("DecodeStrictJSON(over depth cap) error = %v, want ErrJSONContract", err)
		}
	})
}

func TestSpecializedContractIdentitiesPreserveRootClassification(t *testing.T) {
	t.Parallel()

	identities := []error{ErrLicenseContract, ErrCustodyContract, ErrReleaseContract, ErrJSONContract, ErrContextContract, ErrCurrencyContract, ErrNilContext}
	for _, identity := range identities {
		if !errors.Is(identity, ErrFoundationContract) {
			t.Fatalf("errors.Is(%v, ErrFoundationContract) = false", identity)
		}
	}
	for i, identity := range identities {
		for j, sibling := range identities {
			if i != j && errors.Is(identity, sibling) {
				t.Fatalf("specialized identities alias: errors.Is(%v, %v) = true", identity, sibling)
			}
		}
	}
}

func TestDecodeStrictJSONValidLineTable(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		raw  string
		want strictJSONHostilePayload
	}{
		{
			name: "compact object without newline",
			raw:  `{"name":"ok","at":1783484211688677000,"ok":true}`,
			want: strictJSONHostilePayload{Name: "ok", At: 1783484211688677000, OK: true},
		},
		{
			name: "escaped angle brackets from operation log line",
			raw:  `{"name":"Ase Deliri \u003cdeliri.ase@gmail.com\u003e","at":1783484211688677000,"ok":true}`,
			want: strictJSONHostilePayload{Name: "Ase Deliri <deliri.ase@gmail.com>", At: 1783484211688677000, OK: true},
		},
		{
			name: "valid object with trailing json whitespace",
			raw:  "{\"name\":\"ok\",\"at\":1783484211688677000,\"ok\":true}\n\t ",
			want: strictJSONHostilePayload{Name: "ok", At: 1783484211688677000, OK: true},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := DecodeStrictJSON[strictJSONHostilePayload]([]byte(tc.raw))
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
	value := apiEnvelopeHostileData{Value: "ok"}
	invalidValue := apiEnvelopeHostileData{}
	errBody := APIErrorBody{Code: APICodeForbidden, Message: "no"}
	for _, tc := range []struct {
		wantError error
		envelope  APIEnvelope[apiEnvelopeHostileData]
		name      string
		success   bool
	}{
		{
			name: "success envelope accepts data only",
			envelope: APIEnvelope[apiEnvelopeHostileData]{
				RequestID: NewAPIRequestID("req-1"),
				Data:      &value,
			},
			success: true,
		},
		{
			name: "success rejects error arm",
			envelope: APIEnvelope[apiEnvelopeHostileData]{
				RequestID: NewAPIRequestID("req-1"),
				Data:      &value,
				Error:     &errBody,
			},
			success:   true,
			wantError: ErrFoundationContract,
		},
		{
			name: "success validates typed data",
			envelope: APIEnvelope[apiEnvelopeHostileData]{
				RequestID: NewAPIRequestID("req-1"),
				Data:      &invalidValue,
			},
			success:   true,
			wantError: ErrFoundationContract,
		},
		{
			name: "failure envelope accepts error only",
			envelope: APIEnvelope[apiEnvelopeHostileData]{
				RequestID: NewAPIRequestID("req-1"),
				Error:     &errBody,
			},
		},
		{
			name: "failure rejects data arm",
			envelope: APIEnvelope[apiEnvelopeHostileData]{
				RequestID: NewAPIRequestID("req-1"),
				Data:      &value,
				Error:     &errBody,
			},
			wantError: ErrFoundationContract,
		},
		{
			name: "failure rejects whitespace message",
			envelope: APIEnvelope[apiEnvelopeHostileData]{
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

func TestAPIErrorBodyRejectsUnboundedAndControlText(t *testing.T) {
	t.Parallel()

	for _, body := range []APIErrorBody{
		{Code: APICodeInvalidInput, Message: strings.Repeat("m", APIErrorMessageMaxRunes+1)},
		{Code: APICodeInvalidInput, Message: "line one\nline two"},
		{Code: APICodeInvalidInput, Message: "invalid input", Tip: strings.Repeat("t", APIErrorTipMaxRunes+1)},
		{Code: APICodeInvalidInput, Message: "invalid input", Tip: "retry\rlater"},
	} {
		if err := body.Validate(); !errors.Is(err, ErrFoundationContract) {
			t.Fatalf("APIErrorBody.Validate() error = %v, want ErrFoundationContract", err)
		}
	}
}

package core

import (
	"errors"
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

func TestSHA256HexHostileTable(t *testing.T) {
	t.Parallel()
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
